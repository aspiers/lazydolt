package components

// visibleRange computes the start (inclusive) and end (exclusive) indices
// for a scrollable list so that the cursor is always visible within the
// given viewHeight. If the list is shorter than the view, all items are
// shown. Otherwise the window scrolls to keep the cursor in view.
func visibleRange(cursor, total, viewHeight int) (start, end int) {
	if viewHeight <= 0 {
		viewHeight = 1
	}
	if total <= viewHeight {
		return 0, total
	}

	// Keep cursor visible: put it roughly in the middle of the window,
	// but clamp to list bounds.
	start = cursor - viewHeight/2
	if start < 0 {
		start = 0
	}
	end = start + viewHeight
	if end > total {
		end = total
		start = end - viewHeight
	}

	return start, end
}
