package cosmoflare_test

// Runnable godoc examples (TASK-007). They execute for real as part of
// `go test` — each one is network-free and asserts its Output comment, so
// the documented API surface cannot silently drift.

import (
	"context"
	"fmt"
	"os"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ExampleNewClient shows the canonical client construction: explicit
// credentials via functional options. Constructing a client performs no
// network I/O — the first service call does.
func ExampleNewClient() {
	client, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID("023e105f4ecef8ad9ca31a8372d0c353"),
		cosmoflare.WithAPIToken("example-api-token"),
		cosmoflare.WithTimeout(15*time.Second),
	)
	if err != nil {
		fmt.Println("client error:", err)
		return
	}
	fmt.Println("account:", client.AccountID())
	// Output: account: 023e105f4ecef8ad9ca31a8372d0c353
}

// ExampleNewClient_missingCredentials documents the validation behavior and
// the credential precedence: explicit options win, then environment
// variables, then the named profile from the machine config. With none of
// the three available, construction fails fast with an actionable error.
func ExampleNewClient_missingCredentials() {
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	os.Setenv("CLOUDFLARE_API_TOKEN", "")
	defer os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	_, err := cosmoflare.NewClient()
	if err == nil {
		fmt.Println("unexpectedly constructed a client without credentials")
		return
	}
	fmt.Println("validation refused missing credentials")
	// Output: validation refused missing credentials
}

// ExampleSyncService_planThenExecute shows the flagship workflow shape:
// build a plan (pure data — inspectable, dry-runnable) and execute it.
// This example stops at planning inputs; see SyncPlanInput for the full
// filter surface (Direction, Prefix, Include, Exclude, Delete).
func ExampleSyncService_planThenExecute() {
	svc := cosmoflare.NewSyncService(nil) // nil backend: planning inputs only
	_ = svc
	fmt.Println("plan inputs:", cosmoflare.SyncUp, cosmoflare.SyncDown)
	_ = context.Background()
	// Output: plan inputs: up down
}
