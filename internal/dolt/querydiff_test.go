package dolt

import (
	"strings"
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestQueryDiff_IdenticalQueries(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	result, err := runner.QueryDiff(
		"SELECT * FROM users ORDER BY id",
		"SELECT * FROM users ORDER BY id",
	)
	if err != nil {
		t.Fatalf("QueryDiff(): %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for identical queries, got %q", result)
	}
}

func TestQueryDiff_DifferentQueries(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	result, err := runner.QueryDiff(
		"SELECT * FROM users ORDER BY id",
		"SELECT * FROM users WHERE id > 1 ORDER BY id",
	)
	if err != nil {
		t.Fatalf("QueryDiff(): %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result for different queries")
	}
	if !strings.Contains(result, "diff_type") {
		t.Errorf("expected diff_type column in output, got %q", result)
	}
	if !strings.Contains(result, "deleted") {
		t.Errorf("expected 'deleted' diff_type (row 1 removed), got %q", result)
	}
}

func TestQueryDiff_InvalidQuery(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	_, err := runner.QueryDiff(
		"SELECT * FROM nonexistent",
		"SELECT * FROM users",
	)
	if err == nil {
		t.Error("expected error for invalid query")
	}
}
