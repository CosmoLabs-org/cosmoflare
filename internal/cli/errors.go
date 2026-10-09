/*
Package cli — stable machine-readable error codes (FEAT-p8KYM5K wave 1).

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cli

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ErrorCode is a stable, machine-readable error classification surfaced in
// the --json error envelope as "error_code". The string values are an API
// contract: agents and scripts match on them, so they must never change
// once shipped (FEAT-p8KYM5K wave 1).
type ErrorCode string

// Stable error codes. Full coverage of all 40+ commands is a follow-up
// wave; this wave ships the mechanism plus auth/config and the r2, kv,
// dns, d1, and workers command groups.
const (
	// CodeAuthMissingToken: no API token was provided (flag, env, or profile).
	CodeAuthMissingToken ErrorCode = "AUTH_MISSING_TOKEN"
	// CodeAuthInvalidToken: the token was rejected (401/403) or fails
	// local format validation.
	CodeAuthInvalidToken ErrorCode = "AUTH_INVALID_TOKEN"
	// CodeConfigInvalid: configuration exists but is malformed or names
	// something unknown (e.g. an invalid --env profile).
	CodeConfigInvalid ErrorCode = "CONFIG_INVALID"
	// CodeConfigMissing: a required configuration value is absent
	// (e.g. no Cloudflare Account ID).
	CodeConfigMissing ErrorCode = "CONFIG_MISSING"
	// CodeNetworkError: transport-level failure (DNS, dial, timeout, TLS).
	CodeNetworkError ErrorCode = "NETWORK_ERROR"
	// CodeAPIError: the Cloudflare API (or fallback) failed for any other
	// reason — the default classification for unclassified errors.
	CodeAPIError ErrorCode = "API_ERROR"
	// CodeResourceNotFound: the requested resource does not exist.
	CodeResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	// CodeValidationError: the request input is invalid.
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
)

// CodedError attaches a stable ErrorCode to an underlying error without
// altering its message: Error() delegates to the wrapped error, so human
// output is byte-identical to the uncoded error.
type CodedError struct {
	Code ErrorCode
	Err  error
}

// Error implements the error interface, preserving the wrapped message.
func (e *CodedError) Error() string {
	if e.Err == nil {
		return string(e.Code)
	}
	return e.Err.Error()
}

// Unwrap exposes the wrapped error for errors.Is / errors.As.
func (e *CodedError) Unwrap() error { return e.Err }

// NewCoded wraps err with a stable machine-readable code.
func NewCoded(code ErrorCode, err error) *CodedError {
	return &CodedError{Code: code, Err: err}
}

// CodeFor returns the stable ErrorCode for err: an explicit CodedError code
// anywhere in the wrap chain wins; otherwise the error is classified from
// its type (shared pkg/cosmoflare error taxonomy, HTTP status, net.Error)
// or, last resort, from its message. Plain errors fall back to CodeAPIError.
func CodeFor(err error) ErrorCode {
	if err == nil {
		return CodeAPIError
	}

	// Explicit code anywhere in the chain wins.
	var coded *CodedError
	if errors.As(err, &coded) {
		return coded.Code
	}

	// Shared service-layer taxonomy (pkg/cosmoflare).
	var notFound *cosmoflare.R2NotFoundError
	if errors.As(err, &notFound) {
		return CodeResourceNotFound
	}
	var authErr *cosmoflare.R2AuthError
	if errors.As(err, &authErr) {
		return CodeAuthInvalidToken
	}
	var denied *cosmoflare.R2AccessDeniedError
	if errors.As(err, &denied) {
		return CodeAuthInvalidToken
	}
	var validation *cosmoflare.R2ValidationError
	if errors.As(err, &validation) {
		return CodeValidationError
	}

	// HTTP status carried by the chain (cloudflare-go, AWS SDK, R2Error).
	if status := cosmoflare.ErrorStatus(err); status != 0 {
		switch {
		case status == http.StatusUnauthorized || status == http.StatusForbidden:
			return CodeAuthInvalidToken
		case status == http.StatusNotFound:
			return CodeResourceNotFound
		case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
			return CodeValidationError
		}
	}

	// Transport-level failures.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return CodeNetworkError
	}

	// Message heuristics mirror pkg/cosmoflare's not-found classification
	// for errors that reach the CLI as plain wrapped strings.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not found"),
		strings.Contains(msg, "could not find"),
		strings.Contains(msg, "404"):
		return CodeResourceNotFound
	case strings.Contains(msg, "unauthorized"),
		strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "authentication"):
		return CodeAuthInvalidToken
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "tls:"):
		return CodeNetworkError
	}

	return CodeAPIError
}

// Envelope builds the --json error envelope (FEAT-p8KYM5K wave 1): the
// historical OutputResponse error shape (success/error, plus dry_run only
// when active, matching its omitempty) EXTENDED with the stable
// machine-readable error_code. No existing fields are removed, so
// human-mode output is untouched and JSON consumers see the same keys plus
// the new one.
func Envelope(code ErrorCode, message string, dryRun bool) map[string]any {
	env := map[string]any{
		"success":    false,
		"error":      message,
		"error_code": string(code),
	}
	if dryRun {
		env["dry_run"] = true
	}
	return env
}

// compile-time interface check
var _ error = (*CodedError)(nil)
