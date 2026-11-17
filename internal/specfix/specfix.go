// Package specfix normalizes the vendored Strombauer OpenAPI document so that
// it can be consumed by oapi-codegen and produces a complete SDK.
//
// The published document contains constructs that oapi-codegen cannot consume
// directly, and schemas that are incomplete:
//
//   - $ref values that point below a component schema (for example
//     #/components/schemas/Product/properties/type). Those targets are hoisted
//     into top-level components and the references are rewritten.
//   - An empty tuple-style "items": [] array. It is replaced with an empty
//     schema object, which means "any value".
//   - Properties that differ only by letter case. One of them is dropped so
//     that the generated Go struct does not declare the same field twice.
//   - Response schemas with inline enums. They are hoisted into components so
//     the generated enum types are declared.
//   - The CompactCart line-item schema declares only listPrice, although the
//     documented response example and the cart endpoints use the full line
//     item. The missing fields are added and the inline schema is hoisted into
//     the CompactCartLineItem component.
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
	if responses, err := objectAt(components, "/responses"); err == nil {
		for _, name := range []string{"ProductListResponse", "ProductSearchResponse"} {
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
	}

	completeCompactCartLineItems(doc)

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

// completeCompactCartLineItems fills in the line-item fields that the
// published CompactCart schema omits. The response example documents id,
// productId, productNumber, name, image, measureUnit, quantity, price,
// availableStock and stockForecast, and the cart endpoints need the line-item
// ID. The completed schema is hoisted into its own component so the generated
// SDK exposes a named CompactCartLineItem type.
//
// The function is a no-op when the upstream document already references a
// named line-item schema, so a fixed upstream document wins.
func completeCompactCartLineItems(doc map[string]any) {
	schemas, err := objectAt(doc, "/components/schemas")
	if err != nil {
		return
	}
	compactCart, err := objectAt(schemas, "/CompactCart")
	if err != nil {
		return
	}
	items, err := objectAt(compactCart, "/properties/items")
	if err != nil {
		return
	}
	itemSchema, ok := items["items"].(map[string]any)
	if !ok {
		return
	}
	if _, isRef := itemSchema["$ref"]; isRef {
		return
	}

	if _, exists := schemas["CompactCartLineItem"]; !exists {
		addCompactCartLineItemProperties(itemSchema)
		schemas["CompactCartLineItem"] = itemSchema
	}
	items["items"] = map[string]any{"$ref": "#/components/schemas/CompactCartLineItem"}
}

// addCompactCartLineItemProperties adds the documented line-item fields that
// are missing from the inline schema. Existing fields are left untouched.
func addCompactCartLineItemProperties(schema map[string]any) {
	addProperty(schema, "id", map[string]any{
		"description": "Line-item ID. Pass it to `PATCH` or `DELETE /store-api/v1/checkout/cart/line-item/{id}`.",
		"type":        "string",
		"pattern":     "^[0-9a-f]{32}$",
	}, true)
	addProperty(schema, "productId", map[string]any{
		"description": "Shopware product ID.",
		"type":        "string",
		"pattern":     "^[0-9a-f]{32}$",
	}, true)
	addProperty(schema, "productNumber", map[string]any{
		"description": "Product number of the line item.",
		"type":        "string",
	}, true)
	addProperty(schema, "name", map[string]any{
		"description": "Product name.",
		"type":        "string",
		"nullable":    true,
	}, true)
	addProperty(schema, "image", map[string]any{
		"description": "Absolute product cover image URL without query parameters.",
		"type":        "string",
		"nullable":    true,
	}, false)
	addProperty(schema, "measureUnit", map[string]any{
		"description": "Measure unit of the line item; `null` when the product has no unit.",
		"type":        "object",
		"nullable":    true,
		"required":    []any{"code", "quantityPerUnit"},
		"properties": map[string]any{
			"code":            map[string]any{"type": "string"},
			"name":            map[string]any{"type": "string", "nullable": true},
			"label":           map[string]any{"type": "string", "nullable": true},
			"quantityPerUnit": map[string]any{"type": "integer", "minimum": 1},
		},
	}, false)
	addProperty(schema, "quantity", map[string]any{
		"description": "Quantity of the line item.",
		"type":        "integer",
		"minimum":     1,
	}, true)
	addProperty(schema, "price", map[string]any{
		"description": "Price of the line item: `unit` is the price of one unit, `total` is `unit` × `quantity`.",
		"type":        "object",
		"nullable":    true,
		"required":    []any{"unit", "total"},
		"properties": map[string]any{
			"unit":  map[string]any{"type": "number"},
			"total": map[string]any{"type": "number"},
		},
	}, false)
	addProperty(schema, "availableStock", map[string]any{
		"description": "Available stock of the line item; `null` for guests and customers that are not unlocked for orders.",
		"type":        "integer",
		"nullable":    true,
	}, false)
	addProperty(schema, "stockForecast", map[string]any{
		"description":          "Forecast of the available quantity as map `{ \"YYYY-MM-DD\": quantity }`; `null` for guests and customers that are not unlocked for orders.",
		"type":                 "object",
		"nullable":             true,
		"additionalProperties": map[string]any{"type": "number"},
	}, false)
}

// addProperty adds a property to an object schema unless it already exists.
// New required properties are appended to the required list.
func addProperty(schema map[string]any, name string, property map[string]any, required bool) {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		properties = make(map[string]any)
		schema["properties"] = properties
	}
	if _, exists := properties[name]; exists {
		return
	}
	properties[name] = property

	if !required {
		return
	}
	requiredList, _ := schema["required"].([]any)
	schema["required"] = append(requiredList, name)
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
