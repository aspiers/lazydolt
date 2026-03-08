# lazydolt demo — a TUI for Dolt databases
#
# This tape generates the animated GIF used in the README.
# Run with: vhs demo.tape
#
# Prerequisites:
#   - lazydolt binary at ~/bin/lazydolt
#   - Test database at tmp/test-dolt (run scripts/setup-test-db.sh)

Output demo.gif

Require lazydolt

Set Shell "bash"
Set FontSize 14
Set FontFamily "JetBrains Mono,Fira Code,monospace"
Set Width 1200
Set Height 700
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 75ms
Set WindowBar Colorful
Set BorderRadius 8
Set Framerate 24
Set LoopOffset 10%

# --- Launch lazydolt ---
# NOTE: Run scripts/setup-test-db.sh before running this tape
Type "lazydolt tmp/test-dolt"
Sleep 300ms
Enter
Sleep 2s

# Pause to let user see the initial view:
# Tables panel with unstaged/staged changes, diff on the right
Sleep 2s

# --- Browse tables and view diffs ---
# Navigate down to see products diff
Type "j"
Sleep 1.5s

# --- Stage tables ---
# Stage products with Space
Space
Sleep 1.5s

# Navigate back up to orders
Type "k"
Sleep 1s

# Stage orders too
Space
Sleep 1.5s

# --- Switch right panel to Browse tab ---
# ] cycles tabs on the right panel (Tables panel must be focused)
Type "]"
Sleep 2s

# Switch to Schema tab
Type "]"
Sleep 2s

# Back to Diff tab
Type "]"
Sleep 1.5s

# --- Move to Branches panel ---
Tab
Sleep 1.5s

# See both branches — navigate down
Type "j"
Sleep 1s
Type "k"
Sleep 1s

# --- Move to Commits panel ---
Tab
Sleep 1.5s

# Browse through commits
Type "j"
Sleep 700ms
Type "j"
Sleep 700ms
Type "j"
Sleep 1.2s

# --- Filter commits ---
# Use / to filter
Type "/"
Sleep 400ms
Type@100ms "product"
Sleep 1.2s
Enter
Sleep 2s

# Clear filter with Esc
Escape
Sleep 1.5s

# --- Zoom the commits panel ---
Type "+"
Sleep 2s

# Back to normal
Type "+"
Sleep 1s
Type "+"
Sleep 1.5s

# --- Switch to Tables panel for commit ---
# Press 1 to jump to Tables panel
Type "1"
Sleep 1s

# --- Open commit dialog ---
Type "c"
Sleep 800ms
Type@80ms "Update pricing and add new order"
Sleep 1.5s

# Cancel commit (Esc)
Escape
Sleep 1s

# --- Open help overlay (multi-column layout) ---
Type "?"
Sleep 2.5s

# Scroll down to show scrolling works
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1.5s

# Close help
Escape
Sleep 1s

# --- Quit ---
Type "q"
Sleep 1s
