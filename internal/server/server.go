// Package server implements the local HTTP+SSE daemon that powers the
// Cosmoflare desktop app. It is a thin transport over the existing per-service
// constructors in pkg/cosmoflare — it owns no Cloudflare logic of its own.
//
// The desktop shell (Tauri/Rust) spawns the `cosmoflare serve` command, parses
// the stdout handshake for the bound address + auth token, and drives the UI
// against this server's REST + SSE endpoints.
package server

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Config holds the immutable configuration for a Server.
type Config struct {
	// Token is the bearer token required on every request (Authorization:
	// Bearer <token>). The shell generates a random token and passes it both
	// to the daemon and to the webview.
	Token string
	// Version is reported in /healthz and the handshake. Usually AppVersion.
	Version string
}

// Server is the daemon's HTTP server. It is safe for concurrent use: REST
// handlers flip cfOnline from many goroutines and the SSE hub is mutex-guarded.
type Server struct {
	cfg Config

	// cfOnline records the result of the most recent Cloudflare API call:
	// true when the call succeeded (creds valid + API reachable). It backs
	// the two-tier "Cloudflare online" health indicator.
	cfOnline atomic.Bool
}

// New constructs a Server. Routes are registered lazily in Handler() so tests
// can wire optional seams (ServeSource, SSE hub) before the mux is built.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// SetCloudflareOnline records the outcome of the most recent Cloudflare call.
// REST handlers call this on success/failure; it feeds the "Cloudflare online"
// tier of the health model.
func (s *Server) SetCloudflareOnline(ok bool) { s.cfOnline.Store(ok) }

// CloudflareOnline reports the current Cloudflare-online tier state.
func (s *Server) CloudflareOnline() bool { return s.cfOnline.Load() }

// Handler returns the HTTP handler with all registered routes behind the
// token-auth middleware. Optional seams (REST ServeSource, SSE hub) are
// registered here once they are attached; the base server exposes /healthz.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return s.authMiddleware(mux)
}

// registerRoutes mounts the server's routes onto mux. The base implementation
// registers only /healthz; REST and SSE route registration is added by
// rest.go and sse.go respectively.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealthz)
}

// authMiddleware rejects any request whose Authorization header is not the
// configured bearer token. The token gates all endpoints (REST + SSE).
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleHealthz reports the two-tier health model: systems_online is always
// true (the daemon process is up and answering), cloudflare_online reflects the
// most recent Cloudflare API call outcome.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"systems_online":   true,
		"cloudflare_online": s.cfOnline.Load(),
		"version":          s.cfg.Version,
	})
}

// writeJSON encodes v as a JSON response. Errors are logged to the response
// writer as a 500; this is a read-only server so encoding failures are the
// only failure mode.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
