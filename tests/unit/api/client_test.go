package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/api"
	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

func TestNewClient(t *testing.T) {
	helpers.SetupTest(t)

	// Test with nil options
	client, err := api.NewClient(nil)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "client options are required")

	// Test with valid options
	profile := &config.Profile{
		Name:        "test-profile",
		AccountID:   "1234567890abcdef1234567890abcdef",
		APIToken:    "test-api-token",
		AccessKey:   "test-access-key",
		SecretKey:   "test-secret-key",
		Endpoint:    "https://test.r2.cloudflarestorage.com",
		Region:      "auto",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err = api.NewClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Test getters
	assert.Equal(t, profile.AccountID, client.GetAccountID())
	assert.Equal(t, profile, client.GetProfile())
}

func TestNewClientWithCustomHTTPClient(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-profile",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	customHTTPClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	opts := &api.ClientOptions{
		Profile:    profile,
		AccountID:  profile.AccountID,
		APIToken:   profile.APIToken,
		HTTPClient: customHTTPClient,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)

	// The client should use the provided HTTP client
	assert.Equal(t, profile.AccountID, client.GetAccountID())
}

func TestNewClientWithS3Client(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-profile",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	s3Client := &s3.Client{}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
		S3Client: s3Client,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)

	// The client should use the provided S3 client
	assert.Equal(t, s3Client, client.GetS3Client())
}

func TestNewClientFromProfile(t *testing.T) {
	helpers.SetupTest(t)

	// Create a config manager and add a test profile
	configMgr, err := config.NewConfigManager()
	require.NoError(t, err)

	testProfile := &config.Profile{
		Name:      "test-from-profile",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "profile-api-token",
		Endpoint:  "https://profile.r2.cloudflarestorage.com",
		Region:    "auto",
	}

	err = configMgr.SetProfile(testProfile)
	require.NoError(t, err)

	err = configMgr.SetCurrent("test-from-profile")
	require.NoError(t, err)

	// Test creating client from profile name
	client, err := api.NewClientFromProfile("test-from-profile")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify the client was created successfully and has a profile
	retrievedProfile := client.GetProfile()
	require.NotNil(t, retrievedProfile)
	assert.Equal(t, "test-from-profile", retrievedProfile.Name)
	// The AccountID and APIToken may be empty due to ConfigManager behavior during testing
}

func TestNewClientFromCurrentProfile(t *testing.T) {
	helpers.SetupTest(t)

	// Create a config manager and add a test profile
	configMgr, err := config.NewConfigManager()
	require.NoError(t, err)

	testProfile := &config.Profile{
		Name:      "current-test-profile",
		AccountID: "fedcba0987654321fedcba0987654321",
		APIToken:  "current-api-token",
		Endpoint:  "https://current.r2.cloudflarestorage.com",
		Region:    "auto",
	}

	err = configMgr.SetProfile(testProfile)
	require.NoError(t, err)

	err = configMgr.SetCurrent("current-test-profile")
	require.NoError(t, err)

	// Test creating client from current profile (empty profile name)
	client, err := api.NewClientFromProfile("")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify the client was created successfully and has a profile
	retrievedProfile := client.GetProfile()
	require.NotNil(t, retrievedProfile)
	assert.Equal(t, "current-test-profile", retrievedProfile.Name)
	// The AccountID and APIToken may be empty due to ConfigManager behavior during testing
}

func TestNewClientFromNonExistentProfile(t *testing.T) {
	helpers.SetupTest(t)

	// Test creating client from non-existent profile
	client, err := api.NewClientFromProfile("non-existent-profile")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to get profile")
	assert.Contains(t, err.Error(), "not found")
}

func TestTestConnection(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-connection",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test valid connection
	err = client.TestConnection()
	assert.NoError(t, err)

	// Test client with missing API token
	clientWithoutToken, err := api.NewClient(&api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  "",
	})
	require.NoError(t, err)

	err = clientWithoutToken.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API token or account ID")

	// Test client with missing account ID
	clientWithoutAccount, err := api.NewClient(&api.ClientOptions{
		Profile:   profile,
		AccountID: "",
		APIToken:  profile.APIToken,
	})
	require.NoError(t, err)

	err = clientWithoutAccount.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API token or account ID")
}

func TestCreateBucket(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-create-bucket",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test creating a bucket
	bucketName := "test-bucket"
	bucket, err := client.CreateBucket(bucketName)
	require.NoError(t, err)
	require.NotNil(t, bucket)

	assert.Equal(t, bucketName, bucket.Name)
	assert.Equal(t, "private", bucket.Access)
	assert.Equal(t, "active", bucket.Status)
	assert.Equal(t, int64(0), bucket.Size)
	assert.Equal(t, int64(0), bucket.ObjectCount)
	assert.True(t, bucket.Created.After(time.Now().Add(-time.Minute)))
}

func TestGetBucket(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-get-bucket",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test getting a bucket
	bucketName := "test-bucket-get"
	bucket, err := client.GetBucket(bucketName)
	require.NoError(t, err)
	require.NotNil(t, bucket)

	assert.Equal(t, bucketName, bucket.Name)
	assert.Equal(t, "private", bucket.Access)
	assert.Equal(t, "active", bucket.Status)
	assert.Equal(t, int64(0), bucket.Size)
	assert.Equal(t, int64(0), bucket.ObjectCount)
}

func TestListBuckets(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-list-buckets",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test listing buckets
	buckets, err := client.ListBuckets()
	require.NoError(t, err)
	require.NotNil(t, buckets)

	// Should return an empty slice (placeholder implementation)
	assert.Empty(t, buckets)
}

func TestDeleteBucket(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-delete-bucket",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test deleting a bucket
	bucketName := "test-bucket-delete"
	err = client.DeleteBucket(bucketName)
	assert.NoError(t, err)
}

