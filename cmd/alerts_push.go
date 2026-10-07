package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/alertspush"
)

// alertsPushStdin reads the subscription JSON from stdin instead of an
// argument (alerts push add --stdin).
var alertsPushStdin bool

// alertsPushStorePathFn resolves the push.json store path. It is a package
// var so tests can point it at a temp dir (same seam as getAlertServiceFn).
// The path reuses internal/config's ~/.cosmoflare resolution rather than
// re-deriving HOME: push.json lives next to config.yaml in ~/.cosmoflare.
var alertsPushStorePathFn = func() (string, error) {
	configPath, err := config.ConfigPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve ~/.cosmoflare directory: %w", err)
	}
	return filepath.Join(filepath.Dir(configPath), "push.json"), nil
}

// loadAlertsPushStore loads the push store at the resolved path. A missing
// file is an empty store (LoadStore contract), so first-run flows need no
// special casing.
func loadAlertsPushStore() (*alertspush.Store, string, error) {
	path, err := alertsPushStorePathFn()
	if err != nil {
		return nil, "", err
	}
	st, err := alertspush.LoadStore(path)
	if err != nil {
		return nil, "", err
	}
	return st, path, nil
}

// alertsPushSubShape is the subscription JSON shape quoted in every
// agent-readable add error.
const alertsPushSubShape = `{"endpoint":"https://…","p256dh":"…","auth":"…"}`

var alertsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Manage Web Push (VAPID) delivery for the Cosmoflare pager",
	Long: `Manage the Web Push (VAPID) side of the Cosmoflare pager: generate the
server keypair, register browser push subscriptions, list and remove them.

State is local: keys and subscriptions live in ~/.cosmoflare/push.json
(file mode 0600). Nothing is sent anywhere until 'cosmoflare alerts watch'
fires a push; the pager clients never see your Cloudflare credentials.

Pairing flow (one device):
  1. cosmoflare alerts push keygen
  2. Paste the printed public key into the pager PWA settings and enable
     notifications
  3. Copy the subscription JSON the PWA shows
  4. cosmoflare alerts push add '<subscription JSON>'

Commands:
  keygen   Generate the VAPID keypair (idempotent)
  add      Register a push subscription
  list     List registered subscriptions
  remove   Remove a subscription by endpoint

Examples:
  cosmoflare alerts push keygen
  cosmoflare alerts push add '{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh":"BPk…","auth":"aGVsbG8="}'
  cat subscription.json | cosmoflare alerts push add --stdin
  cosmoflare alerts push list --json
  cosmoflare alerts push remove https://fcm.googleapis.com/fcm/send/abc`,
	// Push commands are purely local (~/.cosmoflare/push.json) — pairing
	// must work before any Cloudflare account is configured, so the root
	// credential validation is intentionally not inherited here.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

var alertsPushKeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate the VAPID keypair for push delivery (idempotent)",
	Long: `Generate the VAPID keypair used to sign Web Push messages and save it to
~/.cosmoflare/push.json (file mode 0600 — the private key never leaves the
machine).

Idempotent: if a keypair already exists it is kept (rotating it would
silently break every paired device) and subscriptions are preserved.

The public key is printed for the one-time pairing of a pager client.

Examples:
  cosmoflare alerts push keygen
  cosmoflare alerts push keygen --json`,
	RunE: runAlertsPushKeygen,
}

var alertsPushAddCmd = &cobra.Command{
	Use:   "add [SUBSCRIPTION_JSON]",
	Short: "Register a browser push subscription",
	Long: `Register a Web Push subscription so pushes are delivered to that device.

The subscription is the JSON produced by the browser's
pushManager.subscribe() — the pager PWA pairing view shows it ready to
copy. Pass it as an argument or read it from stdin with --stdin.

Expected shape (all fields required):
  ` + alertsPushSubShape + `

Examples:
  cosmoflare alerts push add '{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh":"BPk…","auth":"aGVsbG8="}'
  cat subscription.json | cosmoflare alerts push add --stdin
  cosmoflare alerts push add --stdin < subscription.json --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAlertsPushAdd,
}

var alertsPushListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered push subscriptions and the VAPID public key",
	Long: `List every push subscription registered for this machine plus the VAPID
public key pager clients need to subscribe.

Examples:
  cosmoflare alerts push list
  cosmoflare alerts push list --json`,
	RunE: runAlertsPushList,
}

var alertsPushRemoveCmd = &cobra.Command{
	Use:   "remove <ENDPOINT>",
	Short: "Remove a push subscription by endpoint",
	Long: `Remove the push subscription whose endpoint matches ENDPOINT exactly.
Use 'cosmoflare alerts push list' to see the registered endpoints. Removing
an already-expired endpoint is the manual counterpart of the automatic
404/410 pruning performed when pushes are dispatched.

Examples:
  cosmoflare alerts push remove https://fcm.googleapis.com/fcm/send/abc
  cosmoflare alerts push remove https://fcm.googleapis.com/fcm/send/abc --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsPushRemove,
}

func init() {
	alertsCmd.AddCommand(alertsPushCmd)

	alertsPushCmd.AddCommand(alertsPushKeygenCmd)
	alertsPushCmd.AddCommand(alertsPushAddCmd)
	alertsPushCmd.AddCommand(alertsPushListCmd)
	alertsPushCmd.AddCommand(alertsPushRemoveCmd)

	alertsPushAddCmd.Flags().BoolVar(&alertsPushStdin, "stdin", false, "Read the subscription JSON from stdin instead of an argument")
}

