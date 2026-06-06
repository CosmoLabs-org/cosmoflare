package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestStreamCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "stream" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("streamCmd not registered on rootCmd")
	}
}

func TestStreamCmd_Metadata(t *testing.T) {
	if streamCmd.Use != "stream" {
		t.Errorf("streamCmd.Use = %q, want %q", streamCmd.Use, "stream")
	}
	if streamCmd.Short == "" {
		t.Error("streamCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestStreamCmd_Subcommands(t *testing.T) {
	expected := []string{"upload", "list", "get", "delete", "token", "live"}
	for _, name := range expected {
		found := false
		for _, sub := range streamCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stream subcommand %q not registered", name)
		}
	}
}

func TestStreamLiveCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range streamLiveCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stream live subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestStreamCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		streamUploadCmd,
		streamListCmd,
		streamGetCmd,
		streamDeleteCmd,
		streamTokenCmd,
		streamLiveCreateCmd,
		streamLiveListCmd,
		streamLiveDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestStreamUpload_Flags(t *testing.T) {
	expected := []string{"url", "metadata", "watermark", "require-signed-urls"}
	for _, name := range expected {
		if streamUploadCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on streamUploadCmd", name)
		}
	}
}

func TestStreamList_StatusFlag(t *testing.T) {
	f := streamListCmd.Flags().Lookup("status")
	if f == nil {
		t.Fatal("--status flag not registered on streamListCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--status default = %q, want empty string", f.DefValue)
	}
}

func TestStreamDelete_ForceFlag(t *testing.T) {
	f := streamDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on streamDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestStreamToken_ExpiresFlag(t *testing.T) {
	f := streamTokenCmd.Flags().Lookup("expires")
	if f == nil {
		t.Fatal("--expires flag not registered on streamTokenCmd")
	}
	if f.DefValue != "1h" {
		t.Errorf("--expires default = %q, want %q", f.DefValue, "1h")
	}
}

func TestStreamLiveCreate_ModeFlag(t *testing.T) {
	f := streamLiveCreateCmd.Flags().Lookup("mode")
	if f == nil {
		t.Fatal("--mode flag not registered on streamLiveCreateCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--mode default = %q, want empty string", f.DefValue)
	}
}

func TestStreamLiveDelete_ForceFlag(t *testing.T) {
	f := streamLiveDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on streamLiveDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestStreamUpload_NoFile(t *testing.T) {
	// Reset URL flag to ensure file mode
	origURL := streamURL
	defer func() { streamURL = origURL }()
	streamURL = ""

	err := runStreamUpload(streamUploadCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("file path")) {
		t.Errorf("error = %q, want it to mention 'file path'", err.Error())
	}
}

func TestStreamGet_NoID(t *testing.T) {
	err := runStreamGet(streamGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no video ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("video ID")) {
		t.Errorf("error = %q, want it to mention 'video ID'", err.Error())
	}
}

func TestStreamDelete_NoID(t *testing.T) {
	err := runStreamDelete(streamDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no video ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("video ID")) {
		t.Errorf("error = %q, want it to mention 'video ID'", err.Error())
	}
}

func TestStreamToken_NoID(t *testing.T) {
	err := runStreamToken(streamTokenCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no video ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("video ID")) {
		t.Errorf("error = %q, want it to mention 'video ID'", err.Error())
	}
}

func TestStreamLiveCreate_NoName(t *testing.T) {
	err := runStreamLiveCreate(streamLiveCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("live input name")) {
		t.Errorf("error = %q, want it to mention 'live input name'", err.Error())
	}
}

func TestStreamLiveDelete_NoID(t *testing.T) {
	err := runStreamLiveDelete(streamLiveDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no input ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("live input ID")) {
		t.Errorf("error = %q, want it to mention 'live input ID'", err.Error())
	}
}

// --- DryRun mode ---

func TestStreamUpload_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origURL := streamURL
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		streamURL = origURL
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	streamURL = ""

	err := runStreamUpload(streamUploadCmd, []string{"/tmp/test.mp4"})
	if err != nil {
		t.Errorf("runStreamUpload(DryRun) returned error: %v", err)
	}
}

func TestStreamUpload_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origURL := streamURL
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		streamURL = origURL
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"
	streamURL = ""

	err := runStreamUpload(streamUploadCmd, []string{"/tmp/test.mp4"})
	if err != nil {
		t.Errorf("runStreamUpload(DryRun+JSON) returned error: %v", err)
	}
}

func TestStreamUploadByURL_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origURL := streamURL
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		streamURL = origURL
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	streamURL = "https://example.com/video.mp4"

	err := runStreamUpload(streamUploadCmd, []string{})
	if err != nil {
		t.Errorf("runStreamUpload(DryRun URL) returned error: %v", err)
	}
}

func TestStreamDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	err := runStreamDelete(streamDeleteCmd, []string{"vid-123"})
	if err != nil {
		t.Errorf("runStreamDelete(DryRun) returned error: %v", err)
	}
}

func TestStreamDelete_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	err := runStreamDelete(streamDeleteCmd, []string{"vid-123"})
	if err != nil {
		t.Errorf("runStreamDelete(DryRun+JSON) returned error: %v", err)
	}
}

func TestStreamToken_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origExpires := streamExpires
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		streamExpires = origExpires
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	streamExpires = "1h"

	err := runStreamToken(streamTokenCmd, []string{"vid-123"})
	if err != nil {
		t.Errorf("runStreamToken(DryRun) returned error: %v", err)
	}
}

func TestStreamLiveCreate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origMode := streamLiveMode
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		streamLiveMode = origMode
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	streamLiveMode = "automatic"

	err := runStreamLiveCreate(streamLiveCreateCmd, []string{"my-stream"})
	if err != nil {
		t.Errorf("runStreamLiveCreate(DryRun) returned error: %v", err)
	}
}

func TestStreamLiveDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	err := runStreamLiveDelete(streamLiveDeleteCmd, []string{"input-123"})
	if err != nil {
		t.Errorf("runStreamLiveDelete(DryRun) returned error: %v", err)
	}
}

// --- parseExpiresDuration ---

func TestParseExpiresDuration(t *testing.T) {
	cases := []struct {
		input   string
		wantSec float64
		wantErr bool
	}{
		{"1h", 3600, false},
		{"30m", 1800, false},
		{"24h", 86400, false},
		{"7d", 604800, false},
		{"1d", 86400, false},
		{"invalid", 0, true},
		{"xd", 0, true},
	}
	for _, tc := range cases {
		d, err := parseExpiresDuration(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseExpiresDuration(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseExpiresDuration(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if d.Seconds() != tc.wantSec {
			t.Errorf("parseExpiresDuration(%q) = %v, want %v seconds", tc.input, d.Seconds(), tc.wantSec)
		}
	}
}

