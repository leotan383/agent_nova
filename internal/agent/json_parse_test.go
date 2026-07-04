package agent

import (
	"testing"
)

func TestSanitizeJSONObject_trailingComma(t *testing.T) {
	raw := `{"entities":[{"name":"林枫","state":{"goal":"test",},},],}`
	fixed := SanitizeJSONObject(raw)
	var m map[string]any
	if err := UnmarshalJSONObject(fixed, &m); err != nil {
		t.Fatalf("unmarshal: %v\nfixed: %s", err, fixed)
	}
}

func TestUnmarshalJSONObject_storyFactsShape(t *testing.T) {
	raw := `{
  "entities": [{"type":"character","name":"林枫","state":{"LOCATION":"归墟"}}],
  "foreshadows": [],
  "cool_points": [],
  "memories": [],
}`
	var m map[string]any
	if err := UnmarshalJSONObject(raw, &m); err != nil {
		t.Fatal(err)
	}
}
