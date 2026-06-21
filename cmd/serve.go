package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/server"
)

var (
	serveAddr  string // --addr, default 127.0.0.1:0 (ephemeral port)
	serveToken string // --token, optional; generated if empty
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local HTTP+SSE daemon for the Cosmoflare desktop app",
	Long: `Start the local daemon the Cosmoflare desktop app connects to.

The daemon binds a localhost address (default 127.0.0.1:0 — the OS picks an
ephemeral port) and protects every endpoint with a bearer token. On startup it
prints a single JSON handshake line to stdout:

  {"addr":"127.0.0.1:54123","token":"<random>"}

The desktop shell (Tauri) parses this line for the address + token, then drives
the webview against the daemon's REST + SSE endpoints:

  GET /healthz                   two-tier health (systems + Cloudflare online)
  GET /accounts|/zones|/r2/buckets|/workers|/kv
  GET /events                    SSE: metrics / notifications / status

Credentials are resolved lazily per request from the config profile selected
via ?profile=<name>. The daemon starts with NO valid credentials and reports
cloudflare_online=false so the first-run setup screen can run. It never probes
the OS keychain (COSMOFLARE_NO_KEYCHAIN=1 is set in-process).

Examples:
  cosmoflare serve                                # ephemeral port, random token
  cosmoflare serve --addr 127.0.0.1:8421          # fixed port
  cosmoflare serve --token $(openssl rand -hex 16) # caller-supplied token`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:0", "address to bind (host:port; port 0 = ephemeral)")
	serveCmd.Flags().StringVar(&serveToken, "token", "", "bearer auth token (random if empty)")
	rootCmd.AddCommand(serveCmd)
}

// runServe starts the daemon. It must run without valid Cloudflare credentials
// (cmd/root.go's PersistentPreRun skips validation for "serve"), so it is
// registered there. The daemon resolves creds per-request from the selected
// config profile.
func runServe(cmd *cobra.Command, args []string) error {
	// Hard keychain gate (BR-04): never probe the OS keychain. The desktop
	// shell also sets this in the child env; setting it in-process is
	// belt-and-suspenders and makes `cosmoflare serve` safe to run directly.
	os.Setenv("COSMOFLARE_NO_KEYCHAIN", "1")

	token := serveToken
	if token == "" {
		generated, err := randomToken()
		if err != nil {
			return fmt.Errorf("generate token: %w", err)
		}
		token = generated
	}

	ln, err := net.Listen("tcp", serveAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", serveAddr, err)
	}

	srv := server.New(server.Config{Token: token, Version: AppVersion})
	httpServer := &http.Server{Handler: srv.Handler()}

	// Handshake: ONE JSON line on stdout. Everything else (logging, errors)
	// goes to stderr so the shell can parse stdout cleanly.
	handshake := map[string]string{
		"addr":  ln.Addr().String(),
		"token": token,
	}
	if err := json.NewEncoder(os.Stdout).Encode(handshake); err != nil {
		return fmt.Errorf("write handshake: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		_ = httpServer.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// randomToken returns 32 hex chars (128 bits) of cryptographic randomness.
func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
