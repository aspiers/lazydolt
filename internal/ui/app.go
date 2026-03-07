package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/ui/components"
)

var (
	focusedBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("2"))
	blurredBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	titleStyle     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	commitBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("5")).Padding(1, 2)

	// Main (right) panel has no left border — the left column's right
	// border serves as the shared divider, eliminating the double ││.
	mainBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, false)
)

// MainView tracks which content is shown in the right panel.
type MainView int

const (
	MainViewDiff MainView = iota
	MainViewSchema
	MainViewBrowser
)

// ScreenMode controls the column split ratio (like lazygit's +/_ cycling).
type ScreenMode int

const (
	ScreenNormal     ScreenMode = iota // default split (~30% left)
	ScreenHalf                         // roughly 50/50
	ScreenFullscreen                   // focused column takes 100%
	screenModeCount                    // sentinel for wrapping
)

// App is the root Bubble Tea model.
type App struct {
	runner   *dolt.Runner
	repoName string
	width    int
	height   int

	// Focus
	focused components.Panel

	// Side panels
	statusBar components.StatusBar
	tables    components.TablesModel
	branches  components.BranchesModel
	commits   components.CommitsModel

	// Main content
	mainView    MainView
	diffView    components.DiffView
	schemaView  components.SchemaView
	browserView components.BrowserView

	// Commit dialog
	commitInput textinput.Model
	showCommit  bool
	commitErr   string

	// Error flash
	errMsg string

	// Help
	showHelp bool

	// Layout
	screenMode ScreenMode
}

// NewApp creates a new App with the given dolt runner.
func NewApp(runner *dolt.Runner) App {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.CharLimit = 200

	app := App{
		runner:      runner,
		repoName:    filepath.Base(runner.RepoDir),
		focused:     components.PanelTables,
		diffView:    components.NewDiffView(80, 20),
		schemaView:  components.NewSchemaView(80, 20),
		browserView: components.NewBrowserView(80, 20),
		commitInput: ti,
	}
	app.syncFocus()
	return app
}

// Init loads initial data from dolt.
func (a App) Init() tea.Cmd {
	return a.loadData()
}

