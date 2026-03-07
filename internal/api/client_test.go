package api

import (
	"testing"

	r2config "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
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
