package scape

import (
	"encoding/json"
	"testing"
)

func TestDefaultValidates(t *testing.T) {
	raw, err := Marshal(Default())
	if err != nil {
		t.Fatalf("marshal default: %v", err)
	}
	if err := Validate(raw); err != nil {
		t.Fatalf("default spec should validate, got: %v", err)
	}
}

func TestParseRoundTrip(t *testing.T) {
	raw, _ := Marshal(Default())
	spec, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if spec.Version != CurrentVersion {
		t.Errorf("version = %d, want %d", spec.Version, CurrentVersion)
	}
	if spec.Layout.Columns != 3 {
		t.Errorf("columns = %d, want 3", spec.Layout.Columns)
	}
	if len(spec.Widgets) == 0 {
		t.Error("expected seeded widgets")
	}
}

func TestDefaultServiceHealthBinding(t *testing.T) {
	spec := Default()
	var health *Widget
	for i := range spec.Widgets {
		if spec.Widgets[i].Type == "service-health" {
			health = &spec.Widgets[i]
			break
		}
	}
	if health == nil {
		t.Fatal("default spec should include a service-health widget")
	}
	// Beginner-bridge default: show all active integrations (no hard-coded service list).
	if health.Options["source"] != "all" {
		t.Errorf("service-health source = %v, want \"all\"", health.Options["source"])
	}
	if _, hasServices := health.Options["services"]; hasServices {
		t.Error("default service-health must not pin an explicit services list")
	}
}

func TestValidateRejectsBadSpec(t *testing.T) {
	cases := map[string]string{
		"unknown widget type": `{"version":1,"meta":{"name":"x","theme":"t","accent":"#E6A95C"},"layout":{"columns":3},"widgets":[{"id":"a","type":"nope","column":0}]}`,
		"bad accent":          `{"version":1,"meta":{"name":"x","theme":"t","accent":"red"},"layout":{"columns":3},"widgets":[]}`,
		"missing layout":      `{"version":1,"meta":{"name":"x","theme":"t","accent":"#E6A95C"},"widgets":[]}`,
		"wrong version":       `{"version":2,"meta":{"name":"x","theme":"t","accent":"#E6A95C"},"layout":{"columns":3},"widgets":[]}`,
		"columns too high":    `{"version":1,"meta":{"name":"x","theme":"t","accent":"#E6A95C"},"layout":{"columns":99},"widgets":[]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Validate([]byte(raw)); err == nil {
				t.Errorf("expected validation error for %q", name)
			}
		})
	}
}

func TestValidateAcceptsMinimalValid(t *testing.T) {
	raw := `{"version":1,"meta":{"name":"x","theme":"midnight-glass","accent":"#57C4C0"},"layout":{"columns":1},"widgets":[]}`
	if err := Validate([]byte(raw)); err != nil {
		t.Errorf("minimal valid spec should pass: %v", err)
	}
	// sanity: it must be parseable JSON too
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("test fixture is not valid JSON: %v", err)
	}
}
