package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/domain"
	"github.com/aspiers/lazydolt/internal/ui/components"
)

var (
	focusedBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("2"))
	blurredBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	titleStyle     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	commitBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("5")).Padding(1, 2)

	// Tab styles for the Tables panel sub-tab bar
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")) // bold green
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // dim
	tabSepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // dim separator

)

// MainView tracks which content is shown in the right panel.
type MainView int

const (
	MainViewDiff    MainView = iota // "Status" tab — shows diff
	MainViewBrowser                 // "Browse" tab — shows table data
	MainViewSchema                  // "Schema" tab — shows DDL
	MainViewLog                     // "Log" tab — shows command log
	mainViewCount                   // sentinel for wrapping
)

// mainViewTabNames returns the display names for the right panel tabs.
var mainViewTabNames = [mainViewCount]string{"Status", "Browse", "Schema", "Log"}

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
	runner     *dolt.Runner
	repoName   string
	repoParent string
	width      int
	height     int

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
	logView     components.LogView

	// Commit dialog
	commitInput textinput.Model
	showCommit  bool
	commitErr   string
	amendMode   bool

	// Reset menu
	showResetMenu        bool
	resetCommitHash      string
	showHardResetConfirm bool

	// New branch dialog
	branchInput textinput.Model
	showBranch  bool
	branchErr   string

	// Merge menu
	showMergeMenu bool
	mergeBranch   string

	// Delete branch confirmation
	showDeleteBranchConfirm bool
	deleteBranchName        string

	// Rename branch dialog
	showRenameBranch bool
	renameBranchOld  string
	renameBranchErr  string

	// Discard confirmation
	showDiscardConfirm bool
	discardTable       string

	// Stash list
	showStashList bool
	stashEntries  []domain.StashEntry
	stashCursor   int

	// SQL query dialog
	showSQL       bool
	sqlInput      textinput.Model
	sqlErr        string
	sqlHistory    []string // past queries for up/down cycling
	sqlHistoryIdx int      // -1 means current input, 0..N-1 = history

	// Panel filter
	filterInput  textinput.Model
	filterActive bool // text input is focused

	// Error flash
	errMsg string

	// Help
	showHelp   bool
	helpFilter textinput.Model

	// Layout
	screenMode ScreenMode
	leftRatio  int // left column width percentage (default 30)

	// Loading state
	spinner    spinner.Model
	dataLoaded bool
}

// NewApp creates a new App with the given dolt runner.
func NewApp(runner *dolt.Runner) App {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.CharLimit = 200

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan

	bi := textinput.New()
	bi.Placeholder = "Enter branch name..."
	bi.CharLimit = 200

	si := textinput.New()
	si.Placeholder = "SELECT * FROM ..."
	si.CharLimit = 500
	si.Prompt = "SQL> "

	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 100
	fi.Prompt = "/ "

	hf := textinput.New()
	hf.Placeholder = "Type to filter..."
	hf.CharLimit = 50
	hf.Prompt = "/ "

	app := App{
		runner:        runner,
		repoName:      filepath.Base(runner.RepoDir),
		repoParent:    filepath.Dir(runner.RepoDir),
		focused:       components.PanelTables,
		diffView:      components.NewDiffView(80, 20),
		schemaView:    components.NewSchemaView(80, 20),
		browserView:   components.NewBrowserView(80, 20),
		logView:       components.NewLogView(80, 20),
		commitInput:   ti,
		branchInput:   bi,
		sqlInput:      si,
		sqlHistoryIdx: -1,
		filterInput:   fi,
		helpFilter:    hf,
		spinner:       s,
		leftRatio:     30,
	}
	app.syncFocus()
	return app
}

