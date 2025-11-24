/*
Package mock provides mock Cloudflare R2 API testing

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package mock

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSimpleMockR2Client tests the simple mock client functionality
func TestSimpleMockR2Client(t *testing.T) {
	t.Run("Success Scenario", func(t *testing.T) {
		client := NewSimpleMockR2Client(false, "")

		buckets, err := client.ListBuckets()
		assert.NoError(t, err, "Should not return error")
		assert.Len(t, buckets, 3, "Should return 3 buckets")

		// Test bucket creation
		err = client.CreateBucket("new-test-bucket")
		assert.NoError(t, err, "Should create bucket successfully")

		// Verify bucket was added
		buckets, err = client.ListBuckets()
		assert.NoError(t, err, "Should not return error")
		assert.Len(t, buckets, 4, "Should now have 4 buckets")
	})

	t.Run("Error Scenario", func(t *testing.T) {
		client := NewSimpleMockR2Client(true, "API connection failed")

		buckets, err := client.ListBuckets()
		assert.Error(t, err, "Should return error")
		assert.Nil(t, buckets, "Buckets should be nil on error")
		assert.Contains(t, err.Error(), "API connection failed", "Error message should match")
	})

	t.Run("Bucket Creation", func(t *testing.T) {
		client := NewSimpleMockR2Client(false, "")

		testNames := []string{"bucket1", "bucket-2", "bucket_3", "Bucket4"}
		for _, name := range testNames {
			err := client.CreateBucket(name)
			assert.NoError(t, err, fmt.Sprintf("Should create bucket: %s", name))
		}

		// Verify all buckets exist
		buckets, err := client.ListBuckets()
		assert.NoError(t, err, "Should not return error")
		assert.GreaterOrEqual(t, len(buckets), len(testNames), "Should have created buckets")
	})
}

// TestMockDataValidation tests mock data validation
func TestMockDataValidation(t *testing.T) {
	t.Run("Mock Bucket Generation", func(t *testing.T) {
		buckets := generateMockBuckets(5)
		assert.Len(t, buckets, 5, "Should generate 5 buckets")

		for i, bucket := range buckets {
			assert.NotEmpty(t, bucket.Name, "Bucket name should not be empty")
			assert.Contains(t, bucket.Name, "mock-bucket-", "Bucket name should follow pattern")
			assert.Equal(t, "active", bucket.Status, "Status should be active")
			assert.Equal(t, "auto", bucket.Region, "Region should be auto")
			assert.Greater(t, bucket.Size, int64(0), "Size should be positive")
			assert.Greater(t, bucket.ObjectCount, int64(0), "Object count should be positive")

			// Ensure unique names
			for j := 0; j < i; j++ {
				assert.NotEqual(t, bucket.Name, buckets[j].Name, "Bucket names should be unique")
			}
		}
	})

	t.Run("Mock Object Generation", func(t *testing.T) {
		objects := generateMockObjects(10, "test-prefix/")
		assert.Len(t, objects, 10, "Should generate 10 objects")

		for i, obj := range objects {
			assert.NotEmpty(t, obj.Key, "Object key should not be empty")
			assert.Contains(t, obj.Key, "test-prefix/", "Object key should have prefix")
			assert.Greater(t, obj.Size, int64(0), "Size should be positive")
			assert.NotEmpty(t, obj.ETag, "ETag should not be empty")
			assert.Equal(t, "STANDARD", obj.StorageClass, "Storage class should be STANDARD")

			// Ensure unique keys
			for j := 0; j < i; j++ {
				assert.NotEqual(t, obj.Key, objects[j].Key, "Object keys should be unique")
			}
		}
	})
}

// TestErrorSimulation tests various error scenarios
func TestErrorSimulation(t *testing.T) {
	t.Run("Network Timeout Error", func(t *testing.T) {
		client := NewSimpleMockR2Client(true, "network timeout: request took too long")

		_, err := client.ListBuckets()
		assert.Error(t, err, "Should return timeout error")
		assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")
	})

	t.Run("Authentication Error", func(t *testing.T) {
		client := NewSimpleMockR2Client(true, "authentication failed: invalid credentials")

		_, err := client.ListBuckets()
		assert.Error(t, err, "Should return auth error")
		assert.Contains(t, err.Error(), "authentication", "Error should mention authentication")
	})

	t.Run("Permission Denied Error", func(t *testing.T) {
		client := NewSimpleMockR2Client(true, "permission denied: insufficient privileges")

		err := client.CreateBucket("restricted-bucket")
		assert.Error(t, err, "Should return permission error")
		assert.Contains(t, err.Error(), "permission denied", "Error should mention permission")
	})

	t.Run("Rate Limit Error", func(t *testing.T) {
		client := NewSimpleMockR2Client(true, "rate limit exceeded: retry after 60 seconds")

		_, err := client.ListBuckets()
		assert.Error(t, err, "Should return rate limit error")
		assert.Contains(t, err.Error(), "rate limit", "Error should mention rate limit")
	})
}