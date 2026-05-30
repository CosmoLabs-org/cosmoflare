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

func imagesMockSetup(handler http.HandlerFunc) (*ImagesService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewImagesService(cf, "acct-img-123")
	return svc, server
}

func imagesWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewImagesServiceValidation(t *testing.T) {
	_, err := NewImagesService(nil, "acct123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewImagesService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewImagesServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewImagesService(cf, "acct123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

func TestNewImagesServiceFromCredsValidation(t *testing.T) {
	_, err := NewImagesServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewImagesServiceFromCreds("acct123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewImagesServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewImagesServiceFromCreds("acct123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

// --- Method validation ---

func TestImagesGetValidation(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.GetImage(context.Background(), "")
	if err == nil {
		t.Error("expected error when image ID is empty")
	}
}

func TestImagesDeleteValidation(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteImage(context.Background(), "")
	if err == nil {
		t.Error("expected error when image ID is empty")
	}
}

func TestImagesUploadURLValidation(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.UploadByURL(context.Background(), "")
	if err == nil {
		t.Error("expected error when URL is empty")
	}
}

// --- Mock server tests ---

func TestImagesListSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"images": []map[string]interface{}{
					{
						"id":                "img-001",
						"filename":          "photo.jpg",
						"requireSignedURLs": false,
						"variants":          []string{"public"},
						"uploaded":          "2025-01-15T10:00:00Z",
					},
					{
						"id":                "img-002",
						"filename":          "banner.png",
						"requireSignedURLs": true,
						"variants":          []string{"public", "thumbnail"},
						"uploaded":          "2025-02-20T14:30:00Z",
					},
				},
			},
		})
	})
	defer server.Close()

	images, err := svc.ListImages(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].ID != "img-001" || images[0].Filename != "photo.jpg" {
		t.Errorf("first image mismatch: %+v", images[0])
	}
	if !images[1].RequireSignedURLs {
		t.Error("expected second image to require signed URLs")
	}
}

func TestImagesListError(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		imagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.ListImages(context.Background())
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

func TestImagesGetSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                "img-001",
				"filename":          "photo.jpg",
				"requireSignedURLs": false,
				"variants":          []string{"public"},
				"uploaded":          "2025-01-15T10:00:00Z",
				"meta":              map[string]interface{}{"author": "test"},
			},
		})
	})
	defer server.Close()

	img, err := svc.GetImage(context.Background(), "img-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.ID != "img-001" {
		t.Errorf("expected ID=img-001, got %s", img.ID)
	}
	if img.Filename != "photo.jpg" {
		t.Errorf("expected Filename=photo.jpg, got %s", img.Filename)
	}
}

func TestImagesDeleteSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
		})
	})
	defer server.Close()

	err := svc.DeleteImage(context.Background(), "img-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImagesDeleteError(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		imagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.DeleteImage(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

func TestImagesUploadByURLSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                "img-new",
				"filename":          "remote.jpg",
				"requireSignedURLs": false,
				"variants":          []string{"public"},
				"uploaded":          time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	img, err := svc.UploadByURL(context.Background(), "https://example.com/photo.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.ID != "img-new" {
		t.Errorf("expected ID=img-new, got %s", img.ID)
	}
}

// --- Variant tests ---

func TestImagesVariantsListSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"variants": map[string]interface{}{
					"public": map[string]interface{}{
						"id":                     "public",
						"neverRequireSignedURLs": true,
						"options": map[string]interface{}{
							"fit":    "scale-down",
							"width":  1920,
							"height": 1080,
						},
					},
					"thumbnail": map[string]interface{}{
						"id":                     "thumbnail",
						"neverRequireSignedURLs": false,
						"options": map[string]interface{}{
							"fit":    "cover",
							"width":  150,
							"height": 150,
						},
					},
				},
			},
		})
	})
	defer server.Close()

	variants, err := svc.ListVariants(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
}

func TestImagesVariantsCreateSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"variant": map[string]interface{}{
					"id": "hero",
					"options": map[string]interface{}{
						"fit":    "cover",
						"width":  1200,
						"height": 630,
					},
				},
			},
		})
	})
	defer server.Close()

	v, err := svc.CreateVariant(context.Background(), "hero", "cover", 1200, 630, "none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "hero" {
		t.Errorf("expected ID=hero, got %s", v.ID)
	}
}

func TestImagesVariantsCreateValidation(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.CreateVariant(context.Background(), "", "cover", 100, 100, "none")
	if err == nil {
		t.Error("expected error when variant name is empty")
	}
}

func TestImagesVariantsDeleteSuccess(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		imagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
		})
	})
	defer server.Close()

	err := svc.DeleteVariant(context.Background(), "thumbnail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImagesVariantsDeleteValidation(t *testing.T) {
	svc, server := imagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteVariant(context.Background(), "")
	if err == nil {
		t.Error("expected error when variant name is empty")
	}
}
