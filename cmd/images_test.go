package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestImagesCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "images" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("imagesCmd not registered on rootCmd")
	}
}

func TestImagesCmd_Metadata(t *testing.T) {
	if imagesCmd.Use != "images" {
		t.Errorf("imagesCmd.Use = %q, want %q", imagesCmd.Use, "images")
	}
	if imagesCmd.Short == "" {
		t.Error("imagesCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestImagesCmd_Subcommands(t *testing.T) {
	expected := []string{"upload", "list", "get", "delete", "variants"}
	for _, name := range expected {
		found := false
		for _, sub := range imagesCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("images subcommand %q not registered", name)
		}
	}
}

func TestImagesVariantsCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "create", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range imagesVariantsCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("images variants subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestImagesCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		imagesUploadCmd,
		imagesListCmd,
		imagesGetCmd,
		imagesDeleteCmd,
		imagesVariantsListCmd,
		imagesVariantsCreateCmd,
		imagesVariantsDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestImagesUpload_Flags(t *testing.T) {
	expected := []string{"url", "metadata", "require-signed-urls"}
	for _, name := range expected {
		if imagesUploadCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on imagesUploadCmd", name)
		}
	}
}

func TestImagesDelete_ForceFlag(t *testing.T) {
	f := imagesDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on imagesDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestImagesVariantsCreate_Flags(t *testing.T) {
	expected := []string{"fit", "width", "height", "metadata-mode", "never-require-signed-urls"}
	for _, name := range expected {
		if imagesVariantsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on imagesVariantsCreateCmd", name)
		}
	}
}

func TestImagesVariantsCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"fit", "scale-down"},
		{"width", "0"},
		{"height", "0"},
		{"metadata-mode", "none"},
	}
	for _, tc := range cases {
		f := imagesVariantsCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on imagesVariantsCreateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Arg validation ---

func TestImagesUpload_NoFile(t *testing.T) {
	err := runImagesUpload(imagesUploadCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("file path")) {
		t.Errorf("error = %q, want it to mention 'file path'", err.Error())
	}
}

func TestImagesGet_NoID(t *testing.T) {
	err := runImagesGet(imagesGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no image ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("image ID")) {
		t.Errorf("error = %q, want it to mention 'image ID'", err.Error())
	}
}

func TestImagesDelete_NoID(t *testing.T) {
	err := runImagesDelete(imagesDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no image ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("image ID")) {
		t.Errorf("error = %q, want it to mention 'image ID'", err.Error())
	}
}

func TestImagesVariantsCreate_NoName(t *testing.T) {
	err := runImagesVariantsCreate(imagesVariantsCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no variant name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("variant name")) {
		t.Errorf("error = %q, want it to mention 'variant name'", err.Error())
	}
}

func TestImagesVariantsDelete_NoName(t *testing.T) {
	err := runImagesVariantsDelete(imagesVariantsDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no variant name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("variant name")) {
		t.Errorf("error = %q, want it to mention 'variant name'", err.Error())
	}
}

// --- DryRun mode ---

func TestImagesUpload_DryRun(t *testing.T) {
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

	err := runImagesUpload(imagesUploadCmd, []string{"/tmp/test.png"})
	if err != nil {
		t.Errorf("runImagesUpload(DryRun) returned error: %v", err)
	}
}

func TestImagesUpload_DryRunJSON(t *testing.T) {
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

	err := runImagesUpload(imagesUploadCmd, []string{"/tmp/test.png"})
	if err != nil {
		t.Errorf("runImagesUpload(DryRun+JSON) returned error: %v", err)
	}
}

func TestImagesDelete_DryRun(t *testing.T) {
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

	err := runImagesDelete(imagesDeleteCmd, []string{"img-123"})
	if err != nil {
		t.Errorf("runImagesDelete(DryRun) returned error: %v", err)
	}
}

func TestImagesDelete_DryRunJSON(t *testing.T) {
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

	err := runImagesDelete(imagesDeleteCmd, []string{"img-123"})
	if err != nil {
		t.Errorf("runImagesDelete(DryRun+JSON) returned error: %v", err)
	}
}
