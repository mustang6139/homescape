package scape

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// specWith wraps a single composed widget's options into a full, otherwise-valid spec.
func specWith(t *testing.T, options string) []byte {
	t.Helper()
	spec := fmt.Sprintf(`{
		"version":1,
		"meta":{"name":"x","theme":"midnight-glass","accent":"#E6A95C"},
		"layout":{"columns":3},
		"widgets":[{"id":"c1","type":"composed","column":0,"options":%s}]
	}`, options)
	return []byte(spec)
}

func TestComposedValid(t *testing.T) {
	opts := `{
		"source":{"kind":"resource","resource":"sonarr-main|service.status"},
		"view":{"el":"row","children":[
			{"el":"icon","bind":"up","map":{"true":"check","false":"x"},"style":{"color":"up"}},
			{"el":"text","bind":"name","style":{"weight":"600"}},
			{"el":"text","bind":"latencyMs","fmt":"ms","style":{"color":"muted"}}
		]}
	}`
	if err := Validate(specWith(t, opts)); err != nil {
		t.Fatalf("valid composed widget should pass: %v", err)
	}
}

// The service-health oracle: it must be expressible with the starter primitive/source set.
func TestComposedServiceHealthOracle(t *testing.T) {
	opts := `{
		"source":{"kind":"services"},
		"view":{"el":"stack","children":[
			{"el":"text","bind":"upCount","fmt":"round"},
			{"el":"list","of":"items","item":{"el":"row","children":[
				{"el":"icon","bind":"up","map":{"true":"check","false":"x"}},
				{"el":"text","bind":"name"},
				{"el":"text","bind":"latencyMs","fmt":"ms"}
			]}}
		]}
	}`
	if err := Validate(specWith(t, opts)); err != nil {
		t.Fatalf("service-health-as-composed should validate: %v", err)
	}
}

func TestComposedRejectsBad(t *testing.T) {
	cases := map[string]string{
		"unknown el":     `{"source":{"kind":"host"},"view":{"el":"blink"}}`,
		"unknown fmt":    `{"source":{"kind":"host"},"view":{"el":"text","bind":"x","fmt":"explode"}}`,
		"unknown style":  `{"source":{"kind":"host"},"view":{"el":"text","style":{"color":"chartreuse"}}}`,
		"unknown source": `{"source":{"kind":"telepathy"},"view":{"el":"text"}}`,
		"missing view":   `{"source":{"kind":"host"}}`,
		"extra prop":     `{"source":{"kind":"host"},"view":{"el":"text","wat":1}}`,
		"missing el":     `{"source":{"kind":"host"},"view":{"bind":"x"}}`,
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Validate(specWith(t, opts)); err == nil {
				t.Errorf("expected validation error for %q", name)
			}
		})
	}
}

func TestComposedDepthLimit(t *testing.T) {
	// Build a deeply nested stack chain exceeding maxViewDepth.
	leaf := map[string]any{"el": "text", "text": "deep"}
	node := leaf
	for i := 0; i < maxViewDepth+2; i++ {
		node = map[string]any{"el": "stack", "children": []any{node}}
	}
	opts := map[string]any{
		"source": map[string]any{"kind": "host"},
		"view":   node,
	}
	raw, _ := json.Marshal(opts)
	if err := Validate(specWith(t, string(raw))); err == nil || !strings.Contains(err.Error(), "deep") {
		t.Errorf("expected depth-limit error, got %v", err)
	}
}

func TestComposedNodeLimit(t *testing.T) {
	children := make([]any, 0, maxViewNodes+5)
	for i := 0; i < maxViewNodes+5; i++ {
		children = append(children, map[string]any{"el": "text", "text": "n"})
	}
	opts := map[string]any{
		"source": map[string]any{"kind": "host"},
		"view":   map[string]any{"el": "stack", "children": children},
	}
	raw, _ := json.Marshal(opts)
	if err := Validate(specWith(t, string(raw))); err == nil || !strings.Contains(err.Error(), "many nodes") {
		t.Errorf("expected node-limit error, got %v", err)
	}
}

func TestNonComposedUnaffected(t *testing.T) {
	// A normal spec (no composed widgets) must still validate.
	raw, _ := Marshal(Default())
	if err := Validate(raw); err != nil {
		t.Fatalf("default spec should still validate: %v", err)
	}
}