// Init loads initial data from dolt.
func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadData(), a.spinner.Tick)
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
		// Delete branch confirmation intercepts all keys when active
		if a.showDeleteBranchConfirm {
			return a.updateDeleteBranchConfirm(msg)
		}
		// Discard confirmation intercepts all keys when active
		if a.showDiscardConfirm {
			return a.updateDiscardConfirm(msg)
		}
		// Hard reset confirmation intercepts all keys when active
		if a.showHardResetConfirm {
			return a.updateHardResetConfirm(msg)
		}
		// Reset menu intercepts all keys when active
		if a.showResetMenu {
			return a.updateResetMenu(msg)
		}
		// Merge menu intercepts all keys when active
		if a.showMergeMenu {
			return a.updateMergeMenu(msg)
		}
		// SQL dialog intercepts all keys when active
		if a.showSQL {
			return a.updateSQLDialog(msg)
		}
		// Stash list intercepts all keys when active
		if a.showStashList {
			return a.updateStashList(msg)
		}
		// Commit dialog intercepts all keys when active
		if a.showCommit {
			return a.updateCommitDialog(msg)
		}
		// New branch dialog intercepts all keys when active
		if a.showBranch {
			return a.updateNewBranchDialog(msg)
		}
		// Rename branch dialog intercepts all keys when active
		if a.showRenameBranch {
			return a.updateRenameBranchDialog(msg)
		}
		// Panel filter intercepts all keys when active
		if a.filterActive {
			switch msg.String() {
			case "esc":
				// Clear filter and exit filter mode
				a.filterActive = false
				a.filterInput.Reset()
				a.filterInput.Blur()
				a.tables.Filter = ""
				a.branches.Filter = ""
				a.commits.Filter = ""
				return a, nil
			case "enter":
				// Confirm filter and return to navigation
				a.filterActive = false
				a.filterInput.Blur()
				return a, nil
			default:
				var cmd tea.Cmd
				a.filterInput, cmd = a.filterInput.Update(msg)
				// Apply filter text to the focused panel
				f := a.filterInput.Value()
				switch a.focused {
				case components.PanelTables:
					a.tables.Filter = f
				case components.PanelBranches:
					a.branches.Filter = f
				case components.PanelCommits:
					a.commits.Filter = f
				}
				return a, cmd
			}
		}
		if a.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				a.showHelp = false
				a.helpFilter.Reset()
				a.helpFilter.Blur()
				return a, nil
			}
			// Forward to filter input
			var cmd tea.Cmd
			a.helpFilter, cmd = a.helpFilter.Update(msg)
			return a, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "esc":
			// Esc clears active filter first
			if a.tables.Filter != "" || a.branches.Filter != "" || a.commits.Filter != "" {
				a.tables.Filter = ""
				a.branches.Filter = ""
				a.commits.Filter = ""
				a.filterInput.Reset()
				return a, nil
			}
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
			if a.focused != components.PanelTables && a.focused != components.PanelMain {
				a.mainView = MainViewDiff
			}
			return a, a.autoPreview()
		case "shift+tab":
			a.cycleFocusReverse()
			if a.focused != components.PanelTables && a.focused != components.PanelMain {
				a.mainView = MainViewDiff
			}
			return a, a.autoPreview()
		case "1":
			if a.focused == components.PanelTables {
				// Already on Tables — cycle to next tab
				a.mainView = (a.mainView + 1) % mainViewCount
				return a, a.autoPreview()
			}
			a.setFocus(components.PanelTables)
			return a, a.autoPreview()
		case "2":
			a.setFocus(components.PanelBranches)
			a.mainView = MainViewDiff
			return a, a.autoPreview()
		case "3":
			a.setFocus(components.PanelCommits)
			a.mainView = MainViewDiff
			return a, a.autoPreview()
		case "]":
			if a.focused == components.PanelTables {
				a.mainView = (a.mainView + 1) % mainViewCount
				return a, a.autoPreview()
			}
		case "[":
			if a.focused == components.PanelTables {
				a.mainView = (a.mainView + mainViewCount - 1) % mainViewCount
				return a, a.autoPreview()
			}
		case "c":
			return a, a.startCommit()
		case "+":
			a.screenMode = (a.screenMode + 1) % screenModeCount
			return a, nil
		case "_":
			a.screenMode = (a.screenMode + screenModeCount - 1) % screenModeCount
			return a, nil
		case "<":
			if a.leftRatio > 10 {
				a.leftRatio -= 5
			}
			return a, nil
		case ">":
			if a.leftRatio < 90 {
				a.leftRatio += 5
			}
			return a, nil
		case "=":
			a.leftRatio = 30
			return a, nil
		case "P":
			return a, a.pushCmd()
		case "p":
			return a, a.pullCmd()
		case "f":
			return a, a.fetchCmd()
		case "/":
			// Only activate filter for left panels (not main)
			if a.focused != components.PanelMain {
				a.filterActive = true
				a.filterInput.Reset()
				a.filterInput.Focus()
				return a, textinput.Blink
			}
		case "R":
			return a, a.loadData()
		case ":":
			return a, a.startSQL()
		case "S":
			if a.statusBar.Dirty {
				return a, a.stashCmd()
			}
			return a, a.loadStashList()
		case "?":
			a.showHelp = true
			a.helpFilter.Reset()
			a.helpFilter.Focus()
			return a, textinput.Blink
		}

		// Route to focused panel, tracking cursor changes for auto-preview.
		prevCursor := a.focusedCursor()
		switch a.focused {
		case components.PanelTables:
			switch msg.String() {
			case "O":
				if a.tables.SelectedIsConflict() {
					table := a.tables.SelectedTable()
					return a, a.resolveConflicts(table, true)
				}
			case "T":
				if a.tables.SelectedIsConflict() {
					table := a.tables.SelectedTable()
					return a, a.resolveConflicts(table, false)
				}
			case "X":
				if a.tables.HasConflicts() {
					return a, a.abortMerge()
				}
			case "d":
				table := a.tables.SelectedTable()
				if table != "" {
					a.showDiscardConfirm = true
					a.discardTable = table
					return a, nil
				}
			}
			var cmd tea.Cmd
			a.tables, cmd = a.tables.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelBranches:
			if msg.String() == "r" {
				if b := a.branches.SelectedBranch(); b != "" {
					return a, a.startRenameBranch(b)
				}
			}
			if msg.String() == "m" {
				if b := a.branches.SelectedBranch(); b != "" {
					a.showMergeMenu = true
					a.mergeBranch = b
					return a, nil
				}
			}
			var cmd tea.Cmd
			a.branches, cmd = a.branches.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelCommits:
			if msg.String() == "A" {
				return a, a.startAmend()
			}
			if msg.String() == "g" {
				if h := a.commits.SelectedHash(); h != "" {
					a.showResetMenu = true
					a.resetCommitHash = h
					return a, nil
				}
			}
			var cmd tea.Cmd
			a.commits, cmd = a.commits.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelMain:
			// Forward key events to the active right-panel viewport.
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
			case MainViewLog:
				var cmd tea.Cmd
				a.logView, cmd = a.logView.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
		if a.focusedCursor() != prevCursor {
			cmds = append(cmds, a.autoPreview())
		}

	case spinner.TickMsg:
		if !a.dataLoaded {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case DataLoadedMsg:
		a.dataLoaded = true
		a.statusBar.Dirty = msg.Dirty
		a.statusBar.RepoDir = a.repoName
		a.statusBar.ParentDir = a.repoParent
		a.tables.Tables = msg.Tables
		a.branches.Branches = msg.Branches
		a.commits.Commits = msg.Commits
		a.errMsg = ""
		// Clamp cursors
		a.tables.ClampCursor()
		if a.branches.Cursor >= len(a.branches.Branches) {
			a.branches.Cursor = max(0, len(a.branches.Branches)-1)
		}
		// Auto-load content for the active tab
		cmds = append(cmds, a.autoPreview())

	case DiffContentMsg:
		a.diffView.SetContent(msg.Table, msg.Content)

	case SchemaContentMsg:
		a.schemaView.SetContent(msg.Table, msg.Schema)

	case RefreshMsg:
		cmds = append(cmds, a.loadData())

	case ErrorMsg:
		a.errMsg = msg.Err.Error()

	case CommitSuccessMsg:
		a.showCommit = false
		a.commitErr = ""
		cmds = append(cmds, a.loadData())

	case ResetSuccessMsg:
		cmds = append(cmds, a.loadData())

	case RemoteOpSuccessMsg:
		cmds = append(cmds, a.loadData())

	case MergeSuccessMsg:
		cmds = append(cmds, a.loadData())

	case MergeConflictMsg:
		a.setFocus(components.PanelTables)
		cmds = append(cmds, a.loadData())

	case ConflictResolveMsg:
		cmds = append(cmds, a.loadData())

	case MergeAbortMsg:
		cmds = append(cmds, a.loadData())

	case SQLResultMsg:
		// Display SQL results in the diff view
		title := fmt.Sprintf("SQL: %s", msg.Query)
		a.diffView.SetContent(title, msg.Result)
		a.mainView = MainViewDiff

	case StashSuccessMsg:
		cmds = append(cmds, a.loadData())

	case StashListMsg:
		a.stashEntries = msg.Entries
		a.stashCursor = 0
		if len(msg.Entries) > 0 {
			a.showStashList = true
		}

	case StashPopMsg:
		a.showStashList = false
		cmds = append(cmds, a.loadData())

	case StashDropMsg:
		// Refresh the stash list after drop
		a.showStashList = false
		cmds = append(cmds, a.loadData())

	case NewBranchSuccessMsg:
		a.showBranch = false
		a.branchErr = ""
		cmds = append(cmds, a.loadData())

	case RenameBranchSuccessMsg:
		a.showRenameBranch = false
		a.renameBranchErr = ""
		cmds = append(cmds, a.loadData())

	// Component messages that bubble up
	case stageTableMsg:
		cmds = append(cmds, a.stageCmd(msg.Table))
	case unstageTableMsg:
		cmds = append(cmds, a.unstageCmd(msg.Table))
	case stageAllMsg:
		cmds = append(cmds, a.stageAllCmd())
	case unstageAllMsg:
		cmds = append(cmds, a.unstageAllCmd())
	case viewDiffMsg:
		cmds = append(cmds, a.loadDiff(msg.Table, a.tables.SelectedIsStaged()))
	case viewSchemaMsg:
		cmds = append(cmds, a.loadSchema(msg.Table))
	case viewTableDataMsg:
		cmds = append(cmds, a.loadTableData(msg.Table))
	case checkoutBranchMsg:
		cmds = append(cmds, a.checkoutCmd(msg.Branch))
	case deleteBranchMsg:
		a.showDeleteBranchConfirm = true
		a.deleteBranchName = msg.Branch
	case newBranchPromptMsg:
		return a, a.startNewBranch()
	case viewCommitMsg:
		cmds = append(cmds, a.loadCommitDiff(msg.Hash))
	case BrowserDataMsg:
		a.browserView.SetData(msg.Table, msg.Columns, msg.Rows, msg.Total, msg.Offset)
	case browserPageMsg:
		cmds = append(cmds, a.loadTableDataPage(msg.Table, msg.Offset))
	}

	// Forward non-key messages (e.g. mouse, resize) to the active viewport
	// regardless of focus. Key events reach the viewport only when PanelMain
	// is focused (handled in the key-routing switch above).
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
		case MainViewLog:
			var cmd tea.Cmd
			a.logView, cmd = a.logView.Update(msg)
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

	// Ensure status bar width is set for Lines() calculation.
	// In ScreenNormal, leftColumnWidth()-2; in other modes, width-2.
	if a.statusBar.Width == 0 {
		a.statusBar.Width = a.leftColumnWidth() - 2
	}
	statusInnerH := a.statusBar.Lines() // 2 or 3 depending on path wrapping
	const borderH = 2                   // top + bottom border per panel
	const hintsH = 1                    // key hints bar

	var body string

	if a.screenMode == ScreenHalf {
		// Half mode: vertical split — focused left panel on top,
		// main panel on bottom, both full terminal width.
		topH := (a.height - hintsH) / 3 // roughly one-third of the screen
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
		a.logView.SetSize(mainInnerW, botInnerH-1)

		mainTitle := a.mainPanelTitle()
		mainContent := a.mainPanelContent()
		mainFocused := a.focused == components.PanelMain
		mainStyle := blurredBorder
		if mainFocused {
			mainStyle = focusedBorder
		}
		mainRendered := mainStyle.Width(mainInnerW).Height(botInnerH).Render(mainContent)
		mainLines := strings.Split(mainRendered, "\n")
		if len(mainLines) > 0 {
			mainLines[0] = buildTitleBorder(mainTitle, mainInnerW+2, mainFocused)
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

		// The focused panel gets more height; the other two share
		// the remainder equally (like lazygit's auto-grow behavior).
		// When PanelMain is focused, all three left panels split evenly.
		var tablesH, branchesH, commitsH int
		if a.focused == components.PanelMain {
			each := availForPanels / 3
			remainder := availForPanels - 3*each
			tablesH = each + remainder // give remainder to first panel
			branchesH = each
			commitsH = each
		} else {
			focusedH, unfocusedH := a.panelHeights(availForPanels)
			heightFor := func(panel components.Panel) int {
				if panel == a.focused {
					return focusedH
				}
				return unfocusedH
			}
			tablesH = heightFor(components.PanelTables)
			branchesH = heightFor(components.PanelBranches)
			commitsH = heightFor(components.PanelCommits)
		}

		// Total outer height of the left column
		leftOuterH := (statusInnerH + borderH) + (tablesH + borderH) + (branchesH + borderH) + (commitsH + borderH)

		// Main panel inner height = left column outer height minus its own border
		mainInnerH := leftOuterH - borderH

		// Status bar
		a.statusBar.Width = leftW - 2 // account for border
		statusBox := a.panelBox(-1, leftW, statusInnerH, "Database", a.statusBar.View())

		// Loading spinner or real content for each panel
		loading := a.loadingText()
		panelView := func(view string) string {
			if !a.dataLoaded {
				return loading
			}
			return view
		}

		// Tables panel — title includes sub-tab indicators
		a.tables.Height = tablesH
		tablesTitle := fmt.Sprintf("[1]─%s", a.tabBar())
		tablesBox := a.panelBox(components.PanelTables, leftW, tablesH, tablesTitle, panelView(a.tables.View()))

		// Branches panel
		a.branches.Height = branchesH
		branchesTitle := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		branchesBox := a.panelBox(components.PanelBranches, leftW, branchesH, branchesTitle, panelView(a.branches.View()))

		// Commits panel
		a.commits.Height = commitsH
		commitsTitle := fmt.Sprintf("[3]─Commits (%d)", len(a.commits.Commits))
		commitsBox := a.panelBox(components.PanelCommits, leftW, commitsH, commitsTitle, panelView(a.commits.View()))

		// Left column
		left := lipgloss.JoinVertical(lipgloss.Left, statusBox, tablesBox, branchesBox, commitsBox)

		if a.screenMode == ScreenFullscreen {
			// Fullscreen: only left column visible
			body = left
		} else {
			// Normal: side-by-side columns.
			// Both borders (left + right) add 2 chars.
			mainInnerW := a.width - leftW - 2
			a.diffView.SetSize(mainInnerW, mainInnerH-1)
			a.schemaView.SetSize(mainInnerW, mainInnerH-1)
			a.browserView.SetSize(mainInnerW, mainInnerH-1)
			a.logView.SetSize(mainInnerW, mainInnerH-1)

			mainTitle := a.mainPanelTitle()
			mainContent := a.mainPanelContent()
			mainFocused := a.focused == components.PanelMain
			mainStyle := blurredBorder
			if mainFocused {
				mainStyle = focusedBorder
			}
			mainRendered := mainStyle.Width(mainInnerW).Height(mainInnerH).Render(mainContent)
			// Embed title in the top border
			mainLines := strings.Split(mainRendered, "\n")
			if len(mainLines) > 0 {
				mainLines[0] = buildTitleBorder(mainTitle, mainInnerW+2, mainFocused)
			}
			mainBox := strings.Join(mainLines, "\n")
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, mainBox)
		}
	}

	// Key hints or filter input
	var hints string
	if a.filterActive {
		hints = a.filterInput.View() + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] confirm  [Esc] clear")
	} else {
		hints = components.RenderKeyHints(a.focused, a.width, a.tables.HasConflicts())
		// Show active filter indicator
		activeFilter := ""
		switch a.focused {
		case components.PanelTables:
			activeFilter = a.tables.Filter
		case components.PanelBranches:
			activeFilter = a.branches.Filter
		case components.PanelCommits:
			activeFilter = a.commits.Filter
		}
		if activeFilter != "" {
			filterIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("/" + activeFilter)
			hints = filterIndicator + "  " + hints
		}
	}

	// Error bar
	if a.errMsg != "" {
		hints = errorStyle.Render("Error: "+a.errMsg) + "  " + hints
	}

	result := body + "\n" + hints

	// Overlays
	if a.showDeleteBranchConfirm {
		result = a.overlayDeleteBranchConfirm(result)
	}
	if a.showDiscardConfirm {
		result = a.overlayDiscardConfirm(result)
	}
	if a.showHardResetConfirm {
		result = a.overlayHardResetConfirm(result)
	}
	if a.showResetMenu {
		result = a.overlayResetMenu(result)
	}
	if a.showMergeMenu {
		result = a.overlayMergeMenu(result)
	}
	if a.showStashList {
		result = a.overlayStashList(result)
	}
	if a.showSQL {
		result = a.overlaySQLDialog(result)
	}
	if a.showCommit {
		result = a.overlayCommitDialog(result)
	}
	if a.showBranch {
		result = a.overlayNewBranchDialog(result)
	}
	if a.showRenameBranch {
		result = a.overlayRenameBranchDialog(result)
	}

	return result
}

