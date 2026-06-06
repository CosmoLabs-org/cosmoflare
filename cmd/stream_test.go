package cmd

import (
	"bytes"
	"strings"
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

// --- streamCmd group command has no RunE ---

func TestStreamCmd_NoRunE(t *testing.T) {
	if streamCmd.RunE != nil {
		t.Error("streamCmd.RunE should be nil (it is a command group, not a leaf command)")
	}
}

// --- streamLiveCmd metadata ---

func TestStreamLiveCmd_Metadata(t *testing.T) {
	if streamLiveCmd.Use != "live" {
		t.Errorf("streamLiveCmd.Use = %q, want %q", streamLiveCmd.Use, "live")
	}
	if streamLiveCmd.Short == "" {
		t.Error("streamLiveCmd.Short is empty")
	}
	if streamLiveCmd.Long == "" {
		t.Error("streamLiveCmd.Long is empty")
	}
}

func TestStreamLiveCmd_NoRunE(t *testing.T) {
	if streamLiveCmd.RunE != nil {
		t.Error("streamLiveCmd.RunE should be nil (it is a command group, not a leaf command)")
	}
}

// --- Subcommand metadata (Use, Short, Long) ---

func TestStreamUploadCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamUploadCmd.Use, "upload") {
		t.Errorf("streamUploadCmd.Use = %q, want prefix 'upload'", streamUploadCmd.Use)
	}
	if streamUploadCmd.Short == "" {
		t.Error("streamUploadCmd.Short is empty")
	}
	if streamUploadCmd.Long == "" {
		t.Error("streamUploadCmd.Long is empty")
	}
}

func TestStreamListCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamListCmd.Use, "list") {
		t.Errorf("streamListCmd.Use = %q, want prefix 'list'", streamListCmd.Use)
	}
	if streamListCmd.Short == "" {
		t.Error("streamListCmd.Short is empty")
	}
	if streamListCmd.Long == "" {
		t.Error("streamListCmd.Long is empty")
	}
}

func TestStreamGetCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamGetCmd.Use, "get") {
		t.Errorf("streamGetCmd.Use = %q, want prefix 'get'", streamGetCmd.Use)
	}
	if streamGetCmd.Short == "" {
		t.Error("streamGetCmd.Short is empty")
	}
	if streamGetCmd.Long == "" {
		t.Error("streamGetCmd.Long is empty")
	}
}

func TestStreamDeleteCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamDeleteCmd.Use, "delete") {
		t.Errorf("streamDeleteCmd.Use = %q, want prefix 'delete'", streamDeleteCmd.Use)
	}
	if streamDeleteCmd.Short == "" {
		t.Error("streamDeleteCmd.Short is empty")
	}
	if streamDeleteCmd.Long == "" {
		t.Error("streamDeleteCmd.Long is empty")
	}
}

func TestStreamTokenCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamTokenCmd.Use, "token") {
		t.Errorf("streamTokenCmd.Use = %q, want prefix 'token'", streamTokenCmd.Use)
	}
	if streamTokenCmd.Short == "" {
		t.Error("streamTokenCmd.Short is empty")
	}
	if streamTokenCmd.Long == "" {
		t.Error("streamTokenCmd.Long is empty")
	}
}

func TestStreamLiveCreateCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamLiveCreateCmd.Use, "create") {
		t.Errorf("streamLiveCreateCmd.Use = %q, want prefix 'create'", streamLiveCreateCmd.Use)
	}
	if streamLiveCreateCmd.Short == "" {
		t.Error("streamLiveCreateCmd.Short is empty")
	}
	if streamLiveCreateCmd.Long == "" {
		t.Error("streamLiveCreateCmd.Long is empty")
	}
}

func TestStreamLiveListCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamLiveListCmd.Use, "list") {
		t.Errorf("streamLiveListCmd.Use = %q, want prefix 'list'", streamLiveListCmd.Use)
	}
	if streamLiveListCmd.Short == "" {
		t.Error("streamLiveListCmd.Short is empty")
	}
}

func TestStreamLiveDeleteCmd_Metadata(t *testing.T) {
	if !strings.HasPrefix(streamLiveDeleteCmd.Use, "delete") {
		t.Errorf("streamLiveDeleteCmd.Use = %q, want prefix 'delete'", streamLiveDeleteCmd.Use)
	}
	if streamLiveDeleteCmd.Short == "" {
		t.Error("streamLiveDeleteCmd.Short is empty")
	}
}