// Update handles messages.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Viewport sizes will be recalculated in View()
		return a, nil

	case tea.KeyMsg:
		// Commit dialog intercepts all keys when active
		if a.showCommit {
			return a.updateCommitDialog(msg)
		}
		if a.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "esc":
			// Esc goes back: zoom → normal, or reset main view to auto-diff
			if a.screenMode != ScreenNormal {
				a.screenMode = ScreenNormal
				return a, nil
			}
			// If viewing non-default content, reset to auto-diff
			if a.mainView != MainViewDiff {
				a.mainView = MainViewDiff
				return a, a.autoViewDiff()
			}
			return a, nil
		case "tab":
			a.cycleFocus()
			return a, a.autoViewDiff()
		case "1":
			a.setFocus(components.PanelTables)
			return a, a.autoViewDiff()
		case "2":
			a.setFocus(components.PanelBranches)
			return a, nil
		case "3":
			a.setFocus(components.PanelCommits)
			return a, nil
		case "c":
			return a, a.startCommit()
		case "+":
			a.screenMode = (a.screenMode + 1) % screenModeCount
			return a, nil
		case "_":
			a.screenMode = (a.screenMode + screenModeCount - 1) % screenModeCount
			return a, nil
		case "?":
			a.showHelp = true
			return a, nil
		}

		// Route to focused panel
		switch a.focused {
		case components.PanelTables:
			var cmd tea.Cmd
			a.tables, cmd = a.tables.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelBranches:
			var cmd tea.Cmd
			a.branches, cmd = a.branches.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelCommits:
			var cmd tea.Cmd
			a.commits, cmd = a.commits.Update(msg)
			cmds = append(cmds, cmd)
		}

	case DataLoadedMsg:
		a.statusBar.Branch = msg.Branch
		a.statusBar.Dirty = msg.Dirty
		a.statusBar.RepoDir = a.repoName
		a.tables.Tables = msg.Tables
		a.branches.Branches = msg.Branches
		a.commits.Commits = msg.Commits
		a.errMsg = ""
		// Clamp cursors
		if a.tables.Cursor >= len(a.tables.Tables) {
			a.tables.Cursor = max(0, len(a.tables.Tables)-1)
		}
		if a.branches.Cursor >= len(a.branches.Branches) {
			a.branches.Cursor = max(0, len(a.branches.Branches)-1)
		}
		// Auto-load diff for selected table
		cmds = append(cmds, a.autoViewDiff())

	case DiffContentMsg:
		a.mainView = MainViewDiff
		a.diffView.SetContent(msg.Table, msg.Content)

	case SchemaContentMsg:
		a.mainView = MainViewSchema
		a.schemaView.SetContent(msg.Table, msg.Schema)

	case RefreshMsg:
		cmds = append(cmds, a.loadData())

	case ErrorMsg:
		a.errMsg = msg.Err.Error()

	case CommitSuccessMsg:
		a.showCommit = false
		a.commitErr = ""
		cmds = append(cmds, a.loadData())

	// Component messages that bubble up
	case stageTableMsg:
		cmds = append(cmds, a.stageCmd(msg.Table))
	case unstageTableMsg:
		cmds = append(cmds, a.unstageCmd(msg.Table))
	case stageAllMsg:
		cmds = append(cmds, a.stageAllCmd())
	case viewDiffMsg:
		cmds = append(cmds, a.loadDiff(msg.Table))
	case viewSchemaMsg:
		cmds = append(cmds, a.loadSchema(msg.Table))
	case viewTableDataMsg:
		cmds = append(cmds, a.loadTableData(msg.Table))
	case checkoutBranchMsg:
		cmds = append(cmds, a.checkoutCmd(msg.Branch))
	case deleteBranchMsg:
		cmds = append(cmds, a.deleteBranchCmd(msg.Branch))
	case viewCommitMsg:
		cmds = append(cmds, a.loadCommitDiff(msg.Hash))
	case BrowserDataMsg:
		a.mainView = MainViewBrowser
		a.browserView.SetData(msg.Table, msg.Columns, msg.Rows, msg.Total, msg.Offset)
	case browserPageMsg:
		cmds = append(cmds, a.loadTableDataPage(msg.Table, msg.Offset))
	}

	// Update main panel viewport, but don't forward key events that
	// were already handled by a focused left panel — otherwise j/k
	// scrolls both the panel list and the viewport simultaneously.
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		switch a.mainView {
		case MainViewDiff:
			var cmd tea.Cmd
			a.diffView, cmd = a.diffView.Update(msg)
			cmds = append(cmds, cmd)
		case MainViewSchema:
			var cmd tea.Cmd
			a.schemaView, cmd = a.schemaView.Update(msg)
			cmds = append(cmds, cmd)
		case MainViewBrowser:
			var cmd tea.Cmd
			a.browserView, cmd = a.browserView.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

// View renders the entire UI.
func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Help overlay
	if a.showHelp {
		return a.renderHelp()
	}

	const statusInnerH = 2 // branch line + dirty marker (title is in border)
	const borderH = 2      // top + bottom border per panel
	const hintsH = 1       // key hints bar

	var body string

	if a.screenMode == ScreenHalf {
		// Half mode: vertical split — focused left panel on top,
		// main panel on bottom, both full terminal width.
		topH := (a.height - hintsH) / 2 // roughly half the screen
		topInnerH := topH - borderH
		if topInnerH < 2 {
			topInnerH = 2
		}
		botOuterH := a.height - hintsH - topH
		botInnerH := botOuterH - borderH
		if botInnerH < 2 {
			botInnerH = 2
		}

		topBox := a.renderFocusedPanel(a.width, topInnerH)

		mainInnerW := a.width - 2
		a.diffView.SetSize(mainInnerW, botInnerH-1)
		a.schemaView.SetSize(mainInnerW, botInnerH-1)
		a.browserView.SetSize(mainInnerW, botInnerH-1)

		mainTitle := a.mainPanelTitle()
		mainContent := a.mainPanelContent()
		mainRendered := blurredBorder.Width(mainInnerW).Height(botInnerH).Render(mainContent)
		mainLines := strings.Split(mainRendered, "\n")
		if len(mainLines) > 0 {
			mainLines[0] = buildTitleBorder(mainTitle, mainInnerW+2, false)
		}
		mainBox := strings.Join(mainLines, "\n")

		body = topBox + "\n" + mainBox
	} else {
		leftW := a.leftColumnWidth()

		// Height budget: terminal height minus 1 row for the key hints bar.
		// Each bordered panel adds 2 rows (top + bottom border) beyond its
		// inner content height.  The left column has 4 panels (status + 3 lists).
		//
		// Layout:  statusBox(inner=statusH, outer=statusH+2)
		//        + 3 * listBox(inner=panelH, outer=panelH+2)
		//        + hintsBar(1 row)
		//        = a.height
		//
		// So: (statusH+2) + 3*(panelH+2) + 1 = a.height
		//     statusH + 3*panelH + 9 = a.height
		availForPanels := a.height - hintsH - (statusInnerH + borderH) - 3*borderH
		panelH := max(2, availForPanels/3)

		// Total outer height of the left column
		leftOuterH := (statusInnerH + borderH) + 3*(panelH+borderH)

		// Main panel inner height = left column outer height minus its own border
		mainInnerH := leftOuterH - borderH

		// Status bar
		a.statusBar.Width = leftW - 2 // account for border
		statusBox := a.panelBox(-1, leftW, statusInnerH, "Status", a.statusBar.View())

		// Tables panel
		a.tables.Height = panelH
		tablesTitle := fmt.Sprintf("[1]─Tables (%d)", len(a.tables.Tables))
		tablesBox := a.panelBox(components.PanelTables, leftW, panelH, tablesTitle, a.tables.View())

		// Branches panel
		a.branches.Height = panelH
		branchesTitle := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		branchesBox := a.panelBox(components.PanelBranches, leftW, panelH, branchesTitle, a.branches.View())

		// Commits panel
		a.commits.Height = panelH
		commitsTitle := fmt.Sprintf("[3]─Commits (%d)", len(a.commits.Commits))
		commitsBox := a.panelBox(components.PanelCommits, leftW, panelH, commitsTitle, a.commits.View())

		// Left column
		left := lipgloss.JoinVertical(lipgloss.Left, statusBox, tablesBox, branchesBox, commitsBox)

		if a.screenMode == ScreenFullscreen {
			// Fullscreen: only left column visible
			body = left
		} else {
			// Normal: side-by-side columns.
			// The main panel has no left border (shared with left column's
			// right border), so only the right border adds 1 char.
			mainInnerW := a.width - leftW - 1
			a.diffView.SetSize(mainInnerW, mainInnerH-1)
			a.schemaView.SetSize(mainInnerW, mainInnerH-1)
			a.browserView.SetSize(mainInnerW, mainInnerH-1)

			mainTitle := a.mainPanelTitle()
			mainContent := a.mainPanelContent()
			mainRendered := mainBorder.Width(mainInnerW).Height(mainInnerH).Render(mainContent)
			// Embed title in the top border (no left corner)
			mainLines := strings.Split(mainRendered, "\n")
			if len(mainLines) > 0 {
				mainLines[0] = buildMainTitleBorder(mainTitle, mainInnerW+1)
			}
			mainBox := strings.Join(mainLines, "\n")
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, mainBox)
		}
	}

	// Key hints
	hints := components.RenderKeyHints(a.focused, a.width)

	// Error bar
	if a.errMsg != "" {
		hints = errorStyle.Render("Error: "+a.errMsg) + "  " + hints
	}

	result := body + "\n" + hints

	// Commit dialog overlay
	if a.showCommit {
		result = a.overlayCommitDialog(result)
	}

	return result
}