// panelHeights computes the inner heights for the focused and unfocused
// list panels. The focused panel gets roughly half the available space,
// and the two unfocused panels share the rest equally. Any remainder
// from integer division is added to the focused panel so no rows are wasted.
func (a App) panelHeights(availForPanels int) (focused, unfocused int) {
	if availForPanels < 6 {
		// Degenerate: split equally
		h := max(2, availForPanels/3)
		return h, h
	}
	focused = availForPanels / 2
	remaining := availForPanels - focused
	unfocused = remaining / 2
	// Add any remainder from the unfocused division to the focused panel
	// so all available rows are used (avoids blank line at the bottom).
	focused += remaining - 2*unfocused
	if focused < 2 {
		focused = 2
	}
	if unfocused < 2 {
		unfocused = 2
	}
	return focused, unfocused
}

// renderFocusedPanel renders only the currently focused left panel at the
// given width and inner height. Used in ScreenHalf for the vertical split.
func (a App) renderFocusedPanel(width, innerH int) string {
	loading := a.loadingText()
	content := func(view string) string {
		if !a.dataLoaded {
			return loading
		}
		return view
	}

	switch a.focused {
	case components.PanelTables:
		a.tables.Height = innerH
		title := fmt.Sprintf("[1]─%s", a.tabBar())
		return a.panelBox(components.PanelTables, width, innerH, title, content(a.tables.View()))
	case components.PanelBranches:
		a.branches.Height = innerH
		title := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		return a.panelBox(components.PanelBranches, width, innerH, title, content(a.branches.View()))
	case components.PanelCommits:
		a.commits.Height = innerH
		title := fmt.Sprintf("[3]─Commits (%d)", len(a.commits.Commits))
		return a.panelBox(components.PanelCommits, width, innerH, title, content(a.commits.View()))
	default:
		// Status panel or unknown — show status
		a.statusBar.Width = width - 2
		return a.panelBox(-1, width, innerH, "Database", a.statusBar.View())
	}
}

