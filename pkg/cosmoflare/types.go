/*
Package r2go2 provides a Go library for managing the full Cloudflare developer platform.

It covers R2 (storage), Workers (compute), KV (key-value), D1 (SQL database),
Pages (static hosting), and Queues (message queues) — all configured through
a single .r2go2.yaml project config.

Library-first design: zero CCS dependencies. Importable by any Go project.
*/
package cosmoflare

import (
	"io"
	"time"
)

// Bucket represents a Cloudflare R2 bucket.
type Bucket struct {
	Name        string            `json:"name"`
	CreatedAt   time.Time         `json:"created_at"`
	Location    string            `json:"location,omitempty"`
	Storage     string            `json:"storage,omitempty"`
	Size        int64             `json:"size"`
	ObjectCount int64             `json:"object_count"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// Object represents an object stored in an R2 bucket.
type Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	StorageClass string            `json:"storage_class"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	Key       string    `json:"key"`
	Bucket    string    `json:"bucket"`
	Size      int64     `json:"size"`
	ETag      string    `json:"etag"`
	VersionID string    `json:"version_id,omitempty"`
	Uploaded  time.Time `json:"uploaded"`
	Parts     int       `json:"parts,omitempty"`
}

// DownloadResult contains the result of a download operation.
type DownloadResult struct {
	Key       string `json:"key"`
	Bucket    string `json:"bucket"`
	Size      int64  `json:"size"`
	Content   io.ReadCloser
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ListResult is a generic paginated list result.
type ListResult[T any] struct {
	Items          []T      `json:"items"`
	NextToken      string   `json:"next_token,omitempty"`
	IsTruncated    bool     `json:"is_truncated"`
	CommonPrefixes []string `json:"common_prefixes,omitempty"`
}

// CopyResult contains the result of a copy operation.
type CopyResult struct {
	Key        string `json:"key"`
	SourceKey  string `json:"source_key"`
	Bucket     string `json:"bucket"`
	ETag       string `json:"etag"`
	VersionID  string `json:"version_id,omitempty"`
}

// HeadResult contains object metadata without the body.
type HeadResult struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type,omitempty"`
	CacheControl string            `json:"cache_control,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
