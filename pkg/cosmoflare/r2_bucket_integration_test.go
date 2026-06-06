//go:build integration

package cosmoflare

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestR2BucketCRUD_Integration(t *testing.T) {
	buckets := make(map[string]map[string]interface{})

	mux := http.NewServeMux()

	// List buckets
	mux.HandleFunc("/accounts/test-account-id/r2/buckets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list := make([]map[string]interface{}, 0, len(buckets))
			for name, b := range buckets {
				b["name"] = name
				list = append(list, b)
			}
			writeCFJSON(w, map[string]interface{}{"buckets": list})
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			if err := decodeJSON(r.Body, &body); err != nil {
				writeCFError(w, 400, "invalid request body")
				return
			}
			if _, exists := buckets[body.Name]; exists {
				writeCFError(w, 409, "bucket already exists")
				return
			}
			buckets[body.Name] = map[string]interface{}{
				"name":         body.Name,
				"creation_date": time.Now().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusOK)
			writeCFJSON(w, buckets[body.Name])
		}
	})

	// Single bucket operations
	mux.HandleFunc("/accounts/test-account-id/r2/buckets/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/accounts/test-account-id/r2/buckets/"):]
		switch r.Method {
		case http.MethodGet:
			b, exists := buckets[name]
			if !exists {
				writeCFError(w, 404, "bucket not found")
				return
			}
			b["name"] = name
			writeCFJSON(w, b)
		case http.MethodDelete:
			if _, exists := buckets[name]; !exists {
				writeCFError(w, 404, "bucket not found")
				return
			}
			delete(buckets, name)
			w.WriteHeader(http.StatusOK)
			writeCFJSON(w, nil)
		}
	})

	c, _ := newTestClientWithServer(t, mux)
	ctx := context.Background()

	t.Run("CreateBucket", func(t *testing.T) {
		bucket, err := c.CreateBucket(ctx, "test-bucket")
		if err != nil {
			t.Fatalf("CreateBucket failed: %v", err)
		}
		if bucket.Name != "test-bucket" {
			t.Errorf("expected bucket name 'test-bucket', got %q", bucket.Name)
		}
	})

	t.Run("ListBuckets", func(t *testing.T) {
		list, err := c.ListBuckets(ctx)
		if err != nil {
			t.Fatalf("ListBuckets failed: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 bucket, got %d", len(list))
		}
	})

	t.Run("GetBucket", func(t *testing.T) {
		bucket, err := c.GetBucket(ctx, "test-bucket")
		if err != nil {
			t.Fatalf("GetBucket failed: %v", err)
		}
		if bucket.Name != "test-bucket" {
			t.Errorf("expected 'test-bucket', got %q", bucket.Name)
		}
	})

	t.Run("BucketExists", func(t *testing.T) {
		exists, err := c.BucketExists(ctx, "test-bucket")
		if err != nil {
			t.Fatalf("BucketExists failed: %v", err)
		}
		if !exists {
			t.Error("expected bucket to exist")
		}

		exists, err = c.BucketExists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("BucketExists failed: %v", err)
		}
		if exists {
			t.Error("expected bucket to not exist")
		}
	})

	t.Run("DeleteBucket", func(t *testing.T) {
		err := c.DeleteBucket(ctx, "test-bucket")
		if err != nil {
			t.Fatalf("DeleteBucket failed: %v", err)
		}

		list, err := c.ListBuckets(ctx)
		if err != nil {
			t.Fatalf("ListBuckets after delete failed: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 buckets after delete, got %d", len(list))
		}
	})

	t.Run("DeleteNonexistent", func(t *testing.T) {
		err := c.DeleteBucket(ctx, "does-not-exist")
		if err == nil {
			t.Error("expected error deleting nonexistent bucket")
		}
	})
}
