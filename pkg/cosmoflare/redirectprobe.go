package cosmoflare

// RedirectProbeResult is one redirect-destination probe outcome.
type RedirectProbeResult struct {
	Destination string `json:"destination"`
	Status      int    `json:"status"`            // final HTTP status; 0 on transport error or loop
	Loop        bool   `json:"loop,omitempty"`    // a hop was revisited
	Err         string `json:"err,omitempty"`     // transport error / hop-cap message
	Skipped     bool   `json:"skipped,omitempty"` // unprobeable ($n capture reference)
}
