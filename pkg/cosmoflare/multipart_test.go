package cosmoflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- State file path generation ---

func TestStateFilePath_Deterministic(t *testing.T) {
	path1 := stateFilePath("my-bucket", "my-key")
	path2 := stateFilePath("my-bucket", "my-key")
	if path1 != path2 {
		t.Errorf("stateFilePath not deterministic: %q != %q", path1, path2)
	}
}

func TestStateFilePath_DifferentKeys(t *testing.T) {
	path1 := stateFilePath("bucket", "key1")
	path2 := stateFilePath("bucket", "key2")
	if path1 == path2 {
		t.Error("different keys should produce different state file paths")
	}
}

func TestStateFilePath_DifferentBuckets(t *testing.T) {
	path1 := stateFilePath("bucket1", "key")
	path2 := stateFilePath("bucket2", "key")
	if path1 == path2 {
		t.Error("different buckets should produce different state file paths")
	}
}

func TestStateFilePath_HasPrefix(t *testing.T) {
	path := stateFilePath("bucket", "key")
	base := filepath.Base(path)
	if !strings.HasPrefix(base, stateFilePrefix) {
		t.Errorf("state file should have prefix %q, got base %q", stateFilePrefix, base)
	}
}

func TestStateFilePath_HasJSONExtension(t *testing.T) {
	path := stateFilePath("bucket", "key")
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("state file should end with .json, got %q", path)
	}
}

func TestStateFilePath_ExportedMatchesInternal(t *testing.T) {
	internal := stateFilePath("bucket", "key")
	exported := StateFilePath("bucket", "key")
	if internal != exported {
		t.Errorf("StateFilePath should match stateFilePath: %q != %q", exported, internal)
	}
}

// --- Save and Load round-trip ---

func TestSaveLoadUploadState_RoundTrip(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "test-upload-id-123",
		Bucket:     "test-bucket-roundtrip",
		Key:        "test-key-roundtrip.dat",
		TotalSize:  1024 * 1024 * 200,
		PartSize:   8 * 1024 * 1024,
		TotalParts: 25,
		CompletedParts: []CompletedPartInfo{
			{PartNumber: 1, ETag: "etag-1", Size: 8 * 1024 * 1024},
			{PartNumber: 2, ETag: "etag-2", Size: 8 * 1024 * 1024},
		},
		StartedAt:    time.Now().UTC().Truncate(time.Second),
		ContentType:  "application/octet-stream",
		CacheControl: "no-cache",
		Metadata:     map[string]string{"env": "test"},
	}

	err := SaveUploadState(state)
	if err != nil {
		t.Fatalf("SaveUploadState failed: %v", err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)

	loaded, err := LoadUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatalf("LoadUploadState failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded state is nil")
	}

	if loaded.UploadID != state.UploadID {
		t.Errorf("UploadID: got %q, want %q", loaded.UploadID, state.UploadID)
	}
	if loaded.Bucket != state.Bucket {
		t.Errorf("Bucket: got %q, want %q", loaded.Bucket, state.Bucket)
	}
	if loaded.Key != state.Key {
		t.Errorf("Key: got %q, want %q", loaded.Key, state.Key)
	}
	if loaded.TotalSize != state.TotalSize {
		t.Errorf("TotalSize: got %d, want %d", loaded.TotalSize, state.TotalSize)
	}
	if loaded.PartSize != state.PartSize {
		t.Errorf("PartSize: got %d, want %d", loaded.PartSize, state.PartSize)
	}
	if loaded.TotalParts != state.TotalParts {
		t.Errorf("TotalParts: got %d, want %d", loaded.TotalParts, state.TotalParts)
	}
	if len(loaded.CompletedParts) != 2 {
		t.Fatalf("CompletedParts: got %d, want 2", len(loaded.CompletedParts))
	}
	if loaded.ContentType != state.ContentType {
		t.Errorf("ContentType: got %q, want %q", loaded.ContentType, state.ContentType)
	}
	if loaded.CacheControl != state.CacheControl {
		t.Errorf("CacheControl: got %q, want %q", loaded.CacheControl, state.CacheControl)
	}
	if loaded.Metadata["env"] != "test" {
		t.Errorf("Metadata[env]: got %q, want %q", loaded.Metadata["env"], "test")
	}
}

// --- Save validation ---