// tabBar renders the sub-tab indicator for the Tables panel title.
// Active tab is bold green, inactive tabs are dim, separated by " - ".
func (a App) tabBar() string {
	sep := tabSepStyle.Render(" - ")
	var parts []string
	for i, name := range mainViewTabNames {
		if MainView(i) == a.mainView {
			parts = append(parts, activeTabStyle.Render(name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(name))
		}
	}
	return strings.Join(parts, sep)
}

// --- Layout helpers ---

func (a App) leftColumnWidth() int {
	switch a.screenMode {
	case ScreenHalf:
		return a.width / 2
	case ScreenFullscreen:
		return a.width
	default: // ScreenNormal
		w := a.width * a.leftRatio / 100
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

	// Max visual width for the title: totalWidth - "╭─" (2) - "─╮" (2)
	maxTitleW := totalWidth - 4
	if maxTitleW < 3 {
		maxTitleW = 3
	}

	// Title may already contain ANSI codes (e.g. coloured tab bar),
	// so use ANSI-aware width and truncation.
	titleRendered := titleStyle.Render(title)
	if lipgloss.Width(titleRendered) > maxTitleW {
		titleRendered = ansi.Truncate(titleRendered, maxTitleW-1, "…")
	}

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
	case MainViewLog:
		return fmt.Sprintf("Command Log (%d)", len(a.runner.CommandLog()))
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
	case MainViewLog:
		// Refresh log entries from runner before rendering
		a.logView.Entries = a.runner.CommandLog()
		a.logView.RefreshContent()
		return a.logView.View()
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
		a.focused = components.PanelMain
	case components.PanelMain:
		a.focused = components.PanelTables
	}
	a.syncFocus()
}

func (a *App) cycleFocusReverse() {
	switch a.focused {
	case components.PanelTables:
		a.focused = components.PanelMain
	case components.PanelBranches:
		a.focused = components.PanelTables
	case components.PanelCommits:
		a.focused = components.PanelBranches
	case components.PanelMain:
		a.focused = components.PanelCommits
	}
	a.syncFocus()
}

func (a *App) setFocus(p components.Panel) {
	// Clear filter when switching panels
	a.filterActive = false
	a.filterInput.Reset()
	a.filterInput.Blur()
	a.tables.Filter = ""
	a.branches.Filter = ""
	a.commits.Filter = ""
	a.focused = p
	a.syncFocus()
}

// syncFocus updates the Focused field on all panel models to match a.focused.
func (a *App) syncFocus() {
	a.tables.Focused = a.focused == components.PanelTables
	a.branches.Focused = a.focused == components.PanelBranches
	a.commits.Focused = a.focused == components.PanelCommits
}

// loadingText returns the spinner animation text shown while data is loading.
func (a App) loadingText() string {
	return a.spinner.View() + " Loading…"
}

// --- Data loading commands ---

func (a *App) loadData() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		type branchResult struct {
			branch string
			err    error
		}
		type tablesResult struct {
			tables []domain.Table
			err    error
		}
		type branchesResult struct {
			branches []domain.Branch
			err      error
		}
		type commitsResult struct {
			commits []domain.Commit
			err     error
		}

		branchCh := make(chan branchResult, 1)
		tablesCh := make(chan tablesResult, 1)
		branchesCh := make(chan branchesResult, 1)
		commitsCh := make(chan commitsResult, 1)

		go func() {
			b, err := runner.CurrentBranch()
			branchCh <- branchResult{b, err}
		}()
		go func() {
			t, err := runner.Tables()
			tablesCh <- tablesResult{t, err}
		}()
		go func() {
			br, err := runner.Branches()
			branchesCh <- branchesResult{br, err}
		}()
		go func() {
			c, err := runner.Log(50)
			commitsCh <- commitsResult{c, err}
		}()

		brRes := <-branchCh
		tblRes := <-tablesCh
		brchRes := <-branchesCh
		cmtRes := <-commitsCh

		// Derive dirty from tables — any table with a non-nil Status
		// has uncommitted changes, avoiding a redundant Status() call.
		dirty := false
		for _, t := range tblRes.tables {
			if t.Status != nil {
				dirty = true
				break
			}
		}

		return DataLoadedMsg{
			Branch:   brRes.branch,
			Dirty:    dirty,
			Tables:   tblRes.tables,
			Branches: brchRes.branches,
			Commits:  cmtRes.commits,
		}
	}
}

