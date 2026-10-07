// Package alertspush implements the FEAT-045 pager: VAPID key management,
// push subscriptions, and the Web Push payload protocol.
package alertspush

import (
	"encoding/json"
	"fmt"
	"time"
)

// Severity is the alert severity vocabulary pushed to pager clients.
type Severity string

// The severity vocabulary is exactly info | warning | critical.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// maxEnvelopeBytes is the hard cap on the marshaled push payload. Web Push
// message bodies must stay within 2048 bytes.
const maxEnvelopeBytes = 2048

// Payload is the JSON envelope delivered to pager clients. The wire keys are
// exactly id, severity, service, title, detail, fired_at. No credentials may
// ever be placed in a payload.
type Payload struct {
	ID       string    `json:"id"`
	Severity Severity  `json:"severity"`
	Service  string    `json:"service"`
	Title    string    `json:"title"`
	Detail   string    `json:"detail"`
	FiredAt  time.Time `json:"fired_at"` // RFC3339 via time.Time's default JSON encoding
}

// Validate reports whether the payload carries the required fields (non-empty
// id, service, title) and a severity from the known vocabulary.
func (p Payload) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("payload: id is required")
	}
	switch p.Severity {
	case SeverityInfo, SeverityWarning, SeverityCritical:
	default:
		return fmt.Errorf("payload: severity must be one of info, warning, critical (got %q)", string(p.Severity))
	}
	if p.Service == "" {
		return fmt.Errorf("payload: service is required")
	}
	if p.Title == "" {
		return fmt.Errorf("payload: title is required")
	}
	return nil
}

// MarshalEnvelope validates the payload and encodes it as the JSON push
// envelope. It errors if the marshaled payload exceeds the 2048-byte cap.
func (p Payload) MarshalEnvelope() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("payload: marshal envelope: %w", err)
	}
	if len(b) > maxEnvelopeBytes {
		return nil, fmt.Errorf("payload: payload exceeds 2048 bytes (%d)", len(b))
	}
	return b, nil
}
