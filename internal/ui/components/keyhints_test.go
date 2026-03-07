package components

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

func TestRenderKeyHints_TablesPanelHints(t *testing.T) {
	golden.RequireEqual(t, RenderKeyHints(PanelTables, 120))
}

func TestRenderKeyHints_TablesPanelHintsWide(t *testing.T) {
	golden.RequireEqual(t, RenderKeyHints(PanelTables, 200))
}

func TestRenderKeyHints_BranchesPanelHints(t *testing.T) {
	golden.RequireEqual(t, RenderKeyHints(PanelBranches, 120))
}

func TestRenderKeyHints_CommitsPanelHints(t *testing.T) {
	golden.RequireEqual(t, RenderKeyHints(PanelCommits, 120))
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
