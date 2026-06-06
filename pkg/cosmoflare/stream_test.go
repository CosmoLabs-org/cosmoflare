package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func streamMockSetup(handler http.HandlerFunc) (*StreamService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewStreamService(cf, "acct-stream-123")
	return svc, server
}

func streamWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewStreamServiceValidation(t *testing.T) {
	_, err := NewStreamService(nil, "acct123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewStreamService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewStreamServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewStreamService(cf, "acct123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

func TestNewStreamServiceFromCredsValidation(t *testing.T) {
	_, err := NewStreamServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewStreamServiceFromCreds("acct123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewStreamServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewStreamServiceFromCreds("acct123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

// --- Method validation ---

func TestStreamGetVideoValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.GetVideo(context.Background(), "")
	if err == nil {
		t.Error("expected error when video ID is empty")
	}
}

func TestStreamDeleteVideoValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteVideo(context.Background(), "")
	if err == nil {
		t.Error("expected error when video ID is empty")
	}
}

func TestStreamUploadByURLValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.UploadByURL(context.Background(), "")
	if err == nil {
		t.Error("expected error when URL is empty")
	}
}

func TestStreamUploadFileValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.UploadFile(context.Background(), "", nil)
	if err == nil {
		t.Error("expected error when file path is empty")
	}
}

func TestStreamCreateSignedTokenValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.CreateSignedToken(context.Background(), "", 3600)
	if err == nil {
		t.Error("expected error when video ID is empty")
	}
}

func TestStreamCreateLiveInputValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.CreateLiveInput(context.Background(), "", "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestStreamDeleteLiveInputValidation(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteLiveInput(context.Background(), "")
	if err == nil {
		t.Error("expected error when input ID is empty")
	}
}

// --- Mock server tests ---

func TestStreamListVideosSuccess(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"uid":               "vid-001",
					"readyToStream":     true,
					"requireSignedURLs": false,
					"duration":          120.5,
					"size":              1048576,
					"status":            map[string]interface{}{"state": "ready"},
					"created":           now.Format(time.RFC3339),
					"playback": map[string]interface{}{
						"hls":  "https://example.com/vid-001/manifest.m3u8",
						"dash": "https://example.com/vid-001/manifest.mpd",
					},
				},
				{
					"uid":               "vid-002",
					"readyToStream":     false,
					"requireSignedURLs": true,
					"duration":          0,
					"status":            map[string]interface{}{"state": "processing", "pctComplete": "45"},
					"created":           now.Format(time.RFC3339),
				},
			},
		})
	})
	defer server.Close()

	videos, err := svc.ListVideos(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}
	if videos[0].UID != "vid-001" {
		t.Errorf("expected UID=vid-001, got %s", videos[0].UID)
	}
	if !videos[0].ReadyToStream {
		t.Error("expected first video to be ready to stream")
	}
	if videos[0].Duration != 120.5 {
		t.Errorf("expected duration=120.5, got %f", videos[0].Duration)
	}
	if videos[1].Status.State != "processing" {
		t.Errorf("expected second video state=processing, got %s", videos[1].Status.State)
	}
}

func TestStreamListVideosWithStatusFilter(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status != "ready" {
			t.Errorf("expected status=ready query param, got %q", status)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	_, err := svc.ListVideos(context.Background(), "ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamListVideosError(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		streamWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.ListVideos(context.Background(), "")
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

func TestStreamGetVideoSuccess(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"uid":               "vid-001",
				"readyToStream":     true,
				"requireSignedURLs": false,
				"duration":          300.0,
				"size":              5242880,
				"status":            map[string]interface{}{"state": "ready"},
				"meta":              map[string]interface{}{"name": "my-video.mp4"},
				"playback": map[string]interface{}{
					"hls":  "https://example.com/vid-001/manifest.m3u8",
					"dash": "https://example.com/vid-001/manifest.mpd",
				},
				"input": map[string]interface{}{
					"width":  1920,
					"height": 1080,
				},
			},
		})
	})
	defer server.Close()

	video, err := svc.GetVideo(context.Background(), "vid-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if video.UID != "vid-001" {
		t.Errorf("expected UID=vid-001, got %s", video.UID)
	}
	if video.Duration != 300.0 {
		t.Errorf("expected duration=300, got %f", video.Duration)
	}
	if video.Input.Width != 1920 || video.Input.Height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", video.Input.Width, video.Input.Height)
	}
	if video.Playback.HLS == "" {
		t.Error("expected HLS playback URL to be set")
	}
}

