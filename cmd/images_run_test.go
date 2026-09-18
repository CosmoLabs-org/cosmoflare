package cmd

import (
	"strings"
	"testing"
)

// imagesRunSnapshot snapshots the credential, output and images flag variables
// so tests cannot leak state into each other.
func imagesRunSnapshot(t *testing.T) {
	t.Helper()
	savedAccount, savedToken := AccountID, APIToken
	savedDry, savedJSON := DryRun, JSONOutput
	savedURL := imagesURL
	t.Cleanup(func() {
		AccountID, APIToken = savedAccount, savedToken
		DryRun, JSONOutput = savedDry, savedJSON
		imagesURL = savedURL
	})
}

// TestRunImagesUploadByURL_DryRun verifies the URL upload path short-circuits
// in dry-run mode without credentials and without contacting the API.
func TestRunImagesUploadByURL_DryRun(t *testing.T) {
	imagesRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = true
	imagesURL = "https://example.com/pic.png"

	if err := runImagesUploadByURL(imagesUploadCmd); err != nil {
		t.Fatalf("dry-run URL upload returned error: %v", err)
	}
}

// TestRunImagesUploadByURL_ServiceCreationFailsOffline verifies the URL upload
// path aborts with a wrapped service error when credentials are empty.
func TestRunImagesUploadByURL_ServiceCreationFailsOffline(t *testing.T) {
	imagesRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false
	imagesURL = "https://example.com/pic.png"

	err := runImagesUploadByURL(imagesUploadCmd)
	if err == nil || !strings.Contains(err.Error(), "failed to create Images service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunImagesList_ServiceCreationFailsOffline verifies the list runner
// aborts with a wrapped service error before any network call.
func TestRunImagesList_ServiceCreationFailsOffline(t *testing.T) {
	imagesRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runImagesList(imagesListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create Images service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunImagesVariantsList_ServiceCreationFailsOffline verifies the variants
// list runner aborts with a wrapped service error before any network call.
func TestRunImagesVariantsList_ServiceCreationFailsOffline(t *testing.T) {
	imagesRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runImagesVariantsList(imagesVariantsListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create Images service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}