func TestSaveUploadState_NilState(t *testing.T) {
	err := SaveUploadState(nil)
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestSaveUploadState_EmptyUploadID(t *testing.T) {
	state := &MultipartUploadState{Bucket: "bucket", Key: "key"}
	err := SaveUploadState(state)
	if err == nil {
		t.Fatal("expected error for empty upload ID")
	}
}

func TestSaveUploadState_EmptyBucket(t *testing.T) {
	state := &MultipartUploadState{UploadID: "id", Key: "key"}
	err := SaveUploadState(state)
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
}

func TestSaveUploadState_EmptyKey(t *testing.T) {
	state := &MultipartUploadState{UploadID: "id", Bucket: "bucket"}
	err := SaveUploadState(state)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

// --- Load validation ---

func TestLoadUploadState_EmptyBucket(t *testing.T) {
	_, err := LoadUploadState("", "key")
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
}

func TestLoadUploadState_EmptyKey(t *testing.T) {
	_, err := LoadUploadState("bucket", "")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestLoadUploadState_NoStateFile(t *testing.T) {
	state, err := LoadUploadState("nonexistent-bucket-abc123", "nonexistent-key-xyz789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != nil {
		t.Fatal("expected nil state for nonexistent file")
	}
}

func TestLoadUploadState_CorruptedJSON(t *testing.T) {
	bucket := "corrupt-test-bucket"
	key := "corrupt-test-key"
	path := stateFilePath(bucket, key)

	if err := os.WriteFile(path, []byte("not valid json{{{"), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	_, err := LoadUploadState(bucket, key)
	if err == nil {
		t.Fatal("expected error for corrupted JSON")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption, got: %v", err)
	}
}

func TestLoadUploadState_ChecksumMismatch(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "test-id",
		Bucket:     "checksum-test-bucket",
		Key:        "checksum-test-key",
		TotalSize:  100,
		PartSize:   50,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	err := SaveUploadState(state)
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)

	// Tamper with the file
	path := stateFilePath(state.Bucket, state.Key)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), `"total_size": 100`, `"total_size": 999`, 1)
	if err := os.WriteFile(path, []byte(tampered), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = LoadUploadState(state.Bucket, state.Key)
	if err == nil {
		t.Fatal("expected error for checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("error should mention checksum, got: %v", err)
	}
}

// --- Remove state ---

func TestRemoveUploadState_Existing(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "remove-test-id",
		Bucket:     "remove-test-bucket",
		Key:        "remove-test-key",
		TotalSize:  100,
		PartSize:   50,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}

	err := RemoveUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatalf("RemoveUploadState failed: %v", err)
	}

	loaded, err := LoadUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Fatal("state should be nil after removal")
	}
}

func TestRemoveUploadState_Nonexistent(t *testing.T) {
	err := RemoveUploadState("no-bucket", "no-key")
	if err != nil {
		t.Errorf("RemoveUploadState for nonexistent file should not error: %v", err)
	}
}

// --- Part tracking ---

func TestIsPartCompleted(t *testing.T) {
	state := &MultipartUploadState{
		CompletedParts: []CompletedPartInfo{
			{PartNumber: 1, ETag: "e1", Size: 100},
			{PartNumber: 3, ETag: "e3", Size: 100},
			{PartNumber: 5, ETag: "e5", Size: 100},
		},
	}

	tests := []struct {
		partNumber int32
		want       bool
	}{
		{1, true},
		{2, false},
		{3, true},
		{4, false},
		{5, true},
		{6, false},
	}

	for _, tt := range tests {
		got := state.IsPartCompleted(tt.partNumber)
		if got != tt.want {
			t.Errorf("IsPartCompleted(%d) = %v, want %v", tt.partNumber, got, tt.want)
		}
	}
}

func TestCompletedBytes(t *testing.T) {
	state := &MultipartUploadState{
		CompletedParts: []CompletedPartInfo{
			{PartNumber: 1, ETag: "e1", Size: 1000},
			{PartNumber: 2, ETag: "e2", Size: 2000},
			{PartNumber: 3, ETag: "e3", Size: 500},
		},
	}

	got := state.CompletedBytes()
	want := int64(3500)
	if got != want {
		t.Errorf("CompletedBytes() = %d, want %d", got, want)
	}
}

func TestCompletedBytes_Empty(t *testing.T) {
	state := &MultipartUploadState{}
	got := state.CompletedBytes()
	if got != 0 {
		t.Errorf("CompletedBytes() for empty state = %d, want 0", got)
	}
}

func TestRemainingParts(t *testing.T) {
	state := &MultipartUploadState{
		TotalParts: 5,
		CompletedParts: []CompletedPartInfo{
			{PartNumber: 1, ETag: "e1", Size: 100},
			{PartNumber: 3, ETag: "e3", Size: 100},
		},
	}

	remaining := state.RemainingParts()
	expected := []int32{2, 4, 5}
	if len(remaining) != len(expected) {
		t.Fatalf("RemainingParts() returned %d parts, want %d", len(remaining), len(expected))
	}
	for i, got := range remaining {
		if got != expected[i] {
			t.Errorf("RemainingParts()[%d] = %d, want %d", i, got, expected[i])
		}
	}
}

