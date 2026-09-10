package cosmoflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// --- newError ---

func TestNewError_ReturnsNonNil(t *testing.T) {
	err := newError("TestOp", "something broke", nil)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestNewError_FieldsPopulated(t *testing.T) {
	cause := fmt.Errorf("underlying")
	err := newError("ListBuckets", "connection refused", cause)
	if err.Op != "ListBuckets" {
		t.Errorf("Op = %q, want %q", err.Op, "ListBuckets")
	}
	if err.Message != "connection refused" {
		t.Errorf("Message = %q, want %q", err.Message, "connection refused")
	}
	if err.Err != cause {
		t.Error("Err should be the wrapped cause")
	}
}

func TestNewError_ErrorStringContainsOpAndMessage(t *testing.T) {
	err := newError("GetObject", "timeout", nil)
	s := err.Error()
	if !strings.Contains(s, "GetObject") {
		t.Errorf("error string %q missing operation", s)
	}
	if !strings.Contains(s, "timeout") {
		t.Errorf("error string %q missing message", s)
	}
	if !strings.HasPrefix(s, "r2go2:") {
		t.Errorf("error string %q missing r2go2 prefix", s)
	}
}

func TestNewError_EmptyStrings(t *testing.T) {
	err := newError("", "", nil)
	if err == nil {
		t.Fatal("expected non-nil error even with empty strings")
	}
	// Should still produce a valid error string
	s := err.Error()
	if s == "" {
		t.Error("error string should not be empty")
	}
}

func TestNewError_NilWrappedError(t *testing.T) {
	err := newError("Op", "msg", nil)
	if err.Unwrap() != nil {
		t.Error("Unwrap should return nil when no cause provided")
	}
}

func TestNewError_UnwrapReturnsCause(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := newError("Op", "msg", cause)
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

// --- R2Error.Error() formatting variants ---

func TestR2Error_ErrorFormat_BucketAndKey(t *testing.T) {
	err := &R2Error{Op: "GetObject", Bucket: "my-bucket", Key: "photo.jpg", Message: "not found"}
	s := err.Error()
	want := "r2go2: GetObject: bucket=my-bucket key=photo.jpg: not found"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

func TestR2Error_ErrorFormat_BucketOnly(t *testing.T) {
	err := &R2Error{Op: "ListObjects", Bucket: "data-bucket", Message: "access denied"}
	s := err.Error()
	want := "r2go2: ListObjects: bucket=data-bucket: access denied"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

func TestR2Error_ErrorFormat_OpAndMessageOnly(t *testing.T) {
	err := &R2Error{Op: "ListBuckets", Message: "network error"}
	s := err.Error()
	want := "r2go2: ListBuckets: network error"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

func TestR2Error_ErrorFormat_KeyWithoutBucket(t *testing.T) {
	// Key is set but Bucket is empty — should use the op+message-only format
	err := &R2Error{Op: "GetObject", Key: "orphan-key", Message: "bad state"}
	s := err.Error()
	// Bucket is empty so the key branch is not taken
	if strings.Contains(s, "orphan-key") {
		t.Error("key should not appear in output when bucket is empty")
	}
	want := "r2go2: GetObject: bad state"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

// --- notFound ---

func TestNotFound_ReturnsNonNil(t *testing.T) {
	err := notFound("GetObject", "bucket", "key.txt", nil)
	if err == nil {
		t.Fatal("expected non-nil")
	}
}

func TestNotFound_FieldsPopulated(t *testing.T) {
	cause := fmt.Errorf("404")
	err := notFound("GetObject", "assets", "logo.png", cause)
	if err.Op != "GetObject" {
		t.Errorf("Op = %q", err.Op)
	}
	if err.Bucket != "assets" {
		t.Errorf("Bucket = %q", err.Bucket)
	}
	if err.Key != "logo.png" {
		t.Errorf("Key = %q", err.Key)
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %q, want 'resource not found'", err.Message)
	}
	if err.Err != cause {
		t.Error("wrapped error mismatch")
	}
}

func TestNotFound_ErrorString(t *testing.T) {
	err := notFound("HeadObject", "media", "vid.mp4", nil)
	s := err.Error()
	if !strings.Contains(s, "HeadObject") || !strings.Contains(s, "media") || !strings.Contains(s, "vid.mp4") {
		t.Errorf("error string %q missing expected fields", s)
	}
}

func TestNotFound_EmptyBucketAndKey(t *testing.T) {
	err := notFound("GetObject", "", "", nil)
	s := err.Error()
	// With empty bucket, should fall through to the op+message format
	if strings.Contains(s, "bucket=") {
		t.Errorf("should not contain bucket= when bucket is empty: %q", s)
	}
}

// --- authError ---

func TestAuthError_ReturnsNonNil(t *testing.T) {
	err := authError("ListBuckets", "invalid credentials", nil)
	if err == nil {
		t.Fatal("expected non-nil")
	}
}

func TestAuthError_ErrorString(t *testing.T) {
	err := authError("PutObject", "token expired", nil)
	s := err.Error()
	if !strings.Contains(s, "PutObject") || !strings.Contains(s, "token expired") {
		t.Errorf("error string %q missing expected content", s)
	}
}

func TestAuthError_UnwrapsCause(t *testing.T) {
	cause := fmt.Errorf("HTTP 401")
	err := authError("ListBuckets", "unauthorized", cause)
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find wrapped cause")
	}
}

// --- quotaError ---

func TestQuotaError_ReturnsNonNil(t *testing.T) {
	err := quotaError("PutObject", "rate limit exceeded", nil)
	if err == nil {
		t.Fatal("expected non-nil")
	}
}

func TestQuotaError_ErrorString(t *testing.T) {
	err := quotaError("Upload", "429 too many requests", nil)
	s := err.Error()
	if !strings.Contains(s, "Upload") || !strings.Contains(s, "429 too many requests") {
		t.Errorf("error string %q missing expected content", s)
	}
}

// --- accessDenied ---

func TestAccessDenied_ReturnsNonNil(t *testing.T) {
	err := accessDenied("DeleteBucket", "private-bucket", "forbidden", nil)
	if err == nil {
		t.Fatal("expected non-nil")
	}
}

func TestAccessDenied_FieldsPopulated(t *testing.T) {
	err := accessDenied("PutObject", "secure-bucket", "insufficient permissions", nil)
	if err.Op != "PutObject" {
		t.Errorf("Op = %q", err.Op)
	}
	if err.Bucket != "secure-bucket" {
		t.Errorf("Bucket = %q", err.Bucket)
	}
	if err.Message != "insufficient permissions" {
		t.Errorf("Message = %q", err.Message)
	}
}

func TestAccessDenied_ErrorStringIncludesBucket(t *testing.T) {
	err := accessDenied("GetObject", "locked-bucket", "no access", nil)
	s := err.Error()
	if !strings.Contains(s, "locked-bucket") {
		t.Errorf("error string %q should include bucket name", s)
	}
}

// --- validationError ---

func TestValidationError_ReturnsNonNil(t *testing.T) {
	err := validationError("CreateBucket", "bucket name too short")
	if err == nil {
		t.Fatal("expected non-nil")
	}
}

func TestValidationError_NoWrappedError(t *testing.T) {
	err := validationError("CreateBucket", "invalid name")
	if err.Unwrap() != nil {
		t.Error("validationError should not have a wrapped error")
	}
}

func TestValidationError_ErrorString(t *testing.T) {
	err := validationError("PutObject", "key cannot be empty")
	s := err.Error()
	if !strings.Contains(s, "PutObject") || !strings.Contains(s, "key cannot be empty") {
		t.Errorf("error string %q missing expected content", s)
	}
}

func TestValidationError_EmptyStrings(t *testing.T) {
	err := validationError("", "")
	if err == nil {
		t.Fatal("expected non-nil even with empty strings")
	}
	s := err.Error()
	if s == "" {
		t.Error("error string should not be empty")
	}
}

// --- Type distinguishability ---

func TestErrorTypes_AreDistinguishable(t *testing.T) {
	nf := notFound("op", "b", "k", nil)
	ae := authError("op", "msg", nil)
	qe := quotaError("op", "msg", nil)
	ad := accessDenied("op", "b", "msg", nil)
	ve := validationError("op", "msg")
	be := newError("op", "msg", nil)

	// Each should be assignable to its own type via errors.As
	var targetNF *R2NotFoundError
	if !errors.As(nf, &targetNF) {
		t.Error("notFound should match R2NotFoundError via errors.As")
	}

	var targetAE *R2AuthError
	if !errors.As(ae, &targetAE) {
		t.Error("authError should match R2AuthError via errors.As")
	}

	var targetQE *R2QuotaError
	if !errors.As(qe, &targetQE) {
		t.Error("quotaError should match R2QuotaError via errors.As")
	}

	var targetAD *R2AccessDeniedError
	if !errors.As(ad, &targetAD) {
		t.Error("accessDenied should match R2AccessDeniedError via errors.As")
	}

	var targetVE *R2ValidationError
	if !errors.As(ve, &targetVE) {
		t.Error("validationError should match R2ValidationError via errors.As")
	}

	// Cross-type: notFound should NOT match R2AuthError
	var wrongType *R2AuthError
	if errors.As(nf, &wrongType) {
		t.Error("notFound should NOT match R2AuthError")
	}

	// newError returns *R2Error directly, so errors.As works for it
	var baseErr *R2Error
	if !errors.As(be, &baseErr) {
		t.Error("newError result should match *R2Error via errors.As")
	}

	// Subtypes embed R2Error by value (not *R2Error), so errors.As to *R2Error
	// does NOT match them. This is correct Go behavior — they are distinct types.
	// Verify this is the case (documents the design).
	for _, e := range []error{nf, ae, qe, ad, ve} {
		if errors.As(e, &baseErr) {
			t.Errorf("%T should NOT match *R2Error via errors.As (embedded by value)", e)
		}
	}

	// But all subtypes still satisfy the error interface and produce valid strings
	for _, e := range []error{nf, ae, qe, ad, ve, be} {
		if e.Error() == "" {
			t.Errorf("%T.Error() should not be empty", e)
		}
	}
}

// --- errors.Is chain ---

func TestErrorChain_DeepUnwrap(t *testing.T) {
	root := fmt.Errorf("network timeout")
	mid := fmt.Errorf("API call failed: %w", root)
	top := authError("ListBuckets", "auth failed", mid)

	if !errors.Is(top, root) {
		t.Error("errors.Is should traverse the chain to find root cause")
	}
	if !errors.Is(top, mid) {
		t.Error("errors.Is should find the intermediate error")
	}
}

// --- isNotFound ---

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"cf error 404", &cloudflare.Error{StatusCode: http.StatusNotFound}, true},
		{"cf error code 1000", &cloudflare.Error{StatusCode: http.StatusInternalServerError, ErrorCodes: []int{1000}}, true},
		{"cf error 500 no codes", &cloudflare.Error{StatusCode: http.StatusInternalServerError}, false},
		{"bucket not found", errors.New("bucket not found"), true},
		{"could not find resource", errors.New("could not find resource"), true},
		{"HTTP 404 string", errors.New("HTTP 404"), true},
		{"something else", errors.New("something else entirely"), false},
		{"wrapped cf 404", fmt.Errorf("wrap: %w", &cloudflare.Error{StatusCode: http.StatusNotFound}), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotFound(tc.err); got != tc.want {
				t.Errorf("isNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