// renderFocusedPanel renders only the currently focused left panel at the
// given width and inner height. Used in ScreenHalf for the vertical split.
func (a App) renderFocusedPanel(width, innerH int) string {
	switch a.focused {
	case components.PanelTables:
		a.tables.Height = innerH
		title := fmt.Sprintf("[1]─Tables (%d)", len(a.tables.Tables))
		return a.panelBox(components.PanelTables, width, innerH, title, a.tables.View())
	case components.PanelBranches:
		a.branches.Height = innerH
		title := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		return a.panelBox(components.PanelBranches, width, innerH, title, a.branches.View())
	case components.PanelCommits:
		a.commits.Height = innerH
		title := fmt.Sprintf("[3]─Commits (%d)", len(a.commits.Commits))
		return a.panelBox(components.PanelCommits, width, innerH, title, a.commits.View())
	default:
		// Status panel or unknown — show status
		a.statusBar.Width = width - 2
		return a.panelBox(-1, width, innerH, "Status", a.statusBar.View())
	}
}

// --- Layout helpers ---

func (a App) leftColumnWidth() int {
	switch a.screenMode {
	case ScreenHalf:
		return a.width / 2
	case ScreenFullscreen:
		return a.width
	default: // ScreenNormal
		w := a.width * 30 / 100
		if w < 24 {
			w = 24
		}
		maxW := a.width - 30
		if w > maxW {
			w = maxW
		}
		return w
	}
}

