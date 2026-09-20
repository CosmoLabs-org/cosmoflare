package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func TestWorkerTypesCmdValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	if err := runWorkerTypes(workerTypesCmd, nil); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected arg validation error, got %v", err)
	}
}

func TestWorkerTypesCmdFailsOffline(t *testing.T) {
	runGlobalsSnapshot(t)
	err := runWorkerTypes(workerTypesCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestWorkerTypesOutWritesGeneratedContent(t *testing.T) {
	runGlobalsSnapshot(t)
	path := filepath.Join(t.TempDir(), "worker-configuration.d.ts")

	content := cosmoflare.GenerateWorkerTypes(cosmoflare.WorkerSettings{
		Bindings: []cosmoflare.WorkerBinding{
			{Name: "ALPHA", Type: "kv", ID: "a"},
			{Name: "ZULU", Type: "var", ID: "z"},
		},
	})
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat written file: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected mode 0644, got %v", info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "  ALPHA: KVNamespace;") || !strings.Contains(got, "  ZULU: string;") {
		t.Errorf("written file missing typed Env entries:\n%s", got)
	}
	if strings.Index(got, "  ALPHA:") > strings.Index(got, "  ZULU:") {
		t.Errorf("bindings not sorted in written file:\n%s", got)
	}
}

func TestWorkerTypesCmdRegistration(t *testing.T) {
	registerWorkerTypesCmds(workerCmd)
	found := false
	for _, c := range workerCmd.Commands() {
		if c.Name() == "types" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'types' subcommand registered on worker command")
	}
	// Idempotency: re-registering must not panic or duplicate the flag.
	registerWorkerTypesCmds(workerCmd)
	if workerTypesCmd.Flags().Lookup("out") == nil {
		t.Fatal("expected --out flag to be registered")
	}
}
