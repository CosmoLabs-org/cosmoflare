package alertspush

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// validPayload returns a payload that passes Validate and fits the envelope cap.
func validPayload() Payload {
	return Payload{
		ID:       "alert-123",
		Severity: SeverityWarning,
		Service:  "web",
		Title:    "High 5xx rate",
		Detail:   "5xx rate is 12% over the last 5 minutes",
		FiredAt:  time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
	}
}

func TestValidate_ValidPayload(t *testing.T) {
	if err := validPayload().Validate(); err != nil {
		t.Fatalf("Validate on valid payload: %v", err)
	}
}

func TestValidate_BogusSeverity(t *testing.T) {
	p := validPayload()
	p.Severity = Severity("bogus")
	if err := p.Validate(); err == nil {
		t.Fatal("Validate: want error for severity \"bogus\"")
	} else if !strings.Contains(err.Error(), "severity") {
		t.Errorf("Validate error %q should mention severity", err)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	cases := map[string]func(p Payload) Payload{
		"empty id":      func(p Payload) Payload { p.ID = ""; return p },
		"empty service": func(p Payload) Payload { p.Service = ""; return p },
		"empty title":   func(p Payload) Payload { p.Title = ""; return p },
	}
	for name, mangle := range cases {
		t.Run(name, func(t *testing.T) {
			if err := mangle(validPayload()).Validate(); err == nil {
				t.Fatalf("Validate: want error for %s", name)
			}
		})
	}
}

func TestMarshalEnvelope_ExactKeysAndRFC3339(t *testing.T) {
	p := validPayload()
	b, err := p.MarshalEnvelope()
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\n%s", err, b)
	}
	want := []string{"id", "severity", "service", "title", "detail", "fired_at"}
	if len(raw) != len(want) {
		t.Fatalf("envelope has %d keys (%v), want exactly %v", len(raw), raw, want)
	}
	for _, k := range want {
		if _, ok := raw[k]; !ok {
			t.Errorf("envelope missing key %q:\n%s", k, b)
		}
	}
	if raw["id"] != p.ID {
		t.Errorf("id = %v, want %q", raw["id"], p.ID)
	}
	if raw["severity"] != string(SeverityWarning) {
		t.Errorf("severity = %v, want %q", raw["severity"], SeverityWarning)
	}
	firedAt, ok := raw["fired_at"].(string)
	if !ok {
		t.Fatalf("fired_at is not a string: %v", raw["fired_at"])
	}
	if _, err := time.Parse(time.RFC3339, firedAt); err != nil {
		t.Errorf("fired_at %q does not parse as RFC3339: %v", firedAt, err)
	}
	if firedAt != "2026-10-07T12:00:00Z" {
		t.Errorf("fired_at = %q, want RFC3339 \"2026-10-07T12:00:00Z\"", firedAt)
	}
	if len(b) > 2048 {
		t.Errorf("valid payload envelope is %d bytes, must stay under the 2048-byte cap", len(b))
	}
}

func TestMarshalEnvelope_ExceedsCap(t *testing.T) {
	p := validPayload()
	p.Detail = strings.Repeat("x", 2100)
	b, err := p.MarshalEnvelope()
	if err == nil {
		t.Fatalf("MarshalEnvelope: want error for oversized payload, got %d bytes", len(b))
	}
	if !strings.Contains(err.Error(), "payload exceeds 2048 bytes") {
		t.Errorf("error %q should contain \"payload exceeds 2048 bytes\"", err)
	}
	if b != nil {
		t.Errorf("MarshalEnvelope should return nil bytes on error, got %d bytes", len(b))
	}
}

func TestMarshalEnvelope_RejectsInvalidPayload(t *testing.T) {
	p := validPayload()
	p.Severity = Severity("bogus")
	if _, err := p.MarshalEnvelope(); err == nil {
		t.Fatal("MarshalEnvelope: want error for payload failing Validate")
	}
}
