package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestTables_ListsAll(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	tables, err := runner.Tables()
	if err != nil {
		t.Fatalf("Tables(): %v", err)
	}
	if len(tables) < 2 {
		t.Fatalf("expected at least 2 tables, got %d", len(tables))
	}

	names := make(map[string]bool)
	for _, tbl := range tables {
		names[tbl.Name] = true
	}
	for _, expected := range []string{"users", "orders"} {
		if !names[expected] {
			t.Errorf("expected table %q in list, got %v", expected, names)
		}
	}
}

func TestTables_ExcludesSystem(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	tables, err := runner.Tables()
	if err != nil {
		t.Fatalf("Tables(): %v", err)
	}
	for _, tbl := range tables {
		if len(tbl.Name) > 5 && tbl.Name[:5] == "dolt_" {
			t.Errorf("system table %q should not appear in Tables()", tbl.Name)
		}
	}
}

func TestTables_WithStatus(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	tables, err := runner.Tables()
	if err != nil {
		t.Fatalf("Tables(): %v", err)
	}

	// orders is new and unstaged; users is modified and unstaged
	for _, tbl := range tables {
		if tbl.Name == "orders" || tbl.Name == "users" {
			if tbl.Status == nil {
				t.Errorf("table %q should have status (has uncommitted changes)", tbl.Name)
			}
		}
	}
}
