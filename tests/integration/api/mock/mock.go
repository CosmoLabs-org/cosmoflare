/*
Package mock provides mock Cloudflare R2 API testing

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package mock

import (
	"fmt"
	"time"
)

// MockBucket represents a mock bucket
type MockBucket struct {
	Name        string
	CreatedAt   time.Time
	Size        int64
	ObjectCount int64
	Status      string
	Region      string
}

// MockObject represents a mock object
type MockObject struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
}

// MockR2Client provides a mock implementation of the R2 API client
type MockR2Client struct {
	shouldFail   bool
	errorMessage string
	buckets     []MockBucket
	objects     []MockObject
}

// NewSimpleMockR2Client creates a new simple mock client
func NewSimpleMockR2Client(shouldFail bool, errorMessage string) *MockR2Client {
	return &MockR2Client{
		shouldFail:   shouldFail,
		errorMessage: errorMessage,
		buckets:     generateMockBuckets(3),
		objects:     generateMockObjects(5, "mock/"),
	}
}

// ListBuckets returns mock bucket data or error
func (m *MockR2Client) ListBuckets() ([]MockBucket, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.errorMessage)
	}
	return m.buckets, nil
}

// CreateBucket simulates bucket creation
func (m *MockR2Client) CreateBucket(name string) error {
	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.errorMessage)
	}

	// Add to mock buckets
	newBucket := MockBucket{
		Name:        name,
		CreatedAt:   time.Now(),
		Size:        0,
		ObjectCount: 0,
		Status:      "active",
		Region:      "auto",
	}
	m.buckets = append(m.buckets, newBucket)
	return nil
}

// generateMockBuckets creates mock bucket data
func generateMockBuckets(count int) []MockBucket {
	buckets := make([]MockBucket, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		buckets[i] = MockBucket{
			Name:        fmt.Sprintf("mock-bucket-%d", i),
			CreatedAt:   now.Add(-time.Duration(i) * time.Hour * 24),
			Size:        int64((i + 1) * 1024 * 1024),
			ObjectCount: int64((i + 1) * 10),
			Status:      "active",
			Region:      "auto",
		}
	}

	return buckets
}

// ListObjects returns mock object data or error
func (m *MockR2Client) ListObjects(bucketName string, prefix string) ([]MockObject, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.errorMessage)
	}
	return m.objects, nil
}

// GetBucket returns a specific mock bucket or error
func (m *MockR2Client) GetBucket(name string) (*MockBucket, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.errorMessage)
	}

	for _, bucket := range m.buckets {
		if bucket.Name == name {
			return &bucket, nil
		}
	}

	return nil, fmt.Errorf("bucket not found: %s", name)
}

// UploadFile simulates file upload
func (m *MockR2Client) UploadFile(bucketName, objectKey, filePath string) error {
	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.errorMessage)
	}

	// Simulate adding object
	newObject := MockObject{
		Key:          objectKey,
		Size:         1024, // Mock size
		LastModified: time.Now(),
		ETag:         fmt.Sprintf("\"mock-upload-etag-%d\"", len(m.objects)),
		StorageClass: "STANDARD",
	}
	m.objects = append(m.objects, newObject)
	return nil
}

// DownloadFile simulates file download
func (m *MockR2Client) DownloadFile(bucketName, objectKey, filePath string) error {
	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.errorMessage)
	}
	return nil
}

// generateMockObjects creates mock object data
func generateMockObjects(count int, prefix string) []MockObject {
	objects := make([]MockObject, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		objects[i] = MockObject{
			Key:          fmt.Sprintf("%smock-object-%d.txt", prefix, i),
			Size:         int64((i + 1) * 1024),
			LastModified: now.Add(-time.Duration(i) * time.Minute),
			ETag:         fmt.Sprintf("\"mock-etag-%d\"", i),
			StorageClass: "STANDARD",
		}
	}

	return objects
}