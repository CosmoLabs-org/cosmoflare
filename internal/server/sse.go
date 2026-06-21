package server

import "net/http"

// registerSSERoutes mounts the SSE endpoints. The base implementation is a
// no-op; the real /events handler + SSE hub is implemented in G-03 (P-03).
// Keeping the call site wired in Server.registerRoutes lets each wave build
// and test in isolation.
func (s *Server) registerSSERoutes(_ *http.ServeMux) {
	// no-op until SSE lands
}

// Publish is the seam REST/health code uses to push an SSE frame to all
// subscribers on a channel. The G-03 SSE hub fans this out; until then it is a
// no-op so callers can wire it without depending on wave ordering.
func (s *Server) Publish(_ string, _ any) {
	// no-op until SSE lands
}
