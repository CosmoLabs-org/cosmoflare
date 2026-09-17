package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// --- Resource-prefix scoping (FEAT-026 part 2) ---

// withEnvProfile sets the package-level ActiveProfile for the test and
// restores a nil profile afterwards.
func withEnvProfile(t *testing.T, p *config.Profile) {
	t.Helper()
	ActiveProfile = p
	t.Cleanup(func() { ActiveProfile = nil })
}

// captureCmdOutput runs fn with os.Stdout piped and returns what was
// printed. printInfo/Presenter read os.Stdout at call time, so the swap is
// visible. (Distinct from root_test.go's captureStdout, which swaps the
// global for the whole test.)
func captureCmdOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	w.Close()
	var sb strings.Builder
	if _, err := io.Copy(&sb, r); err != nil {
		t.Fatalf("read captured output: %v", err)
	}
	return sb.String()
}

func TestD1Create_AppliesResourcePrefix(t *testing.T) {
	origDryRun, origJSON := DryRun, JSONOutput
	origAccount, origToken := AccountID, APIToken
	defer func() {
		DryRun, JSONOutput = origDryRun, origJSON
		AccountID, APIToken = origAccount, origToken
	}()
	DryRun, JSONOutput = true, true
	AccountID, APIToken = "test-account", "test-token"
	withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})

	out := captureCmdOutput(t, func() {
		if err := runD1Create(d1CreateCmd, []string{"users"}); err != nil {
			t.Errorf("runD1Create(DryRun) returned error: %v", err)
		}
	})
	if !strings.Contains(out, `"stg-users"`) {
		t.Errorf("dry-run create output missing prefixed name \"stg-users\": %s", out)
	}
}

func TestD1Create_PrefixIsIdempotent(t *testing.T) {
	origDryRun, origJSON := DryRun, JSONOutput
	origAccount, origToken := AccountID, APIToken
	defer func() {
		DryRun, JSONOutput = origDryRun, origJSON
		AccountID, APIToken = origAccount, origToken
	}()
	DryRun, JSONOutput = true, true
	AccountID, APIToken = "test-account", "test-token"
	withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})

	out := captureCmdOutput(t, func() {
		if err := runD1Create(d1CreateCmd, []string{"stg-users"}); err != nil {
			t.Errorf("runD1Create(DryRun) returned error: %v", err)
		}
	})
	if strings.Contains(out, "stg-stg-users") {
		t.Errorf("prefix applied twice: %s", out)
	}
	if !strings.Contains(out, `"stg-users"`) {
		t.Errorf("already-prefixed name not preserved: %s", out)
	}
}

func TestFilterD1ByPrefix(t *testing.T) {
	databases := []*cosmoflare.D1Database{
		{Name: "stg-users"},
		{Name: "prod-users"},
	}

	withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})
	got := filterD1ByPrefix(databases)
	if len(got) != 1 || got[0].Name != "stg-users" {
		t.Errorf("filterD1ByPrefix with active prefix = %v, want [stg-users]", got)
	}

	withEnvProfile(t, nil)
	got = filterD1ByPrefix(databases)
	if len(got) != 2 {
		t.Errorf("filterD1ByPrefix with no profile = %v, want all 2", got)
	}
}

// --- --local/--remote flags and the execute alias ---

func TestD1Query_LocalRemoteFlags(t *testing.T) {
	for _, name := range []string{"local", "remote"} {
		if d1QueryCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on d1QueryCmd", name)
		}
	}
}

func TestD1Query_ExecuteAlias(t *testing.T) {
	found := false
	for _, a := range d1QueryCmd.Aliases {
		if a == "execute" {
			found = true
		}
	}
	if !found {
		t.Errorf("d1QueryCmd.Aliases = %v, want execute", d1QueryCmd.Aliases)
	}
}

func TestD1Query_LocalAndRemoteConflict(t *testing.T) {
	origLocal, origRemote, origSQL := d1Local, d1Remote, d1SQL
	defer func() { d1Local, d1Remote, d1SQL = origLocal, origRemote, origSQL }()
	d1Local, d1Remote, d1SQL = true, true, "SELECT 1"

	err := runD1Query(d1QueryCmd, []string{"any-db"})
	if err == nil || !strings.Contains(err.Error(), "use only one of --local or --remote") {
		t.Errorf("runD1Query(--local --remote) = %v, want flag-conflict error", err)
	}
}

// --- Local execution engine ---

// chdirTemp moves the test into a temp working directory so local state
// files land there, and restores the original directory afterwards.
func chdirTemp(t *testing.T) string {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

func TestLocalD1Path_EnvSelectionAndSanitization(t *testing.T) {
	chdirTemp(t)

	withEnvProfile(t, nil)
	p, err := localD1Path("team/app")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(d1StateDir, "default", "team_app.sqlite")
	if p != want {
		t.Errorf("localD1Path(no profile) = %q, want %q", p, want)
	}

	withEnvProfile(t, &config.Profile{Name: "staging"})
	p, err = localD1Path("users")
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(d1StateDir, "staging", "users.sqlite")
	if p != want {
		t.Errorf("localD1Path(staging) = %q, want %q", p, want)
	}
}

func TestExecLocalD1_RoundTrip(t *testing.T) {
	dir := chdirTemp(t)
	withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})

	if _, err := execLocalD1("users",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO users (name) VALUES (?1)",
		[]string{"alice"}); err != nil {
		t.Fatalf("create+insert: %v", err)
	}

	state := filepath.Join(dir, d1StateDir, "staging", "users.sqlite")
	if _, err := os.Stat(state); err != nil {
		t.Fatalf("state file not created at %s: %v", state, err)
	}

	results, err := execLocalD1("users", "SELECT id, name FROM users", nil)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if len(r.Rows) != 1 || r.Rows[0]["name"] != "alice" {
		t.Errorf("SELECT rows = %v, want one row with name=alice", r.Rows)
	}
	if len(r.Columns) != 2 || r.Columns[0] != "id" || r.Columns[1] != "name" {
		t.Errorf("SELECT columns = %v, want [id name]", r.Columns)
	}
}

func TestRunD1QueryLocal_JSON(t *testing.T) {
	chdirTemp(t)
	withEnvProfile(t, nil)

	origSQL, origJSON := d1SQL, JSONOutput
	defer func() { d1SQL, JSONOutput = origSQL, origJSON }()
	d1SQL, JSONOutput = "CREATE TABLE t (x INTEGER); INSERT INTO t VALUES (1)", true

	out := captureCmdOutput(t, func() {
		if err := runD1QueryLocal("json-db"); err != nil {
			t.Errorf("runD1QueryLocal returned error: %v", err)
		}
	})
	if !strings.Contains(out, `"success"`) {
		t.Errorf("--local --json output missing success field: %s", out)
	}
}
