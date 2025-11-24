/*
Package mock provides mock Cloudflare R2 API testing

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
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