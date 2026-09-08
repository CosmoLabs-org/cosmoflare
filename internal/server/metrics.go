package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"
)

// MetricsSnapshot is one poll of account state. Sources that fail leave
// their field unset and carry the error message in Errors — a partial
// snapshot publishes instead of no snapshot.
type MetricsSnapshot struct {
	Profile      string            `json:"profile"`
	CollectedAt  time.Time         `json:"collected_at"`
	Zones        any               `json:"zones,omitempty"`
	R2Buckets    any               `json:"r2_buckets,omitempty"`
	Workers      any               `json:"workers,omitempty"`
	KVNamespaces any               `json:"kv_namespaces,omitempty"`
	Errors       map[string]string `json:"errors,omitempty"`
}

func (s *MetricsSnapshot) recordError(source string, err error) {
	if s.Errors == nil {
		s.Errors = make(map[string]string)
	}
	s.Errors[source] = err.Error()
}

type MetricsProducer struct {
	srv       *Server
	src       ServeSource
	interval  time.Duration
	lastBytes []byte
}

func NewMetricsProducer(srv *Server, src ServeSource, interval time.Duration) *MetricsProducer {
	return &MetricsProducer{srv: srv, src: src, interval: interval}
}

func (m *MetricsProducer) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *MetricsProducer) run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.srv.Subscribers() == 0 {
				continue
			}
			m.poll(ctx)
		}
	}
}

func (m *MetricsProducer) poll(ctx context.Context) {
	snap := MetricsSnapshot{
		Profile:     m.src.CurrentProfileName(),
		CollectedAt: time.Now().UTC(),
	}

	if data, err := m.src.Zones(ctx, ""); err != nil {
		log.Printf("[metrics] zones: %v", err)
		snap.recordError("zones", err)
	} else {
		snap.Zones = data
	}
	if data, err := m.src.R2Buckets(ctx, ""); err != nil {
		log.Printf("[metrics] r2: %v", err)
		snap.recordError("r2_buckets", err)
	} else {
		snap.R2Buckets = data
	}
	if data, err := m.src.Workers(ctx, ""); err != nil {
		log.Printf("[metrics] workers: %v", err)
		snap.recordError("workers", err)
	} else {
		snap.Workers = data
	}
	if data, err := m.src.KV(ctx, ""); err != nil {
		log.Printf("[metrics] kv: %v", err)
		snap.recordError("kv_namespaces", err)
	} else {
		snap.KVNamespaces = data
	}

	// Change detection compares serialized bytes with CollectedAt zeroed —
	// the collection timestamp changes every poll and must not count as a
	// data change.
	probe := snap
	probe.CollectedAt = time.Time{}
	data, err := json.Marshal(probe)
	if err != nil {
		log.Printf("[metrics] marshal: %v", err)
		return
	}
	if m.lastBytes != nil && bytes.Equal(m.lastBytes, data) {
		return
	}
	m.lastBytes = data
	m.srv.Publish(ChannelMetrics, snap)
}
