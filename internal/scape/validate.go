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

// compiledSchema is built once at init from the embedded canonical schema.
var compiledSchema *jsonschema.Schema

func init() {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		panic(fmt.Sprintf("scape: invalid embedded schema: %v", err))
	}
	c := jsonschema.NewCompiler()
	const id = "https://homescape.dev/schema/scape.schema.json"
	if err := c.AddResource(id, doc); err != nil {
		panic(fmt.Sprintf("scape: cannot add schema resource: %v", err))
	}
	s, err := c.Compile(id)
	if err != nil {
		panic(fmt.Sprintf("scape: cannot compile schema: %v", err))
	}
	compiledSchema = s
}

// SchemaJSON returns the raw canonical schema bytes (served to the frontend if needed).
func SchemaJSON() []byte { return schemaJSON }

// Validate checks raw spec JSON against the canonical schema and returns a readable error.
func Validate(raw []byte) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := compiledSchema.Validate(v); err != nil {
		return fmt.Errorf("spec does not match schema: %w", err)
	}
	return nil
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
