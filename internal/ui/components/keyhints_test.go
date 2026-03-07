package components

import (
	"strings"
	"testing"
)

func TestRenderKeyHints_TablesPanelHints(t *testing.T) {
	got := RenderKeyHints(PanelTables, 120)
	for _, expected := range []string{"stage", "diff", "schema", "browse"} {
		if !strings.Contains(got, expected) {
			t.Errorf("RenderKeyHints(PanelTables) should contain %q, got %q", expected, got)
		}
	}
	// At wide width, all hints should fit including quit
	wide := RenderKeyHints(PanelTables, 200)
	if !strings.Contains(wide, "quit") {
		t.Errorf("RenderKeyHints(PanelTables, 200) should contain %q, got %q", "quit", wide)
	}
}

func TestRenderKeyHints_BranchesPanelHints(t *testing.T) {
	got := RenderKeyHints(PanelBranches, 120)
	for _, expected := range []string{"checkout", "new", "delete", "quit"} {
		if !strings.Contains(got, expected) {
			t.Errorf("RenderKeyHints(PanelBranches) should contain %q, got %q", expected, got)
		}
	}
}

func TestRenderKeyHints_CommitsPanelHints(t *testing.T) {
	got := RenderKeyHints(PanelCommits, 120)
	for _, expected := range []string{"details", "quit"} {
		if !strings.Contains(got, expected) {
			t.Errorf("RenderKeyHints(PanelCommits) should contain %q, got %q", expected, got)
		}
	}
}

func TestRenderKeyHints_Truncation(t *testing.T) {
	wide := RenderKeyHints(PanelTables, 200)
	narrow := RenderKeyHints(PanelTables, 20)

	// Narrow should have fewer hints (or at least not be longer)
	wideW := len(wide)
	narrowW := len(narrow)
	if narrowW > wideW {
		t.Errorf("narrow hints (%d chars) should not be longer than wide hints (%d chars)", narrowW, wideW)
	}
}
