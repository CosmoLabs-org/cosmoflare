// Package alertspush owns the VAPID keypair, push subscriptions and the
// push.json store backing Cosmoflare Pager (FEAT-045) web push delivery.
package alertspush

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SherClockHolmes/webpush-go"
)

// Subscription is a single Web Push endpoint registration as delivered by a
// browser pushManager.subscribe() call.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// Store is the persisted pager state: the VAPID keypair plus the list of
// subscribed devices. It is stored as JSON at ~/.cosmoflare/push.json with
// file mode 0600.
type Store struct {
	VAPIDPublicKey  string         `json:"vapid_public_key"`
	VAPIDPrivateKey string         `json:"vapid_private_key"`
	Subscriptions   []Subscription `json:"subscriptions"`
}

// LoadStore reads the store at path. A missing file yields an empty store
// with no error, so first-run flows need no special casing.
func LoadStore(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("read push store: %w", err)
	}
	s := &Store{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse push store %s: %w", path, err)
	}
	return s, nil
}

// Save persists the store at path, creating parent directories as needed.
// The file is written with mode 0600 — it holds the VAPID private key.
func (s *Store) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create push store dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode push store: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write push store: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write push store: %w", err)
	}
	return nil
}

// Keygen generates and persists a VAPID keypair if the store has none.
// It is idempotent: existing keys are never rotated and subscriptions are
// kept untouched.
func (s *Store) Keygen(path string) error {
	if s.HasVAPIDKeys() {
		return nil
	}
	// webpush.GenerateVAPIDKeys returns (privateKey, publicKey).
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return fmt.Errorf("generate VAPID keys: %w", err)
	}
	s.VAPIDPrivateKey = privateKey
	s.VAPIDPublicKey = publicKey
	return s.Save(path)
}

// AddSubscription registers sub, replacing any existing subscription with
// the same endpoint (keys may have been refreshed).
func (s *Store) AddSubscription(sub Subscription) error {
	for i, existing := range s.Subscriptions {
		if existing.Endpoint == sub.Endpoint {
			s.Subscriptions[i] = sub
			return nil
		}
	}
	s.Subscriptions = append(s.Subscriptions, sub)
	return nil
}

// RemoveSubscription deletes the subscription with the given endpoint and
// reports whether one was removed.
func (s *Store) RemoveSubscription(endpoint string) bool {
	for i, existing := range s.Subscriptions {
		if existing.Endpoint == endpoint {
			s.Subscriptions = append(s.Subscriptions[:i], s.Subscriptions[i+1:]...)
			return true
		}
	}
	return false
}

// HasVAPIDKeys reports whether the store holds a VAPID keypair.
func (s *Store) HasVAPIDKeys() bool {
	return s.VAPIDPublicKey != "" && s.VAPIDPrivateKey != ""
}
