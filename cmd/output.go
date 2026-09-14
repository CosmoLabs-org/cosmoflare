package cmd

import "fmt"

// Presenter renders command output in the active output mode (FEAT-040).
//
// It exists to collapse the per-command `if JSONOutput { ... } else { ... }`
// duplication (618 sites at audit time) into one unconditional call per
// concern. Commands build one presenter per run and call Error/Result
// without branching on the mode themselves.
//
// Contracts preserved from the legacy branches:
//   - Error (plain): returns a formatted error — the command exits non-zero.
//   - Error (JSON): prints the standard error envelope and returns
//     printErrorJSON's result (nil when the print succeeds — the historical
//     exit-0-in-JSON-mode behavior; changing that is a separate decision).
//   - Result (JSON): marshals data; the human renderer is NEVER invoked in
//     JSON mode so tables never pollute machine-readable stdout.
type Presenter struct {
	json bool
}

// NewPresenter captures the output mode at construction time. Construct it
// after flag parsing (inside RunE), not at init time.
func NewPresenter() *Presenter {
	return &Presenter{json: JSONOutput}
}

// IsJSON reports whether the presenter runs in machine mode.
func (p *Presenter) IsJSON() bool { return p.json }

// Error renders a command error in the active mode.
func (p *Presenter) Error(format string, args ...any) error {
	if p.json {
		return printErrorJSON(fmt.Sprintf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

// ErrorWrap renders a wrapped upstream error (the `%w` legacy shape) in the
// active mode: "msg: err" both ways, %w-preserved for errors.Is/As in plain
// mode.
func (p *Presenter) ErrorWrap(msg string, err error) error {
	if p.json {
		return printErrorJSON(fmt.Sprintf("%s: %v", msg, err))
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Result renders a success payload: data marshalled in JSON mode, the human
// renderer invoked in plain mode. Returns the print error, if any.
func (p *Presenter) Result(data any, human func()) error {
	if p.json {
		return printJSON(data)
	}
	human()
	return nil
}

// SuccessPayload collapses the shaped-payload species: the json closure
// builds the command's typed output and is invoked ONLY in JSON mode (the
// standard success envelope wraps it); the human renderer runs ONLY in
// plain mode. Both closures stay at the call site, the mode branch lives
// here — once.
func (p *Presenter) SuccessPayload(message string, json func() any, human func()) error {
	if p.json {
		return printSuccessJSON(message, json())
	}
	human()
	return nil
}

// Success renders a human success message, or the standard success envelope
// in JSON mode.
func (p *Presenter) Success(message string, data any) error {
	if p.json {
		return printSuccessJSON(message, data)
	}
	if data == nil {
		printSuccess("%s", message)
		return nil
	}
	printSuccess("%s", message)
	return nil
}
