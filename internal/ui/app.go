package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/ui/components"
)

var (
	focusedBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("6"))
	blurredBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8"))
	titleStyle     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	commitBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("5")).Padding(1, 2)
)

// MainView tracks which content is shown in the right panel.
type MainView int

const (
	MainViewDiff MainView = iota
	MainViewSchema
	MainViewBrowser
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
	mainView   MainView
	diffView   components.DiffView
	schemaView components.SchemaView

	// Commit dialog
	commitInput textinput.Model
	showCommit  bool
	commitErr   string

	// Error flash
	errMsg string

	// Help
	showHelp bool
}

// NewApp creates a new App with the given dolt runner.
func NewApp(runner *dolt.Runner) App {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.CharLimit = 200

	return App{
		runner:      runner,
		repoName:    filepath.Base(runner.RepoDir),
		focused:     components.PanelTables,
		diffView:    components.NewDiffView(80, 20),
		schemaView:  components.NewSchemaView(80, 20),
		commitInput: ti,
	}
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
		// TODO: implement browser view
	}

	// Update main panel viewport
	switch a.mainView {
	case MainViewDiff:
		var cmd tea.Cmd
		a.diffView, cmd = a.diffView.Update(msg)
		cmds = append(cmds, cmd)
	case MainViewSchema:
		var cmd tea.Cmd
		a.schemaView, cmd = a.schemaView.Update(msg)
		cmds = append(cmds, cmd)
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
	const statusInnerH = 3 // "Status" title + branch line + dirty marker
	const borderH = 2      // top + bottom border per panel
	const hintsH = 1       // key hints bar

	availForPanels := a.height - hintsH - (statusInnerH + borderH) - 3*borderH
	panelH := max(2, availForPanels/3)

	// Total outer height of the left column
	leftOuterH := (statusInnerH + borderH) + 3*(panelH+borderH)

	// Main panel inner height = left column outer height minus its own border
	mainInnerH := leftOuterH - borderH
	mainInnerW := a.width - leftW - 4

	// Update viewport sizes
	a.diffView.SetSize(mainInnerW, mainInnerH-1) // -1 for title line
	a.schemaView.SetSize(mainInnerW, mainInnerH-1)

	// Status bar
	a.statusBar.Width = leftW - 4 // account for border
	statusContent := titleStyle.Render("Status") + "\n" + a.statusBar.View()
	statusBox := a.panelBox(-1, leftW, statusInnerH, statusContent) // -1 = never focused

	// Tables panel
	a.tables.Focused = a.focused == components.PanelTables
	a.tables.Height = panelH
	tablesTitle := fmt.Sprintf("Tables (%d)", len(a.tables.Tables))
	tablesBox := a.panelBox(components.PanelTables, leftW, panelH, titleStyle.Render(tablesTitle)+"\n"+a.tables.View())

	// Branches panel
	a.branches.Focused = a.focused == components.PanelBranches
	a.branches.Height = panelH
	branchesTitle := fmt.Sprintf("Branches (%d)", len(a.branches.Branches))
	branchesBox := a.panelBox(components.PanelBranches, leftW, panelH, titleStyle.Render(branchesTitle)+"\n"+a.branches.View())

	// Commits panel
	a.commits.Focused = a.focused == components.PanelCommits
	a.commits.Height = panelH
	commitsTitle := fmt.Sprintf("Commits (%d)", len(a.commits.Commits))
	commitsBox := a.panelBox(components.PanelCommits, leftW, panelH, titleStyle.Render(commitsTitle)+"\n"+a.commits.View())

	// Left column
	left := lipgloss.JoinVertical(lipgloss.Left, statusBox, tablesBox, branchesBox, commitsBox)

	// Main panel — same outer height as left column
	mainTitle := a.mainPanelTitle()
	mainContent := a.mainPanelContent()
	mainBox := blurredBorder.Width(mainInnerW).Height(mainInnerH).Render(
		titleStyle.Render(mainTitle) + "\n" + mainContent,
	)

	// Join left and main
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, mainBox)

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

// --- Layout helpers ---

func (a App) leftColumnWidth() int {
	w := a.width * 30 / 100
	if w < 24 {
		w = 24
	}
	if w > 40 {
		w = 40
	}
	return w
}

func (a App) panelBox(panel components.Panel, width, height int, content string) string {
	style := blurredBorder
	if panel == a.focused {
		style = focusedBorder
	}
	return style.Width(width - 4).Height(height).Render(content)
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
}

func (a *App) setFocus(p components.Panel) {
	a.focused = p
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
	runner := a.runner
	return func() tea.Msg {
		result, err := runner.QueryPage(table, 100, 0)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return BrowserDataMsg{
			Table:   table,
			Columns: result.Columns,
			Rows:    result.Rows,
			Total:   result.Total,
			Offset:  0,
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
