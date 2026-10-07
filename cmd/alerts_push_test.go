package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/alertspush"
)

// setupAlertsPushTestEnv points alertsPushStorePathFn at a temp push.json
// path (same seam pattern as getAlertServiceFn), resets the command's flag
// globals, and returns the path.
func setupAlertsPushTestEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "push.json")
	origPathFn := alertsPushStorePathFn
	alertsPushStorePathFn = func() (string, error) { return path, nil }
	origStdin := alertsPushStdin
	alertsPushStdin = false
	t.Cleanup(func() {
		alertsPushStorePathFn = origPathFn
		alertsPushStdin = origStdin
	})
	return path
}

// alertsPushSetJSON pins the global JSON output mode for a test.
func alertsPushSetJSON(t *testing.T, on bool) {
	t.Helper()
	old := JSONOutput
	JSONOutput = on
	t.Cleanup(func() { JSONOutput = old })
}

// executeAlertsPushCommand runs the root command against args, capturing
// cobra's out/err buffers plus the real stdout the print* helpers write to.
// stdin, when non-nil, is wired as the command's input.
func executeAlertsPushCommand(t *testing.T, stdin *bytes.Buffer, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	if stdin != nil {
		rootCmd.SetIn(bytes.NewReader(stdin.Bytes()))
		defer rootCmd.SetIn(nil)
	}
	rootCmd.SetArgs(args)
	var execErr error
	stdout := capturePrint(t, func() { execErr = rootCmd.Execute() })
	return buf.String() + stdout, execErr
}

// alertsPushEnvelope is the standard success envelope the push commands
// emit in --json mode; Data is decoded loosely so each command's shape can
// be asserted independently.
type alertsPushEnvelope struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Error   string         `json:"error"`
	Data    map[string]any `json:"data"`
}

func alertsPushDecode(t *testing.T, out string) alertsPushEnvelope {
	t.Helper()
	var env alertsPushEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("decode envelope: %v\noutput: %s", err, out)
	}
	return env
}

// alertsPushDataSubs extracts the subscriptions array from list output,
// failing the test when the key is absent or null — an empty store must
// still render a JSON [] so agents can iterate without nil checks.
func alertsPushDataSubs(t *testing.T, env alertsPushEnvelope) []alertspush.Subscription {
	t.Helper()
	raw, ok := env.Data["subscriptions"]
	if !ok {
		t.Fatalf("list data missing \"subscriptions\" key: %v", env.Data)
	}
	arr, ok := raw.([]any)
	if !ok {
		t.Fatalf("\"subscriptions\" is %T, want array (null is not acceptable)", raw)
	}
	subs := make([]alertspush.Subscription, 0, len(arr))
	for _, item := range arr {
		m, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("re-marshal subscription: %v", err)
		}
		var s alertspush.Subscription
		if err := json.Unmarshal(m, &s); err != nil {
			t.Fatalf("decode subscription: %v", err)
		}
		subs = append(subs, s)
	}
	return subs
}

func TestAlertsPushKeygenTwiceSameKey(t *testing.T) {
	path := setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "keygen", "--json")
	if err != nil {
		t.Fatalf("keygen #1: %v\n%s", err, out)
	}
	env := alertsPushDecode(t, out)
	if !env.Success {
		t.Fatalf("keygen #1 not successful: %+v", env)
	}
	key1, _ := env.Data["public_key"].(string)
	if key1 == "" {
		t.Fatalf("keygen #1 returned empty public_key: %v", env.Data)
	}

	out, err = executeAlertsPushCommand(t, nil, "alerts", "push", "keygen", "--json")
	if err != nil {
		t.Fatalf("keygen #2: %v\n%s", err, out)
	}
	env = alertsPushDecode(t, out)
	key2, _ := env.Data["public_key"].(string)
	if key2 != key1 {
		t.Fatalf("keygen is not idempotent: %q != %q", key1, key2)
	}

	// The store must exist on disk with the 0600 mode the store contract
	// requires (it holds the VAPID private key).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat push store: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("push store mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestAlertsPushKeygenKeepsSubscriptions(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	sub := `{"endpoint":"https://push.example/e1","p256dh":"k","auth":"a"}`
	if _, err := executeAlertsPushCommand(t, nil, "alerts", "push", "add", sub, "--json"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := executeAlertsPushCommand(t, nil, "alerts", "push", "keygen", "--json"); err != nil {
		t.Fatalf("keygen: %v", err)
	}
	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "list", "--json")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	subs := alertsPushDataSubs(t, alertsPushDecode(t, out))
	if len(subs) != 1 || subs[0].Endpoint != "https://push.example/e1" {
		t.Fatalf("keygen disturbed subscriptions: %+v", subs)
	}
}

