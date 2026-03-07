package dolt

import (
	"strings"
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestSchema_ValidTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	schema, err := runner.Schema("users")
	if err != nil {
		t.Fatalf("Schema(users): %v", err)
	}
	if schema.CreateStatement == "" {
		t.Error("Schema(users) returned empty CreateStatement")
	}
	if !strings.Contains(strings.ToUpper(schema.CreateStatement), "CREATE TABLE") {
		t.Errorf("Schema(users) should contain CREATE TABLE, got %q", schema.CreateStatement)
	}
}

func TestSchema_InvalidTable(t *testing.T) {
	runner, _ := newTestRunner(t)
	schema, err := runner.Schema("nonexistent_table")
	if err != nil {
		// dolt may or may not error — either behavior is acceptable
		return
	}
	// If no error, the CreateStatement should be empty
	if schema.CreateStatement != "" {
		t.Errorf("Schema(nonexistent_table) should have empty CreateStatement, got %q", schema.CreateStatement)
	}
}
