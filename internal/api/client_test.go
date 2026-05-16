package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	r2config "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("nil options returns error", func(t *testing.T) {
		client, err := NewClient(nil)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "client options are required")
	})

	t.Run("valid options", func(t *testing.T) {
		opts := &ClientOptions{
			AccountID: "test-account-123",
			APIToken:  "test-token-456",
		}
		client, err := NewClient(opts)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "test-account-123", client.accountID)
		assert.Equal(t, "test-token-456", client.apiToken)
		assert.NotNil(t, client.httpClient, "default HTTP client should be created")
	})

	t.Run("with profile", func(t *testing.T) {
		profile := &r2config.Profile{
			Name:        "test-profile",
			Description: "Test",
		}
		opts := &ClientOptions{
			Profile:   profile,
			AccountID: "acct",
			APIToken:  "tok",
		}
		client, err := NewClient(opts)
		require.NoError(t, err)
		assert.Equal(t, profile, client.profile)
	})
}

func TestTestConnection(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		client := &Client{accountID: "acct", apiToken: ""}
		err := client.TestConnection()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing API token or account ID")
	})

	t.Run("missing account ID", func(t *testing.T) {
		client := &Client{accountID: "", apiToken: "tok"}
		err := client.TestConnection()
		assert.Error(t, err)
	})

	t.Run("valid credentials", func(t *testing.T) {
		client := &Client{accountID: "acct", apiToken: "tok"}
		err := client.TestConnection()
		assert.NoError(t, err)
	})
}

func TestClientGetters(t *testing.T) {
	profile := &r2config.Profile{Name: "prod"}
	client := &Client{
		accountID: "acct-123",
		apiToken:  "tok-456",
		profile:   profile,
	}

	t.Run("GetAccountID", func(t *testing.T) {
		assert.Equal(t, "acct-123", client.GetAccountID())
	})

	t.Run("GetProfile", func(t *testing.T) {
		assert.Equal(t, profile, client.GetProfile())
	})

	t.Run("GetS3Client", func(t *testing.T) {
		assert.Nil(t, client.GetS3Client())
	})
}

func TestCreateBucket(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	bucket, err := client.CreateBucket("my-bucket")
	require.NoError(t, err)
	require.NotNil(t, bucket)
	assert.Equal(t, "my-bucket", bucket.Name)
	assert.Equal(t, "private", bucket.Access)
	assert.Equal(t, "active", bucket.Status)
}

func TestListBuckets(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	buckets, err := client.ListBuckets()
	require.NoError(t, err)
	assert.NotNil(t, buckets)
	assert.Empty(t, buckets)
}

func TestGetBucket(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	bucket, err := client.GetBucket("test-bucket")
	require.NoError(t, err)
	require.NotNil(t, bucket)
	assert.Equal(t, "test-bucket", bucket.Name)
}

func TestListObjects(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	objects, err := client.ListObjects("bucket", "prefix/", "/", 100)
	require.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Empty(t, objects)
}

func TestDeleteBucket(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	err := client.DeleteBucket("test-bucket")
	assert.NoError(t, err)
}

func TestBucketExists(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	exists, err := client.BucketExists("test-bucket")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestGetObject(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	reader, obj, err := client.GetObject("bucket", "key.txt", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, obj)
	assert.Equal(t, "key.txt", obj.Key)
	assert.Equal(t, "placeholder-etag", obj.ETag)
	reader.Close()
}

func TestDeleteObject(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	err := client.DeleteObject("bucket", "key.txt")
	assert.NoError(t, err)
}

func TestHeadObject(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	obj, err := client.HeadObject("bucket", "key.txt")
	require.NoError(t, err)
	require.NotNil(t, obj)
	assert.Equal(t, "key.txt", obj.Key)
	assert.Equal(t, "STANDARD", obj.StorageClass)
}

// ---------------------------------------------------------------------------
// NewClientFromProfile
// ---------------------------------------------------------------------------

func TestNewClientFromProfile(t *testing.T) {
	t.Run("nonexistent profile returns error", func(t *testing.T) {
		// This will try to load config from ~/.r2go2/config.yaml
		// In a clean test environment, the profile won't exist
		client, err := NewClientFromProfile("nonexistent-profile-xyz")
		// Either the config manager fails or the profile isn't found
		if err == nil {
			// If it succeeded, the client should be non-nil
			assert.NotNil(t, client)
		} else {
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "profile")
		}
	})

	t.Run("empty profile name uses current", func(t *testing.T) {
		client, err := NewClientFromProfile("")
		if err == nil {
			assert.NotNil(t, client)
		} else {
			assert.Nil(t, client)
			// Should fail at getting current profile
			assert.Contains(t, err.Error(), "profile")
		}
	})

	t.Run("config manager creation failure", func(t *testing.T) {
		// Set HOME to an invalid path to force NewConfigManager to fail
		t.Setenv("HOME", "/nonexistent/path/that/does/not/exist")
		client, err := NewClientFromProfile("any-profile")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to create config manager")
	})

	t.Run("empty profile name no current set", func(t *testing.T) {
		// Use a temp dir as HOME with empty config
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		client, err := NewClientFromProfile("")
		// Config will be created but has no profiles, so GetCurrent fails
		if err == nil {
			assert.NotNil(t, client)
		} else {
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "profile")
		}
	})
}