func TestAlertsPushListEmptyGoldenShape(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "list", "--json")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	env := alertsPushDecode(t, out)
	if !env.Success {
		t.Fatalf("list not successful: %+v", env)
	}
	if pk, _ := env.Data["public_key"].(string); pk != "" {
		t.Errorf("public_key = %q, want empty on a fresh store", pk)
	}
	if subs := alertsPushDataSubs(t, env); len(subs) != 0 {
		t.Errorf("subscriptions = %+v, want empty array", subs)
	}
}

func TestAlertsPushListOneSubGoldenShape(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	sub := `{"endpoint":"https://push.example/e1","p256dh":"BPk2","auth":"aGVsbG8"}`
	if _, err := executeAlertsPushCommand(t, nil, "alerts", "push", "add", sub, "--json"); err != nil {
		t.Fatalf("add: %v", err)
	}
	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "list", "--json")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	env := alertsPushDecode(t, out)
	subs := alertsPushDataSubs(t, env)
	if len(subs) != 1 {
		t.Fatalf("subscriptions = %+v, want exactly one", subs)
	}
	got := subs[0]
	if got.Endpoint != "https://push.example/e1" || got.P256dh != "BPk2" || got.Auth != "aGVsbG8" {
		t.Errorf("subscription round-trip mismatch: %+v", got)
	}
}

func TestAlertsPushAddMissingFieldError(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	_, err := executeAlertsPushCommand(t, nil, "alerts", "push", "add",
		`{"endpoint":"https://push.example/e1","auth":"a"}`, "--json")
	if err == nil {
		t.Fatal("add with missing p256dh must fail")
	}
	for _, want := range []string{"p256dh", "endpoint", "auth"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
	if !strings.Contains(err.Error(), "p256dh") {
		t.Errorf("error must name the missing field, got: %v", err)
	}
}

func TestAlertsPushAddInvalidJSONError(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	_, err := executeAlertsPushCommand(t, nil, "alerts", "push", "add", "not-json", "--json")
	if err == nil {
		t.Fatal("add with invalid JSON must fail")
	}
	if !strings.Contains(err.Error(), "subscription JSON") {
		t.Errorf("error %q does not mention the subscription JSON", err.Error())
	}
}

func TestAlertsPushAddStdin(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	in := bytes.NewBufferString(`{"endpoint":"https://push.example/e2","p256dh":"k2","auth":"a2"}`)
	out, err := executeAlertsPushCommand(t, in, "alerts", "push", "add", "--stdin", "--json")
	if err != nil {
		t.Fatalf("add --stdin: %v\n%s", err, out)
	}
	env := alertsPushDecode(t, out)
	if !env.Success {
		t.Fatalf("add --stdin not successful: %+v", env)
	}
	if ep, _ := env.Data["endpoint"].(string); ep != "https://push.example/e2" {
		t.Errorf("add data endpoint = %v, want the added subscription", env.Data)
	}
}

func TestAlertsPushRemoveFlow(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, true)

	if _, err := executeAlertsPushCommand(t, nil, "alerts", "push", "add",
		`{"endpoint":"https://push.example/e1","p256dh":"k","auth":"a"}`, "--json"); err != nil {
		t.Fatalf("add: %v", err)
	}

	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "remove", "https://push.example/e1", "--json")
	if err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	if env := alertsPushDecode(t, out); !env.Success {
		t.Fatalf("remove not successful: %+v", env)
	}

	out, err = executeAlertsPushCommand(t, nil, "alerts", "push", "list", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if subs := alertsPushDataSubs(t, alertsPushDecode(t, out)); len(subs) != 0 {
		t.Fatalf("subscriptions after remove = %+v, want empty", subs)
	}

	_, err = executeAlertsPushCommand(t, nil, "alerts", "push", "remove", "https://push.example/e1", "--json")
	if err == nil {
		t.Fatal("removing an unknown endpoint must fail")
	}
	if !strings.Contains(err.Error(), "no push subscription") {
		t.Errorf("error %q should say no push subscription was found", err.Error())
	}
}

func TestAlertsPushHumanOutputSmoke(t *testing.T) {
	setupAlertsPushTestEnv(t)
	alertsPushSetJSON(t, false)

	out, err := executeAlertsPushCommand(t, nil, "alerts", "push", "keygen")
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	if !strings.Contains(out, "Public key") {
		t.Errorf("human keygen output missing public key section:\n%s", out)
	}
	if !strings.Contains(out, "alerts push add") {
		t.Errorf("human keygen output missing pairing instructions:\n%s", out)
	}

	out, err = executeAlertsPushCommand(t, nil, "alerts", "push", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "keygen") {
		t.Errorf("human list output should hint at pairing flow:\n%s", out)
	}
}