// --- Help text content ---

func TestStreamCmd_LongContainsExamples(t *testing.T) {
	long := streamCmd.Long
	examples := []string{"cosmoflare stream upload", "cosmoflare stream list", "cosmoflare stream get"}
	for _, ex := range examples {
		if !strings.Contains(long, ex) {
			t.Errorf("streamCmd.Long does not contain example %q", ex)
		}
	}
}

func TestStreamLiveCmd_LongMentionsRecordingModes(t *testing.T) {
	long := streamLiveCmd.Long
	if !strings.Contains(long, "off") {
		t.Error("streamLiveCmd.Long does not mention 'off' recording mode")
	}
	if !strings.Contains(long, "automatic") {
		t.Error("streamLiveCmd.Long does not mention 'automatic' recording mode")
	}
}

func TestStreamTokenCmd_LongMentionsDurationFormats(t *testing.T) {
	long := streamTokenCmd.Long
	if !strings.Contains(long, "1h") {
		t.Error("streamTokenCmd.Long does not mention '1h' duration example")
	}
}

func TestStreamDeleteCmd_LongMentionsWarning(t *testing.T) {
	long := streamDeleteCmd.Long
	if !strings.Contains(strings.ToUpper(long), "WARNING") && !strings.Contains(long, "irreversible") {
		t.Error("streamDeleteCmd.Long should warn about irreversibility")
	}
}

// --- Flag defaults and types ---

func TestStreamUpload_FlagDefaults(t *testing.T) {
	tests := []struct {
		flag     string
		defValue string
	}{
		{"url", ""},
		{"metadata", ""},
		{"watermark", ""},
		{"require-signed-urls", "false"},
	}
	for _, tc := range tests {
		f := streamUploadCmd.Flags().Lookup(tc.flag)
		if f == nil {
			t.Errorf("flag --%s not found", tc.flag)
			continue
		}
		if f.DefValue != tc.defValue {
			t.Errorf("--%s default = %q, want %q", tc.flag, f.DefValue, tc.defValue)
		}
	}
}

func TestStreamUpload_FlagTypes(t *testing.T) {
	stringFlags := []string{"url", "metadata", "watermark"}
	for _, name := range stringFlags {
		f := streamUploadCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag --%s not found", name)
			continue
		}
		if f.Value.Type() != "string" {
			t.Errorf("--%s type = %q, want %q", name, f.Value.Type(), "string")
		}
	}
	boolFlag := streamUploadCmd.Flags().Lookup("require-signed-urls")
	if boolFlag == nil {
		t.Fatal("--require-signed-urls not found")
	}
	if boolFlag.Value.Type() != "bool" {
		t.Errorf("--require-signed-urls type = %q, want %q", boolFlag.Value.Type(), "bool")
	}
}

func TestStreamDelete_FlagType(t *testing.T) {
	f := streamDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force not found on streamDeleteCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--force type = %q, want %q", f.Value.Type(), "bool")
	}
}

func TestStreamToken_FlagType(t *testing.T) {
	f := streamTokenCmd.Flags().Lookup("expires")
	if f == nil {
		t.Fatal("--expires not found on streamTokenCmd")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--expires type = %q, want %q", f.Value.Type(), "string")
	}
}

func TestStreamLiveCreate_FlagType(t *testing.T) {
	f := streamLiveCreateCmd.Flags().Lookup("mode")
	if f == nil {
		t.Fatal("--mode not found on streamLiveCreateCmd")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--mode type = %q, want %q", f.Value.Type(), "string")
	}
}

func TestStreamLiveDelete_FlagType(t *testing.T) {
	f := streamLiveDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force not found on streamLiveDeleteCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--force type = %q, want %q", f.Value.Type(), "bool")
	}
}

// --- DryRun+JSON variants for remaining commands ---

func TestStreamToken_DryRunJSON(t *testing.T) {
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
	JSONOutput = true
	APIToken = "test-token"
	streamExpires = "24h"

	err := runStreamToken(streamTokenCmd, []string{"vid-abc"})
	if err != nil {
		t.Errorf("runStreamToken(DryRun+JSON) returned error: %v", err)
	}
}