func runAlertsPushKeygen(cmd *cobra.Command, args []string) error {
	st, path, err := loadAlertsPushStore()
	if err != nil {
		return outErr("failed to load push store", err)
	}
	existed := st.HasVAPIDKeys()
	if err := st.Keygen(path); err != nil {
		return outErr("failed to generate VAPID keys", err)
	}
	return outPayload("VAPID keys ready", func() any {
		return map[string]string{
			"public_key": st.VAPIDPublicKey,
			"store_path": path,
		}
	}, func() {
		if existed {
			printSuccess("VAPID keypair already exists — keeping existing keys")
		} else {
			printSuccess("VAPID keypair generated and saved to %s", path)
		}
		fmt.Printf("\nPublic key (share with your pager clients):\n  %s\n", st.VAPIDPublicKey)
		fmt.Printf("\nOne-time pairing:\n")
		fmt.Printf("  1. Open the pager PWA settings and paste the public key.\n")
		fmt.Printf("  2. Enable notifications, then copy the subscription JSON it shows.\n")
		fmt.Printf("  3. Run: cosmoflare alerts push add '<subscription JSON>'\n")
	})
}

func runAlertsPushAdd(cmd *cobra.Command, args []string) error {
	if alertsPushStdin && len(args) > 0 {
		return fmt.Errorf("pass the subscription either as an argument or with --stdin, not both\n\nExpected shape: %s", alertsPushSubShape)
	}

	var raw string
	switch {
	case alertsPushStdin:
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return outErr("failed to read subscription JSON from stdin", err)
		}
		raw = string(data)
	case len(args) == 1:
		raw = args[0]
	default:
		return fmt.Errorf("missing subscription JSON\n\nExpected shape: %s\nCopy it from the pager PWA pairing view (\"Copy subscription\"), then run:\n  cosmoflare alerts push add '<subscription JSON>'\n  cat subscription.json | cosmoflare alerts push add --stdin", alertsPushSubShape)
	}

	var sub alertspush.Subscription
	if err := json.Unmarshal([]byte(raw), &sub); err != nil {
		return fmt.Errorf("invalid subscription JSON: %v\n\nExpected shape: %s\nCopy it from the pager PWA pairing view (\"Copy subscription\")", err, alertsPushSubShape)
	}

	var missing []string
	if strings.TrimSpace(sub.Endpoint) == "" {
		missing = append(missing, "endpoint")
	}
	if strings.TrimSpace(sub.P256dh) == "" {
		missing = append(missing, "p256dh")
	}
	if strings.TrimSpace(sub.Auth) == "" {
		missing = append(missing, "auth")
	}
	if len(missing) > 0 {
		return fmt.Errorf("subscription JSON is missing required field(s): %s\n\nExpected shape: %s\nEach field is required: endpoint (https push-service URL), p256dh (client public key), auth (auth secret)", strings.Join(missing, ", "), alertsPushSubShape)
	}
	if !strings.HasPrefix(sub.Endpoint, "https://") && !strings.HasPrefix(sub.Endpoint, "http://") {
		return fmt.Errorf("subscription endpoint %q is not a valid push-service URL\n\nExpected shape: %s\nThe endpoint is the https URL the browser's pushManager.subscribe() returned", sub.Endpoint, alertsPushSubShape)
	}

	st, path, err := loadAlertsPushStore()
	if err != nil {
		return outErr("failed to load push store", err)
	}
	if err := st.AddSubscription(sub); err != nil {
		return outErr("failed to register subscription", err)
	}
	if err := st.Save(path); err != nil {
		return outErr("failed to save push store", err)
	}

	return outPayload("Push subscription added", func() any {
		return sub
	}, func() {
		printSuccess("Push subscription added (%d registered)", len(st.Subscriptions))
		if !st.HasVAPIDKeys() {
			printInfo("No VAPID keys yet — run 'cosmoflare alerts push keygen' so pager clients can subscribe")
		}
	})
}

func runAlertsPushList(cmd *cobra.Command, args []string) error {
	st, _, err := loadAlertsPushStore()
	if err != nil {
		return outErr("failed to load push store", err)
	}
	subs := st.Subscriptions
	if subs == nil {
		subs = []alertspush.Subscription{}
	}
	return outPayload("Push subscriptions", func() any {
		return map[string]any{
			"public_key":    st.VAPIDPublicKey,
			"subscriptions": subs,
		}
	}, func() {
		if len(subs) == 0 {
			printInfo("No push subscriptions registered. Pair a device: 'cosmoflare alerts push keygen', then 'cosmoflare alerts push add'.")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ENDPOINT\tP256DH\tAUTH")
		for _, s := range subs {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				truncate(s.Endpoint, 60), truncate(s.P256dh, 16), truncate(s.Auth, 16))
		}
		w.Flush()
	})
}

func runAlertsPushRemove(cmd *cobra.Command, args []string) error {
	endpoint := args[0]

	st, path, err := loadAlertsPushStore()
	if err != nil {
		return outErr("failed to load push store", err)
	}
	if !st.RemoveSubscription(endpoint) {
		return fmt.Errorf("no push subscription with endpoint %q\n\nRun 'cosmoflare alerts push list' to see the registered endpoints", endpoint)
	}
	if err := st.Save(path); err != nil {
		return outErr("failed to save push store", err)
	}

	return outPayload("Push subscription removed", func() any {
		return map[string]string{"endpoint": endpoint}
	}, func() {
		printSuccess("Push subscription removed (%d remaining)", len(st.Subscriptions))
	})
}
