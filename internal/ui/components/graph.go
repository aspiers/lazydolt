package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Graph characters using Unicode box-drawing.
const (
	graphCommit  = "●"
	graphLine    = "│"
	graphMergeIn = "╯"
	graphForkOut = "╮"
	graphHoriz   = "─"
	graphCross   = "┼"
	graphEmpty   = " "
)

// graphColors are the colors used for each lane, cycling through them.
var graphColors = []lipgloss.Color{"1", "2", "3", "4", "5", "6", "9", "10", "11", "12", "13", "14"}

// GraphLine holds the styled graph prefix for a single commit row.
type GraphLine struct {
	Text string // styled string to prepend to the commit line
}

// BuildGraph computes graph line prefixes for a list of commits.
// Commits must be ordered most-recent first (topological/date order).
// Each commit's Parents field lists its parent hashes.
//
// The algorithm tracks "lanes" — vertical columns representing active
// branches passing through the log. Each lane holds the expected next
// commit hash for that lane. When a commit matches a lane, it occupies
// that lane; otherwise a new lane is created.
func BuildGraph(commits []domain.Commit) []GraphLine {
	if len(commits) == 0 {
		return nil
	}

	// lanes holds the expected hash for each active lane.
	// A lane is "waiting" for a specific parent commit.
	var lanes []string

	result := make([]GraphLine, len(commits))

	for i, c := range commits {
		// Find which lane this commit occupies.
		myLane := -1
		for j, laneHash := range lanes {
			if laneHash == c.Hash {
				myLane = j
				break
			}
		}

		// If no lane is waiting for this commit, add a new lane.
		if myLane == -1 {
			myLane = len(lanes)
			lanes = append(lanes, c.Hash)
		}

		// Find additional lanes waiting for this same commit (merge convergence).
		// These extra lanes should be removed (the branch merged in).
		var mergingLanes []int
		for j, laneHash := range lanes {
			if j != myLane && laneHash == c.Hash {
				mergingLanes = append(mergingLanes, j)
			}
		}

		// Build the graph line for this commit.
		var parts []string
		for j := 0; j < len(lanes); j++ {
			color := graphColors[j%len(graphColors)]
			style := lipgloss.NewStyle().Foreground(color)

			if j == myLane {
				parts = append(parts, style.Render(graphCommit))
			} else if containsInt(mergingLanes, j) {
				// This lane merges into our lane
				if j > myLane {
					parts = append(parts, style.Render(graphMergeIn))
				} else {
					parts = append(parts, style.Render(graphMergeIn))
				}
			} else {
				parts = append(parts, style.Render(graphLine))
			}
		}

		result[i] = GraphLine{Text: strings.Join(parts, "")}

		// Remove merging lanes (right-to-left to preserve indices).
		for k := len(mergingLanes) - 1; k >= 0; k-- {
			idx := mergingLanes[k]
			lanes = append(lanes[:idx], lanes[idx+1:]...)
		}

		// Update lanes with this commit's parents.
		// First parent continues in the same lane.
		if len(c.Parents) > 0 {
			lanes[myLane] = c.Parents[0]
		} else {
			// Root commit — remove the lane.
			lanes = append(lanes[:myLane], lanes[myLane+1:]...)
		}

		// Additional parents create new lanes (fork points for merges).
		for p := 1; p < len(c.Parents); p++ {
			// Check if any existing lane already has this parent.
			found := false
			for _, lh := range lanes {
				if lh == c.Parents[p] {
					found = true
					break
				}
			}
			if !found {
				lanes = append(lanes, c.Parents[p])
			}
		}
	}

	return result
}

// MaxGraphWidth returns the maximum character width of graph lines.
func MaxGraphWidth(lines []GraphLine) int {
	maxW := 0
	for _, gl := range lines {
		w := lipgloss.Width(gl.Text)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