func (a *App) autoViewDiff() tea.Cmd {
	if a.focused != components.PanelTables {
		return nil
	}
	// On a section header, show all diffs for that section
	if a.tables.IsOnHeader() {
		if a.tables.IsOnCleanHeader() {
			return nil
		}
		return a.loadDiff("", a.tables.SelectedIsStaged())
	}
	table := a.tables.SelectedTable()
	if table == "" {
		return nil
	}
	return a.loadDiff(table, a.tables.SelectedIsStaged())
}

// focusedCursor returns the cursor position of the currently focused panel.
func (a App) focusedCursor() int {
	switch a.focused {
	case components.PanelTables:
		return a.tables.Cursor
	case components.PanelBranches:
		return a.branches.Cursor
	case components.PanelCommits:
		return a.commits.Cursor
	default:
		return -1
	}
}

// autoPreview loads the appropriate content for the currently selected item
// in the focused panel, updating the right panel automatically.
func (a *App) autoPreview() tea.Cmd {
	switch a.focused {
	case components.PanelTables:
		// On a section header, show all diffs for that section
		if a.tables.IsOnHeader() {
			if a.tables.IsOnCleanHeader() {
				return nil
			}
			switch a.mainView {
			case MainViewDiff:
				return a.loadDiff("", a.tables.SelectedIsStaged())
			case MainViewBrowser:
				a.browserView.Clear("Select a table to browse its data")
				return nil
			case MainViewSchema:
				a.schemaView.Clear("Select a table to view its schema")
				return nil
			}
			return nil
		}
		table := a.tables.SelectedTable()
		if table == "" {
			return nil
		}
		// Show conflict details for conflicted tables
		if a.tables.SelectedIsConflict() {
			return a.loadConflicts(table)
		}
		switch a.mainView {
		case MainViewSchema:
			return a.loadSchema(table)
		case MainViewBrowser:
			return a.loadTableData(table)
		default:
			return a.loadDiff(table, a.tables.SelectedIsStaged())
		}
	case components.PanelCommits:
		hash := a.commits.SelectedHash()
		if hash == "" {
			return nil
		}
		return a.loadCommitDiff(hash)
	default:
		return nil
	}
}

func (a *App) loadDiff(table string, staged bool) tea.Cmd {
	runner := a.runner
	label := table
	if label == "" {
		if staged {
			label = "all staged"
		} else {
			label = "all unstaged"
		}
	}
	return func() tea.Msg {
		content, err := runner.DiffText(table, staged)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return DiffContentMsg{Table: label, Content: content}
	}
}

func (a App) resolveConflicts(table string, ours bool) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		var err error
		if ours {
			err = runner.ConflictsResolveOurs(table)
		} else {
			err = runner.ConflictsResolveTheirs(table)
		}
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ConflictResolveMsg{Table: table, Ours: ours}
	}
}

func (a App) abortMerge() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.MergeAbort(); err != nil {
			return ErrorMsg{Err: err}
		}
		return MergeAbortMsg{}
	}
}

func (a App) stashCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.Stash(); err != nil {
			return ErrorMsg{Err: err}
		}
		return StashSuccessMsg{}
	}
}