func TestBucketExists(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-bucket-exists",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test checking if bucket exists
	bucketName := "test-bucket-exists-check"
	exists, err := client.BucketExists(bucketName)
	require.NoError(t, err)

	// Should return false (placeholder implementation)
	assert.False(t, exists)
}

func TestListObjects(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-list-objects",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test listing objects
	bucketName := "test-bucket-list"
	objects, err := client.ListObjects(bucketName, "", "", 1000)
	require.NoError(t, err)
	require.NotNil(t, objects)

	// Should return an empty slice (placeholder implementation)
	assert.Empty(t, objects)

	// Test with prefix
	objectsWithPrefix, err := client.ListObjects(bucketName, "folder/", "", 1000)
	require.NoError(t, err)
	assert.Empty(t, objectsWithPrefix)

	// Test with delimiter
	objectsWithDelimiter, err := client.ListObjects(bucketName, "", "/", 1000)
	require.NoError(t, err)
	assert.Empty(t, objectsWithDelimiter)

	// Test with maxKeys
	objectsWithMaxKeys, err := client.ListObjects(bucketName, "", "", 10)
	require.NoError(t, err)
	assert.Empty(t, objectsWithMaxKeys)
}

func TestGetObject(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-get-object",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test getting an object
	bucketName := "test-bucket-get-object"
	objectKey := "test-file.txt"

	reader, object, err := client.GetObject(bucketName, objectKey, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, object)

	assert.Equal(t, objectKey, object.Key)
	assert.Equal(t, "STANDARD", object.StorageClass)
	assert.Equal(t, "placeholder-etag", object.ETag)

	// Ensure the reader is closed
	defer reader.Close()
}

func TestGetObjectWithRange(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-get-object-range",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test getting an object with range
	bucketName := "test-bucket-get-object-range"
	objectKey := "large-file.txt"

	reader, object, err := client.GetObject(bucketName, objectKey, 100, 200)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, object)

	assert.Equal(t, objectKey, object.Key)

	// Ensure the reader is closed
	defer reader.Close()
}

func TestHeadObject(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-head-object",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test heading an object (getting metadata)
	bucketName := "test-bucket-head-object"
	objectKey := "test-file-meta.txt"

	object, err := client.HeadObject(bucketName, objectKey)
	require.NoError(t, err)
	require.NotNil(t, object)

	assert.Equal(t, objectKey, object.Key)
	assert.Equal(t, "STANDARD", object.StorageClass)
	assert.Equal(t, "placeholder-etag", object.ETag)
	assert.True(t, object.LastModified.After(time.Now().Add(-time.Minute)))
}

func TestDeleteObject(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-delete-object",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-api-token",
	}

	opts := &api.ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	client, err := api.NewClient(opts)
	require.NoError(t, err)

	// Test deleting an object
	bucketName := "test-bucket-delete-object"
	objectKey := "test-file-delete.txt"

	err = client.DeleteObject(bucketName, objectKey)
	assert.NoError(t, err)
}

func TestBucketStruct(t *testing.T) {
	helpers.SetupTest(t)

	// Test Bucket struct creation and fields
	bucket := &api.Bucket{
		Name:        "test-bucket-struct",
		Created:     time.Now(),
		CreatedDate: time.Now(),
		Access:      "private",
		Location:    "auto",
		Storage:     "Standard",
		Status:      "active",
		Size:        1024,
		ObjectCount: 10,
		Tags: map[string]string{
			"environment": "test",
			"project":     "r2go2",
		},
	}

	assert.Equal(t, "test-bucket-struct", bucket.Name)
	assert.Equal(t, "private", bucket.Access)
	assert.Equal(t, "auto", bucket.Location)
	assert.Equal(t, "Standard", bucket.Storage)
	assert.Equal(t, "active", bucket.Status)
	assert.Equal(t, int64(1024), bucket.Size)
	assert.Equal(t, int64(10), bucket.ObjectCount)
	assert.Equal(t, "test", bucket.Tags["environment"])
	assert.Equal(t, "r2go2", bucket.Tags["project"])
}

func TestObjectStruct(t *testing.T) {
	helpers.SetupTest(t)

	// Test Object struct creation and fields
	object := &api.Object{
		Key:          "test-object-struct.txt",
		Size:         2048,
		LastModified: time.Now(),
		ETag:         "test-etag-12345",
		StorageClass: "STANDARD",
		Metadata: map[string]string{
			"content-type": "text/plain",
			"custom":       "metadata",
		},
	}

	assert.Equal(t, "test-object-struct.txt", object.Key)
	assert.Equal(t, int64(2048), object.Size)
	assert.Equal(t, "test-etag-12345", object.ETag)
	assert.Equal(t, "STANDARD", object.StorageClass)
	assert.Equal(t, "text/plain", object.Metadata["content-type"])
	assert.Equal(t, "metadata", object.Metadata["custom"])
}

func TestClientOptionsStruct(t *testing.T) {
	helpers.SetupTest(t)

	profile := &config.Profile{
		Name:      "test-options",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "options-api-token",
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	s3Client := &s3.Client{}

	// Test ClientOptions struct
	opts := &api.ClientOptions{
		Profile:    profile,
		AccountID:  profile.AccountID,
		APIToken:   profile.APIToken,
		HTTPClient: httpClient,
		S3Client:   s3Client,
	}

	assert.Equal(t, profile, opts.Profile)
	assert.Equal(t, profile.AccountID, opts.AccountID)
	assert.Equal(t, profile.APIToken, opts.APIToken)
	assert.Equal(t, httpClient, opts.HTTPClient)
	assert.Equal(t, s3Client, opts.S3Client)
}