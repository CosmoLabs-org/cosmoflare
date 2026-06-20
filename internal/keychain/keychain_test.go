package keychain

import (
	"testing"
)

func TestNoopBackend(t *testing.T) {
	b := &noopBackend{}

	_, err := b.Get("prof", "key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	if err := b.Set("prof", "key", "val"); err != nil {
		t.Errorf("Set should not error: %v", err)
	}

	if err := b.Delete("prof", "key"); err != nil {
		t.Errorf("Delete should not error: %v", err)
	}
}

func TestItemKey(t *testing.T) {
	got := itemKey("production", "api_token")
	want := "production/api_token"
	if got != want {
		t.Errorf("itemKey() = %q, want %q", got, want)
	}
}

func TestSecretStoreFallback(t *testing.T) {
	store := &SecretStore{
		backend:  &failingBackend{},
		fallback: &memoryBackend{store: make(map[string]string)},
	}

	if err := store.Set("prof", "token", "secret123"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := store.Get("prof", "token")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "secret123" {
		t.Errorf("Get = %q, want %q", val, "secret123")
	}

	if err := store.Delete("prof", "token"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("prof", "token")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSecretStorePrimaryBackend(t *testing.T) {
	primary := &memoryBackend{store: make(map[string]string)}
	store := &SecretStore{
		backend:  primary,
		fallback: &noopBackend{},
	}

	if err := store.Set("p1", "key1", "val1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := store.Get("p1", "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "val1" {
		t.Errorf("Get = %q, want %q", val, "val1")
	}
}

func TestSecretStoreNotFound(t *testing.T) {
	store := &SecretStore{
		backend:  &noopBackend{},
		fallback: &noopBackend{},
	}

	_, err := store.Get("nonexistent", "key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestNew_NoOSKeychainUnderTest verifies that New() never touches the OS
// keychain when running under `go test`: Available() must report false (so no
// shelling out to macOS `security`), yet the in-memory backend must still
// round-trip secrets so credential-dependent tests keep working.
func TestNew_NoOSKeychainUnderTest(t *testing.T) {
	s := New()

	if s.Available() {
		t.Fatal("Available() must be false under test — the OS keychain must not be used")
	}
	if _, ok := s.backend.(*memBackend); !ok {
		t.Fatalf("expected in-memory backend under test, got %T", s.backend)
	}

	if err := s.Set("profile", "api_token", "secret123"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("profile", "api_token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "secret123" {
		t.Errorf("Get = %q, want %q", got, "secret123")
	}

	if err := s.Delete("profile", "api_token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("profile", "api_token"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// failingBackend always fails (simulates unavailable keychain).
type failingBackend struct{}

func (f *failingBackend) Get(_, _ string) (string, error) { return "", ErrNotFound }
func (f *failingBackend) Set(_, _, _ string) error         { return ErrNotFound }
func (f *failingBackend) Delete(_, _ string) error         { return ErrNotFound }

// memoryBackend stores secrets in memory (for tests).
type memoryBackend struct {
	store map[string]string
}

func (m *memoryBackend) Get(profile, key string) (string, error) {
	v, ok := m.store[itemKey(profile, key)]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *memoryBackend) Set(profile, key, value string) error {
	m.store[itemKey(profile, key)] = value
	return nil
}

func (m *memoryBackend) Delete(profile, key string) error {
	delete(m.store, itemKey(profile, key))
	return nil
}