func TestRemainingParts_AllComplete(t *testing.T) {
	state := &MultipartUploadState{
		TotalParts: 3,
		CompletedParts: []CompletedPartInfo{
			{PartNumber: 1, ETag: "e1", Size: 100},
			{PartNumber: 2, ETag: "e2", Size: 100},
			{PartNumber: 3, ETag: "e3", Size: 100},
		},
	}

	remaining := state.RemainingParts()
	if len(remaining) != 0 {
		t.Errorf("RemainingParts() for complete upload = %v, want empty", remaining)
	}
}

func TestRemainingParts_NoneComplete(t *testing.T) {
	state := &MultipartUploadState{TotalParts: 4}

	remaining := state.RemainingParts()
	expected := []int32{1, 2, 3, 4}
	if len(remaining) != len(expected) {
		t.Fatalf("RemainingParts() returned %d parts, want %d", len(remaining), len(expected))
	}
	for i, got := range remaining {
		if got != expected[i] {
			t.Errorf("RemainingParts()[%d] = %d, want %d", i, got, expected[i])
		}
	}
}

// --- Threshold detection ---

func TestShouldUseMultipart(t *testing.T) {
	tests := []struct {
		size      int64
		threshold int64
		want      bool
	}{
		{50 * 1024 * 1024, 0, false},                        // 50MB < 100MB default
		{150 * 1024 * 1024, 0, true},                        // 150MB > 100MB default
		{100 * 1024 * 1024, 0, false},                       // exactly 100MB, strict >
		{10 * 1024 * 1024, 5 * 1024 * 1024, true},           // 10MB > 5MB custom
		{5 * 1024 * 1024, 5 * 1024 * 1024, false},           // exactly at custom
		{1024, 5 * 1024 * 1024, false},                      // 1KB < 5MB custom
	}

	for _, tt := range tests {
		got := ShouldUseMultipart(tt.size, tt.threshold)
		if got != tt.want {
			t.Errorf("ShouldUseMultipart(%d, %d) = %v, want %v",
				tt.size, tt.threshold, got, tt.want)
		}
	}
}

// --- Part size validation ---

func TestValidatePartSize(t *testing.T) {
	tests := []struct {
		name     string
		partSize int64
		wantErr  bool
	}{
		{"below minimum", 1024 * 1024, true},
		{"exactly minimum", MinPartSize, false},
		{"default size", defaultPartSize, false},
		{"large size", 100 * 1024 * 1024, false},
		{"exactly maximum", MaxPartSize, false},
		{"above maximum", MaxPartSize + 1, true},
		{"zero", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePartSize(tt.partSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePartSize(%d) error = %v, wantErr %v", tt.partSize, err, tt.wantErr)
			}
		})
	}
}

// --- Constants ---

func TestMinPartSizeConstant(t *testing.T) {
	expected := int64(5 * 1024 * 1024)
	if MinPartSize != expected {
		t.Errorf("MinPartSize = %d, want %d (5MB)", MinPartSize, expected)
	}
}

func TestMaxPartSizeConstant(t *testing.T) {
	expected := int64(5 * 1024 * 1024 * 1024)
	if MaxPartSize != expected {
		t.Errorf("MaxPartSize = %d, want %d (5GB)", MaxPartSize, expected)
	}
}

// --- Checksum ---

func TestComputeStateChecksum_Deterministic(t *testing.T) {
	state := &MultipartUploadState{
		UploadID: "id1", Bucket: "b1", Key: "k1",
		TotalSize: 100, PartSize: 50, TotalParts: 2,
	}

	cs1 := computeStateChecksum(state)
	cs2 := computeStateChecksum(state)
	if cs1 != cs2 {
		t.Errorf("checksum not deterministic: %q != %q", cs1, cs2)
	}
}

func TestComputeStateChecksum_DifferentStates(t *testing.T) {
	state1 := &MultipartUploadState{
		UploadID: "id1", Bucket: "b1", Key: "k1",
		TotalSize: 100, PartSize: 50,
	}
	state2 := &MultipartUploadState{
		UploadID: "id2", Bucket: "b1", Key: "k1",
		TotalSize: 100, PartSize: 50,
	}

	cs1 := computeStateChecksum(state1)
	cs2 := computeStateChecksum(state2)
	if cs1 == cs2 {
		t.Error("different states should produce different checksums")
	}
}

// --- Save updates UpdatedAt ---

