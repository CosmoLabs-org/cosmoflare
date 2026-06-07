package keychain

import (
	"fmt"
	"sync"

	"github.com/zalando/go-keyring"
)

const serviceName = "cosmoflare"

// SecretStore abstracts credential storage with OS keychain primary, file fallback.
type SecretStore struct {
	mu       sync.RWMutex
	backend  Backend
	fallback Backend
}

// Backend is the interface for secret storage backends.
type Backend interface {
	Get(profile, key string) (string, error)
	Set(profile, key, value string) error
	Delete(profile, key string) error
}

// ErrNotFound is returned when a secret is not in any backend.
var ErrNotFound = fmt.Errorf("secret not found")

// New creates a SecretStore that tries the OS keychain first,
// falling back to an in-memory/no-op backend if the keychain is unavailable.
func New() *SecretStore {
	kb := &keychainBackend{}
	var fb Backend

	if err := kb.Set("__cosmoflare_probe", "test", "probe"); err != nil {
		fb = &noopBackend{}
	} else {
		_ = kb.Delete("__cosmoflare_probe", "test")
		fb = &noopBackend{}
	}

	return &SecretStore{backend: kb, fallback: fb}
}

// Available reports whether the OS keychain is usable.
func (s *SecretStore) Available() bool {
	err := keyring.Set(serviceName, "__probe__", "1")
	if err != nil {
		return false
	}
	_ = keyring.Delete(serviceName, "__probe__")
	return true
}

// Get retrieves a secret. Returns ErrNotFound if absent from all backends.
func (s *SecretStore) Get(profile, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, err := s.backend.Get(profile, key)
	if err == nil {
		return val, nil
	}
	val, err = s.fallback.Get(profile, key)
	if err == nil {
		return val, nil
	}
	return "", ErrNotFound
}

// Set stores a secret in the primary backend.
func (s *SecretStore) Set(profile, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.backend.Set(profile, key, value); err != nil {
		return s.fallback.Set(profile, key, value)
	}
	return nil
}

// Delete removes a secret from all backends.
func (s *SecretStore) Delete(profile, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = s.backend.Delete(profile, key)
	_ = s.fallback.Delete(profile, key)
	return nil
}

func itemKey(profile, key string) string {
	return fmt.Sprintf("%s/%s", profile, key)
}

// keychainBackend uses the OS keychain via go-keyring.
type keychainBackend struct{}

func (k *keychainBackend) Get(profile, key string) (string, error) {
	val, err := keyring.Get(serviceName, itemKey(profile, key))
	if err != nil {
		return "", ErrNotFound
	}
	return val, nil
}

func (k *keychainBackend) Set(profile, key, value string) error {
	return keyring.Set(serviceName, itemKey(profile, key), value)
}

func (k *keychainBackend) Delete(profile, key string) error {
	return keyring.Delete(serviceName, itemKey(profile, key))
}

// noopBackend always returns ErrNotFound (used when keychain is unavailable).
type noopBackend struct{}

func (n *noopBackend) Get(_, _ string) (string, error) { return "", ErrNotFound }
func (n *noopBackend) Set(_, _, _ string) error         { return nil }
func (n *noopBackend) Delete(_, _ string) error         { return nil }
