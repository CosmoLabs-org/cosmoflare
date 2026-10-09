/*
Tests for stable machine-readable error codes (FEAT-p8KYM5K wave 1)

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT (https://opensource.org/licenses/MIT)
*/

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestErrorCodeConstantsExist pins the stable wire values: scripts and
// agents match on these strings, so any rename is a breaking change.
func TestErrorCodeConstantsExist(t *testing.T) {
	cases := map[ErrorCode]string{
		CodeAuthMissingToken: "AUTH_MISSING_TOKEN",
		CodeAuthInvalidToken: "AUTH_INVALID_TOKEN",
		CodeConfigInvalid:    "CONFIG_INVALID",
		CodeConfigMissing:    "CONFIG_MISSING",
		CodeNetworkError:     "NETWORK_ERROR",
		CodeAPIError:         "API_ERROR",
		CodeResourceNotFound: "RESOURCE_NOT_FOUND",
		CodeValidationError:  "VALIDATION_ERROR",
	}
	for code, want := range cases {
		if string(code) != want {
			t.Errorf("ErrorCode %q != stable value %q", string(code), want)
		}
	}
}

// TestCodeForUnwrapsNestedWraps: an explicit code survives arbitrary
// fmt.Errorf %w wrapping around the CodedError.
func TestCodeForUnwrapsNestedWraps(t *testing.T) {
	inner := NewCoded(CodeAuthMissingToken, errors.New("token missing"))
	nested := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", inner))
	if got := CodeFor(nested); got != CodeAuthMissingToken {
		t.Errorf("CodeFor(nested) = %q, want %q", got, CodeAuthMissingToken)
	}
}

// TestCodeForFallbackPlainError: plain errors classify as API_ERROR.
func TestCodeForFallbackPlainError(t *testing.T) {
	if got := CodeFor(errors.New("something went wrong")); got != CodeAPIError {
		t.Errorf("CodeFor(plain) = %q, want %q", got, CodeAPIError)
	}
	if got := CodeFor(nil); got != CodeAPIError {
		t.Errorf("CodeFor(nil) = %q, want %q", got, CodeAPIError)
	}
}

// TestCodeForClassifications covers the type- and message-based classes.
func TestCodeForClassifications(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "coded auth invalid",
			err:  NewCoded(CodeAuthInvalidToken, errors.New("bad token")),
			want: CodeAuthInvalidToken,
		},
		{
			name: "r2 not found type",
			err:  &cosmoflare.R2NotFoundError{},
			want: CodeResourceNotFound,
		},
		{
			name: "r2 auth type",
			err:  &cosmoflare.R2AuthError{},
			want: CodeAuthInvalidToken,
		},
		{
			name: "r2 validation type",
			err:  &cosmoflare.R2ValidationError{},
			want: CodeValidationError,
		},
		{
			name: "http 401 status",
			err:  fmt.Errorf("wrapped: %w", &fakeStatusError{status: http.StatusUnauthorized}),
			want: CodeAuthInvalidToken,
		},
		{
			name: "http 404 status",
			err:  &fakeStatusError{status: http.StatusNotFound},
			want: CodeResourceNotFound,
		},
		{
			name: "net.Error",
			err:  &fakeNetError{},
			want: CodeNetworkError,
		},
		{
			name: "message not found",
			err:  errors.New("namespace 'prod' not found"),
			want: CodeResourceNotFound,
		},
		{
			name: "message network",
			err:  errors.New("dial tcp: connection refused"),
			want: CodeNetworkError,
		},
	}
	for _, tc := range cases {
		if got := CodeFor(tc.err); got != tc.want {
			t.Errorf("%s: CodeFor = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestCodedErrorPreservesMessage: Error() delegates so human output is
// unchanged, and errors.Is/As reach the wrapped error.
func TestCodedErrorPreservesMessage(t *testing.T) {
	sentinel := errors.New("the ground truth")
	err := NewCoded(CodeConfigMissing, sentinel)
	if err.Error() != "the ground truth" {
		t.Errorf("CodedError.Error() = %q, want wrapped message", err.Error())
	}
	if !errors.Is(err, sentinel) {
		t.Error("errors.Is should reach the wrapped sentinel through Unwrap")
	}
}

// TestEnvelopeMarshalsWithErrorCode: the JSON error envelope carries the
// stable error_code alongside the historical fields.
func TestEnvelopeMarshalsWithErrorCode(t *testing.T) {
	raw, err := json.Marshal(Envelope(CodeAuthMissingToken, "token missing", false))
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if got, _ := decoded["error_code"].(string); got != string(CodeAuthMissingToken) {
		t.Errorf("error_code = %q, want %q", got, string(CodeAuthMissingToken))
	}
	if decoded["success"] != false {
		t.Errorf("success = %v, want false", decoded["success"])
	}
	if msg, _ := decoded["error"].(string); msg != "token missing" {
		t.Errorf("error = %q, want %q", msg, "token missing")
	}
	if _, ok := decoded["dry_run"]; ok {
		t.Error("dry_run should be omitted when false (historical omitempty)")
	}
	// dry-run keeps the historical key.
	raw, err = json.Marshal(Envelope(CodeAPIError, "boom", true))
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if !strings.Contains(string(raw), `"dry_run":true`) {
		t.Errorf("dry-run envelope missing dry_run:true: %s", raw)
	}
}

// --- test doubles ---

// fakeStatusError carries an HTTP status via the StatusCode() carrier
// shape that cosmoflare.ErrorStatus walks (TASK-012).
type fakeStatusError struct{ status int }

func (e *fakeStatusError) Error() string   { return fmt.Sprintf("http %d", e.status) }
func (e *fakeStatusError) StatusCode() int { return e.status }

// fakeNetError satisfies net.Error for the transport classification.
type fakeNetError struct{}

func (e *fakeNetError) Error() string   { return "connection refused" }
func (e *fakeNetError) Timeout() bool   { return false }
func (e *fakeNetError) Temporary() bool { return false }