func TestSaveUploadState_SetsUpdatedAt(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "updated-at-test",
		Bucket:     "updated-at-bucket",
		Key:        "updated-at-key",
		TotalSize:  100,
		PartSize:   50,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	before := time.Now().UTC()
	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)
	after := time.Now().UTC()

	loaded, err := LoadUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.UpdatedAt.Before(before.Add(-time.Second)) || loaded.UpdatedAt.After(after.Add(time.Second)) {
		t.Errorf("UpdatedAt %v not between %v and %v", loaded.UpdatedAt, before, after)
	}
}

// --- CompletedPartInfo ---

func TestCompletedPartInfo_Fields(t *testing.T) {
	p := CompletedPartInfo{
		PartNumber: 42,
		ETag:       "\"abc123\"",
		Size:       8388608,
	}
	if p.PartNumber != 42 {
		t.Errorf("PartNumber = %d, want 42", p.PartNumber)
	}
	if p.ETag != "\"abc123\"" {
		t.Errorf("ETag = %q, want %q", p.ETag, "\"abc123\"")
	}
	if p.Size != 8388608 {
		t.Errorf("Size = %d, want 8388608", p.Size)
	}
}

// --- State file permissions ---

func TestSaveUploadState_FilePermissions(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "perms-test",
		Bucket:     "perms-bucket",
		Key:        "perms-key",
		TotalSize:  100,
		PartSize:   50,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)

	path := stateFilePath(state.Bucket, state.Key)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("state file permissions = %o, want 0600", perm)
	}
}

// --- MultipartUploadState struct defaults ---

func TestMultipartUploadState_ZeroValue(t *testing.T) {
	state := &MultipartUploadState{}
	if state.UploadID != "" {
		t.Error("zero-value UploadID should be empty")
	}
	if state.CompletedParts != nil {
		t.Error("zero-value CompletedParts should be nil")
	}
	if state.CompletedBytes() != 0 {
		t.Error("zero-value CompletedBytes should be 0")
	}
}

// --- Save overwrite behavior ---

func TestSaveUploadState_Overwrite(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "overwrite-test",
		Bucket:     "overwrite-bucket",
		Key:        "overwrite-key",
		TotalSize:  200,
		PartSize:   100,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	// Save initial state with no completed parts
	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)

	// Add a completed part and save again
	state.CompletedParts = append(state.CompletedParts, CompletedPartInfo{
		PartNumber: 1, ETag: "etag-1", Size: 100,
	})
	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}

	// Load and verify the update persisted
	loaded, err := LoadUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CompletedParts) != 1 {
		t.Fatalf("CompletedParts: got %d, want 1", len(loaded.CompletedParts))
	}
	if loaded.CompletedParts[0].PartNumber != 1 {
		t.Errorf("CompletedParts[0].PartNumber = %d, want 1", loaded.CompletedParts[0].PartNumber)
	}
}

// --- Validate part size edge cases ---

func TestValidatePartSize_ExactlyFiveMB(t *testing.T) {
	err := ValidatePartSize(5 * 1024 * 1024)
	if err != nil {
		t.Errorf("exactly 5MB should be valid: %v", err)
	}
}

func TestValidatePartSize_JustBelowMinimum(t *testing.T) {
	err := ValidatePartSize(MinPartSize - 1)
	if err == nil {
		t.Fatal("one byte below minimum should fail")
	}
}

func TestValidatePartSize_JustAboveMaximum(t *testing.T) {
	err := ValidatePartSize(MaxPartSize + 1)
	if err == nil {
		t.Fatal("one byte above maximum should fail")
	}
}

// --- ShouldUseMultipart with negative threshold ---

func TestShouldUseMultipart_NegativeThreshold(t *testing.T) {
	// Negative threshold should use default (100MB)
	got := ShouldUseMultipart(50*1024*1024, -1)
	if got != false {
		t.Error("50MB with negative threshold (uses default 100MB) should return false")
	}
}

// --- State with special characters in key ---

func TestSaveLoadUploadState_SpecialCharsInKey(t *testing.T) {
	state := &MultipartUploadState{
		UploadID:   "special-chars-test",
		Bucket:     "special-bucket",
		Key:        "path/to/file with spaces & symbols!.tar.gz",
		TotalSize:  1000,
		PartSize:   500,
		TotalParts: 2,
		StartedAt:  time.Now().UTC(),
	}

	if err := SaveUploadState(state); err != nil {
		t.Fatalf("SaveUploadState with special chars failed: %v", err)
	}
	defer RemoveUploadState(state.Bucket, state.Key)

	loaded, err := LoadUploadState(state.Bucket, state.Key)
	if err != nil {
		t.Fatalf("LoadUploadState with special chars failed: %v", err)
	}
	if loaded.Key != state.Key {
		t.Errorf("Key: got %q, want %q", loaded.Key, state.Key)
	}
}
