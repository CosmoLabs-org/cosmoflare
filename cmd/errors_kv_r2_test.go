package cmd

import (
	"errors"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli"
)

// FEAT-058 wave 2a: the service/client construction failures carry stable
// machine-readable codes in --json mode. credErrorCode is the classifier
// shared by getKVService and getAPIClient; the runner tests below assert
// the codes survive to the JSON envelope.
func TestCredErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want cli.ErrorCode
	}{
		{"missing token", errors.New("API token is required"), cli.CodeAuthMissingToken},
		{"token mention wins over account", errors.New("api token or account id missing"), cli.CodeAuthMissingToken},
		{"missing account", errors.New("account ID is required"), cli.CodeConfigMissing},
		{"malformed client", errors.New("invalid base64 credentials"), cli.CodeAuthInvalidToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := credErrorCode(tc.err); got != tc.want {
				t.Errorf("credErrorCode(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestKVServiceConstruction_ErrorCodes(t *testing.T) {
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() { AccountID, APIToken = savedAccount, savedToken })

	// Pure credential validation — no ambient state, deterministic in any
	// suite order (runners additionally validate args first, which makes
	// runner-level assertions order-fragile).
	AccountID, APIToken = "acct", ""
	if _, err := getKVService(); err == nil {
		t.Fatal("missing token: expected error, got nil")
	} else {
		var coded *cli.CodedError
		if !errors.As(err, &coded) {
			t.Fatalf("missing token: not a CodedError: %v", err)
		}
		if coded.Code != cli.CodeAuthMissingToken {
			t.Errorf("missing token: code = %v, want AUTH_MISSING_TOKEN", coded.Code)
		}
	}

	AccountID, APIToken = "", ""
	if _, err := getKVService(); err == nil {
		t.Fatal("missing account: expected error, got nil")
	} else {
		var coded *cli.CodedError
		if !errors.As(err, &coded) {
			t.Fatalf("missing account: not a CodedError: %v", err)
		}
		if coded.Code != cli.CodeConfigMissing && coded.Code != cli.CodeAuthMissingToken {
			t.Errorf("missing account: code = %v, want CONFIG_MISSING or AUTH_MISSING_TOKEN", coded.Code)
		}
	}
}