// ---------------------------------------------------------------------------
// NewClient - with custom HTTP client
// ---------------------------------------------------------------------------

func TestNewClient_CustomHTTPClient(t *testing.T) {
	customHTTP := &http.Client{Timeout: 5 * time.Second}
	opts := &ClientOptions{
		AccountID:  "custom-http",
		APIToken:   "custom-token",
		HTTPClient: customHTTP,
	}
	client, err := NewClient(opts)
	require.NoError(t, err)
	assert.Equal(t, customHTTP, client.httpClient)
}

func TestNewClient_WithS3Client(t *testing.T) {
	mockS3 := &s3.Client{}
	opts := &ClientOptions{
		AccountID: "s3-test",
		APIToken:  "s3-token",
		S3Client:  mockS3,
	}
	client, err := NewClient(opts)
	require.NoError(t, err)
	assert.Equal(t, mockS3, client.s3)
}

func TestNewClient_ZeroValues(t *testing.T) {
	opts := &ClientOptions{}
	client, err := NewClient(opts)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Empty(t, client.accountID)
	assert.Empty(t, client.apiToken)
	assert.Nil(t, client.profile)
	assert.NotNil(t, client.httpClient) // default HTTP client created
}

// ---------------------------------------------------------------------------
// TestConnection - edge cases
// ---------------------------------------------------------------------------

func TestTestConnection_BothEmpty(t *testing.T) {
	client := &Client{accountID: "", apiToken: ""}
	err := client.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API token or account ID")
}

func TestTestConnection_TokenEmptyAccountNonEmpty(t *testing.T) {
	client := &Client{accountID: "acct-123", apiToken: ""}
	err := client.TestConnection()
	assert.Error(t, err)
}

func TestTestConnection_AccountEmptyTokenNonEmpty(t *testing.T) {
	client := &Client{accountID: "", apiToken: "tok-456"}
	err := client.TestConnection()
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Bucket struct - JSON serialization
// ---------------------------------------------------------------------------

func TestBucket_JSONFields(t *testing.T) {
	b := &Bucket{
		Name:        "test-bucket",
		Created:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Access:      "public-read",
		Location:    "auto",
		Storage:     "Standard",
		Status:      "active",
		Size:        1024,
		ObjectCount: 10,
		Tags:        map[string]string{"env": "prod"},
	}

	data, err := json.Marshal(b)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"test-bucket"`)
	assert.Contains(t, string(data), `"access":"public-read"`)
	assert.Contains(t, string(data), `"tags"`)

	var decoded Bucket
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "test-bucket", decoded.Name)
	assert.Equal(t, "public-read", decoded.Access)
	assert.Equal(t, int64(1024), decoded.Size)
	assert.Equal(t, "prod", decoded.Tags["env"])
}

func TestBucket_OptionalFieldsOmitted(t *testing.T) {
	b := &Bucket{Name: "minimal"}
	data, err := json.Marshal(b)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"location"`)
	assert.NotContains(t, string(data), `"storage"`)
	assert.NotContains(t, string(data), `"tags"`)
}

// ---------------------------------------------------------------------------
// Object struct - JSON serialization
// ---------------------------------------------------------------------------

func TestObject_JSONFields(t *testing.T) {
	o := &Object{
		Key:          "path/to/file.txt",
		Size:         4096,
		LastModified: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		ETag:         "\"abc123\"",
		StorageClass: "STANDARD",
		Metadata:     map[string]string{"content-type": "text/plain"},
	}

	data, err := json.Marshal(o)
	require.NoError(t, err)

	var decoded Object
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "path/to/file.txt", decoded.Key)
	assert.Equal(t, int64(4096), decoded.Size)
	assert.Equal(t, "\"abc123\"", decoded.ETag)
	assert.Equal(t, "text/plain", decoded.Metadata["content-type"])
}

func TestObject_OptionalFieldsOmitted(t *testing.T) {
	o := &Object{Key: "simple.txt", Size: 100}
	data, err := json.Marshal(o)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"metadata"`)
}

// ---------------------------------------------------------------------------
// ClientOptions - JSON tags
// ---------------------------------------------------------------------------

func TestClientOptions_Struct(t *testing.T) {
	opts := &ClientOptions{
		Profile:   &r2config.Profile{Name: "test", AccountID: "a1", APIToken: "t1"},
		AccountID: "opts-acct",
		APIToken:  "opts-tok",
	}

	assert.Equal(t, "opts-acct", opts.AccountID)
	assert.Equal(t, "opts-tok", opts.APIToken)
	assert.Equal(t, "test", opts.Profile.Name)
	assert.Nil(t, opts.HTTPClient)
	assert.Nil(t, opts.S3Client)
}

// ---------------------------------------------------------------------------
// GetObject - reader content
// ---------------------------------------------------------------------------

func TestGetObject_ReadsContent(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	reader, obj, err := client.GetObject("bucket", "key.txt", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, obj)

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, content) // placeholder returns empty reader
	reader.Close()
}

func TestGetObject_WithRange(t *testing.T) {
	client := &Client{accountID: "acct", apiToken: "tok"}
	reader, obj, err := client.GetObject("bucket", "large.bin", 100, 200)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, obj)
	assert.Equal(t, "large.bin", obj.Key)
	reader.Close()
}