func TestStreamDeleteVideoSuccess(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
		})
	})
	defer server.Close()

	err := svc.DeleteVideo(context.Background(), "vid-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamDeleteVideoError(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		streamWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.DeleteVideo(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

func TestStreamUploadByURLSuccess(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"uid":           "vid-new",
				"readyToStream": false,
				"status":        map[string]interface{}{"state": "downloading"},
				"created":       time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	video, err := svc.UploadByURL(context.Background(), "https://example.com/video.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if video.UID != "vid-new" {
		t.Errorf("expected UID=vid-new, got %s", video.UID)
	}
}

func TestStreamUploadByURLWithOptions(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"uid":               "vid-opts",
				"readyToStream":     false,
				"requireSignedURLs": true,
				"status":            map[string]interface{}{"state": "downloading"},
			},
		})
	})
	defer server.Close()

	video, err := svc.UploadByURL(context.Background(), "https://example.com/video.mp4",
		WithStreamRequireSignedURLs(true),
		WithStreamMetadata(map[string]interface{}{"project": "test"}),
		WithStreamWatermark("wm-123"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if video.UID != "vid-opts" {
		t.Errorf("expected UID=vid-opts, got %s", video.UID)
	}
}

func TestStreamCreateSignedTokenSuccess(t *testing.T) {
	svc, server := streamMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		streamWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-token",
			},
		})
	})
	defer server.Close()

	token, err := svc.CreateSignedToken(context.Background(), "vid-001", 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if token != "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-token" {
		t.Errorf("unexpected token: %s", token)
	}
}

// --- Conversion helper test ---

func TestToCFStreamVideo(t *testing.T) {
	now := time.Now().UTC()
	sv := cloudflare.StreamVideo{
		UID:               "test-uid",
		Created:           &now,
		Duration:          60.5,
		Size:              1024,
		ReadyToStream:     true,
		RequireSignedURLs: true,
		Status: cloudflare.StreamVideoStatus{
			State:       "ready",
			PctComplete: "100",
		},
		Input: cloudflare.StreamVideoInput{
			Width:  1280,
			Height: 720,
		},
		Playback: cloudflare.StreamVideoPlayback{
			HLS:  "https://hls.example.com",
			Dash: "https://dash.example.com",
		},
		Preview:   "https://preview.example.com",
		Thumbnail: "https://thumb.example.com",
		Meta:      map[string]interface{}{"name": "test"},
		Creator:   "creator-1",
		LiveInput: "live-input-1",
		Watermark: cloudflare.StreamVideoWatermark{UID: "wm-1"},
	}

	result := toCFStreamVideo(sv)

	if result.UID != "test-uid" {
		t.Errorf("UID mismatch: got %s", result.UID)
	}
	if result.Duration != 60.5 {
		t.Errorf("Duration mismatch: got %f", result.Duration)
	}
	if !result.ReadyToStream {
		t.Error("expected ReadyToStream=true")
	}
	if !result.RequireSignedURLs {
		t.Error("expected RequireSignedURLs=true")
	}
	if result.Status.State != "ready" {
		t.Errorf("Status.State mismatch: got %s", result.Status.State)
	}
	if result.Input.Width != 1280 || result.Input.Height != 720 {
		t.Errorf("Input mismatch: got %dx%d", result.Input.Width, result.Input.Height)
	}
	if result.Playback.HLS != "https://hls.example.com" {
		t.Errorf("Playback.HLS mismatch: got %s", result.Playback.HLS)
	}
	if result.Creator != "creator-1" {
		t.Errorf("Creator mismatch: got %s", result.Creator)
	}
	if result.Watermark.UID != "wm-1" {
		t.Errorf("Watermark.UID mismatch: got %s", result.Watermark.UID)
	}
}

// --- Upload options test ---

func TestStreamUploadOptions(t *testing.T) {
	cfg := &streamUploadConfig{}

	WithStreamRequireSignedURLs(true)(cfg)
	if !cfg.requireSignedURLs {
		t.Error("expected requireSignedURLs=true")
	}

	meta := map[string]interface{}{"key": "value"}
	WithStreamMetadata(meta)(cfg)
	if cfg.metadata["key"] != "value" {
		t.Error("expected metadata key=value")
	}

	WithStreamWatermark("wm-abc")(cfg)
	if cfg.watermarkUID != "wm-abc" {
		t.Errorf("expected watermarkUID=wm-abc, got %s", cfg.watermarkUID)
	}
}
