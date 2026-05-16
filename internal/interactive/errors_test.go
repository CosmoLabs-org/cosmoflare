package interactive

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestHandleError_AllTypes(t *testing.T) {
	types := []struct {
		name     string
		errType  ErrorType
		header   string
	}{
		{"network", ErrorTypeNetwork, "Network Error"},
		{"auth", ErrorTypeAuth, "Authentication Error"},
		{"config", ErrorTypeConfig, "Configuration Error"},
		{"input", ErrorTypeInput, "Input Error"},
		{"permission", ErrorTypePermission, "Permission Error"},
		{"not_found", ErrorTypeNotFound, "Not Found Error"},
		{"validation", ErrorTypeValidation, "Validation Error"},
		{"unknown", ErrorTypeUnknown, "Error"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ErrorContext{
				Error:     errors.New("test error"),
				Type:      tt.errType,
				Operation: "test-op",
			}
			// Just verify it doesn't panic
			HandleError(ctx)
		})
	}
}

func TestHandleError_AllFields(t *testing.T) {
	ctx := ErrorContext{
		Error:        errors.New("detailed error"),
		Type:         ErrorTypeNetwork,
		Operation:    "upload file",
		UserAction:   "Check your connection",
		Troubleshoot: []string{"Step 1", "Step 2", "Step 3"},
		NextSteps:    []string{"Retry", "Contact support"},
	}
	HandleError(ctx)
}

func TestHandleError_NilError(t *testing.T) {
	ctx := ErrorContext{
		Error:     nil,
		Type:      ErrorTypeConfig,
		Operation: "config test",
	}
	HandleError(ctx)
}

func TestHandleError_EmptyFields(t *testing.T) {
	ctx := ErrorContext{
		Error:        errors.New("err"),
		Type:         ErrorTypeInput,
		Operation:    "op",
		UserAction:   "",
		Troubleshoot: nil,
		NextSteps:    nil,
	}
	HandleError(ctx)
}

func TestErrorConstructors(t *testing.T) {
	constructors := []struct {
		name string
		ctx  ErrorContext
	}{
		{"NetworkError", NetworkError("connect", errors.New("timeout"))},
		{"AuthError", AuthError("login", errors.New("bad token"))},
		{"ConfigError", ConfigError("load", errors.New("missing file"))},
		{"InputError", InputError("validate", errors.New("bad input"))},
		{"PermissionError", PermissionError("write", errors.New("denied"))},
		{"NotFoundError", NotFoundError("bucket", errors.New("missing"))},
		{"ValidationError", ValidationError("check", errors.New("invalid"))},
	}

	for _, tt := range constructors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ctx.Error == nil {
				t.Error("Error should not be nil")
			}
			if tt.ctx.Operation == "" {
				t.Error("Operation should not be empty")
			}
			if tt.ctx.UserAction == "" {
				t.Error("UserAction should not be empty")
			}
			if len(tt.ctx.Troubleshoot) == 0 {
				t.Error("Troubleshoot should have steps")
			}
			if len(tt.ctx.NextSteps) == 0 {
				t.Error("NextSteps should have items")
			}
		})
	}
}

func TestErrorConstructors_NilError(t *testing.T) {
	constructors := []struct {
		name string
		ctx  ErrorContext
	}{
		{"NetworkError", NetworkError("op", nil)},
		{"AuthError", AuthError("op", nil)},
		{"ConfigError", ConfigError("op", nil)},
		{"InputError", InputError("op", nil)},
		{"PermissionError", PermissionError("op", nil)},
		{"NotFoundError", NotFoundError("op", nil)},
		{"ValidationError", ValidationError("op", nil)},
	}

	for _, tt := range constructors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ctx.Error != nil {
				t.Error("Error should be nil")
			}
			if tt.ctx.Operation != "op" {
				t.Errorf("Operation = %q, want %q", tt.ctx.Operation, "op")
			}
		})
	}
}

func TestErrorPackage_SuccessMessage(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		SuccessMessage("upload", "3 files uploaded")
	})

	t.Run("empty details", func(t *testing.T) {
		SuccessMessage("upload", "")
	})

	t.Run("long operation name", func(t *testing.T) {
		SuccessMessage(strings.Repeat("x", 200), "details")
	})
}

func TestErrorPackage_WarningMessage(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		WarningMessage("deprecated", "Use new-method instead")
	})

	t.Run("empty details", func(t *testing.T) {
		WarningMessage("warning", "")
	})
}

func TestErrorType_String(t *testing.T) {
	types := map[ErrorType]bool{
		ErrorTypeNetwork:    true,
		ErrorTypeAuth:       true,
		ErrorTypeConfig:     true,
		ErrorTypeInput:      true,
		ErrorTypePermission: true,
		ErrorTypeNotFound:   true,
		ErrorTypeValidation: true,
		ErrorTypeUnknown:    true,
	}
	if len(types) != 8 {
		t.Errorf("expected 8 error types, got %d", len(types))
	}
}

func TestErrorContext_Security(t *testing.T) {
	t.Run("error messages should not contain secrets", func(t *testing.T) {
		secret := "super-secret-api-token-12345678"
		ctx := NetworkError("connect", fmt.Errorf("failed with token %s", secret))
		// The error contains the secret in its Error field - that's expected
		// But verify the UserAction/Troubleshoot don't leak it
		if strings.Contains(ctx.UserAction, secret) {
			t.Error("UserAction should not contain secret")
		}
		for _, step := range ctx.Troubleshoot {
			if strings.Contains(step, secret) {
				t.Error("Troubleshoot should not contain secret")
			}
		}
		for _, step := range ctx.NextSteps {
			if strings.Contains(step, secret) {
				t.Error("NextSteps should not contain secret")
			}
		}
	})
}
