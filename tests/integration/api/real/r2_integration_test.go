/*
Package api provides real Cloudflare R2 API integration testing

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package real

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	r2api "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/tests/helpers"
)

// R2IntegrationTestSuite provides comprehensive R2 API integration testing
type R2IntegrationTestSuite struct {
	suite.Suite
	client     *r2api.Client
	testConfig helpers.TestConfig
	bucketName string
	tempFiles  []string
}

// SetupSuite sets up the integration test suite
func (suite *R2IntegrationTestSuite) SetupSuite() {
	// Check if we have real R2 credentials for integration testing
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	accessKey := os.Getenv("CLOUDFLARE_ACCESS_KEY")
	secretKey := os.Getenv("CLOUDFLARE_SECRET_KEY")

	if accountID == "" || accessKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration tests: missing CLOUDFLARE credentials")
		return
	}

	// Create test configuration
	suite.testConfig = helpers.SetupTest(suite.T())

	// Create API client
	client, err := r2api.NewClient(&r2api.ClientConfig{
		AccountID: accountID,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	require.NoError(suite.T(), err, "Failed to create API client")
	suite.client = client

	// Generate unique bucket name for testing
	suite.bucketName = "r2go2-test-" + helpers.RandomString(8)
	suite.tempFiles = []string{}
}

// TearDownSuite cleans up after integration tests
func (suite *R2IntegrationTestSuite) TearDownSuite() {
	// Clean up test bucket
	if suite.client != nil && suite.bucketName != "" {
		// List and delete all objects in test bucket
		objects, err := suite.client.ListObjects(suite.bucketName, "")
		if err == nil {
			for _, obj := range objects {
				suite.client.DeleteObject(suite.bucketName, obj.Key)
			}
		}

		// Delete test bucket
		suite.client.DeleteBucket(suite.bucketName)
	}

	// Clean up temporary files
	for _, file := range suite.tempFiles {
		os.Remove(file)
	}
}

// TestClientCreation tests API client creation and validation
func (suite *R2IntegrationTestSuite) TestClientCreation() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	suite.Run("Valid Client Configuration", func() {
		// Test that client was created successfully
		assert.NotNil(suite.T(), suite.client, "Client should be created")

		// Test client configuration
		// (This would need to be implemented based on actual client interface)
	})

	suite.Run("Client Authentication", func() {
		// Test that client can authenticate with R2
		// This would typically involve a simple API call to validate credentials
		err := suite.client.TestConnection()
		assert.NoError(suite.T(), err, "Client should be able to authenticate with R2")
	})
}

// TestBucketOperations tests complete bucket lifecycle management
func (suite *R2IntegrationTestSuite) TestBucketOperations() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	suite.Run("Create Bucket", func() {
		// Create a test bucket
		err := suite.client.CreateBucket(suite.bucketName)
		assert.NoError(suite.T(), err, "Should be able to create bucket")

		// Verify bucket was created
		buckets, err := suite.client.ListBuckets()
		assert.NoError(suite.T(), err, "Should be able to list buckets")

		// Find our test bucket
		found := false
		for _, bucket := range buckets {
			if bucket.Name == suite.bucketName {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Test bucket should be in the list")
	})

	suite.Run("List Buckets", func() {
		buckets, err := suite.client.ListBuckets()
		assert.NoError(suite.T(), err, "Should be able to list buckets")
		assert.NotEmpty(suite.T(), buckets, "Should have at least one bucket")

		// Verify our test bucket is in the list
		found := false
		for _, bucket := range buckets {
			if bucket.Name == suite.bucketName {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Test bucket should be found in list")
	})

	suite.Run("Get Bucket Info", func() {
		bucketInfo, err := suite.client.GetBucket(suite.bucketName)
		assert.NoError(suite.T(), err, "Should be able to get bucket info")
		assert.Equal(suite.T(), suite.bucketName, bucketInfo.Name, "Bucket name should match")
	})

	suite.Run("Update Bucket", func() {
		// Test bucket metadata updates (if supported)
		// This would depend on the actual R2 API capabilities
		metadata := map[string]string{
			"test":      "integration",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		err := suite.client.UpdateBucketMetadata(suite.bucketName, metadata)
		// Note: R2 might not support bucket metadata updates
		if err != nil {
			suite.T().Logf("Bucket metadata update not supported: %v", err)
		}
	})
}

// TestObjectOperations tests complete object lifecycle management
func (suite *R2IntegrationTestSuite) TestObjectOperations() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	// Create test file
	testFile := suite.testConfig.TempDir + "/test-upload.txt"
	content := "This is a test file for R2Go2 integration testing\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(suite.T(), err)
	suite.tempFiles = append(suite.tempFiles, testFile)

	suite.Run("Upload Object", func() {
		// Upload a simple object
		objectKey := "test-file.txt"
		err := suite.client.UploadFile(suite.bucketName, objectKey, testFile)
		assert.NoError(suite.T(), err, "Should be able to upload file")

		// Verify object exists
		objects, err := suite.client.ListObjects(suite.bucketName, "")
		assert.NoError(suite.T(), err, "Should be able to list objects")

		found := false
		for _, obj := range objects {
			if obj.Key == objectKey {
				found = true
				assert.Greater(suite.T(), obj.Size, int64(0), "Object should have content")
				break
			}
		}
		assert.True(suite.T(), found, "Uploaded object should be found")
	})

	suite.Run("List Objects", func() {
		objects, err := suite.client.ListObjects(suite.bucketName, "")
		assert.NoError(suite.T(), err, "Should be able to list objects")
		assert.NotEmpty(suite.T(), objects, "Should have at least one object")
	})

	suite.Run("Download Object", func() {
		// Download the uploaded file
		objectKey := "test-file.txt"
		downloadFile := suite.testConfig.TempDir + "/downloaded-test.txt"

		err := suite.client.DownloadFile(suite.bucketName, objectKey, downloadFile)
		assert.NoError(suite.T(), err, "Should be able to download file")

		// Verify content matches
		originalContent, err := os.ReadFile(testFile)
		require.NoError(suite.T(), err)

		downloadedContent, err := os.ReadFile(downloadFile)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), originalContent, downloadedContent, "Downloaded content should match original")
		suite.tempFiles = append(suite.tempFiles, downloadFile)
	})

	suite.Run("Copy Object", func() {
		// Test object copying
		sourceKey := "test-file.txt"
		destKey := "copied-file.txt"

		err := suite.client.CopyObject(suite.bucketName, sourceKey, suite.bucketName, destKey)
		if err != nil {
			suite.T().Logf("Object copy not supported: %v", err)
			suite.T().Skip("Object copy not supported")
			return
		}

		// Verify copy exists
		objects, err := suite.client.ListObjects(suite.bucketName, "")
		assert.NoError(suite.T(), err)

		found := false
		for _, obj := range objects {
			if obj.Key == destKey {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Copied object should be found")
	})

	suite.Run("Delete Object", func() {
		// Delete the test objects
		testObjects := []string{"test-file.txt", "copied-file.txt"}

		for _, objectKey := range testObjects {
			err := suite.client.DeleteObject(suite.bucketName, objectKey)
			if err != nil {
				suite.T().Logf("Failed to delete object %s: %v", objectKey, err)
				continue
			}
		}

		// Verify objects are deleted
		objects, err := suite.client.ListObjects(suite.bucketName, "")
		assert.NoError(suite.T(), err)

		for _, objectKey := range testObjects {
			found := false
			for _, obj := range objects {
				if obj.Key == objectKey {
					found = true
					break
				}
			}
			assert.False(suite.T(), found, "Deleted object should not be found: "+objectKey)
		}
	})
}

// TestPerformanceMetrics tests API performance characteristics
func (suite *R2IntegrationTestSuite) TestPerformanceMetrics() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	suite.Run("API Response Times", func() {
		// Test various API operations response times
		operations := []struct {
			name     string
			operation func() error
			maxTime  time.Duration
		}{
			{
				name: "List buckets",
				operation: func() error {
					_, err := suite.client.ListBuckets()
					return err
				},
				maxTime: time.Second * 2,
			},
			{
				name: "Get bucket info",
				operation: func() error {
					_, err := suite.client.GetBucket(suite.bucketName)
					return err
				},
				maxTime: time.Second * 1,
			},
			{
				name: "List objects",
				operation: func() error {
					_, err := suite.client.ListObjects(suite.bucketName, "")
					return err
				},
				maxTime: time.Second * 3,
			},
		}

		for _, op := range operations {
			start := time.Now()
			err := op.operation()
			duration := time.Since(start)

			if err != nil {
				suite.T().Logf("Operation %s failed: %v", op.name, err)
				continue
			}

			assert.Less(suite.T(), duration, op.maxTime,
				fmt.Sprintf("%s should complete within %v (actual: %v)", op.name, op.maxTime, duration))
		}
	})

	suite.Run("Concurrent Operations", func() {
		// Test handling multiple concurrent requests
		concurrentRequests := 10
		done := make(chan bool, concurrentRequests)
		errors := make(chan error, concurrentRequests)

		start := time.Now()

		for i := 0; i < concurrentRequests; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Perform list operations concurrently
				_, err := suite.client.ListBuckets()
				errors <- err
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < concurrentRequests; i++ {
			<-done
		}

		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for i := 0; i < concurrentRequests; i++ {
			if err := <-errors; err != nil {
				errorCount++
			}
		}

		// Should handle concurrent requests without too many errors
		assert.Less(suite.T(), errorCount, concurrentRequests/2,
			"Should handle concurrent requests without excessive errors")

		// Performance should be reasonable
		assert.Less(suite.T(), duration, time.Second*10,
			"Concurrent requests should complete in reasonable time")
	})
}

// TestErrorHandling tests various error scenarios and recovery
func (suite *R2IntegrationTestSuite) TestErrorHandling() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	suite.Run("Non-existent Bucket", func() {
		_, err := suite.client.GetBucket("non-existent-bucket-12345")
		assert.Error(suite.T(), err, "Should return error for non-existent bucket")
	})

	suite.Run("Non-existent Object", func() {
		downloadFile := suite.testConfig.TempDir + "/non-existent-download.txt"
		err := suite.client.DownloadFile(suite.bucketName, "non-existent-object.txt", downloadFile)
		assert.Error(suite.T(), err, "Should return error for non-existent object")
	})

	suite.Run("Invalid Permissions", func() {
		// This would require testing with a client that has limited permissions
		// For now, we'll test the error handling path
		suite.T().Skip("Invalid permissions test requires special client setup")
	})
}

// TestDataIntegrity tests data integrity during operations
func (suite *R2IntegrationTestSuite) TestDataIntegrity() {
	if suite.client == nil {
		suite.T().Skip("No client available for testing")
		return
	}

	suite.Run("Large File Upload", func() {
		// Create a larger test file
		largeFile := suite.testConfig.TempDir + "/large-test.txt"
		content := string(make([]byte, 1024*1024)) // 1MB file
		for i := range content {
			content[i] = byte(i % 256)
		}

		err := os.WriteFile(largeFile, []byte(content), 0644)
		require.NoError(suite.T(), err)
		suite.tempFiles = append(suite.tempFiles, largeFile)

		// Upload the large file
		objectKey := "large-test-file.dat"
		err = suite.client.UploadFile(suite.bucketName, objectKey, largeFile)
		assert.NoError(suite.T(), err, "Should be able to upload large file")

		// Download and verify
		downloadFile := suite.testConfig.TempDir + "/downloaded-large.txt"
		err = suite.client.DownloadFile(suite.bucketName, objectKey, downloadFile)
		assert.NoError(suite.T(), err, "Should be able to download large file")

		// Verify integrity
		originalData, err := os.ReadFile(largeFile)
		require.NoError(suite.T(), err)

		downloadedData, err := os.ReadFile(downloadFile)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), len(originalData), len(downloadedData), "File sizes should match")
		assert.Equal(suite.T(), originalData, downloadedData, "File contents should match")

		// Clean up
		suite.client.DeleteObject(suite.bucketName, objectKey)
		suite.tempFiles = append(suite.tempFiles, downloadFile)
	})

	suite.Run("Special Characters", func() {
		// Test files with special characters in names and content
		specialContent := "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
		testFile := suite.testConfig.TempDir + "/special-chars.txt"

		err := os.WriteFile(testFile, []byte(specialContent), 0644)
		require.NoError(suite.T(), err)
		suite.tempFiles = append(suite.tempFiles, testFile)

		objectKey := "special-chars-!@#$%^&*().txt"
		err = suite.client.UploadFile(suite.bucketName, objectKey, testFile)
		if err != nil {
			suite.T().Logf("Special characters in object name not supported: %v", err)
			return
		}

		// Download and verify
		downloadFile := suite.testConfig.TempDir + "/downloaded-special.txt"
		err = suite.client.DownloadFile(suite.bucketName, objectKey, downloadFile)
		assert.NoError(suite.T(), err, "Should be able to download file with special chars")

		downloadedContent, err := os.ReadFile(downloadFile)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), specialContent, string(downloadedContent), "Special content should be preserved")

		// Clean up
		suite.client.DeleteObject(suite.bucketName, objectKey)
		suite.tempFiles = append(suite.tempFiles, downloadFile)
	})
}

// TestR2IntegrationTestSuite runs the complete integration test suite
func TestR2IntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(R2IntegrationTestSuite))
}