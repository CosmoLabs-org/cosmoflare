package keychain

import (
	"fmt"
	"os"
	"sync"
	"testing"

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

// New creates a SecretStore backed by the OS keychain.
//
// The OS keychain is deliberately NOT used under `go test` or when
// COSMOFLARE_NO_KEYCHAIN=1 is set. macOS shells out to the `security` binary,
// which prints "item not found" to stderr even when the error is handled, and
// writing to the developer's real keychain from a test run is both noisy and
// invasive. In those contexts an in-memory backend is used instead, so
// credentials are gathered/managed through the project's credentials flow
// (see `/credentials`) and config file rather than the OS keychain.
func New() *SecretStore {
	if testing.Testing() || os.Getenv("COSMOFLARE_NO_KEYCHAIN") == "1" {
		return &SecretStore{backend: newMemBackend(), fallback: &noopBackend{}}
	}
	return &SecretStore{backend: &keychainBackend{}, fallback: &noopBackend{}}
}

// Available reports whether the OS keychain is in use and usable. It only probes
// the OS keychain when the OS backend is actually active — under test / no-keychain
// mode it returns false without shelling out to `security`.
func (s *SecretStore) Available() bool {
	if _, ok := s.backend.(*keychainBackend); !ok {
		return false
	}
	if err := keyring.Set(serviceName, "__probe__", "1"); err != nil {
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

// memBackend is a process-local, in-memory secret store. It never shells out to
// the OS keychain, so it produces no stderr noise and does not touch the real
// keychain. Used under `go test` and when COSMOFLARE_NO_KEYCHAIN=1.
type memBackend struct {
	mu sync.RWMutex
	m  map[string]string
}

func newMemBackend() *memBackend { return &memBackend{m: make(map[string]string)} }

func (b *memBackend) Get(profile, key string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if v, ok := b.m[itemKey(profile, key)]; ok {
		return v, nil
	}
	return "", ErrNotFound
}

func (b *memBackend) Set(profile, key, value string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[itemKey(profile, key)] = value
	return nil
}

func (b *memBackend) Delete(profile, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, itemKey(profile, key))
	return nil
}
