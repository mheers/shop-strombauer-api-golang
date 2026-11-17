package specfix

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func loadSpec(t *testing.T) []byte {
	t.Helper()

	src, err := os.ReadFile("../../spec/openapi.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	return src
}

func decode(t *testing.T, src []byte) map[string]any {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal(src, &doc); err != nil {
		t.Fatalf("decode normalized spec: %v", err)
	}
	return doc
}

func TestNormalizeHoistsDeepReferences(t *testing.T) {
	out, err := Normalize(loadSpec(t))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	doc := decode(t, out)

	schemas := mustMap(t, mustMap(t, doc, "components"), "schemas")
	for _, name := range []string{
		"LineItemType",
		"ProductType",
		"StrombauerProductType",
		"StrombauerProductBundleProducts",
	} {
		if _, ok := schemas[name]; !ok {
			t.Errorf("expected hoisted component %q", name)
		}
	}

	walk(doc, func(m map[string]any) {
		ref, ok := m["$ref"].(string)
		if !ok {
			return
		}
		if strings.Count(ref, "/") > 3 {
			t.Errorf("deep $ref left in document: %s", ref)
		}
	})
}

func TestNormalizeReplacesEmptyTupleItems(t *testing.T) {
	out, err := Normalize(loadSpec(t))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	doc := decode(t, out)

	features := mustMap(t,
		mustMap(t,
			mustMap(t,
				mustMap(t, mustMap(t, mustMap(t, doc, "components"), "schemas"), "OrderLineItem"),
				"properties"),
			"payload"),
		"properties")
	items, ok := mustMap(t, features, "features")["items"]
	if !ok {
		t.Fatal("features.items is missing")
	}
	if _, isArray := items.([]any); isArray {
		t.Fatalf("features.items is still an array: %#v", items)
	}
}

func TestNormalizeHoistsResponseSchemas(t *testing.T) {
	out, err := Normalize(loadSpec(t))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	doc := decode(t, out)

	schemas := mustMap(t, mustMap(t, doc, "components"), "schemas")
	for _, name := range []string{"ProductListResponse", "ProductSearchResponse"} {
		if _, ok := schemas[name]; !ok {
			t.Errorf("expected response schema %q in components.schemas", name)
		}

		responses := mustMap(t, mustMap(t, doc, "components"), "responses")
		response := mustMap(t, responses, name)
		content := mustMap(t, response, "content")
		media := mustMap(t, content, "application/json")
		schema := mustMap(t, media, "schema")
		if schema["$ref"] != "#/components/schemas/"+name {
			t.Errorf("%s schema $ref = %v", name, schema["$ref"])
		}
	}
}

func TestNormalizeRemovesCaseOnlyDuplicateProperties(t *testing.T) {
	out, err := Normalize(loadSpec(t))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	doc := decode(t, out)

	schemas := mustMap(t, mustMap(t, doc, "components"), "schemas")
	cmsSlot := mustMap(t, schemas, "CmsSlot")
	cmsSlotProperties := mustMap(t, cmsSlot, "properties")
	if _, ok := cmsSlotProperties["VersionId"]; ok {
		t.Error("CmsSlot still contains the duplicate VersionId property")
	}
	if _, ok := cmsSlotProperties["versionId"]; !ok {
		t.Error("CmsSlot lost the versionId property")
	}

	theme := mustMap(t, schemas, "Theme")
	themeProperties := mustMap(t, theme, "properties")
	if _, ok := themeProperties["ConfigValues"]; ok {
		t.Error("Theme still contains the duplicate ConfigValues property")
	}
	if _, ok := themeProperties["configValues"]; !ok {
		t.Error("Theme lost the configValues property")
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	first, err := Normalize(loadSpec(t))
	if err != nil {
		t.Fatalf("first Normalize: %v", err)
	}
	second, err := Normalize(first)
	if err != nil {
		t.Fatalf("second Normalize: %v", err)
	}

	if !reflect.DeepEqual(decode(t, first), decode(t, second)) {
		t.Fatal("Normalize is not idempotent")
	}
}

func TestNormalizeRejectsInvalidJSON(t *testing.T) {
	if _, err := Normalize([]byte("{")); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func mustMap(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()

	raw, ok := m[key]
	if !ok {
		t.Fatalf("key %q is missing", key)
	}
	child, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("key %q is %T, want object", key, raw)
	}
	return child
}

func walk(v any, visit func(map[string]any)) {
	switch x := v.(type) {
	case map[string]any:
		visit(x)
		for _, child := range x {
			walk(child, visit)
		}
	case []any:
		for _, child := range x {
			walk(child, visit)
		}
	}
}