func (a App) loadStashList() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		entries, err := runner.StashList()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return StashListMsg{Entries: entries}
	}
}

func (a App) stashPopCmd(index int) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.StashPop(index); err != nil {
			return ErrorMsg{Err: err}
		}
		return StashPopMsg{Index: index}
	}
}

func (a App) stashDropCmd(index int) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.StashDrop(index); err != nil {
			return ErrorMsg{Err: err}
		}
		return StashDropMsg{Index: index}
	}
}

func (a *App) loadConflicts(table string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		content, err := runner.ConflictsCat(table)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return DiffContentMsg{Table: table + " (conflicts)", Content: content}
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

func (a *App) unstageAllCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if err := runner.ResetAll(); err != nil {
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

// --- Remote operations ---

func (a *App) pushCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if _, err := runner.Push(); err != nil {
			return ErrorMsg{Err: err}
		}
		return RemoteOpSuccessMsg{Op: "push"}
	}
}

func (a *App) pullCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if _, err := runner.Pull(); err != nil {
			return ErrorMsg{Err: err}
		}
		return RemoteOpSuccessMsg{Op: "pull"}
	}
}

func (a *App) fetchCmd() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		if _, err := runner.Fetch(); err != nil {
			return ErrorMsg{Err: err}
		}
		return RemoteOpSuccessMsg{Op: "fetch"}
	}
}

// --- SQL query dialog ---

func (a *App) startSQL() tea.Cmd {
	a.showSQL = true
	a.sqlInput.Reset()
	a.sqlInput.Focus()
	a.sqlErr = ""
	a.sqlHistoryIdx = -1
	return textinput.Blink
}

func (a App) updateSQLDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showSQL = false
		return a, nil
	case "enter":
		query := strings.TrimSpace(a.sqlInput.Value())
		if query == "" {
			a.sqlErr = "Query cannot be empty"
			return a, nil
		}
		a.showSQL = false
		// Add to history (avoid duplicating the most recent entry)
		if len(a.sqlHistory) == 0 || a.sqlHistory[len(a.sqlHistory)-1] != query {
			a.sqlHistory = append(a.sqlHistory, query)
		}
		runner := a.runner
		return a, func() tea.Msg {
			result, err := runner.SQLRaw(query)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return SQLResultMsg{Query: query, Result: result}
		}
	case "up":
		// Cycle backward through history
		if len(a.sqlHistory) > 0 {
			if a.sqlHistoryIdx == -1 {
				a.sqlHistoryIdx = len(a.sqlHistory) - 1
			} else if a.sqlHistoryIdx > 0 {
				a.sqlHistoryIdx--
			}
			a.sqlInput.SetValue(a.sqlHistory[a.sqlHistoryIdx])
			a.sqlInput.CursorEnd()
		}
		return a, nil
	case "down":
		// Cycle forward through history
		if a.sqlHistoryIdx >= 0 {
			if a.sqlHistoryIdx < len(a.sqlHistory)-1 {
				a.sqlHistoryIdx++
				a.sqlInput.SetValue(a.sqlHistory[a.sqlHistoryIdx])
			} else {
				a.sqlHistoryIdx = -1
				a.sqlInput.Reset()
			}
			a.sqlInput.CursorEnd()
		}
		return a, nil
	}

	var cmd tea.Cmd
	a.sqlInput, cmd = a.sqlInput.Update(msg)
	return a, cmd
}

func (a App) overlaySQLDialog(base string) string {
	dialogW := 70
	if a.width < 80 {
		dialogW = a.width - 10
	}

	content := titleStyle.Render("SQL Query") + "\n\n"
	content += a.sqlInput.View() + "\n\n"
	if a.sqlErr != "" {
		content += errorStyle.Render(a.sqlErr) + "\n"
	}
	histHint := ""
	if len(a.sqlHistory) > 0 {
		histHint = "  [↑/↓] history"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] execute  [Esc] cancel" + histHint)

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
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

func (a *App) startAmend() tea.Cmd {
	if len(a.commits.Commits) == 0 {
		a.errMsg = "No commits to amend"
		return nil
	}
	a.showCommit = true
	a.amendMode = true
	a.commitInput.Reset()
	a.commitInput.SetValue(a.commits.Commits[0].Message)
	a.commitInput.Focus()
	a.commitErr = ""
	return textinput.Blink
}

func (a App) updateCommitDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showCommit = false
		a.amendMode = false
		return a, nil
	case "enter":
		message := strings.TrimSpace(a.commitInput.Value())
		if message == "" {
			a.commitErr = "Commit message required"
			return a, nil
		}
		a.showCommit = false
		amend := a.amendMode
		a.amendMode = false
		runner := a.runner
		if amend {
			return a, func() tea.Msg {
				hash, err := runner.CommitAmend(message)
				if err != nil {
					return ErrorMsg{Err: err}
				}
				return CommitSuccessMsg{Hash: hash}
			}
		}
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

	title := "Commit Message"
	action := "commit"
	if a.amendMode {
		title = "Amend Commit"
		action = "amend"
	}
	content := titleStyle.Render(title) + "\n\n"
	content += a.commitInput.View() + "\n\n"
	if a.commitErr != "" {
		content += errorStyle.Render(a.commitErr) + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] " + action + "  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	// Center the dialog
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- New branch dialog ---

func (a *App) startNewBranch() tea.Cmd {
	a.showBranch = true
	a.branchInput.Reset()
	a.branchInput.Focus()
	a.branchErr = ""
	return textinput.Blink
}

func (a App) updateNewBranchDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showBranch = false
		return a, nil
	case "enter":
		name := strings.TrimSpace(a.branchInput.Value())
		if name == "" {
			a.branchErr = "Branch name cannot be empty"
			return a, nil
		}
		a.showBranch = false
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.NewBranch(name); err != nil {
				return ErrorMsg{Err: err}
			}
			return NewBranchSuccessMsg{Name: name}
		}
	}

	var cmd tea.Cmd
	a.branchInput, cmd = a.branchInput.Update(msg)
	return a, cmd
}