func (a App) panelBox(panel components.Panel, width, height int, title, content string) string {
	style := blurredBorder
	if panel == a.focused {
		style = focusedBorder
	}
	innerW := width - 2 // account for border (1 char each side)
	// Clip each line to the panel width to prevent wrapping.
	// Users can H/L scroll to see truncated content.
	content = clipLines(content, innerW)
	// Lipgloss Height() sets minimum height but doesn't clip overflow.
	// Truncate content to the panel height.
	content = truncateToVisualHeight(content, height, innerW)
	rendered := style.Width(innerW).Height(height).Render(content)

	if title == "" {
		return rendered
	}

	// Replace the top border line with one that embeds the title.
	// Rendered first line looks like: "╭──────...──────╮"
	// We want:                        "╭─ Title ──...──╮"
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		isFocused := panel == a.focused
		lines[0] = buildTitleBorder(title, innerW+2, isFocused)
	}
	return strings.Join(lines, "\n")
}

// buildTitleBorder creates a top border line with an embedded title.
// Format: ╭─[1]─Title───────────╮  (lazygit style: no spaces, bold+green when focused)
func buildTitleBorder(title string, totalWidth int, focused bool) string {
	var borderStyle, titleStyle lipgloss.Style
	if focused {
		borderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	} else {
		borderStyle = lipgloss.NewStyle()
		titleStyle = lipgloss.NewStyle().Bold(true)
	}
	titleRendered := titleStyle.Render(title)

	// Fixed parts: "╭─" (2 chars) + title + fill + "╮" (1 char)
	titleVisualW := lipgloss.Width(titleRendered)
	fillCount := totalWidth - 2 - titleVisualW - 1
	if fillCount < 1 {
		fillCount = 1
	}

	return borderStyle.Render("╭─") +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", fillCount)+"╮")
}

// buildMainTitleBorder creates a top border for the main (right) panel
// which has no left border. Format: ─Title───────────╮
func buildMainTitleBorder(title string, totalWidth int) string {
	borderStyle := lipgloss.NewStyle()
	titleStyle := lipgloss.NewStyle().Bold(true)
	titleRendered := titleStyle.Render(title)

	// Fixed parts: "─" (1 char) + title + fill + "╮" (1 char)
	titleVisualW := lipgloss.Width(titleRendered)
	fillCount := totalWidth - 1 - titleVisualW - 1
	if fillCount < 1 {
		fillCount = 1
	}

	return borderStyle.Render("─") +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", fillCount)+"╮")
}

