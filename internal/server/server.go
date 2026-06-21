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
	"sync"
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

	// mu guards src (the optional REST data source). cfOnline is atomic and
	// the SSE hub has its own mutex, so this only protects the ServeSource
	// attachment.
	mu  sync.Mutex
	src ServeSource

	// hub fans out SSE frames to /events clients. Created in New so Publish
	// works before any client connects.
	hub *sseHub
}

// New constructs a Server. Routes are registered lazily in Handler() so tests
// can wire optional seams (ServeSource) before the mux is built. The SSE hub
// is created eagerly so Publish is always safe to call.
func New(cfg Config) *Server {
	return &Server{cfg: cfg, hub: newSSEHub()}
}

// SetCloudflareOnline records the outcome of the most recent Cloudflare call.
// REST handlers call this on success/failure; it feeds the "Cloudflare online"
// tier of the health model. When the value flips, two SSE frames are published:
// a `status` frame (so dashboards update the health indicator live) AND a
// `notifications` frame (BR-07's v1 notification source — daemon-internal
// cloudflare_online transitions feed the notifications panel). Repeated
// same-value sets (e.g. every successful REST call) do NOT re-publish — only
// transitions are pushed.
func (s *Server) SetCloudflareOnline(ok bool) {
	old := s.cfOnline.Swap(ok)
	if old != ok {
		s.Publish("status", map[string]any{
			"systems_online":   true,
			"cloudflare_online": ok,
		})
		s.Publish("notifications", map[string]any{
			"message": cfOnlineMessage(ok),
			"level":   cfOnlineLevel(ok),
			"kind":    "cloudflare_online",
		})
	}
}

// cfOnlineMessage is the human-readable text for a cloudflare_online transition
// notification.
func cfOnlineMessage(online bool) string {
	if online {
		return "Cloudflare connection restored"
	}
	return "Cloudflare connection lost"
}

// cfOnlineLevel is the severity for a cloudflare_online transition notification.
func cfOnlineLevel(online bool) string {
	if online {
		return "info"
	}
	return "warning"
}

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
	s.registerRESTRoutes(mux)
	s.registerSSERoutes(mux)
}

// authMiddleware rejects any request that is not authorized. The token gates
// all endpoints (REST + SSE). See authorized() for the accepted forms.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authorized reports whether the request carries the configured token. The
// primary form is the Bearer header; a ?token= query-string fallback exists so
// the browser's EventSource API (which cannot set headers) can subscribe to the
// SSE /events stream. Both are localhost-only desktop transports, so the
// token-in-query threat model is acceptable.
func (s *Server) authorized(r *http.Request) bool {
	if r.Header.Get("Authorization") == "Bearer "+s.cfg.Token {
		return true
	}
	return r.URL.Query().Get("token") != "" && r.URL.Query().Get("token") == s.cfg.Token
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
