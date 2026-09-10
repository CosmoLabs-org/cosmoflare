package knowledge

import (
	"errors"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// KnowledgeError decorates a Cloudflare API error with the knowledge
// layer's cause and fix.
type KnowledgeError struct {
	Code    int
	Message string
	Cause   string
	Fix     string
	Err     error
}

func (e *KnowledgeError) Error() string {
	return fmt.Sprintf("cloudflare error %d (%s): %s — fix: %s", e.Code, e.Message, e.Cause, e.Fix)
}

func (e *KnowledgeError) Unwrap() error { return e.Err }

// DecodeCFError wraps err in a *KnowledgeError when a decode exists for its
// code (context refines: "phase-entrypoint" for entrypoint 1000s). Unknown
// errors return unchanged.
func DecodeCFError(err error, context string) error {
	if err == nil {
		return nil
	}
	var cfErr *cloudflare.Error
	if !errors.As(err, &cfErr) {
		return err
	}
	for _, code := range cfErr.ErrorCodes {
		if d := LookupDecode(code, context); d != nil {
			msg := ""
			if len(cfErr.ErrorMessages) > 0 {
				msg = cfErr.ErrorMessages[0]
			}
			return &KnowledgeError{Code: code, Message: msg, Cause: d.Cause, Fix: d.Fix, Err: err}
		}
	}
	return err
}