// clipLines truncates each line of content to maxWidth visible columns,
// preserving ANSI escape sequences. This prevents lipgloss from wrapping
// long lines within bordered panels.
func clipLines(content string, maxWidth int) string {
	if maxWidth <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > maxWidth {
			lines[i] = ansi.Truncate(line, maxWidth, "")
		}
	}
	return strings.Join(lines, "\n")
}

// truncateToVisualHeight clips content to at most maxLines visual lines,
// accounting for text wrapping within the given width.
func truncateToVisualHeight(content string, maxLines, width int) string {
	if maxLines <= 0 {
		return ""
	}
	if width <= 0 {
		width = 1
	}

	lines := strings.Split(content, "\n")
	visualCount := 0
	var kept []string

	for _, line := range lines {
		lineW := lipgloss.Width(line)
		// How many visual lines does this logical line occupy?
		wrapped := 1
		if lineW > width {
			wrapped = (lineW + width - 1) / width
		}

		if visualCount+wrapped > maxLines {
			// This line would push us over; stop here.
			break
		}
		kept = append(kept, line)
		visualCount += wrapped
	}

	return strings.Join(kept, "\n")
}

func (a App) mainPanelTitle() string {
	switch a.mainView {
	case MainViewDiff:
		if a.diffView.Table != "" {
			return "Diff: " + a.diffView.Table
		}
		return "Diff"
	case MainViewSchema:
		if a.schemaView.Table != "" {
			return "Schema: " + a.schemaView.Table
		}
		return "Schema"
	case MainViewBrowser:
		if a.browserView.Table != "" {
			return "Browse: " + a.browserView.Table
		}
		return "Table Browser"
	}
	return ""
}

func (a App) mainPanelContent() string {
	switch a.mainView {
	case MainViewDiff:
		return a.diffView.View()
	case MainViewSchema:
		return a.schemaView.View()
	case MainViewBrowser:
		return a.browserView.View()
	}
	return ""
}

// --- Focus management ---

func (a *App) cycleFocus() {
	switch a.focused {
	case components.PanelTables:
		a.focused = components.PanelBranches
	case components.PanelBranches:
		a.focused = components.PanelCommits
	case components.PanelCommits:
		a.focused = components.PanelTables
	}
	a.syncFocus()
}

func (a *App) setFocus(p components.Panel) {
	a.focused = p
	a.syncFocus()
}

// syncFocus updates the Focused field on all panel models to match a.focused.
func (a *App) syncFocus() {
	a.tables.Focused = a.focused == components.PanelTables
	a.branches.Focused = a.focused == components.PanelBranches
	a.commits.Focused = a.focused == components.PanelCommits
}

// --- Data loading commands ---

func (a *App) loadData() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		branch, _ := runner.CurrentBranch()
		dirty, _ := runner.IsDirty()
		tables, _ := runner.Tables()
		branches, _ := runner.Branches()
		commits, _ := runner.Log(50)

		return DataLoadedMsg{
			Branch:   branch,
			Dirty:    dirty,
			Tables:   tables,
			Branches: branches,
			Commits:  commits,
		}
	}
}

func (a *App) autoViewDiff() tea.Cmd {
	if a.focused != components.PanelTables {
		return nil
	}
	table := a.tables.SelectedTable()
	if table == "" {
		return nil
	}
	return a.loadDiff(table)
}

func (a *App) loadDiff(table string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		content, err := runner.DiffText(table)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return DiffContentMsg{Table: table, Content: content}
	}
}

func (a *App) loadSchema(table string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		schema, err := runner.Schema(table)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return SchemaContentMsg{Table: table, Schema: schema.CreateStatement}
	}
}

func (a *App) loadTableData(table string) tea.Cmd {
	return a.loadTableDataPage(table, 0)
}

