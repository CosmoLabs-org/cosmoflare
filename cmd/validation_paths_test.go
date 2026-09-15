package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// emailStreamImagesValidationCases drives the arg-validation head of every
// email/stream/images run function with no arguments — each must fail fast
// with its own message before touching credentials or the network.
var emailStreamImagesValidationCases = []struct {
	name string
	fn   func(cmd *cobra.Command, args []string) error
	want string
}{
	{"email rules list", runEmailRulesList, "zone ID is required"},
	{"email rules get", runEmailRulesGet, "zone ID and rule ID are required"},
	{"email rules create", runEmailRulesCreate, "zone ID is required"},
	{"email rules update", runEmailRulesUpdate, "zone ID and rule ID are required"},
	{"email rules delete", runEmailRulesDelete, "zone ID and rule ID are required"},
	{"email dest list", runEmailDestList, "zone ID is required"},
	{"email dest add", runEmailDestAdd, "zone ID is required"},
	{"email dest get", runEmailDestGet, "zone ID and address ID are required"},
	{"email dest delete", runEmailDestDelete, "zone ID and address ID are required"},
	{"email catchall", runEmailCatchall, "zone ID is required"},
	{"email catchall update", runEmailCatchallUpdate, "zone ID is required"},
	{"email settings", runEmailSettings, "zone ID is required"},
	{"email enable", runEmailEnable, "zone ID is required"},
	{"email disable", runEmailDisable, "zone ID is required"},
	{"images upload", runImagesUpload, "required"},
	{"images list", runImagesList, "required"},
	{"images get", runImagesGet, "image ID is required"},
	{"images delete", runImagesDelete, "image ID is required"},
	{"images variants list", runImagesVariantsList, "required"},
	{"images variants create", runImagesVariantsCreate, "variant name is required"},
	{"images variants delete", runImagesVariantsDelete, "variant name is required"},
	{"stream upload", runStreamUpload, "required"},
	{"stream list", runStreamList, "required"},
	{"stream get", runStreamGet, "video ID is required"},
	{"stream delete", runStreamDelete, "video ID is required"},
	{"stream token", runStreamToken, "video ID is required"},
	{"stream live create", runStreamLiveCreate, "live input name is required"},
	{"stream live list", runStreamLiveList, "required"},
	{"stream live delete", runStreamLiveDelete, "live input ID is required"},
	{"ssl status", runSSLStatus, "zone ID is required"},
	{"ssl settings", runSSLSettings, "zone ID is required"},
	{"ssl update", runSSLUpdate, "zone ID is required"},
	{"ssl verify", runSSLVerify, "zone ID is required"},
	{"waf packages", runWAFPackages, "zone ID is required"},
	{"waf rules", runWAFRules, "zone ID and package ID are required"},
	{"waf rule", runWAFRule, "zone ID, package ID, and rule ID are required"},
	{"waf access list", runWAFAccessList, "zone ID is required"},
	{"waf access create", runWAFAccessCreate, "zone ID is required"},
	{"waf access delete", runWAFAccessDelete, "zone ID and rule ID are required"},
	{"dns create", runDNSCreate, "zone ID is required"},
	{"dns list", runDNSList, "zone ID is required"},
	{"dns get", runDNSGet, "zone ID and record ID are required"},
	{"dns update", runDNSUpdate, "zone ID and record ID are required"},
	{"dns delete", runDNSDelete, "zone ID and record ID are required"},
}

// TestServiceBackedRunsFailOffline drives the bucket/object read paths with
// valid arguments but no credentials: the run functions must fail at client
// construction with the library's deterministic validation error, before any
// network dial. Covers the flag-parse and client-gate statements offline.
func TestServiceBackedRunsFailOffline(t *testing.T) {
	oldAcct, oldToken := AccountID, APIToken
	AccountID, APIToken = "", ""
	_ = os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	_ = os.Unsetenv("CLOUDFLARE_API_TOKEN")
	t.Cleanup(func() {
		AccountID, APIToken = oldAcct, oldToken
	})

	cases := []struct {
		name string
		fn   func(cmd *cobra.Command, args []string) error
		cmd  *cobra.Command
		args []string
	}{
		{"bucket list", runBucketList, bucketListCmd, nil},
		{"bucket get", runBucketGet, bucketGetCmd, []string{"my-bucket"}},
		{"object list", runObjectList, objectListCmd, []string{"my-bucket"}},
		{"object head", runObjectHead, objectHeadCmd, []string{"my-bucket", "k.txt"}},
		{"object search", runObjectSearch, objectSearchCmd, []string{"my-bucket", "q"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := c.fn(c.cmd, c.args)
			if err == nil {
				t.Fatal("expected client-construction error, got nil")
			}
			if !strings.Contains(err.Error(), "CLOUDFLARE_ACCOUNT_ID is required") {
				t.Fatalf("error %q does not contain the credential validation message", err.Error())
			}
		})
	}
}

func TestEmailStreamImagesValidationHeads(t *testing.T) {
	for _, c := range emailStreamImagesValidationCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := c.fn(nil, nil)
			if err == nil {
				t.Fatalf("expected a validation error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), c.want)
			}
		})
	}
}

// TestConfirmDeletionDrivesLoop drives the delete confirmation loop with
// scripted stdin: cancel, delete, and an invalid answer followed by delete.
func TestConfirmDeletionDrivesLoop(t *testing.T) {
	feed := func(t *testing.T, input string) bool {
		t.Helper()
		old := os.Stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("pipe: %v", err)
		}
		if _, err := w.WriteString(input); err != nil {
			t.Fatalf("write stdin: %v", err)
		}
		_ = w.Close()
		os.Stdin = r
		t.Cleanup(func() { os.Stdin = old })
		return confirmDeletion("test-bucket")
	}

	if feed(t, "cancel\n") {
		t.Fatal("cancel must not confirm")
	}
	if !feed(t, "DELETE\n") {
		t.Fatal("DELETE must confirm")
	}
	if !feed(t, "maybe\nDELETE\n") {
		t.Fatal("invalid input then DELETE must confirm")
	}
}
