package server

import (
	"context"
	"log"
	"reflect"
	"time"
)

type MetricsSnapshot struct {
	Profile      string `json:"profile"`
	Zones        any    `json:"zones"`
	R2Buckets    any    `json:"r2_buckets"`
	Workers      any    `json:"workers"`
	KVNamespaces any    `json:"kv_namespaces"`
}

type MetricsProducer struct {
	srv      *Server
	src      ServeSource
	interval time.Duration
	last     *MetricsSnapshot
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
	snap := MetricsSnapshot{Profile: ""}
	var failed bool

	if data, err := m.src.Zones(ctx, ""); err != nil {
		log.Printf("[metrics] zones: %v", err)
		failed = true
	} else {
		snap.Zones = data
	}
	if data, err := m.src.R2Buckets(ctx, ""); err != nil {
		log.Printf("[metrics] r2: %v", err)
		failed = true
	} else {
		snap.R2Buckets = data
	}
	if data, err := m.src.Workers(ctx, ""); err != nil {
		log.Printf("[metrics] workers: %v", err)
		failed = true
	} else {
		snap.Workers = data
	}
	if data, err := m.src.KV(ctx, ""); err != nil {
		log.Printf("[metrics] kv: %v", err)
		failed = true
	} else {
		snap.KVNamespaces = data
	}

	if failed {
		return
	}
	if m.last != nil && reflect.DeepEqual(*m.last, snap) {
		return
	}
	m.last = &snap
	m.srv.Publish("metrics", snap)
}
