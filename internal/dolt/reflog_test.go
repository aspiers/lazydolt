package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestParseReflog(t *testing.T) {
	raw := `jucf2jic (HEAD -> main) Update product pricing
e64tblg8 (main) Add bulk order data
a552tplq (feature-branch) Add Charlie and Diana`

	entries := ParseReflog(raw)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	tests := []struct {
		idx     int
		hash    string
		ref     string
		message string
	}{
		{0, "jucf2jic", "HEAD -> main", "Update product pricing"},
		{1, "e64tblg8", "main", "Add bulk order data"},
		{2, "a552tplq", "feature-branch", "Add Charlie and Diana"},
	}

	for _, tt := range tests {
		e := entries[tt.idx]
		if e.Hash != tt.hash {
			t.Errorf("entry[%d].Hash = %q, want %q", tt.idx, e.Hash, tt.hash)
		}
		if e.Ref != tt.ref {
			t.Errorf("entry[%d].Ref = %q, want %q", tt.idx, e.Ref, tt.ref)
		}
		if e.Message != tt.message {
			t.Errorf("entry[%d].Message = %q, want %q", tt.idx, e.Message, tt.message)
		}
	}
}

func TestParseReflog_Empty(t *testing.T) {
	entries := ParseReflog("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestHeadHash(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	hash, err := runner.HeadHash()
	if err != nil {
		t.Fatalf("HeadHash(): %v", err)
	}
	if hash == "" {
		t.Error("HeadHash() returned empty string")
	}
	if len(hash) < 8 {
		t.Errorf("HeadHash() = %q, expected longer hash", hash)
	}
}

func TestReflogEntries(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	entries, err := runner.ReflogEntries()
	if err != nil {
		t.Fatalf("ReflogEntries(): %v", err)
	}
	if len(entries) < 1 {
		t.Error("expected at least 1 reflog entry")
	}
}
