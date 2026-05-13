package r2go2

import (
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func TestNewKVServiceValidation(t *testing.T) {
	_, err := NewKVService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewKVService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewKVService(nil, "")
	if err == nil {
		t.Error("expected error when both are empty")
	}
}

func TestNewKVServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewKVService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestKVOptions(t *testing.T) {
	cfg := &kvConfig{}
	WithKVTTL(3600)(cfg)
	if cfg.ttl != 3600 {
		t.Errorf("expected ttl=3600, got %d", cfg.ttl)
	}

	meta := map[string]string{"env": "prod", "version": "2"}
	WithKVMetadata(meta)(cfg)
	if cfg.metadata["env"] != "prod" {
		t.Errorf("expected metadata env=prod, got %s", cfg.metadata["env"])
	}
}

func TestKVListOptions(t *testing.T) {
	cfg := &kvListConfig{}
	WithKVPrefix("cache/")(cfg)
	if cfg.prefix != "cache/" {
		t.Errorf("expected prefix=cache/, got %s", cfg.prefix)
	}

	WithKVLimit(100)(cfg)
	if cfg.limit != 100 {
		t.Errorf("expected limit=100, got %d", cfg.limit)
	}

	WithKVCursor("abc123")(cfg)
	if cfg.cursor != "abc123" {
		t.Errorf("expected cursor=abc123, got %s", cfg.cursor)
	}
}

func TestKVNamespaceType(t *testing.T) {
	ns := &KVNamespace{ID: "ns-abc", Title: "my-namespace"}
	if ns.ID != "ns-abc" {
		t.Errorf("unexpected ID: %s", ns.ID)
	}
	if ns.Title != "my-namespace" {
		t.Errorf("unexpected title: %s", ns.Title)
	}
}

func TestKVKeyType(t *testing.T) {
	k := &KVKey{Key: "user:123", Expiration: 1735689600}
	if k.Key != "user:123" {
		t.Errorf("unexpected key: %s", k.Key)
	}
	if k.Expiration != 1735689600 {
		t.Errorf("unexpected expiration: %d", k.Expiration)
	}
}

func TestKVCreateNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.CreateNamespace(nil, "")
	if err == nil {
		t.Error("expected error when title is empty")
	}
	if !strings.Contains(err.Error(), "namespace title is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKVGetNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.GetNamespace(nil, "")
	if err == nil {
		t.Error("expected error when ID is empty")
	}
}

func TestKVDeleteNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.DeleteNamespace(nil, "")
	if err == nil {
		t.Error("expected error when ID is empty")
	}
}

func TestKVPutValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.Put(nil, "", "key", strings.NewReader("val"))
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	err = svc.Put(nil, "ns-1", "", strings.NewReader("val"))
	if err == nil {
		t.Error("expected error when key is empty")
	}

	err = svc.Put(nil, "ns-1", "key", nil)
	if err == nil {
		t.Error("expected error when value is nil")
	}
}

func TestKVGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.Get(nil, "", "key")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	_, err = svc.Get(nil, "ns-1", "")
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestKVDeleteKeyValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.Delete(nil, "", "key")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	err = svc.Delete(nil, "ns-1", "")
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestKVListKeysValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.ListKeys(nil, "")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}
}
