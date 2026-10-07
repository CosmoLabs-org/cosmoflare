package alertspush

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStore_MissingFileIsEmpty(t *testing.T) {
	s, err := LoadStore(filepath.Join(t.TempDir(), "push.json"))
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if s == nil || len(s.Subscriptions) != 0 || s.HasVAPIDKeys() {
		t.Fatal("want empty store")
	}
}

func TestKeygen_PersistsKeysAndKeepsSubscriptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "push.json")
	s, _ := LoadStore(path)
	if err := s.Keygen(path); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	if !s.HasVAPIDKeys() {
		t.Fatal("keys not generated")
	}
	reloaded, _ := LoadStore(path)
	if reloaded.VAPIDPublicKey != s.VAPIDPublicKey {
		t.Fatal("keys not persisted")
	}
	reloaded.AddSubscription(Subscription{Endpoint: "https://push.example/e1", P256dh: "k", Auth: "a"})
	if err := reloaded.Save(path); err != nil {
		t.Fatal(err)
	}
	if err := reloaded.Keygen(path); err != nil {
		t.Fatal(err)
	}
	if reloaded.VAPIDPublicKey != s.VAPIDPublicKey {
		t.Fatal("Keygen must not rotate existing keys")
	}
	if len(reloaded.Subscriptions) != 1 {
		t.Fatal("Keygen dropped subscriptions")
	}
}

func TestSave_FileMode0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "push.json")
	s, _ := LoadStore(path)
	s.AddSubscription(Subscription{Endpoint: "e", P256dh: "k", Auth: "a"})
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestAddSubscription_DedupesByEndpoint(t *testing.T) {
	s := &Store{}
	s.AddSubscription(Subscription{Endpoint: "e1", P256dh: "a", Auth: "a"})
	s.AddSubscription(Subscription{Endpoint: "e1", P256dh: "b", Auth: "b"})
	if len(s.Subscriptions) != 1 || s.Subscriptions[0].P256dh != "b" {
		t.Fatal("dedupe/replace by endpoint failed")
	}
	if !s.RemoveSubscription("e1") || s.RemoveSubscription("e1") {
		t.Fatal("remove semantics wrong")
	}
}