func (a App) overlayNewBranchDialog(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	content := titleStyle.Render("New Branch") + "\n\n"
	content += a.branchInput.View() + "\n\n"
	if a.branchErr != "" {
		content += errorStyle.Render(a.branchErr) + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] create  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Rename branch dialog ---

func (a *App) startRenameBranch(name string) tea.Cmd {
	a.showRenameBranch = true
	a.renameBranchOld = name
	a.renameBranchErr = ""
	a.branchInput.SetValue(name)
	a.branchInput.Focus()
	a.branchInput.CursorEnd()
	return textinput.Blink
}

func (a App) updateRenameBranchDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showRenameBranch = false
		return a, nil
	case "enter":
		newName := strings.TrimSpace(a.branchInput.Value())
		if newName == "" {
			a.renameBranchErr = "Branch name cannot be empty"
			return a, nil
		}
		if newName == a.renameBranchOld {
			a.showRenameBranch = false
			return a, nil
		}
		a.showRenameBranch = false
		oldName := a.renameBranchOld
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.RenameBranch(oldName, newName); err != nil {
				return ErrorMsg{Err: err}
			}
			return RenameBranchSuccessMsg{OldName: oldName, NewName: newName}
		}
	}

	var cmd tea.Cmd
	a.branchInput, cmd = a.branchInput.Update(msg)
	return a, cmd
}

func (a App) overlayRenameBranchDialog(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	content := titleStyle.Render("Rename Branch") + "\n\n"
	content += a.branchInput.View() + "\n\n"
	if a.renameBranchErr != "" {
		content += errorStyle.Render(a.renameBranchErr) + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] rename  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Delete branch confirmation ---

func (a App) updateDeleteBranchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	branch := a.deleteBranchName
	runner := a.runner
	switch msg.String() {
	case "esc", "n":
		a.showDeleteBranchConfirm = false
		a.deleteBranchName = ""
		return a, nil
	case "y", "enter":
		a.showDeleteBranchConfirm = false
		a.deleteBranchName = ""
		return a, func() tea.Msg {
			if err := runner.DeleteBranch(branch); err != nil {
				return ErrorMsg{Err: err}
			}
			return RefreshMsg{}
		}
	}
	return a, nil
}

func (a App) overlayDeleteBranchConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red

	content := titleStyle.Render("Delete Branch") + "\n\n"
	content += warnStyle.Render("Delete branch "+a.deleteBranchName+"?") + "\n"
	content += dimStyle.Render("This cannot be undone.") + "\n\n"
	content += "  [y/Enter] delete  " + dimStyle.Render("[n/Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Discard confirmation ---

func (a App) updateDiscardConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	table := a.discardTable
	runner := a.runner
	switch msg.String() {
	case "esc", "n":
		a.showDiscardConfirm = false
		a.discardTable = ""
		return a, nil
	case "y", "enter":
		a.showDiscardConfirm = false
		a.discardTable = ""
		return a, func() tea.Msg {
			if err := runner.CheckoutTable(table); err != nil {
				return ErrorMsg{Err: err}
			}
			return RefreshMsg{}
		}
	}
	return a, nil
}

func (a App) overlayDiscardConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red

	content := titleStyle.Render("Discard Changes") + "\n\n"
	content += warnStyle.Render("Discard all changes to "+a.discardTable+"?") + "\n"
	content += dimStyle.Render("This cannot be undone.") + "\n\n"
	content += dimStyle.Render("[y/Enter] discard  [n/Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Reset menu ---

func (a App) updateResetMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hash := a.resetCommitHash
	runner := a.runner
	switch msg.String() {
	case "esc":
		a.showResetMenu = false
		a.resetCommitHash = ""
		return a, nil
	case "s":
		a.showResetMenu = false
		a.resetCommitHash = ""
		return a, func() tea.Msg {
			if err := runner.ResetSoft(hash); err != nil {
				return ErrorMsg{Err: err}
			}
			return ResetSuccessMsg{Mode: "soft"}
		}
	case "h":
		// Show confirmation before hard reset
		a.showResetMenu = false
		a.showHardResetConfirm = true
		return a, nil
	}
	return a, nil
}

func (a App) overlayResetMenu(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	shortHash := a.resetCommitHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Reset to "+shortHash) + "\n\n"
	content += "  [s] soft reset  " + dimStyle.Render("— keep working changes") + "\n"
	content += "  [h] hard reset  " + dimStyle.Render("— discard all changes") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Hard reset confirmation ---

func (a App) updateHardResetConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hash := a.resetCommitHash
	runner := a.runner
	switch msg.String() {
	case "esc", "n":
		a.showHardResetConfirm = false
		a.resetCommitHash = ""
		return a, nil
	case "y", "enter":
		a.showHardResetConfirm = false
		a.resetCommitHash = ""
		return a, func() tea.Msg {
			if err := runner.ResetHard(hash); err != nil {
				return ErrorMsg{Err: err}
			}
			return ResetSuccessMsg{Mode: "hard"}
		}
	}
	return a, nil
}

func (a App) overlayHardResetConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	shortHash := a.resetCommitHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red

	content := titleStyle.Render("Hard Reset") + "\n\n"
	content += warnStyle.Render("Reset to "+shortHash+" and discard ALL changes?") + "\n"
	content += dimStyle.Render("This cannot be undone.") + "\n\n"
	content += "  [y/Enter] reset  " + dimStyle.Render("[n/Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Merge menu ---

func (a App) updateMergeMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	branch := a.mergeBranch
	runner := a.runner
	switch msg.String() {
	case "esc":
		a.showMergeMenu = false
		a.mergeBranch = ""
		return a, nil
	case "m":
		a.showMergeMenu = false
		a.mergeBranch = ""
		return a, func() tea.Msg {
			if _, err := runner.Merge(branch); errors.Is(err, dolt.ErrMergeConflict) {
				return MergeConflictMsg{Branch: branch}
			} else if err != nil {
				return ErrorMsg{Err: err}
			}
			return MergeSuccessMsg{Branch: branch}
		}
	case "s":
		a.showMergeMenu = false
		a.mergeBranch = ""
		return a, func() tea.Msg {
			if _, err := runner.MergeSquash(branch); err != nil {
				return ErrorMsg{Err: err}
			}
			return MergeSuccessMsg{Branch: branch}
		}
	}
	return a, nil
}

