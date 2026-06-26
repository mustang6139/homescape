package scape

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed scape.schema.json
var schemaJSON []byte

//go:embed composed.schema.json
var composedSchemaJSON []byte

// compiledSchema validates a whole spec; composedSchema validates a composed widget's
// options. Both are built once at init from the embedded canonical schemas.
var (
	compiledSchema *jsonschema.Schema
	composedSchema *jsonschema.Schema
)

// Structural limits on a composed view tree (moderation + performance).
const (
	maxViewDepth = 8
	maxViewNodes = 200
)

func init() {
	compiledSchema = mustCompile(schemaJSON, "https://homescape.dev/schema/scape.schema.json")
	composedSchema = mustCompile(composedSchemaJSON, "https://homescape.dev/schema/composed.schema.json")
}

func mustCompile(raw []byte, id string) *jsonschema.Schema {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("scape: invalid embedded schema %s: %v", id, err))
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(id, doc); err != nil {
		panic(fmt.Sprintf("scape: cannot add schema resource %s: %v", id, err))
	}
	s, err := c.Compile(id)
	if err != nil {
		panic(fmt.Sprintf("scape: cannot compile schema %s: %v", id, err))
	}
	return s
}

// SchemaJSON returns the raw canonical spec schema (served to the frontend if needed).
func SchemaJSON() []byte { return schemaJSON }

// ComposedSchemaJSON returns the raw composed-widget schema (shared with the frontend).
func ComposedSchemaJSON() []byte { return composedSchemaJSON }

// Validate checks raw spec JSON against the canonical schema, then validates every composed
// widget's options against the composed schema and structural limits.
func Validate(raw []byte) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := compiledSchema.Validate(v); err != nil {
		return fmt.Errorf("spec does not match schema: %w", err)
	}

	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("decode spec: %w", err)
	}
	for _, w := range spec.Widgets {
		if w.Type != "composed" {
			continue
		}
		if err := validateComposed(w); err != nil {
			return fmt.Errorf("widget %q: %w", w.ID, err)
		}
	}
	return nil
}

// validateComposed checks a composed widget's options against the composed schema and the
// view-tree depth/node limits.
func validateComposed(w Widget) error {
	optRaw, err := json.Marshal(w.Options)
	if err != nil {
		return err
	}
	var ov any
	if err := json.Unmarshal(optRaw, &ov); err != nil {
		return err
	}
	if err := composedSchema.Validate(ov); err != nil {
		return fmt.Errorf("composed options invalid: %w", err)
	}

	view, ok := w.Options["view"].(map[string]any)
	if !ok {
		return fmt.Errorf("composed widget missing view")
	}
	nodes, depth := 0, 0
	walkView(view, 1, &nodes, &depth)
	if depth > maxViewDepth {
		return fmt.Errorf("view nesting too deep (%d > %d)", depth, maxViewDepth)
	}
	if nodes > maxViewNodes {
		return fmt.Errorf("view has too many nodes (%d > %d)", nodes, maxViewNodes)
	}
	return nil
}

// walkView counts nodes and tracks the maximum depth of a view tree.
func walkView(node map[string]any, depth int, count, maxDepth *int) {
	*count++
	if depth > *maxDepth {
		*maxDepth = depth
	}
	if children, ok := node["children"].([]any); ok {
		for _, c := range children {
			if cm, ok := c.(map[string]any); ok {
				walkView(cm, depth+1, count, maxDepth)
			}
		}
	}
	if item, ok := node["item"].(map[string]any); ok {
		walkView(item, depth+1, count, maxDepth)
	}
}

// Parse validates raw JSON and unmarshals it into a Spec.
func Parse(raw []byte) (Spec, error) {
	if err := Validate(raw); err != nil {
		return Spec{}, err
	}
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf("decode spec: %w", err)
	}
	return s, nil
}

// Marshal serialises a Spec back to canonical JSON.
func Marshal(s Spec) ([]byte, error) {
	return json.Marshal(s)
}