func TestStreamLiveCreate_DryRunJSON(t *testing.T) {
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
	JSONOutput = true
	APIToken = "test-token"
	streamLiveMode = ""

	err := runStreamLiveCreate(streamLiveCreateCmd, []string{"my-stream"})
	if err != nil {
		t.Errorf("runStreamLiveCreate(DryRun+JSON, no mode) returned error: %v", err)
	}
}

func TestStreamLiveDelete_DryRunJSON(t *testing.T) {
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

	err := runStreamLiveDelete(streamLiveDeleteCmd, []string{"input-abc"})
	if err != nil {
		t.Errorf("runStreamLiveDelete(DryRun+JSON) returned error: %v", err)
	}
}

func TestStreamUploadByURL_DryRunJSON(t *testing.T) {
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
	streamURL = "https://example.com/video.mp4"

	err := runStreamUpload(streamUploadCmd, []string{})
	if err != nil {
		t.Errorf("runStreamUploadByURL(DryRun+JSON) returned error: %v", err)
	}
}

// --- DryRun with mode printed ---

func TestStreamLiveCreate_DryRunWithMode(t *testing.T) {
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
	streamLiveMode = "off"

	err := runStreamLiveCreate(streamLiveCreateCmd, []string{"my-stream"})
	if err != nil {
		t.Errorf("runStreamLiveCreate(DryRun, mode=off) returned error: %v", err)
	}
}

// --- Invalid inputs ---

func TestStreamToken_InvalidExpires(t *testing.T) {
	origDryRun := DryRun
	origExpires := streamExpires
	defer func() {
		DryRun = origDryRun
		streamExpires = origExpires
	}()

	DryRun = true
	streamExpires = "notaduration"

	err := runStreamToken(streamTokenCmd, []string{"vid-123"})
	if err == nil {
		t.Fatal("expected error for invalid --expires value")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %q, want mention of 'invalid'", err.Error())
	}
}

func TestStreamUpload_FileNotFound(t *testing.T) {
	origDryRun := DryRun
	origURL := streamURL
	defer func() {
		DryRun = origDryRun
		streamURL = origURL
	}()

	DryRun = false
	streamURL = ""

	err := runStreamUpload(streamUploadCmd, []string{"/nonexistent/path/video.mp4"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error = %q, want mention of 'file not found'", err.Error())
	}
}

// --- parseExpiresDuration edge cases ---

func TestParseExpiresDuration_ZeroDays(t *testing.T) {
	d, err := parseExpiresDuration("0d")
	if err != nil {
		t.Errorf("parseExpiresDuration(\"0d\") unexpected error: %v", err)
	}
	if d.Seconds() != 0 {
		t.Errorf("parseExpiresDuration(\"0d\") = %v, want 0", d)
	}
}

func TestParseExpiresDuration_LargeDays(t *testing.T) {
	d, err := parseExpiresDuration("365d")
	if err != nil {
		t.Errorf("parseExpiresDuration(\"365d\") unexpected error: %v", err)
	}
	want := float64(365 * 24 * 3600)
	if d.Seconds() != want {
		t.Errorf("parseExpiresDuration(\"365d\") = %v, want %v", d.Seconds(), want)
	}
}

func TestParseExpiresDuration_Minutes(t *testing.T) {
	d, err := parseExpiresDuration("90m")
	if err != nil {
		t.Errorf("parseExpiresDuration(\"90m\") unexpected error: %v", err)
	}
	if d.Seconds() != 5400 {
		t.Errorf("parseExpiresDuration(\"90m\") = %v, want 5400", d.Seconds())
	}
}

func TestParseExpiresDuration_EmptyString(t *testing.T) {
	_, err := parseExpiresDuration("")
	if err == nil {
		t.Error("parseExpiresDuration(\"\") expected error for empty string")
	}
}

// --- StreamCmd Long content ---

func TestStreamCmd_LongMentionsLiveCommands(t *testing.T) {
	long := streamCmd.Long
	if !strings.Contains(long, "live") {
		t.Error("streamCmd.Long should mention 'live' subcommand")
	}
}

func TestStreamCmd_LongMentionsToken(t *testing.T) {
	long := streamCmd.Long
	if !strings.Contains(long, "token") {
		t.Error("streamCmd.Long should mention 'token' subcommand")
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