func (a App) overlayMergeMenu(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Merge "+a.mergeBranch) + "\n\n"
	content += "  [m] merge        " + dimStyle.Render("— regular merge") + "\n"
	content += "  [s] squash merge " + dimStyle.Render("— squash into one commit") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Stash List ---

func (a App) updateStashList(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		a.showStashList = false
		return a, nil
	case "j", "down":
		if a.stashCursor < len(a.stashEntries)-1 {
			a.stashCursor++
		}
		return a, nil
	case "k", "up":
		if a.stashCursor > 0 {
			a.stashCursor--
		}
		return a, nil
	case " ", "enter":
		// Pop the selected stash
		if a.stashCursor < len(a.stashEntries) {
			idx := a.stashEntries[a.stashCursor].Index
			a.showStashList = false
			return a, a.stashPopCmd(idx)
		}
	case "d":
		// Drop the selected stash
		if a.stashCursor < len(a.stashEntries) {
			idx := a.stashEntries[a.stashCursor].Index
			a.showStashList = false
			return a, a.stashDropCmd(idx)
		}
	}
	return a, nil
}

func (a App) overlayStashList(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedLine := lipgloss.NewStyle().Reverse(true)

	content := titleStyle.Render("Stash List") + "\n\n"

	if len(a.stashEntries) == 0 {
		content += dimStyle.Render("  No stash entries") + "\n"
	} else {
		for i, entry := range a.stashEntries {
			// Truncate hash to 7 chars
			hash := entry.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			line := fmt.Sprintf("  stash@{%d}: %s %s", entry.Index, hash, entry.Message)
			if len(line) > dialogW-4 {
				line = line[:dialogW-7] + "..."
			}
			if i == a.stashCursor {
				content += selectedLine.Render(line) + "\n"
			} else {
				content += line + "\n"
			}
		}
	}

	content += "\n" + dimStyle.Render("[Space/Enter] pop  [d] drop  [Esc] close")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Help ---

// helpBindings is the structured list of keybindings for the help overlay.
// Each entry is a [section, key, description] triple; section headers have
// an empty key.
var helpBindings = []struct{ Section, Key, Desc string }{
	{"Global", "q / Ctrl+C", "Quit"},
	{"Global", "Tab / S-Tab", "Next / previous panel (1-2-3-main)"},
	{"Global", "1-3", "Jump to left panel"},
	{"Global", "c", "Commit"},
	{"Global", "+ / _", "Zoom panel"},
	{"Global", "< / >", "Narrow / widen left column"},
	{"Global", "=", "Reset column width"},
	{"Global", "P", "Push to remote"},
	{"Global", "p", "Pull from remote"},
	{"Global", "f", "Fetch from remote"},
	{"Global", "R", "Refresh all data"},
	{"Global", "S", "Stash changes / show stash list"},
	{"Global", ":", "Run SQL query"},
	{"Global", "/", "Filter panel items"},
	{"Global", "Esc", "Back / reset zoom / clear filter"},
	{"Global", "?", "Toggle help"},
	{"Tables Panel", "j/k", "Navigate"},
	{"Tables Panel", "Space", "Stage/unstage table"},
	{"Tables Panel", "a", "Stage all"},
	{"Tables Panel", "A", "Unstage all"},
	{"Tables Panel", "d", "Discard changes"},
	{"Tables Panel", "O", "Resolve conflicts (ours)"},
	{"Tables Panel", "T", "Resolve conflicts (theirs)"},
	{"Tables Panel", "X", "Abort merge"},
	{"Tables Panel", "s", "View schema"},
	{"Tables Panel", "Enter", "Browse table data"},
	{"Branches Panel", "j/k", "Navigate"},
	{"Branches Panel", "Enter", "Checkout branch"},
	{"Branches Panel", "m", "Merge into current"},
	{"Branches Panel", "n", "New branch"},
	{"Branches Panel", "r", "Rename branch"},
	{"Branches Panel", "D", "Delete branch"},
	{"Commits Panel", "j/k", "Navigate"},
	{"Commits Panel", "Enter", "View commit details"},
	{"Commits Panel", "A", "Amend last commit"},
	{"Commits Panel", "g", "Reset to commit"},
	{"Main Panel", "j/k", "Scroll up/down"},
	{"Main Panel", "PgUp/PgDn", "Page up/down"},
	{"Main Panel", "u/d", "Half page up/down"},
	{"Main Panel", "H/L", "Scroll left/right"},
}

func (a App) renderHelp() string {
	filter := strings.ToLower(strings.TrimSpace(a.helpFilter.Value()))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("lazydolt - Keyboard Shortcuts"))
	sb.WriteString("\n\n")
	sb.WriteString(a.helpFilter.View())
	sb.WriteString("\n\n")

	lastSection := ""
	matchCount := 0
	for _, b := range helpBindings {
		// Filter: match against key or description (case-insensitive)
		if filter != "" {
			combined := strings.ToLower(b.Key + " " + b.Desc + " " + b.Section)
			if !strings.Contains(combined, filter) {
				continue
			}
		}

		// Section header
		if b.Section != lastSection {
			if lastSection != "" {
				sb.WriteString("\n")
			}
			sb.WriteString("  ")
			sb.WriteString(lipgloss.NewStyle().Bold(true).Render(b.Section))
			sb.WriteString("\n")
			lastSection = b.Section
		}

		sb.WriteString(fmt.Sprintf("    %-14s%s\n", b.Key, b.Desc))
		matchCount++
	}

	if filter != "" && matchCount == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("  No matches"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press ? or Esc to close"))

	dialogW := 54
	if a.width < 64 {
		dialogW = a.width - 10
	}
	box := commitBoxStyle.Width(dialogW).Render(sb.String())
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

// --- Re-export component message types for the switch in Update ---

type stageTableMsg = components.StageTableMsg
type unstageTableMsg = components.UnstageTableMsg
type stageAllMsg = components.StageAllMsg
type unstageAllMsg = components.UnstageAllMsg
type viewDiffMsg = components.ViewDiffMsg
type viewSchemaMsg = components.ViewSchemaMsg
type viewTableDataMsg = components.ViewTableDataMsg
type viewCommitMsg = components.ViewCommitMsg
type checkoutBranchMsg = components.CheckoutBranchMsg
type deleteBranchMsg = components.DeleteBranchMsg
type newBranchPromptMsg = components.NewBranchPromptMsg
type browserPageMsg = components.BrowserPageMsg