func (a *App) loadTableDataPage(table string, offset int) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		result, err := runner.QueryPage(table, 100, offset)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return BrowserDataMsg{
			Table:   table,
			Columns: result.Columns,
			Rows:    result.Rows,
			Total:   result.Total,
			Offset:  offset,
		}
	}
}

func (a *App) loadCommitDiff(hash string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		// Show diff for this commit vs its parent
		content, err := runner.Exec("diff", hash+"^", hash)
		if err != nil {
			// For initial commit, diff against empty
			content, err = runner.Exec("diff", hash)
			if err != nil {
				return ErrorMsg{Err: err}
			}
		}
		return DiffContentMsg{Table: hash[:7], Content: content}
	}
}

// --- Mutation commands ---

func (a *App) stageCmd(table string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.Add(table); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (a *App) unstageCmd(table string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.Reset(table); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (a *App) stageAllCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.AddAll(); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (a *App) checkoutCmd(branch string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.Checkout(branch); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

func (a *App) deleteBranchCmd(branch string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.DeleteBranch(branch); err != nil {
			return ErrorMsg{Err: err}
		}
		return RefreshMsg{}
	}
}

// --- Commit dialog ---

func (a *App) startCommit() tea.Cmd {
	dirty, _ := a.runner.IsDirty()
	if !dirty {
		a.errMsg = "Nothing to commit"
		return nil
	}

	// Check if anything is staged
	status, _ := a.runner.Status()
	hasStaged := false
	for _, s := range status {
		if s.Staged {
			hasStaged = true
			break
		}
	}
	if !hasStaged {
		a.errMsg = "Nothing staged to commit (use Space to stage tables)"
		return nil
	}

	a.showCommit = true
	a.commitInput.Reset()
	a.commitInput.Focus()
	a.commitErr = ""
	return textinput.Blink
}

func (a App) updateCommitDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showCommit = false
		return a, nil
	case "enter":
		message := strings.TrimSpace(a.commitInput.Value())
		if message == "" {
			a.commitErr = "Commit message required"
			return a, nil
		}
		a.showCommit = false
		runner := a.runner
		return a, func() tea.Msg {
			hash, err := runner.Commit(message)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return CommitSuccessMsg{Hash: hash}
		}
	}

	var cmd tea.Cmd
	a.commitInput, cmd = a.commitInput.Update(msg)
	return a, cmd
}

func (a App) overlayCommitDialog(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	content := titleStyle.Render("Commit Message") + "\n\n"
	content += a.commitInput.View() + "\n\n"
	if a.commitErr != "" {
		content += errorStyle.Render(a.commitErr) + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] commit  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	// Center the dialog
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Help ---

func (a App) renderHelp() string {
	help := `lazydolt - Keyboard Shortcuts

  Global
    q / Ctrl+C    Quit
    Tab           Next panel
    1-3           Jump to panel
    c             Commit
    < / >         Resize columns
    =             Reset column sizes
    ?             Toggle help

  Tables Panel
    j/k           Navigate
    Space         Stage/unstage table
    a             Stage all
    d             View diff
    s             View schema
    Enter         Browse table data

  Branches Panel
    j/k           Navigate
    Enter         Checkout branch
    n             New branch
    D             Delete branch

  Commits Panel
    j/k           Navigate
    Enter         View commit details

  Diff/Schema Viewer
    j/k           Scroll
    PgUp/PgDn     Page up/down

Press ? or Esc to close`

	box := commitBoxStyle.Width(50).Render(help)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

// --- Re-export component message types for the switch in Update ---

type stageTableMsg = components.StageTableMsg
type unstageTableMsg = components.UnstageTableMsg
type stageAllMsg = components.StageAllMsg
type viewDiffMsg = components.ViewDiffMsg
type viewSchemaMsg = components.ViewSchemaMsg
type viewTableDataMsg = components.ViewTableDataMsg
type viewCommitMsg = components.ViewCommitMsg
type checkoutBranchMsg = components.CheckoutBranchMsg
type deleteBranchMsg = components.DeleteBranchMsg
type browserPageMsg = components.BrowserPageMsg
