package server

import (
	"encoding/json"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	dst := map[string]any{
		"meta":   map[string]any{"theme": "a", "accent": "#fff"},
		"layout": map[string]any{"columns": float64(3)},
	}
	src := map[string]any{
		"meta": map[string]any{"accent": "#000"},
	}
	out := deepMerge(dst, src)

	meta := out["meta"].(map[string]any)
	if meta["accent"] != "#000" {
		t.Errorf("accent = %v, want #000 (overridden)", meta["accent"])
	}
	if meta["theme"] != "a" {
		t.Errorf("theme = %v, want a (preserved)", meta["theme"])
	}
	if out["layout"].(map[string]any)["columns"] != float64(3) {
		t.Error("layout should be untouched")
	}
}

func TestDeepMergeReplacesArrays(t *testing.T) {
	dst := map[string]any{"widgets": []any{"x", "y"}}
	src := map[string]any{"widgets": []any{"z"}}
	out := deepMerge(dst, src)

	got, _ := json.Marshal(out["widgets"])
	if string(got) != `["z"]` {
		t.Errorf("arrays should be replaced wholesale, got %s", got)
	}
}
