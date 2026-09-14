package cosmoflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// R2Error is the base error type for all R2Go2 errors.
type R2Error struct {
	Op      string // Operation that failed
	Bucket  string // Bucket involved (if any)
	Key     string // Object key involved (if any)
	Err     error  // Underlying error
	Message string // Human-readable message
}

func (e *R2Error) Error() string {
	if e.Bucket != "" && e.Key != "" {
		return fmt.Sprintf("cosmoflare: %s: bucket=%s key=%s: %s", e.Op, e.Bucket, e.Key, e.Message)
	}
	if e.Bucket != "" {
		return fmt.Sprintf("cosmoflare: %s: bucket=%s: %s", e.Op, e.Bucket, e.Message)
	}
	return fmt.Sprintf("cosmoflare: %s: %s", e.Op, e.Message)
}

func (e *R2Error) Unwrap() error { return e.Err }

// R2NotFoundError indicates a requested resource does not exist.
type R2NotFoundError struct {
	R2Error
}

// R2AuthError indicates an authentication or authorization failure.
type R2AuthError struct {
	R2Error
}

// R2QuotaError indicates a quota or rate limit has been exceeded.
type R2QuotaError struct {
	R2Error
}

// R2AccessDeniedError indicates the caller lacks permission for the operation.
type R2AccessDeniedError struct {
	R2Error
}

// R2ValidationError indicates invalid input (bad bucket name, missing key, etc.).
type R2ValidationError struct {
	R2Error
}

// newError creates a base R2Error.
func newError(op, msg string, err error) *R2Error {
	e := &R2Error{Op: op, Message: msg, Err: err}
	// Knowledge layer: decorate globally-decoded CF errors for every
	// service without call-site changes (context-free entries only).
	if ke, ok := knowledge.DecodeCFError(err, "").(*knowledge.KnowledgeError); ok {
		e.Message = msg + " — " + ke.Cause + " (fix: " + ke.Fix + ")"
	}
	return e
}

// notFound creates an R2NotFoundError.
func notFound(op, bucket, key string, err error) *R2NotFoundError {
	return &R2NotFoundError{R2Error{Op: op, Bucket: bucket, Key: key, Message: "resource not found", Err: err}}
}

// authError creates an R2AuthError.
func authError(op, msg string, err error) *R2AuthError {
	return &R2AuthError{R2Error{Op: op, Message: msg, Err: err}}
}

// quotaError creates an R2QuotaError.
func quotaError(op, msg string, err error) *R2QuotaError {
	return &R2QuotaError{R2Error{Op: op, Message: msg, Err: err}}
}

// accessDenied creates an R2AccessDeniedError.
func accessDenied(op, bucket, msg string, err error) *R2AccessDeniedError {
	return &R2AccessDeniedError{R2Error{Op: op, Bucket: bucket, Message: msg, Err: err}}
}

// validationError creates an R2ValidationError.
func validationError(op, msg string) *R2ValidationError {
	return &R2ValidationError{R2Error{Op: op, Message: msg}}
}

// isNotFound reports whether err is a Cloudflare not-found error: a
// *cloudflare.Error with HTTP 404 or error code 1000 (cloudflare-go has
// no ErrNotFound sentinel), or — for non-cloudflare errors (transport,
// wrapped strings) — a message matching not-found shapes.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var cfErr *cloudflare.Error
	if errors.As(err, &cfErr) {
		if cfErr.StatusCode == http.StatusNotFound {
			return true
		}
		for _, c := range cfErr.ErrorCodes {
			if c == 1000 {
				return true
			}
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "404")
}
