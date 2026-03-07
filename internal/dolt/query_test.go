package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestQuery_Select(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	result, err := runner.Query("SELECT * FROM users ORDER BY id")
	if err != nil {
		t.Fatalf("Query(): %v", err)
	}
	// PopulateTestData inserts 3 rows (2 committed + 1 uncommitted)
	if len(result.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(result.Rows))
	}
	if len(result.Columns) == 0 {
		t.Error("expected columns in result")
	}
}

func TestQueryPage_Pagination(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	result, err := runner.QueryPage("users", 1, 0)
	if err != nil {
		t.Fatalf("QueryPage(): %v", err)
	}
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 row with limit=1, got %d", len(result.Rows))
	}
	if result.Total < 3 {
		t.Errorf("expected total >= 3, got %d", result.Total)
	}
}

func TestQueryPage_Offset(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	page1, err := runner.QueryPage("users", 1, 0)
	if err != nil {
		t.Fatalf("QueryPage(offset=0): %v", err)
	}
	page2, err := runner.QueryPage("users", 1, 1)
	if err != nil {
		t.Fatalf("QueryPage(offset=1): %v", err)
	}

	if len(page1.Rows) != 1 || len(page2.Rows) != 1 {
		t.Fatalf("expected 1 row per page, got %d and %d", len(page1.Rows), len(page2.Rows))
	}

	// Pages should return different rows
	id1 := page1.Rows[0]["id"]
	id2 := page2.Rows[0]["id"]
	if id1 == id2 {
		t.Errorf("pages should return different rows, both returned id=%v", id1)
	}
}
