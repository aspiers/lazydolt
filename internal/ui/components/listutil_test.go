package components

import "testing"

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		name       string
		cursor     int
		total      int
		viewHeight int
		wantStart  int
		wantEnd    int
	}{
		{
			name:       "list fits in view",
			cursor:     0,
			total:      3,
			viewHeight: 10,
			wantStart:  0,
			wantEnd:    3,
		},
		{
			name:       "list exactly fits",
			cursor:     0,
			total:      5,
			viewHeight: 5,
			wantStart:  0,
			wantEnd:    5,
		},
		{
			name:       "cursor at top of long list",
			cursor:     0,
			total:      20,
			viewHeight: 5,
			wantStart:  0,
			wantEnd:    5,
		},
		{
			name:       "cursor in middle of long list",
			cursor:     10,
			total:      20,
			viewHeight: 5,
			wantStart:  8,
			wantEnd:    13,
		},
		{
			name:       "cursor near end of long list",
			cursor:     18,
			total:      20,
			viewHeight: 5,
			wantStart:  15,
			wantEnd:    20,
		},
		{
			name:       "cursor at last item",
			cursor:     19,
			total:      20,
			viewHeight: 5,
			wantStart:  15,
			wantEnd:    20,
		},
		{
			name:       "single item list",
			cursor:     0,
			total:      1,
			viewHeight: 5,
			wantStart:  0,
			wantEnd:    1,
		},
		{
			name:       "viewHeight zero treated as 1",
			cursor:     3,
			total:      10,
			viewHeight: 0,
			wantStart:  3,
			wantEnd:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleRange(tt.cursor, tt.total, tt.viewHeight)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("visibleRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.cursor, tt.total, tt.viewHeight, start, end, tt.wantStart, tt.wantEnd)
			}
			// Invariant: cursor must be within [start, end)
			if tt.cursor < start || tt.cursor >= end {
				t.Errorf("cursor %d not in visible range [%d, %d)", tt.cursor, start, end)
			}
		})
	}
}
