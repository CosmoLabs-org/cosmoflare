package cmd

import (
	"errors"
	"sort"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// compareRunSnapshot snapshots the compare prefix flag plus output-mode and
// credential globals so tests stay isolated.
func compareRunSnapshot(t *testing.T) {
	t.Helper()
	savedPrefix := comparePrefix
	savedJSON, savedDry := JSONOutput, DryRun
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() {
		comparePrefix = savedPrefix
		JSONOutput, DryRun = savedJSON, savedDry
		AccountID, APIToken = savedAccount, savedToken
	})
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	comparePrefix = ""
	JSONOutput, DryRun = false, false
	AccountID, APIToken = "", ""
}

// TestRunCompare_ClientCreationError verifies runCompare with two bucket
// arguments surfaces the API-client creation failure instead of proceeding.
func TestRunCompare_ClientCreationError(t *testing.T) {
	compareRunSnapshot(t)

	err := runCompare(nil, []string{"src-bucket", "dst-bucket"})
	if err == nil {
		t.Fatal("expected error when no credentials are configured")
	}
	if !strings.Contains(err.Error(), "failed to create API client") {
		t.Errorf("error = %q, want it to mention failed to create API client", err.Error())
	}
}

// TestCompareListBuckets_BuildsKeyMaps verifies each bucket's objects are
// reduced to a key→size map and the prefix flag is forwarded to ListObjects.
func TestCompareListBuckets_BuildsKeyMaps(t *testing.T) {
	compareRunSnapshot(t)
	comparePrefix = "images/"

	client := &fakeR2Client{objects: map[string][]*cosmoflare.Object{
		"src": {{Key: "images/a.jpg", Size: 10}, {Key: "images/b.jpg", Size: 20}},
		"dst": {{Key: "images/a.jpg", Size: 10}},
	}}

	srcMap, dstMap, err := compareListBuckets(client, "src", "dst")
	if err != nil {
		t.Fatalf("compareListBuckets should succeed: %v", err)
	}

	if len(srcMap) != 2 || srcMap["images/a.jpg"] != 10 || srcMap["images/b.jpg"] != 20 {
		t.Errorf("srcMap = %v, want sizes for images/a.jpg=10 and images/b.jpg=20", srcMap)
	}
	if len(dstMap) != 1 || dstMap["images/a.jpg"] != 10 {
		t.Errorf("dstMap = %v, want single entry images/a.jpg=10", dstMap)
	}
	for _, p := range client.listedPrefixes {
		if p != "images/" {
			t.Errorf("ListObjects received prefix %q, want images/", p)
		}
	}
}

// TestCompareListBuckets_EmptyBuckets verifies both maps come back empty
// (not nil-errored) when neither bucket has objects.
func TestCompareListBuckets_EmptyBuckets(t *testing.T) {
	compareRunSnapshot(t)

	client := &fakeR2Client{}
	srcMap, dstMap, err := compareListBuckets(client, "src", "dst")
	if err != nil {
		t.Fatalf("empty buckets should not error: %v", err)
	}
	if len(srcMap) != 0 || len(dstMap) != 0 {
		t.Errorf("expected empty maps, got src=%v dst=%v", srcMap, dstMap)
	}
}

// TestCompareListBuckets_ListErrors verifies source and destination listing
// failures each return the corresponding wrapped error.
func TestCompareListBuckets_ListErrors(t *testing.T) {
	cases := []struct {
		name    string
		failing string
		wantErr string
	}{
		{"source-list-fails", "src", "failed to list source bucket"},
		{"dest-list-fails", "dst", "failed to list destination bucket"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			compareRunSnapshot(t)
			client := &fakeR2Client{
				objects:  map[string][]*cosmoflare.Object{},
				listErrs: map[string]error{tc.failing: errors.New("simulated listing failure")},
			}

			_, _, err := compareListBuckets(client, "src", "dst")
			if err == nil {
				t.Fatal("expected error from failing bucket list")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestCompareDiffBuckets_Classification verifies every key lands in exactly
// one bucket of the comparison: only-in-source, only-in-dest, different-size,
// or same.
func TestCompareDiffBuckets_Classification(t *testing.T) {
	src := map[string]int64{
		"only-src": 5,
		"same-key": 7,
		"diff-key": 100,
	}
	dst := map[string]int64{
		"same-key": 7,
		"diff-key": 42,
		"only-dst": 9,
	}

	got := compareDiffBuckets(src, dst)

	sort.Strings(got.OnlyInSource)
	sort.Strings(got.OnlyInDest)
	sort.Strings(got.Same)

	if len(got.OnlyInSource) != 1 || got.OnlyInSource[0] != "only-src" {
		t.Errorf("OnlyInSource = %v, want [only-src]", got.OnlyInSource)
	}
	if len(got.OnlyInDest) != 1 || got.OnlyInDest[0] != "only-dst" {
		t.Errorf("OnlyInDest = %v, want [only-dst]", got.OnlyInDest)
	}
	if len(got.Same) != 1 || got.Same[0] != "same-key" {
		t.Errorf("Same = %v, want [same-key]", got.Same)
	}
	if len(got.DifferentSize) != 1 {
		t.Fatalf("DifferentSize = %v, want one entry", got.DifferentSize)
	}
	d := got.DifferentSize[0]
	if d.Key != "diff-key" || d.SourceSize != 100 || d.DestSize != 42 {
		t.Errorf("DifferentSize entry = %+v, want key=diff-key source=100 dest=42", d)
	}
}

// TestCompareDiffBuckets_EmptyAndIdentical verifies nil and empty maps, plus
// identical maps, classify without panicking and leave no phantom entries.
func TestCompareDiffBuckets_EmptyAndIdentical(t *testing.T) {
	cases := []struct {
		name     string
		src      map[string]int64
		dst      map[string]int64
		wantDest int
		wantSame int
	}{
		{"both-nil", nil, nil, 0, 0},
		{"both-empty", map[string]int64{}, map[string]int64{}, 0, 0},
		{"src-nil-dst-populated", nil, map[string]int64{"k": 1}, 1, 0},
		{"identical-maps", map[string]int64{"k": 3}, map[string]int64{"k": 3}, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareDiffBuckets(tc.src, tc.dst)

			if len(got.OnlyInSource) != 0 {
				t.Errorf("OnlyInSource = %v, want none", got.OnlyInSource)
			}
			if len(got.DifferentSize) != 0 {
				t.Errorf("DifferentSize = %v, want none", got.DifferentSize)
			}
			if len(got.OnlyInDest) != tc.wantDest {
				t.Errorf("OnlyInDest = %v, want %d entries", got.OnlyInDest, tc.wantDest)
			}
			if len(got.Same) != tc.wantSame {
				t.Errorf("Same = %v, want %d entries", got.Same, tc.wantSame)
			}
		})
	}
}
