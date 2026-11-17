// Package specfix normalizes the vendored Strombauer OpenAPI document so that
// it can be consumed by oapi-codegen.
//
// The published document uses two constructs that are valid JSON Schema but
// not valid OpenAPI 3.0, and that oapi-codegen rejects:
//
//   - $ref values that point below a component schema (for example
//     #/components/schemas/Product/properties/type). Those targets are hoisted
//     into top-level components and the references are rewritten.
//   - An empty tuple-style "items": [] array. It is replaced with an empty
//     schema object, which means "any value".
package specfix

import (
	"encoding/json"
	"fmt"
	"strings"
)

// hoist describes a deep $ref in the published document and the top-level
// component name its target is moved to.
type hoist struct {
	ref  string
	name string
}

var hoists = []hoist{
	{
		ref:  "#/components/schemas/OrderLineItem/properties/type",
		name: "LineItemType",
	},
	{
		ref:  "#/components/schemas/Product/properties/type",
		name: "ProductType",
	},
	{
		ref:  "#/components/schemas/StrombauerProduct/properties/productType",
		name: "StrombauerProductType",
	},
	{
		ref:  "#/components/schemas/StrombauerProduct/properties/bundleProducts",
		name: "StrombauerProductBundleProducts",
	},
}

// Normalize returns a copy of src with the known OpenAPI incompatibilities
// fixed. It is safe to call Normalize on its own output.
func Normalize(src []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("specfix: parse OpenAPI document: %w", err)
	}

	components, err := objectAt(doc, "/components")
	if err != nil {
		return nil, fmt.Errorf("specfix: %w", err)
	}
	schemas, err := objectAt(components, "/schemas")
	if err != nil {
		return nil, fmt.Errorf("specfix: %w", err)
	}

	for _, h := range hoists {
		target, ok := lookup(doc, h.ref)
		if !ok {
			// The upstream document may have been fixed already.
			continue
		}
		if _, ok := schemas[h.name]; !ok {
			schemas[h.name] = target
		}

		replaceRef(doc, h.ref, "#/components/schemas/"+h.name)
	}

	// Response components with inline enum properties make oapi-codegen emit
	// field types whose declarations are missing. Hoisting the schema into
	// components.schemas fixes the generated type declarations.
	for _, name := range []string{"ProductListResponse", "ProductSearchResponse"} {
		responses, err := objectAt(components, "/responses")
		if err != nil {
			return nil, fmt.Errorf("specfix: %w", err)
		}
		response, err := objectAt(responses, "/"+name)
		if err != nil {
			continue
		}
		content, err := objectAt(response, "/content")
		if err != nil {
			continue
		}
		media, err := objectAt(content, "/application~1json")
		if err != nil {
			continue
		}
		schema, ok := media["schema"].(map[string]any)
		if !ok {
			continue
		}
		if _, exists := schemas[name]; !exists {
			schemas[name] = schema
		}
		media["schema"] = map[string]any{"$ref": "#/components/schemas/" + name}
	}

	if err := removeCaseOnlyDuplicateProperties(doc); err != nil {
		return nil, fmt.Errorf("specfix: %w", err)
	}
	replaceEmptyItems(doc)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("specfix: encode OpenAPI document: %w", err)
	}
	return append(out, '\n'), nil
}

// objectAt resolves a JSON pointer inside root and requires an object.
func objectAt(root map[string]any, pointer string) (map[string]any, error) {
	value, ok := lookup(root, pointer)
	if !ok {
		return nil, fmt.Errorf("missing JSON pointer %q", pointer)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("JSON pointer %q is %T, want object", pointer, value)
	}
	return object, nil
}

// lookup resolves a JSON pointer such as
// #/components/schemas/Product/properties/type.
func lookup(root map[string]any, pointer string) (any, bool) {
	pointer = strings.TrimPrefix(pointer, "#")
	if pointer == "" || pointer == "/" {
		return root, true
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, false
	}

	var current any = root
	for _, token := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")

		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[token]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// replaceRef rewrites every $ref with the given value.
func replaceRef(v any, oldRef, newRef string) {
	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok && ref == oldRef {
			x["$ref"] = newRef
		}
		for _, child := range x {
			replaceRef(child, oldRef, newRef)
		}
	case []any:
		for _, child := range x {
			replaceRef(child, oldRef, newRef)
		}
	}
}

// removeCaseOnlyDuplicateProperties drops properties that differ from another
// property only by letter case. Go struct fields are case-sensitive, but
// oapi-codegen derives the same field name for both and the generated file
// then fails to compile.
func removeCaseOnlyDuplicateProperties(v any) error {
	switch x := v.(type) {
	case map[string]any:
		if properties, ok := x["properties"].(map[string]any); ok {
			seen := make(map[string]string, len(properties))
			for name, schema := range properties {
				lower := strings.ToLower(name)
				existingName, duplicate := seen[lower]
				if !duplicate {
					seen[lower] = name
					continue
				}

				existingSchema, _ := properties[existingName].(map[string]any)
				currentSchema, _ := schema.(map[string]any)
				switch {
				case isSparseSchema(currentSchema) && !isSparseSchema(existingSchema):
					delete(properties, name)
				case !isSparseSchema(currentSchema) && isSparseSchema(existingSchema):
					delete(properties, existingName)
					seen[lower] = name
				case startsUpper(name) && !startsUpper(existingName):
					delete(properties, name)
				case !startsUpper(name) && startsUpper(existingName):
					delete(properties, existingName)
					seen[lower] = name
				default:
					return fmt.Errorf("cannot resolve properties %q and %q that differ only by case", existingName, name)
				}
			}
		}
		for _, child := range x {
			if err := removeCaseOnlyDuplicateProperties(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range x {
			if err := removeCaseOnlyDuplicateProperties(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func isSparseSchema(schema map[string]any) bool {
	for _, key := range []string{"type", "$ref", "allOf", "oneOf", "anyOf", "items", "enum", "properties", "additionalProperties"} {
		if _, ok := schema[key]; ok {
			return false
		}
	}
	return true
}

func startsUpper(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

// replaceEmptyItems replaces the invalid tuple form "items": [] with an empty
// schema object.
func replaceEmptyItems(v any) {
	switch x := v.(type) {
	case map[string]any:
		if items, ok := x["items"].([]any); ok && len(items) == 0 {
			x["items"] = map[string]any{}
		}
		for _, child := range x {
			replaceEmptyItems(child)
		}
	case []any:
		for _, child := range x {
			replaceEmptyItems(child)
		}
	}
}
