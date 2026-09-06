package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSE channel names. These are a wire contract shared by this package's
// producers, the serve daemon's glue code (cmd/serve.go), and the desktop
// client — publish and subscribe through these constants, never literals,
// so a rename cannot silently orphan one side.
const (
	ChannelMetrics       = "metrics"
	ChannelNotifications = "notifications"
	ChannelStatus        = "status"
)

// eventFrame is one SSE event enqueued by Publish and drained by the /events
// handler. channel becomes the `event:` line; data is JSON-encoded as `data:`.
type eventFrame struct {
	channel string
	data    any
}

// sseHub fans out event frames to all connected /events clients. Publish is
// best-effort: a slow client with a full buffer has frames dropped (the
// alternative — blocking the publisher — would stall the whole daemon).
type sseHub struct {
	mu      sync.Mutex
	clients map[chan eventFrame]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[chan eventFrame]struct{})}
}

func (h *sseHub) subscribe() chan eventFrame {
	ch := make(chan eventFrame, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *sseHub) unsubscribe(ch chan eventFrame) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

func (h *sseHub) publish(f eventFrame) {
	h.mu.Lock()
	for ch := range h.clients {
		select {
		case ch <- f:
		default:
			// Slow client: drop the frame rather than block the publisher.
		}
	}
	h.mu.Unlock()
}

func (h *sseHub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// registerSSERoutes mounts the SSE endpoint. The hub is created once in New()
// so Publish works from the moment the server exists (REST handlers + the
// health tier call it before any client connects).
func (s *Server) registerSSERoutes(mux *http.ServeMux) {
	mux.HandleFunc("/events", s.handleEvents)
}

// Subscribers returns the number of currently-connected /events clients. It
// exists primarily so tests can wait for registration before publishing
// (defeating the publish-before-subscribe race).
func (s *Server) Subscribers() int { return s.hub.count() }

// Publish fans an event out to every connected /events client as an SSE frame:
//
//	event: <channel>
//	data: <json>
//
//	(blank line)
//
// With no clients connected the frame is dropped — it is a live push channel,
// not a durable queue. Callers (REST handlers, the health tier) never block.
func (s *Server) Publish(channel string, data any) {
	s.hub.publish(eventFrame{channel: channel, data: data})
}

// handleEvents serves the multiplexed SSE stream. A client connects once and
// receives frames from all channels (metrics / notifications / status),
// filtered client-side by the `event:` line. The connection lives until the
// client disconnects (r.Context().Done()).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	// Flush headers so the client observes Content-Type immediately, before any
	// frame is published.
	flusher.Flush()

	ch := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case f := <-ch:
			payload, err := json.Marshal(f.data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", f.channel, payload)
			flusher.Flush()
		case <-s.done:
			return
		case <-ctx.Done():
			return
		}
	}
}
