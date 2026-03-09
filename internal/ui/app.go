package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
	mainViewCount                   // sentinel for wrapping
)

// mainViewTabNames returns the display names for the right panel tabs.
var mainViewTabNames = [mainViewCount]string{"Status", "Browse", "Schema"}

// ScreenMode controls the column split ratio (like lazygit's +/_ cycling).
type ScreenMode int

const (
	ScreenNormal     ScreenMode = iota // default split (~30% left)
	ScreenHalf                         // roughly 50/50
	ScreenFullscreen                   // focused column takes 100%
	screenModeCount                    // sentinel for wrapping
)

// BranchSort controls how branches are ordered.
type BranchSort int

const (
	BranchSortDate   BranchSort = iota // latest commit date (default)
	BranchSortName                     // alphabetical by name
	BranchSortAuthor                   // by latest committer
	branchSortCount                    // sentinel
)

var branchSortLabels = [branchSortCount]string{"Date", "Name", "Author"}

func (s BranchSort) orderBy() dolt.BranchOrderBy {
	switch s {
	case BranchSortName:
		return dolt.BranchOrderByName
	case BranchSortAuthor:
		return dolt.BranchOrderByAuthor
	default:
		return dolt.BranchOrderByDate
	}
}

// CommitSort controls how commits are ordered.
type CommitSort int

const (
	CommitSortDate    CommitSort = iota // date descending (default)
	CommitSortDateAsc                   // date ascending (oldest first)
	CommitSortAuthor                    // by committer
	CommitSortMessage                   // by message
	commitSortCount                     // sentinel
)

var commitSortLabels = [commitSortCount]string{"Date (newest)", "Date (oldest)", "Author", "Message"}

func (s CommitSort) orderBy() dolt.CommitOrderBy {
	switch s {
	case CommitSortDateAsc:
		return dolt.CommitOrderByDateAsc
	case CommitSortAuthor:
		return dolt.CommitOrderByAuthor
	case CommitSortMessage:
		return dolt.CommitOrderByMessage
	default:
		return dolt.CommitOrderByDateDesc
	}
}

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
	schemaDiff  bool // toggle: show schema diff instead of data diff
	diffStat    bool // toggle: show diff statistics instead of full diff
	blameMode   bool // true when showing blame output
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

	// Rebase confirmation
	showRebaseConfirm bool
	rebaseBranch      string

	// Cherry-pick confirmation
	showCherryPickConfirm bool
	cherryPickHash        string

	// Revert confirmation
	showRevertConfirm bool
	revertHash        string

	// Tag creation dialog
	showTagDialog bool
	tagCommitHash string
	tagErr        string

	// Delete tag confirmation
	showDeleteTagConfirm bool
	deleteTagName        string

	// Delete branch confirmation
	showDeleteBranchConfirm bool
	deleteBranchName        string

	// Add remote dialog
	showAddRemote  bool
	remoteURLInput textinput.Model
	addRemoteErr   string
	addRemoteStep  int // 0 = entering name, 1 = entering URL

	// Delete remote confirmation
	showDeleteRemoteConfirm bool
	deleteRemoteName        string

	// Rename branch dialog
	showRenameBranch bool
	renameBranchOld  string
	renameBranchErr  string

	// Table operations menu and sub-dialogs
	showTableOpsMenu bool
	tableOpsTable    string // the table being operated on
	tableOpsCursor   int    // menu cursor (0=rename, 1=copy, 2=drop, 3=export)
	showTableRename  bool
	showTableCopy    bool
	showTableDrop    bool
	showTableExport  bool
	tableOpsErr      string

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

	// Commit detail mode: shows changed tables for a specific commit
	commitDetailHash   string                 // non-empty when in commit detail mode
	commitDetailTables []domain.DiffStatEntry // tables changed in the commit
	commitDetailCursor int                    // cursor in the changed tables list

	// Panel filter
	filterInput  textinput.Model
	filterActive bool // text input is focused

	// Flash message (error or success)
	flashMsg     string
	flashIsError bool   // true for errors, false for success
	flashFull    string // full untruncated message for overlay display
	flashID      int    // monotonic ID to match timeout messages

	// Branch commit viewing (view another branch's commits without checkout)
	currentBranch string // the checked-out branch name
	viewingBranch string // non-empty when viewing another branch's commits

	// Help
	showHelp     bool
	helpFilter   textinput.Model
	helpViewport viewport.Model

	// Undo/redo
	undoStack       []domain.UndoEntry // stack of pre-mutation states
	redoStack       []domain.UndoEntry // stack of undone states for redo
	showUndoConfirm bool               // true when undo confirmation dialog is shown
	showRedoConfirm bool               // true when redo confirmation dialog is shown

	// Sort options
	branchSort     BranchSort
	commitSort     CommitSort
	showSortMenu   bool
	sortMenuCursor int

	// Config viewer
	showConfig     bool
	configViewport viewport.Model
	configContent  string

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

	ri := textinput.New()
	ri.Placeholder = "https://..."
	ri.CharLimit = 500

	hf := textinput.New()
	hf.Placeholder = "Type to filter..."
	hf.CharLimit = 50
	hf.Prompt = "/ "

	app := App{
		runner:         runner,
		repoName:       filepath.Base(runner.RepoDir),
		repoParent:     filepath.Dir(runner.RepoDir),
		focused:        components.PanelTables,
		diffView:       components.NewDiffView(80, 20),
		schemaView:     components.NewSchemaView(80, 20),
		browserView:    components.NewBrowserView(80, 20),
		logView:        components.NewLogView(80, 20),
		commitInput:    ti,
		branchInput:    bi,
		remoteURLInput: ri,
		sqlInput:       si,
		sqlHistoryIdx:  -1,
		filterInput:    fi,
		helpFilter:     hf,
		spinner:        s,
		leftRatio:      30,
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
		// Recalculate help viewport on resize if it's open
		if a.showHelp {
			a.syncHelpViewport()
		}
		// Viewport sizes will be recalculated in View()
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg)

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
		// Rebase confirmation intercepts all keys when active
		if a.showRebaseConfirm {
			return a.updateRebaseConfirm(msg)
		}
		// Cherry-pick confirmation intercepts all keys when active
		if a.showCherryPickConfirm {
			return a.updateCherryPickConfirm(msg)
		}
		// Revert confirmation intercepts all keys when active
		if a.showRevertConfirm {
			return a.updateRevertConfirm(msg)
		}
		// Undo confirmation intercepts all keys when active
		if a.showUndoConfirm {
			return a.updateUndoConfirm(msg)
		}
		// Redo confirmation intercepts all keys when active
		if a.showRedoConfirm {
			return a.updateRedoConfirm(msg)
		}
		// Tag dialog intercepts all keys when active
		if a.showTagDialog {
			return a.updateTagDialog(msg)
		}
		// Delete tag confirmation intercepts all keys when active
		if a.showDeleteTagConfirm {
			return a.updateDeleteTagConfirm(msg)
		}
		// Table operations menu/dialogs intercept all keys when active
		if a.showTableRename || a.showTableCopy || a.showTableExport {
			return a.updateTableInputDialog(msg)
		}
		if a.showTableDrop {
			return a.updateTableDropConfirm(msg)
		}
		if a.showSortMenu {
			return a.updateSortMenu(msg)
		}
		if a.showTableOpsMenu {
			return a.updateTableOpsMenu(msg)
		}
		// Add remote dialog intercepts all keys when active
		if a.showAddRemote {
			return a.updateAddRemoteDialog(msg)
		}
		// Delete remote confirmation intercepts all keys when active
		if a.showDeleteRemoteConfirm {
			return a.updateDeleteRemoteConfirm(msg)
		}
		// SQL dialog intercepts all keys when active
		if a.showSQL {
			return a.updateSQLDialog(msg)
		}
		// Config viewer intercepts all keys when active
		if a.showConfig {
			return a.updateConfigViewer(msg)
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
			switch msg.String() {
			case "?", "esc":
				a.showHelp = false
				a.helpFilter.Reset()
				a.helpFilter.Blur()
				return a, nil
			case "j", "down":
				a.helpViewport.LineDown(1)
				return a, nil
			case "k", "up":
				a.helpViewport.LineUp(1)
				return a, nil
			case "pgdown":
				a.helpViewport.ViewDown()
				return a, nil
			case "pgup":
				a.helpViewport.ViewUp()
				return a, nil
			case "home":
				a.helpViewport.GotoTop()
				return a, nil
			case "end":
				a.helpViewport.GotoBottom()
				return a, nil
			default:
				// Forward to filter input
				var cmd tea.Cmd
				a.helpFilter, cmd = a.helpFilter.Update(msg)
				// Rebuild viewport content and reset scroll when filter changes
				a.syncHelpViewport()
				a.helpViewport.GotoTop()
				return a, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "esc":
			// Reset blame mode
			if a.blameMode {
				a.blameMode = false
				return a, a.autoPreview()
			}
			// Reset schema diff / diff stat toggles
			if a.schemaDiff || a.diffStat {
				a.schemaDiff = false
				a.diffStat = false
				return a, a.reloadCurrentDiff()
			}
			// Exit commit detail mode first
			if a.commitDetailHash != "" {
				a.commitDetailHash = ""
				a.commitDetailTables = nil
				a.commitDetailCursor = 0
				return a, a.autoViewDiff()
			}
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
			// If viewing another branch's commits, return to current branch
			if a.viewingBranch != "" {
				a.viewingBranch = ""
				return a, a.loadData()
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
		case "y":
			return a, a.copyToClipboard()
		case "ctrl+l":
			return a, func() tea.Msg { return tea.ClearScreen() }
		case "?":
			a.showHelp = true
			a.helpFilter.Reset()
			a.helpFilter.Focus()
			a.syncHelpViewport()
			a.helpViewport.GotoTop()
			return a, textinput.Blink
		case "@":
			return a, a.loadConfig()
		case "z":
			if len(a.undoStack) > 0 {
				a.showUndoConfirm = true
			} else {
				return a, a.setFlashError("Nothing to undo")
			}
			return a, nil
		case "Z":
			if len(a.redoStack) > 0 {
				a.showRedoConfirm = true
			} else {
				return a, a.setFlashError("Nothing to redo")
			}
			return a, nil
		}

		// Commit detail mode: intercept tables panel navigation
		if a.commitDetailHash != "" && a.focused == components.PanelTables {
			switch msg.String() {
			case "j", "down":
				if a.commitDetailCursor < len(a.commitDetailTables)-1 {
					a.commitDetailCursor++
					return a, a.loadCommitTableDiffCurrent()
				}
				return a, nil
			case "k", "up":
				if a.commitDetailCursor > 0 {
					a.commitDetailCursor--
					return a, a.loadCommitTableDiffCurrent()
				}
				return a, nil
			case "s":
				// Toggle schema diff in commit detail mode
				a.schemaDiff = !a.schemaDiff
				a.diffStat = false
				return a, a.reloadCurrentDiff()
			case "w":
				// Toggle diff stat in commit detail mode
				a.diffStat = !a.diffStat
				a.schemaDiff = false
				return a, a.reloadCurrentDiff()
			default:
				// Ignore other keys in commit detail tables view
				return a, nil
			}
		}

		// Route to focused panel, tracking cursor changes for auto-preview.
		prevCursor := a.focusedCursor()
		switch a.focused {
		case components.PanelTables:
			switch msg.String() {
			case "w":
				if a.mainView == MainViewDiff {
					a.diffStat = !a.diffStat
					a.schemaDiff = false
					return a, a.reloadCurrentDiff()
				}
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
			case "b":
				table := a.tables.SelectedTable()
				if table != "" {
					a.blameMode = true
					cmds = append(cmds, a.loadBlame(table))
					return a, tea.Batch(cmds...)
				}
			case "o":
				table := a.tables.SelectedTable()
				if table != "" {
					a.showTableOpsMenu = true
					// Use current name for renamed tables ("old -> new")
					_, a.tableOpsTable, _ = renamedTableParts(table)
					a.tableOpsCursor = 0
					return a, nil
				}
			}
			var cmd tea.Cmd
			a.tables, cmd = a.tables.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelBranches:
			if msg.String() == "s" {
				a.showSortMenu = true
				a.sortMenuCursor = int(a.branchSort)
				return a, nil
			}
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
			if msg.String() == "e" {
				if b := a.branches.SelectedBranch(); b != "" {
					a.showRebaseConfirm = true
					a.rebaseBranch = b
					return a, nil
				}
			}
			if msg.String() == "W" {
				if b := a.branches.SelectedBranch(); b != "" && b != a.currentBranch {
					return a, a.loadBranchDiff(b)
				}
			}
			var cmd tea.Cmd
			a.branches, cmd = a.branches.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelCommits:
			if msg.String() == "s" {
				a.showSortMenu = true
				a.sortMenuCursor = int(a.commitSort)
				return a, nil
			}
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
			if msg.String() == "C" {
				if h := a.commits.SelectedHash(); h != "" {
					a.showCherryPickConfirm = true
					a.cherryPickHash = h
					return a, nil
				}
			}
			if msg.String() == "t" {
				if h := a.commits.SelectedHash(); h != "" {
					a.showRevertConfirm = true
					a.revertHash = h
					return a, nil
				}
			}
			if msg.String() == "T" {
				if h := a.commits.SelectedHash(); h != "" {
					a.showTagDialog = true
					a.tagCommitHash = h
					a.tagErr = ""
					a.branchInput.SetValue("")
					a.branchInput.Placeholder = "Enter tag name..."
					a.branchInput.Focus()
					return a, textinput.Blink
				}
			}
			if msg.String() == "l" {
				a.blameMode = true // reuse blameMode to prevent diff override
				cmds = append(cmds, a.loadReflog())
				return a, tea.Batch(cmds...)
			}
			var cmd tea.Cmd
			a.commits, cmd = a.commits.Update(msg)
			cmds = append(cmds, cmd)
		case components.PanelMain:
			// Toggle schema diff with 's' when viewing diff
			if a.mainView == MainViewDiff && msg.String() == "s" {
				a.schemaDiff = !a.schemaDiff
				a.diffStat = false
				return a, a.reloadCurrentDiff()
			}
			// Toggle diff stat with 'w' when viewing diff
			if a.mainView == MainViewDiff && msg.String() == "w" {
				a.diffStat = !a.diffStat
				a.schemaDiff = false
				return a, a.reloadCurrentDiff()
			}
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
			}
		}
		if a.focusedCursor() != prevCursor {
			a.schemaDiff = false // reset schema diff toggle on navigation
			a.diffStat = false   // reset diff stat toggle on navigation
			a.blameMode = false  // reset blame mode on navigation
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
		a.currentBranch = msg.Branch
		a.statusBar.Dirty = msg.Dirty
		a.statusBar.RepoDir = a.repoName
		a.statusBar.ParentDir = a.repoParent
		a.tables.Tables = msg.Tables
		a.branches.Branches = msg.Branches
		a.branches.Tags = msg.Tags
		a.branches.Remotes = msg.Remotes
		// Only update commits if not viewing a different branch
		if a.viewingBranch == "" {
			a.commits.Commits = msg.Commits
		}
		a.clearFlash()
		// Clamp cursors
		a.tables.ClampCursor()
		if a.branches.Cursor >= a.branches.ItemCount() {
			a.branches.Cursor = max(0, a.branches.ItemCount()-1)
		}
		// Auto-load content for the active tab
		cmds = append(cmds, a.autoPreview())

	case DiffContentMsg:
		// Don't override blame view with auto-loaded diff
		if !a.blameMode {
			a.diffView.SetContent(msg.Table, msg.Content)
		}

	case SchemaContentMsg:
		a.schemaView.SetContent(msg.Table, msg.Schema)

	case BlameContentMsg:
		a.diffView.SetContent("Blame: "+msg.Table, msg.Content)
		a.mainView = MainViewDiff

	case ReflogContentMsg:
		a.diffView.SetContent("Reflog", msg.Content)
		a.mainView = MainViewDiff

	case RefreshMsg:
		cmds = append(cmds, a.loadData())

	case ErrorMsg:
		cmds = append(cmds, a.setFlashError(msg.Err.Error()))

	case CommitSuccessMsg:
		a.showCommit = false
		a.commitErr = ""
		cmds = append(cmds, a.setFlashSuccess("Committed "+msg.Hash[:8]))
		cmds = append(cmds, a.loadData())

	case ResetSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Reset ("+msg.Mode+") successful"))
		cmds = append(cmds, a.loadData())

	case RemoteOpSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess(msg.Op+" successful"))
		cmds = append(cmds, a.loadData())

	case MergeSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Merged "+msg.Branch))
		cmds = append(cmds, a.loadData())

	case MergeConflictMsg:
		a.setFocus(components.PanelTables)
		cmds = append(cmds, a.setFlashError("Merge conflicts with "+msg.Branch))
		cmds = append(cmds, a.loadData())

	case ConflictResolveMsg:
		side := "theirs"
		if msg.Ours {
			side = "ours"
		}
		cmds = append(cmds, a.setFlashSuccess("Resolved "+msg.Table+" with "+side))
		cmds = append(cmds, a.loadData())

	case MergeAbortMsg:
		cmds = append(cmds, a.setFlashSuccess("Merge aborted"))
		cmds = append(cmds, a.loadData())

	case RebaseSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Rebased onto "+msg.Branch))
		cmds = append(cmds, a.loadData())

	case RebaseConflictMsg:
		a.setFocus(components.PanelTables)
		cmds = append(cmds, a.setFlashError("Rebase conflicts with "+msg.Branch))
		cmds = append(cmds, a.loadData())

	case RebaseAbortMsg:
		cmds = append(cmds, a.setFlashSuccess("Rebase aborted"))
		cmds = append(cmds, a.loadData())

	case CherryPickSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Cherry-picked "+msg.Hash[:8]))
		cmds = append(cmds, a.loadData())

	case CherryPickConflictMsg:
		a.setFocus(components.PanelTables)
		cmds = append(cmds, a.setFlashError("Cherry-pick conflicts for "+msg.Hash[:8]))
		cmds = append(cmds, a.loadData())

	case RevertSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Reverted "+msg.Hash[:8]))
		cmds = append(cmds, a.loadData())

	case TagSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Created tag "+msg.Name))
		cmds = append(cmds, a.loadData())

	case DeleteTagSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Deleted tag "+msg.Name))
		cmds = append(cmds, a.loadData())

	case AddRemoteSuccessMsg:
		a.showAddRemote = false
		a.addRemoteErr = ""
		cmds = append(cmds, a.setFlashSuccess("Added remote "+msg.Name))
		cmds = append(cmds, a.loadData())

	case DeleteRemoteSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Deleted remote "+msg.Name))
		cmds = append(cmds, a.loadData())

	case TableRenameSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Renamed "+msg.OldName+" → "+msg.NewName))
		cmds = append(cmds, a.loadData())

	case TableCopySuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Copied "+msg.SrcName+" → "+msg.DstName))
		cmds = append(cmds, a.loadData())

	case TableDropSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Dropped table "+msg.Name))
		cmds = append(cmds, a.loadData())

	case TableExportSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Exported "+msg.Table+" to "+msg.Path))
		a.diffView.SetContent("Export: "+msg.Table, "Exported to "+msg.Path)
		a.mainView = MainViewDiff

	case SQLResultMsg:
		// Display SQL results in the diff view
		title := fmt.Sprintf("SQL: %s", msg.Query)
		a.diffView.SetContent(title, msg.Result)
		a.mainView = MainViewDiff

	case StashSuccessMsg:
		cmds = append(cmds, a.setFlashSuccess("Changes stashed"))
		cmds = append(cmds, a.loadData())

	case ConfigLoadedMsg:
		a.configContent = formatConfigContent(msg.Global, msg.Local)
		a.configViewport = viewport.New(60, 20)
		a.configViewport.SetContent(a.configContent)
		a.showConfig = true

	case StashListMsg:
		a.stashEntries = msg.Entries
		a.stashCursor = 0
		if len(msg.Entries) > 0 {
			a.showStashList = true
		}

	case StashPopMsg:
		a.showStashList = false
		cmds = append(cmds, a.setFlashSuccess(fmt.Sprintf("Popped stash@{%d}", msg.Index)))
		cmds = append(cmds, a.loadData())

	case StashDropMsg:
		// Refresh the stash list after drop
		a.showStashList = false
		cmds = append(cmds, a.setFlashSuccess(fmt.Sprintf("Dropped stash@{%d}", msg.Index)))
		cmds = append(cmds, a.loadData())

	case NewBranchSuccessMsg:
		a.showBranch = false
		a.branchErr = ""
		cmds = append(cmds, a.setFlashSuccess("Created branch "+msg.Name))
		cmds = append(cmds, a.loadData())

	case RenameBranchSuccessMsg:
		a.showRenameBranch = false
		a.renameBranchErr = ""
		cmds = append(cmds, a.setFlashSuccess("Renamed "+msg.OldName+" → "+msg.NewName))
		cmds = append(cmds, a.loadData())

	// Undo/redo messages
	case undoableResultMsg:
		// A mutation succeeded with undo info — push to undo stack
		a.undoStack = append(a.undoStack, msg.Entry)
		a.redoStack = nil // clear redo on new mutation
		// Re-dispatch the inner message
		return a.Update(msg.Inner)

	case undoResultMsg:
		// Undo succeeded — push to redo stack and refresh
		a.redoStack = append(a.redoStack, msg.RedoEntry)
		cmds = append(cmds, a.setFlashSuccess("Undo: reset to "+msg.TargetHash[:8]))
		cmds = append(cmds, a.loadData())

	case redoResultMsg:
		// Redo succeeded — push to undo stack and refresh
		a.undoStack = append(a.undoStack, msg.UndoEntry)
		cmds = append(cmds, a.setFlashSuccess("Redo: reset to "+msg.TargetHash[:8]))
		cmds = append(cmds, a.loadData())

	case flashTimeoutMsg:
		// Auto-clear flash message if it hasn't been replaced
		if msg.ID == a.flashID {
			a.clearFlash()
		}

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
		a.viewingBranch = "" // clear branch viewing on checkout
		cmds = append(cmds, a.checkoutCmd(msg.Branch))
	case viewBranchMsg:
		cmds = append(cmds, a.viewBranchCommits(msg.Branch))
	case branchCommitsMsg:
		a.commits.Commits = msg.Commits
		a.commits.Cursor = 0
	case deleteBranchMsg:
		a.showDeleteBranchConfirm = true
		a.deleteBranchName = msg.Branch
	case deleteTagMsg:
		a.showDeleteTagConfirm = true
		a.deleteTagName = msg.Tag
	case addRemotePromptMsg:
		return a, a.startAddRemote()
	case deleteRemoteMsg:
		a.showDeleteRemoteConfirm = true
		a.deleteRemoteName = msg.Remote
	case newBranchPromptMsg:
		return a, a.startNewBranch()
	case viewCommitMsg:
		cmds = append(cmds, a.loadCommitDetail(msg.Hash))
	case CommitDetailMsg:
		a.commitDetailHash = msg.Hash
		a.commitDetailTables = msg.Tables
		a.commitDetailCursor = 0
		a.focused = components.PanelTables
		// Load diff for the first changed table (or full diff if none)
		if len(msg.Tables) > 0 {
			cmds = append(cmds, a.loadCommitTableDiff(msg.Hash, msg.Tables[0].TableName, msg.Header))
		} else {
			cmds = append(cmds, a.loadCommitDiff(msg.Hash))
		}
	case BrowserDataMsg:
		a.browserView.SetData(msg.Table, msg.Columns, msg.Rows, msg.Total, msg.Offset)
	case browserPageMsg:
		cmds = append(cmds, a.loadTableDataPage(msg.Table, msg.Offset))
	}

	// Forward non-key, non-mouse messages (e.g. resize) to the active
	// viewport. Key events reach the viewport only when PanelMain is focused
	// (handled in the key-routing switch above). Mouse events are handled
	// in handleMouse() with coordinate-based routing.
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		// Already handled above
	default:
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

		// Split bottom area: main panel + command log pane
		const logPaneInnerHHalf = 4
		mainContentH := botInnerH - logPaneInnerHHalf - borderH
		if mainContentH < 3 {
			mainContentH = 3
		}

		a.diffView.SetSize(mainInnerW, mainContentH-1)
		a.schemaView.SetSize(mainInnerW, mainContentH-1)
		a.browserView.SetSize(mainInnerW, mainContentH-1)

		mainTitle := a.mainPanelTitle()
		mainContent := a.mainPanelContent()
		mainFocused := a.focused == components.PanelMain
		mainStyle := blurredBorder
		if mainFocused {
			mainStyle = focusedBorder
		}
		mainRendered := mainStyle.Width(mainInnerW).Height(mainContentH).Render(mainContent)
		mainLines := strings.Split(mainRendered, "\n")
		if len(mainLines) > 0 {
			mainLines[0] = buildTitleBorder(mainTitle, mainInnerW+2, mainFocused)
		}
		mainBox := strings.Join(mainLines, "\n")

		// Command log pane
		a.logView.Entries = a.runner.CommandLog()
		a.logView.RefreshContent()
		a.logView.SetSize(mainInnerW, logPaneInnerHHalf)
		logTitle := fmt.Sprintf("Command Log (%d)", len(a.logView.Entries))
		logRendered := blurredBorder.Width(mainInnerW).Height(logPaneInnerHHalf).Render(a.logView.View())
		logLines := strings.Split(logRendered, "\n")
		if len(logLines) > 0 {
			logLines[0] = buildTitleBorder(logTitle, mainInnerW+2, false)
		}
		logBox := strings.Join(logLines, "\n")

		body = topBox + "\n" + mainBox + "\n" + logBox
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

		// Tables panel — title includes sub-tab indicators or commit detail
		a.tables.Height = tablesH
		var tablesTitle, tablesContent string
		if a.commitDetailHash != "" {
			shortHash := a.commitDetailHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			tablesTitle = fmt.Sprintf("[1]─Changed in %s (%d)", shortHash, len(a.commitDetailTables))
			tablesContent = a.renderCommitDetailTables(tablesH)
		} else {
			tablesTitle = fmt.Sprintf("[1]─%s", a.tabBar())
			tablesContent = panelView(a.tables.View())
		}
		tablesBox := a.panelBox(components.PanelTables, leftW, tablesH, tablesTitle, tablesContent)

		// Branches panel
		a.branches.Height = branchesH
		branchesTitle := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		if len(a.branches.Tags) > 0 {
			branchesTitle += fmt.Sprintf(" Tags (%d)", len(a.branches.Tags))
		}
		if len(a.branches.Remotes) > 0 {
			branchesTitle += fmt.Sprintf(" Remotes (%d)", len(a.branches.Remotes))
		}
		branchesBox := a.panelBox(components.PanelBranches, leftW, branchesH, branchesTitle, panelView(a.branches.View()))

		// Commits panel
		a.commits.Height = commitsH
		commitsBox := a.panelBox(components.PanelCommits, leftW, commitsH, a.commitsTitle(), panelView(a.commits.View()))

		// Left column
		left := lipgloss.JoinVertical(lipgloss.Left, statusBox, tablesBox, branchesBox, commitsBox)

		if a.screenMode == ScreenFullscreen {
			// Fullscreen: only left column visible
			body = left
		} else {
			// Normal: side-by-side columns.
			// Both borders (left + right) add 2 chars.
			mainInnerW := a.width - leftW - 2

			// Split right column: main panel + command log pane
			const logPaneInnerH = 5 // inner height of command log pane
			mainContentH := mainInnerH - logPaneInnerH - borderH
			if mainContentH < 4 {
				mainContentH = 4
			}

			a.diffView.SetSize(mainInnerW, mainContentH-1)
			a.schemaView.SetSize(mainInnerW, mainContentH-1)
			a.browserView.SetSize(mainInnerW, mainContentH-1)

			mainTitle := a.mainPanelTitle()
			mainContent := a.mainPanelContent()
			mainFocused := a.focused == components.PanelMain
			mainStyle := blurredBorder
			if mainFocused {
				mainStyle = focusedBorder
			}
			mainRendered := mainStyle.Width(mainInnerW).Height(mainContentH).Render(mainContent)
			// Embed title in the top border
			mainLines := strings.Split(mainRendered, "\n")
			if len(mainLines) > 0 {
				mainLines[0] = buildTitleBorder(mainTitle, mainInnerW+2, mainFocused)
			}
			mainBox := strings.Join(mainLines, "\n")

			// Command log pane — always visible below main panel
			a.logView.Entries = a.runner.CommandLog()
			a.logView.RefreshContent()
			a.logView.SetSize(mainInnerW, logPaneInnerH)
			logTitle := fmt.Sprintf("Command Log (%d)", len(a.logView.Entries))
			logRendered := blurredBorder.Width(mainInnerW).Height(logPaneInnerH).Render(a.logView.View())
			logLines := strings.Split(logRendered, "\n")
			if len(logLines) > 0 {
				logLines[0] = buildTitleBorder(logTitle, mainInnerW+2, false)
			}
			logBox := strings.Join(logLines, "\n")

			rightColumn := lipgloss.JoinVertical(lipgloss.Left, mainBox, logBox)
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, rightColumn)
		}
	}

	// Key hints or filter input
	var hints string
	if a.filterActive {
		hints = a.filterInput.View() + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("[Enter] confirm  [Esc] clear")
	} else if a.commitDetailHash != "" {
		hints = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).
			Render("j/k navigate tables | s schema diff | Esc back | ? help")
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

	// Flash message bar
	if a.flashMsg != "" {
		if a.flashIsError {
			// Truncate long errors for the hint bar
			display := a.flashMsg
			maxLen := a.width / 2
			if len(display) > maxLen && maxLen > 3 {
				display = display[:maxLen-3] + "..."
			}
			hints = errorStyle.Render("Error: "+display) + "  " + hints
		} else {
			hints = successStyle.Render("✓ "+a.flashMsg) + "  " + hints
		}
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
	if a.showRebaseConfirm {
		result = a.overlayRebaseConfirm(result)
	}
	if a.showCherryPickConfirm {
		result = a.overlayCherryPickConfirm(result)
	}
	if a.showRevertConfirm {
		result = a.overlayRevertConfirm(result)
	}
	if a.showUndoConfirm {
		result = a.overlayUndoConfirm(result)
	}
	if a.showRedoConfirm {
		result = a.overlayRedoConfirm(result)
	}
	if a.showTagDialog {
		result = a.overlayTagDialog(result)
	}
	if a.showDeleteTagConfirm {
		result = a.overlayDeleteTagConfirm(result)
	}
	if a.showSortMenu {
		result = a.overlaySortMenu(result)
	}
	if a.showTableOpsMenu {
		result = a.overlayTableOpsMenu(result)
	}
	if a.showTableRename || a.showTableCopy || a.showTableExport {
		result = a.overlayTableInputDialog(result)
	}
	if a.showTableDrop {
		result = a.overlayTableDropConfirm(result)
	}
	if a.showAddRemote {
		result = a.overlayAddRemoteDialog(result)
	}
	if a.showDeleteRemoteConfirm {
		result = a.overlayDeleteRemoteConfirm(result)
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
	if a.showConfig {
		result = a.overlayConfigViewer(result)
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
		var title, body string
		if a.commitDetailHash != "" {
			shortHash := a.commitDetailHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			title = fmt.Sprintf("[1]─Changed in %s (%d)", shortHash, len(a.commitDetailTables))
			body = a.renderCommitDetailTables(innerH)
		} else {
			title = fmt.Sprintf("[1]─%s", a.tabBar())
			body = content(a.tables.View())
		}
		return a.panelBox(components.PanelTables, width, innerH, title, body)
	case components.PanelBranches:
		a.branches.Height = innerH
		title := fmt.Sprintf("[2]─Branches (%d)", len(a.branches.Branches))
		if len(a.branches.Tags) > 0 {
			title += fmt.Sprintf(" Tags (%d)", len(a.branches.Tags))
		}
		if len(a.branches.Remotes) > 0 {
			title += fmt.Sprintf(" Remotes (%d)", len(a.branches.Remotes))
		}
		return a.panelBox(components.PanelBranches, width, innerH, title, content(a.branches.View()))
	case components.PanelCommits:
		a.commits.Height = innerH
		return a.panelBox(components.PanelCommits, width, innerH, a.commitsTitle(), content(a.commits.View()))
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

// commitsTitle returns the title for the commits panel, including the viewed
// branch name if viewing a different branch's commits.
func (a *App) commitsTitle() string {
	count := len(a.commits.Commits)
	if a.viewingBranch != "" {
		return fmt.Sprintf("[3]─Commits: %s (%d)", a.viewingBranch, count)
	}
	return fmt.Sprintf("[3]─Commits (%d)", count)
}

// viewBranchCommits loads commits for a specific branch without checking it out.
// If branch is the current branch, it clears the viewingBranch state.
func (a *App) viewBranchCommits(branch string) tea.Cmd {
	// If viewing current branch, just clear the viewing state
	if branch == a.currentBranch {
		a.viewingBranch = ""
		return a.loadData()
	}
	a.viewingBranch = branch
	runner := a.runner
	commitOrder := a.commitSort.orderBy()
	return func() tea.Msg {
		commits, err := runner.Log(branch, 50, commitOrder)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("log %s: %w", branch, err)}
		}
		return branchCommitsMsg{Branch: branch, Commits: commits}
	}
}

func (a *App) loadData() tea.Cmd {
	runner := a.runner
	branchOrder := a.branchSort.orderBy()
	commitOrder := a.commitSort.orderBy()
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
		type tagsResult struct {
			tags []domain.Tag
			err  error
		}
		type remotesResult struct {
			remotes []domain.Remote
			err     error
		}

		branchCh := make(chan branchResult, 1)
		tablesCh := make(chan tablesResult, 1)
		branchesCh := make(chan branchesResult, 1)
		commitsCh := make(chan commitsResult, 1)
		tagsCh := make(chan tagsResult, 1)
		remotesCh := make(chan remotesResult, 1)

		go func() {
			b, err := runner.CurrentBranch()
			branchCh <- branchResult{b, err}
		}()
		go func() {
			t, err := runner.Tables()
			tablesCh <- tablesResult{t, err}
		}()
		go func() {
			br, err := runner.Branches(branchOrder)
			branchesCh <- branchesResult{br, err}
		}()
		go func() {
			c, err := runner.Log("", 50, commitOrder)
			commitsCh <- commitsResult{c, err}
		}()
		go func() {
			tg, err := runner.Tags()
			tagsCh <- tagsResult{tg, err}
		}()
		go func() {
			rm, err := runner.Remotes()
			remotesCh <- remotesResult{rm, err}
		}()

		brRes := <-branchCh
		tblRes := <-tablesCh
		brchRes := <-branchesCh
		cmtRes := <-commitsCh
		tagRes := <-tagsCh
		rmtRes := <-remotesCh

		// Derive dirty from tables — any table with a non-nil Status
		// has uncommitted changes, avoiding a redundant Status() call.
		dirty := false
		for _, t := range tblRes.tables {
			if t.Status != nil {
				dirty = true
				break
			}
		}

		// Tags are optional — don't fail the whole load if tags can't be read
		var tags []domain.Tag
		if tagRes.err == nil {
			tags = tagRes.tags
		}

		// Remotes are optional — don't fail the whole load if remotes can't be read
		var remotes []domain.Remote
		if rmtRes.err == nil {
			remotes = rmtRes.remotes
		}

		return DataLoadedMsg{
			Branch:   brRes.branch,
			Dirty:    dirty,
			Tables:   tblRes.tables,
			Branches: brchRes.branches,
			Commits:  cmtRes.commits,
			Tags:     tags,
			Remotes:  remotes,
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

// renamedTableParts splits a renamed table name "old -> new" into its parts.
// Returns (oldName, newName, true) if renamed, or (table, table, false) if not.
func renamedTableParts(table string) (old, new string, renamed bool) {
	if parts := strings.SplitN(table, " -> ", 2); len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return table, table, false
}

func (a *App) loadDiff(table string, staged bool) tea.Cmd {
	runner := a.runner
	schemaOnly := a.schemaDiff
	statOnly := a.diffStat
	label := table
	// Renamed tables appear as "old -> new" in dolt_status;
	// extract the old name for the diff command (either name works).
	diffTable, _, _ := renamedTableParts(table)
	if label == "" {
		if staged {
			label = "all staged"
		} else {
			label = "all unstaged"
		}
	}
	if schemaOnly {
		label += " (schema)"
	} else if statOnly {
		label += " (stat)"
	}
	return func() tea.Msg {
		var content string
		var err error
		if statOnly {
			content, err = runner.DiffStat(diffTable, staged)
		} else if schemaOnly {
			content, err = runner.DiffSchema(diffTable, staged)
		} else {
			content, err = runner.DiffText(diffTable, staged)
		}
		if err != nil {
			return ErrorMsg{Err: err}
		}
		if schemaOnly && strings.TrimSpace(content) == "" {
			content = "No schema changes"
		}
		if statOnly && strings.TrimSpace(content) == "" {
			content = "No changes"
		}
		return DiffContentMsg{Table: label, Content: content}
	}
}

// loadBranchDiff loads the diff between the current branch and the given branch.
func (a *App) loadBranchDiff(branch string) tea.Cmd {
	runner := a.runner
	current := a.currentBranch
	schemaOnly := a.schemaDiff
	statOnly := a.diffStat
	return func() tea.Msg {
		var content string
		var err error
		if statOnly {
			content, err = runner.DiffStatRefs(current, branch, "")
		} else if schemaOnly {
			content, err = runner.DiffSchemaRefs(current, branch, "")
		} else {
			content, err = runner.DiffRefs(current, branch, "")
		}
		if err != nil {
			return ErrorMsg{Err: err}
		}
		if strings.TrimSpace(content) == "" {
			content = "No differences"
		}
		label := current + ".." + branch
		if schemaOnly {
			label += " (schema)"
		} else if statOnly {
			label += " (stat)"
		}
		return DiffContentMsg{Table: label, Content: content}
	}
}

// reloadCurrentDiff reloads the diff for the currently selected table,
// respecting the schemaDiff and diffStat toggles.
func (a *App) reloadCurrentDiff() tea.Cmd {
	// In commit detail mode, reload the per-table diff
	if a.commitDetailHash != "" && len(a.commitDetailTables) > 0 {
		table := a.commitDetailTables[a.commitDetailCursor].TableName
		header := a.commitHeader(a.commitDetailHash)
		if a.diffStat {
			return a.loadCommitStatDiff(a.commitDetailHash, table, header)
		}
		if a.schemaDiff {
			return a.loadCommitSchemaDiff(a.commitDetailHash, table, header)
		}
		return a.loadCommitTableDiff(a.commitDetailHash, table, header)
	}
	// Normal mode
	if a.focused == components.PanelTables || a.focused == components.PanelMain {
		if a.tables.IsOnHeader() {
			return a.loadDiff("", a.tables.SelectedIsStaged())
		}
		table := a.tables.SelectedTable()
		if table == "" {
			return a.loadDiff("", a.tables.SelectedIsStaged())
		}
		return a.loadDiff(table, a.tables.SelectedIsStaged())
	}
	return nil
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
	// Use the new name for renamed tables ("old -> new")
	_, queryTable, _ := renamedTableParts(table)
	return func() tea.Msg {
		schema, err := runner.Schema(queryTable)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return SchemaContentMsg{Table: table, Schema: schema.CreateStatement}
	}
}

func (a *App) loadReflog() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		output, err := runner.Reflog()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ReflogContentMsg{Content: output}
	}
}

func (a *App) loadBlame(table string) tea.Cmd {
	runner := a.runner
	// Use the new name for renamed tables ("old -> new")
	_, queryTable, _ := renamedTableParts(table)
	return func() tea.Msg {
		output, err := runner.Blame(queryTable)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return BlameContentMsg{Table: table, Content: output}
	}
}

func (a *App) loadTableData(table string) tea.Cmd {
	// Use the new name for renamed tables ("old -> new")
	_, queryTable, _ := renamedTableParts(table)
	return a.loadTableDataPage(queryTable, 0)
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
	// Find the commit metadata from our cached list
	var header string
	for _, c := range a.commits.Commits {
		if c.Hash == hash {
			header = fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n",
				c.Hash,
				c.Author, c.Email,
				c.Date.Format("Mon Jan 2 15:04:05 2006"),
				c.Message,
			)
			break
		}
	}
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
		if header != "" {
			content = header + "\n" + content
		}
		return DiffContentMsg{Table: hash[:7], Content: content}
	}
}

// loadCommitDetail loads the diff stat for a commit and enters commit detail mode.
func (a *App) loadCommitDetail(hash string) tea.Cmd {
	runner := a.runner
	var header string
	for _, c := range a.commits.Commits {
		if c.Hash == hash {
			header = fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n",
				c.Hash,
				c.Author, c.Email,
				c.Date.Format("Mon Jan 2 15:04:05 2006"),
				c.Message,
			)
			break
		}
	}
	return func() tea.Msg {
		tables, err := runner.DiffStatBetween(hash+"~1", hash)
		if err != nil {
			// Initial commit has no parent — try without parent
			tables, err = runner.DiffStatBetween("", hash)
			if err != nil {
				// Fall back to empty table list
				tables = nil
			}
		}
		return CommitDetailMsg{Hash: hash, Header: header, Tables: tables}
	}
}

// loadCommitTableDiff loads the diff for a single table within a commit.
func (a *App) loadCommitTableDiff(hash, table, header string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		content, err := runner.Exec("diff", hash+"^", hash, table)
		if err != nil {
			// Initial commit
			content, err = runner.Exec("diff", hash, table)
			if err != nil {
				return ErrorMsg{Err: err}
			}
		}
		if header != "" {
			content = header + "\n" + content
		}
		return DiffContentMsg{Table: table, Content: content}
	}
}

// loadCommitSchemaDiff loads the schema-only diff for a table within a commit.
func (a *App) loadCommitSchemaDiff(hash, table, header string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		content, err := runner.DiffSchemaRefs(hash+"^", hash, table)
		if err != nil {
			content, err = runner.DiffSchemaRefs(hash, "", table)
			if err != nil {
				return ErrorMsg{Err: err}
			}
		}
		if strings.TrimSpace(content) == "" {
			content = "No schema changes for " + table
		}
		if header != "" {
			content = header + "\n" + content
		}
		return DiffContentMsg{Table: table + " (schema)", Content: content}
	}
}

// loadCommitStatDiff loads the diff statistics for a table within a commit.
func (a *App) loadCommitStatDiff(hash, table, header string) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		content, err := runner.DiffStatRefs(hash+"^", hash, table)
		if err != nil {
			content, err = runner.DiffStatRefs(hash, "", table)
			if err != nil {
				return ErrorMsg{Err: err}
			}
		}
		if strings.TrimSpace(content) == "" {
			content = "No changes for " + table
		}
		if header != "" {
			content = header + "\n" + content
		}
		return DiffContentMsg{Table: table + " (stat)", Content: content}
	}
}

// loadCommitTableDiffCurrent loads the diff for the currently selected
// table in commit detail mode.
func (a *App) loadCommitTableDiffCurrent() tea.Cmd {
	if a.commitDetailHash == "" || len(a.commitDetailTables) == 0 {
		return nil
	}
	table := a.commitDetailTables[a.commitDetailCursor].TableName
	header := a.commitHeader(a.commitDetailHash)
	return a.loadCommitTableDiff(a.commitDetailHash, table, header)
}

// commitHeader builds the metadata header for a commit from cached data.
func (a *App) commitHeader(hash string) string {
	for _, c := range a.commits.Commits {
		if c.Hash == hash {
			return fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n",
				c.Hash,
				c.Author, c.Email,
				c.Date.Format("Mon Jan 2 15:04:05 2006"),
				c.Message,
			)
		}
	}
	return ""
}

// --- Clipboard ---

// copyToClipboard copies the relevant value from the focused panel to the
// system clipboard and shows a flash message.
func (a *App) copyToClipboard() tea.Cmd {
	var value string
	switch a.focused {
	case components.PanelTables:
		value = a.tables.SelectedTable()
	case components.PanelBranches:
		// Copy whichever item type is selected: branch, tag, or remote
		if v := a.branches.SelectedBranch(); v != "" {
			value = v
		} else if v := a.branches.SelectedTag(); v != "" {
			value = v
		} else if v := a.branches.SelectedRemote(); v != "" {
			value = v
		}
	case components.PanelCommits:
		value = a.commits.SelectedHash()
	case components.PanelMain:
		// Nothing obvious to copy from the main viewport
	}

	if value == "" {
		return a.setFlashError("Nothing to copy")
	}

	if err := clipboard.WriteAll(value); err != nil {
		return a.setFlashError("Clipboard: " + err.Error())
	}

	// Truncate long values for the flash message
	display := value
	if len(display) > 40 {
		display = display[:37] + "..."
	}
	return a.setFlashSuccess("Copied: " + display)
}

// --- Flash messages ---

const flashDuration = 5 * time.Second

var successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green

// setFlashError sets an error flash message with auto-clear timeout.
func (a *App) setFlashError(msg string) tea.Cmd {
	a.flashID++
	a.flashMsg = msg
	a.flashIsError = true
	a.flashFull = msg
	id := a.flashID
	return tea.Tick(flashDuration, func(time.Time) tea.Msg {
		return flashTimeoutMsg{ID: id}
	})
}

// setFlashSuccess sets a success flash message with auto-clear timeout.
func (a *App) setFlashSuccess(msg string) tea.Cmd {
	a.flashID++
	a.flashMsg = msg
	a.flashIsError = false
	a.flashFull = ""
	id := a.flashID
	return tea.Tick(flashDuration, func(time.Time) tea.Msg {
		return flashTimeoutMsg{ID: id}
	})
}

// clearFlash clears the flash message.
func (a *App) clearFlash() {
	a.flashMsg = ""
	a.flashFull = ""
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

// --- Undo/redo ---

// undoableCmd wraps a mutation function so that the current HEAD position
// is captured before the mutation runs. If the mutation succeeds, the
// result is wrapped in an undoableResultMsg carrying the pre-mutation state.
// If the mutation returns an ErrorMsg, it passes through unwrapped.
func (a *App) undoableCmd(desc string, fn func() tea.Msg) tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		hash, err := runner.HeadHash()
		if err != nil {
			// Can't capture undo state; run mutation anyway
			return fn()
		}
		branch, _ := runner.CurrentBranch()

		result := fn()

		// Don't record undo for failed operations
		if _, ok := result.(ErrorMsg); ok {
			return result
		}

		return undoableResultMsg{
			Inner: result,
			Entry: domain.UndoEntry{
				Branch:      branch,
				Hash:        hash,
				Description: desc,
			},
		}
	}
}

func (a App) updateUndoConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		a.showUndoConfirm = false
		return a, nil
	case "y", "enter":
		a.showUndoConfirm = false
		if len(a.undoStack) == 0 {
			return a, nil
		}
		entry := a.undoStack[len(a.undoStack)-1]
		a.undoStack = a.undoStack[:len(a.undoStack)-1]
		runner := a.runner
		return a, func() tea.Msg {
			// Capture current state for redo before resetting
			curHash, _ := runner.HeadHash()
			curBranch, _ := runner.CurrentBranch()

			// Switch branch if needed
			if entry.Branch != curBranch {
				if err := runner.Checkout(entry.Branch); err != nil {
					return ErrorMsg{Err: fmt.Errorf("undo: checkout %s: %w", entry.Branch, err)}
				}
			}

			if err := runner.ResetHard(entry.Hash); err != nil {
				return ErrorMsg{Err: fmt.Errorf("undo: reset to %s: %w", entry.Hash[:7], err)}
			}

			return undoResultMsg{
				RedoEntry: domain.UndoEntry{
					Branch:      curBranch,
					Hash:        curHash,
					Description: entry.Description,
				},
				TargetHash: entry.Hash,
			}
		}
	}
	return a, nil
}

func (a App) updateRedoConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		a.showRedoConfirm = false
		return a, nil
	case "y", "enter":
		a.showRedoConfirm = false
		if len(a.redoStack) == 0 {
			return a, nil
		}
		entry := a.redoStack[len(a.redoStack)-1]
		a.redoStack = a.redoStack[:len(a.redoStack)-1]
		runner := a.runner
		return a, func() tea.Msg {
			// Capture current state for undo before resetting
			curHash, _ := runner.HeadHash()
			curBranch, _ := runner.CurrentBranch()

			// Switch branch if needed
			if entry.Branch != curBranch {
				if err := runner.Checkout(entry.Branch); err != nil {
					return ErrorMsg{Err: fmt.Errorf("redo: checkout %s: %w", entry.Branch, err)}
				}
			}

			if err := runner.ResetHard(entry.Hash); err != nil {
				return ErrorMsg{Err: fmt.Errorf("redo: reset to %s: %w", entry.Hash[:7], err)}
			}

			return redoResultMsg{
				UndoEntry: domain.UndoEntry{
					Branch:      curBranch,
					Hash:        curHash,
					Description: entry.Description,
				},
				TargetHash: entry.Hash,
			}
		}
	}
	return a, nil
}

func (a App) overlayUndoConfirm(base string) string {
	if len(a.undoStack) == 0 {
		return base
	}
	entry := a.undoStack[len(a.undoStack)-1]
	return a.overlayUndoRedoDialog(base, "Undo", entry)
}

func (a App) overlayRedoConfirm(base string) string {
	if len(a.redoStack) == 0 {
		return base
	}
	entry := a.redoStack[len(a.redoStack)-1]
	return a.overlayUndoRedoDialog(base, "Redo", entry)
}

func (a App) overlayUndoRedoDialog(base, action string, entry domain.UndoEntry) string {
	dialogW := 55
	if a.width < 65 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	shortHash := entry.Hash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	content := titleStyle.Render(action) + "\n\n"
	content += "  " + entry.Description + "\n"
	content += "  Reset to " + shortHash + " on " + entry.Branch + "?\n\n"
	content += "  [y/Enter] confirm  " + dimStyle.Render("— hard reset to previous state") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
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
		return a.setFlashError("Nothing to commit")
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
		return a.setFlashError("Nothing staged to commit (use Space to stage tables)")
	}

	a.showCommit = true
	a.commitInput.Reset()
	a.commitInput.Focus()
	a.commitErr = ""
	return textinput.Blink
}

func (a *App) startAmend() tea.Cmd {
	if len(a.commits.Commits) == 0 {
		return a.setFlashError("No commits to amend")
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
			return a, a.undoableCmd("amend commit", func() tea.Msg {
				hash, err := runner.CommitAmend(message)
				if err != nil {
					return ErrorMsg{Err: err}
				}
				return CommitSuccessMsg{Hash: hash}
			})
		}
		return a, a.undoableCmd("commit: "+message, func() tea.Msg {
			hash, err := runner.Commit(message)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return CommitSuccessMsg{Hash: hash}
		})
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
	a.branchInput.Placeholder = "Enter branch name..."
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
		shortHash := hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		return a, a.undoableCmd("soft reset to "+shortHash, func() tea.Msg {
			if err := runner.ResetSoft(hash); err != nil {
				return ErrorMsg{Err: err}
			}
			return ResetSuccessMsg{Mode: "soft"}
		})
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
		shortHash := hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		return a, a.undoableCmd("hard reset to "+shortHash, func() tea.Msg {
			if err := runner.ResetHard(hash); err != nil {
				return ErrorMsg{Err: err}
			}
			return ResetSuccessMsg{Mode: "hard"}
		})
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
		return a, a.undoableCmd("merge "+branch, func() tea.Msg {
			if _, err := runner.Merge(branch); errors.Is(err, dolt.ErrMergeConflict) {
				return MergeConflictMsg{Branch: branch}
			} else if err != nil {
				return ErrorMsg{Err: err}
			}
			return MergeSuccessMsg{Branch: branch}
		})
	case "s":
		a.showMergeMenu = false
		a.mergeBranch = ""
		return a, a.undoableCmd("squash merge "+branch, func() tea.Msg {
			if _, err := runner.MergeSquash(branch); err != nil {
				return ErrorMsg{Err: err}
			}
			return MergeSuccessMsg{Branch: branch}
		})
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

// --- Rebase confirmation ---

func (a App) updateRebaseConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	branch := a.rebaseBranch
	runner := a.runner
	switch msg.String() {
	case "esc":
		a.showRebaseConfirm = false
		a.rebaseBranch = ""
		return a, nil
	case "y", "enter":
		a.showRebaseConfirm = false
		a.rebaseBranch = ""
		return a, a.undoableCmd("rebase onto "+branch, func() tea.Msg {
			if _, err := runner.Rebase(branch); errors.Is(err, dolt.ErrMergeConflict) {
				return RebaseConflictMsg{Branch: branch}
			} else if err != nil {
				return ErrorMsg{Err: err}
			}
			return RebaseSuccessMsg{Branch: branch}
		})
	}
	return a, nil
}

func (a App) overlayRebaseConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Rebase onto "+a.rebaseBranch) + "\n\n"
	content += "  Rebase current branch onto " + a.rebaseBranch + "?\n\n"
	content += "  [y/Enter] rebase  " + dimStyle.Render("— reapply commits on top") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Cherry-pick confirmation ---

func (a App) updateCherryPickConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hash := a.cherryPickHash
	runner := a.runner
	switch msg.String() {
	case "esc":
		a.showCherryPickConfirm = false
		a.cherryPickHash = ""
		return a, nil
	case "y", "enter":
		a.showCherryPickConfirm = false
		a.cherryPickHash = ""
		shortHash := hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		return a, a.undoableCmd("cherry-pick "+shortHash, func() tea.Msg {
			if _, err := runner.CherryPick(hash); errors.Is(err, dolt.ErrMergeConflict) {
				return CherryPickConflictMsg{Hash: hash}
			} else if err != nil {
				return ErrorMsg{Err: err}
			}
			return CherryPickSuccessMsg{Hash: hash}
		})
	}
	return a, nil
}

func (a App) overlayCherryPickConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	shortHash := a.cherryPickHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	content := titleStyle.Render("Cherry-pick "+shortHash) + "\n\n"
	content += "  Apply this commit onto the current branch?\n\n"
	content += "  [y/Enter] cherry-pick  " + dimStyle.Render("— apply commit changes") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Revert confirmation ---

func (a App) updateRevertConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hash := a.revertHash
	runner := a.runner
	switch msg.String() {
	case "esc":
		a.showRevertConfirm = false
		a.revertHash = ""
		return a, nil
	case "y", "enter":
		a.showRevertConfirm = false
		a.revertHash = ""
		shortHash := hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		return a, a.undoableCmd("revert "+shortHash, func() tea.Msg {
			if _, err := runner.Revert(hash); err != nil {
				return ErrorMsg{Err: err}
			}
			return RevertSuccessMsg{Hash: hash}
		})
	}
	return a, nil
}

func (a App) overlayRevertConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	shortHash := a.revertHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	content := titleStyle.Render("Revert "+shortHash) + "\n\n"
	content += "  Create a new commit undoing this commit's changes?\n\n"
	content += "  [y/Enter] revert  " + dimStyle.Render("— undo commit changes") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Tag dialog ---

func (a App) updateTagDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showTagDialog = false
		a.tagCommitHash = ""
		a.tagErr = ""
		return a, nil
	case "enter":
		name := strings.TrimSpace(a.branchInput.Value())
		if name == "" {
			a.tagErr = "Tag name cannot be empty"
			return a, nil
		}
		a.showTagDialog = false
		hash := a.tagCommitHash
		a.tagCommitHash = ""
		a.tagErr = ""
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.CreateTag(name, hash, ""); err != nil {
				return ErrorMsg{Err: err}
			}
			return TagSuccessMsg{Name: name}
		}
	}
	var cmd tea.Cmd
	a.branchInput, cmd = a.branchInput.Update(msg)
	return a, cmd
}

func (a App) overlayTagDialog(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	shortHash := a.tagCommitHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	content := titleStyle.Render("Create tag at "+shortHash) + "\n\n"
	content += "  Tag name:\n"
	content += "  " + a.branchInput.View() + "\n\n"
	if a.tagErr != "" {
		content += errorStyle.Render("  "+a.tagErr) + "\n\n"
	}
	content += dimStyle.Render("[Enter] create  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Delete tag confirmation ---

func (a App) updateDeleteTagConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showDeleteTagConfirm = false
		a.deleteTagName = ""
		return a, nil
	case "y", "enter":
		name := a.deleteTagName
		a.showDeleteTagConfirm = false
		a.deleteTagName = ""
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.DeleteTag(name); err != nil {
				return ErrorMsg{Err: err}
			}
			return DeleteTagSuccessMsg{Name: name}
		}
	}
	return a, nil
}

func (a App) overlayDeleteTagConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Delete tag "+a.deleteTagName) + "\n\n"
	content += "  Are you sure?\n\n"
	content += "  [y/Enter] delete  " + dimStyle.Render("— remove tag") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Add remote dialog ---

func (a *App) startAddRemote() tea.Cmd {
	a.showAddRemote = true
	a.addRemoteStep = 0
	a.addRemoteErr = ""
	a.branchInput.Reset()
	a.branchInput.Placeholder = "Enter remote name..."
	a.branchInput.Focus()
	a.remoteURLInput.Reset()
	a.remoteURLInput.Blur()
	return nil
}

func (a App) updateAddRemoteDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if a.addRemoteStep == 1 {
			// Go back to name step
			a.addRemoteStep = 0
			a.addRemoteErr = ""
			a.remoteURLInput.Blur()
			a.branchInput.Focus()
			return a, nil
		}
		a.showAddRemote = false
		a.addRemoteErr = ""
		return a, nil
	case "enter":
		if a.addRemoteStep == 0 {
			name := strings.TrimSpace(a.branchInput.Value())
			if name == "" {
				a.addRemoteErr = "Remote name cannot be empty"
				return a, nil
			}
			// Advance to URL step
			a.addRemoteStep = 1
			a.addRemoteErr = ""
			a.branchInput.Blur()
			a.remoteURLInput.Focus()
			return a, nil
		}
		// Step 1: submit
		name := strings.TrimSpace(a.branchInput.Value())
		url := strings.TrimSpace(a.remoteURLInput.Value())
		if url == "" {
			a.addRemoteErr = "Remote URL cannot be empty"
			return a, nil
		}
		a.showAddRemote = false
		a.addRemoteErr = ""
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.RemoteAdd(name, url); err != nil {
				return ErrorMsg{Err: err}
			}
			return AddRemoteSuccessMsg{Name: name}
		}
	}
	var cmd tea.Cmd
	if a.addRemoteStep == 0 {
		a.branchInput, cmd = a.branchInput.Update(msg)
	} else {
		a.remoteURLInput, cmd = a.remoteURLInput.Update(msg)
	}
	return a, cmd
}

func (a App) overlayAddRemoteDialog(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Add remote") + "\n\n"
	content += "  Remote name:\n"
	content += "  " + a.branchInput.View() + "\n\n"
	if a.addRemoteStep >= 1 {
		content += "  Remote URL:\n"
		content += "  " + a.remoteURLInput.View() + "\n\n"
	}
	if a.addRemoteErr != "" {
		content += errorStyle.Render("  "+a.addRemoteErr) + "\n\n"
	}
	if a.addRemoteStep == 0 {
		content += dimStyle.Render("[Enter] next  [Esc] cancel")
	} else {
		content += dimStyle.Render("[Enter] add  [Esc] back")
	}

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Delete remote confirmation ---

func (a App) updateDeleteRemoteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showDeleteRemoteConfirm = false
		a.deleteRemoteName = ""
		return a, nil
	case "y", "enter":
		name := a.deleteRemoteName
		a.showDeleteRemoteConfirm = false
		a.deleteRemoteName = ""
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.RemoteRemove(name); err != nil {
				return ErrorMsg{Err: err}
			}
			return DeleteRemoteSuccessMsg{Name: name}
		}
	}
	return a, nil
}

func (a App) overlayDeleteRemoteConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Remove remote "+a.deleteRemoteName) + "\n\n"
	content += "  Are you sure?\n\n"
	content += "  [y/Enter] remove  " + dimStyle.Render("— delete remote") + "\n\n"
	content += dimStyle.Render("[Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Table operations menu ---

var tableOpsItems = []string{"Rename", "Copy", "Drop", "Export CSV"}

func (a App) updateTableOpsMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showTableOpsMenu = false
		return a, nil
	case "j", "down":
		if a.tableOpsCursor < len(tableOpsItems)-1 {
			a.tableOpsCursor++
		}
		return a, nil
	case "k", "up":
		if a.tableOpsCursor > 0 {
			a.tableOpsCursor--
		}
		return a, nil
	case "enter", " ":
		a.showTableOpsMenu = false
		switch a.tableOpsCursor {
		case 0: // Rename
			a.showTableRename = true
			a.tableOpsErr = ""
			a.branchInput.Reset()
			a.branchInput.Placeholder = "Enter new table name..."
			a.branchInput.SetValue(a.tableOpsTable)
			a.branchInput.Focus()
			a.branchInput.CursorEnd()
		case 1: // Copy
			a.showTableCopy = true
			a.tableOpsErr = ""
			a.branchInput.Reset()
			a.branchInput.Placeholder = "Enter copy name..."
			a.branchInput.SetValue(a.tableOpsTable + "_copy")
			a.branchInput.Focus()
			a.branchInput.CursorEnd()
		case 2: // Drop
			a.showTableDrop = true
		case 3: // Export
			a.showTableExport = true
			a.tableOpsErr = ""
			a.branchInput.Reset()
			a.branchInput.Placeholder = "Enter file path..."
			a.branchInput.SetValue(a.tableOpsTable + ".csv")
			a.branchInput.Focus()
			a.branchInput.CursorEnd()
		}
		return a, nil
	}
	return a, nil
}

func (a App) overlayTableOpsMenu(base string) string {
	dialogW := 40
	if a.width < 50 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	menuSelectedStyle := lipgloss.NewStyle().Reverse(true)

	content := titleStyle.Render("Table: "+a.tableOpsTable) + "\n\n"
	for i, item := range tableOpsItems {
		prefix := "  "
		if i == a.tableOpsCursor {
			prefix = "> "
			content += menuSelectedStyle.Render(prefix+item) + "\n"
		} else {
			content += prefix + item + "\n"
		}
	}
	content += "\n" + dimStyle.Render("[Enter] select  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Sort menu for branches and commits ---

// sortMenuItems returns the labels for the current sort menu context.
func (a App) sortMenuItems() []string {
	if a.focused == components.PanelBranches {
		items := make([]string, branchSortCount)
		for i := range items {
			items[i] = branchSortLabels[i]
		}
		return items
	}
	items := make([]string, commitSortCount)
	for i := range items {
		items[i] = commitSortLabels[i]
	}
	return items
}

// sortMenuTitle returns the title for the sort menu overlay.
func (a App) sortMenuTitle() string {
	if a.focused == components.PanelBranches {
		return "Sort Branches"
	}
	return "Sort Commits"
}

func (a App) updateSortMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := a.sortMenuItems()
	switch msg.String() {
	case "esc":
		a.showSortMenu = false
		return a, nil
	case "j", "down":
		if a.sortMenuCursor < len(items)-1 {
			a.sortMenuCursor++
		}
		return a, nil
	case "k", "up":
		if a.sortMenuCursor > 0 {
			a.sortMenuCursor--
		}
		return a, nil
	case "enter", " ":
		a.showSortMenu = false
		if a.focused == components.PanelBranches {
			a.branchSort = BranchSort(a.sortMenuCursor)
		} else {
			a.commitSort = CommitSort(a.sortMenuCursor)
		}
		return a, a.loadData()
	}
	return a, nil
}

func (a App) overlaySortMenu(base string) string {
	dialogW := 40
	if a.width < 50 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	menuSelectedStyle := lipgloss.NewStyle().Reverse(true)

	items := a.sortMenuItems()
	content := titleStyle.Render(a.sortMenuTitle()) + "\n\n"
	for i, item := range items {
		prefix := "  "
		if i == a.sortMenuCursor {
			prefix = "> "
			content += menuSelectedStyle.Render(prefix+item) + "\n"
		} else {
			content += prefix + item + "\n"
		}
	}
	content += "\n" + dimStyle.Render("[Enter] select  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Config viewer ---

func (a *App) loadConfig() tea.Cmd {
	runner := a.runner
	return func() tea.Msg {
		global, local, err := runner.Config()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ConfigLoadedMsg{Global: global, Local: local}
	}
}

func (a App) updateConfigViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "@":
		a.showConfig = false
		return a, nil
	default:
		var cmd tea.Cmd
		a.configViewport, cmd = a.configViewport.Update(msg)
		return a, cmd
	}
}

func (a App) overlayConfigViewer(base string) string {
	dialogW := a.width - 10
	if dialogW > 80 {
		dialogW = 80
	}
	if dialogW < 30 {
		dialogW = a.width - 4
	}
	dialogH := a.height - 6
	if dialogH < 10 {
		dialogH = a.height - 2
	}

	a.configViewport.Width = dialogW - 4 // border + padding
	a.configViewport.Height = dialogH - 4
	a.configViewport.SetContent(a.configContent)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Dolt Configuration") + "\n\n"
	content += a.configViewport.View()
	content += "\n" + dimStyle.Render("[Esc/q/@] close  [j/k] scroll")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// formatConfigContent builds a display string from config entries.
func formatConfigContent(global, local []dolt.ConfigEntry) string {
	var b strings.Builder

	if len(global) > 0 {
		b.WriteString("── Global ──\n\n")
		maxKeyLen := 0
		for _, e := range global {
			if len(e.Key) > maxKeyLen {
				maxKeyLen = len(e.Key)
			}
		}
		for _, e := range global {
			b.WriteString(fmt.Sprintf("  %-*s = %s\n", maxKeyLen, e.Key, e.Value))
		}
	}

	if len(local) > 0 {
		if len(global) > 0 {
			b.WriteString("\n")
		}
		b.WriteString("── Local ──\n\n")
		maxKeyLen := 0
		for _, e := range local {
			if len(e.Key) > maxKeyLen {
				maxKeyLen = len(e.Key)
			}
		}
		for _, e := range local {
			b.WriteString(fmt.Sprintf("  %-*s = %s\n", maxKeyLen, e.Key, e.Value))
		}
	}

	if len(global) == 0 && len(local) == 0 {
		b.WriteString("  No configuration found.\n")
	}

	return b.String()
}

// --- Table rename/copy/export input dialog ---

func (a App) updateTableInputDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showTableRename = false
		a.showTableCopy = false
		a.showTableExport = false
		a.tableOpsErr = ""
		return a, nil
	case "enter":
		name := strings.TrimSpace(a.branchInput.Value())
		if name == "" {
			a.tableOpsErr = "Name cannot be empty"
			return a, nil
		}
		table := a.tableOpsTable
		runner := a.runner

		if a.showTableRename {
			a.showTableRename = false
			a.tableOpsErr = ""
			return a, func() tea.Msg {
				if err := runner.TableRename(table, name); err != nil {
					return ErrorMsg{Err: err}
				}
				return TableRenameSuccessMsg{OldName: table, NewName: name}
			}
		}
		if a.showTableCopy {
			a.showTableCopy = false
			a.tableOpsErr = ""
			return a, func() tea.Msg {
				if err := runner.TableCopy(table, name); err != nil {
					return ErrorMsg{Err: err}
				}
				return TableCopySuccessMsg{SrcName: table, DstName: name}
			}
		}
		if a.showTableExport {
			a.showTableExport = false
			a.tableOpsErr = ""
			return a, func() tea.Msg {
				if err := runner.TableExport(table, name); err != nil {
					return ErrorMsg{Err: err}
				}
				return TableExportSuccessMsg{Table: table, Path: name}
			}
		}
	}
	var cmd tea.Cmd
	a.branchInput, cmd = a.branchInput.Update(msg)
	return a, cmd
}

func (a App) overlayTableInputDialog(base string) string {
	dialogW := 60
	if a.width < 70 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var title, label string
	if a.showTableRename {
		title = "Rename table " + a.tableOpsTable
		label = "New name:"
	} else if a.showTableCopy {
		title = "Copy table " + a.tableOpsTable
		label = "Copy name:"
	} else {
		title = "Export table " + a.tableOpsTable
		label = "File path:"
	}

	content := titleStyle.Render(title) + "\n\n"
	content += "  " + label + "\n"
	content += "  " + a.branchInput.View() + "\n\n"
	if a.tableOpsErr != "" {
		content += errorStyle.Render("  "+a.tableOpsErr) + "\n\n"
	}
	content += dimStyle.Render("[Enter] confirm  [Esc] cancel")

	dialog := commitBoxStyle.Width(dialogW).Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// --- Table drop confirmation ---

func (a App) updateTableDropConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showTableDrop = false
		return a, nil
	case "y", "enter":
		table := a.tableOpsTable
		a.showTableDrop = false
		runner := a.runner
		return a, func() tea.Msg {
			if err := runner.TableDrop(table); err != nil {
				return ErrorMsg{Err: err}
			}
			return TableDropSuccessMsg{Name: table}
		}
	}
	return a, nil
}

func (a App) overlayTableDropConfirm(base string) string {
	dialogW := 50
	if a.width < 60 {
		dialogW = a.width - 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	content := titleStyle.Render("Drop table "+a.tableOpsTable) + "\n\n"
	content += "  Are you sure? This removes the table from the working set.\n\n"
	content += "  [y/Enter] drop  " + dimStyle.Render("— remove table") + "\n\n"
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

// --- Commit detail tables ---

// renderCommitDetailTables renders the list of changed tables for commit
// detail mode, shown in place of the normal tables panel.
func (a App) renderCommitDetailTables(height int) string {
	if len(a.commitDetailTables) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("  No tables changed")
	}

	selectedStyle := lipgloss.NewStyle().Reverse(true)
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var lines []string
	for i, t := range a.commitDetailTables {
		// Build stat summary: +N -N ~N
		var stats []string
		if t.RowsAdded > 0 {
			stats = append(stats, fmt.Sprintf("+%d", t.RowsAdded))
		}
		if t.RowsDeleted > 0 {
			stats = append(stats, fmt.Sprintf("-%d", t.RowsDeleted))
		}
		if t.RowsModified > 0 {
			stats = append(stats, fmt.Sprintf("~%d", t.RowsModified))
		}
		stat := ""
		if len(stats) > 0 {
			stat = " " + statStyle.Render(strings.Join(stats, " "))
		}

		line := fmt.Sprintf("  %s%s", t.TableName, stat)
		if i == a.commitDetailCursor {
			// Apply reverse to the table name only, keep stat styling
			line = fmt.Sprintf("  %s%s", selectedStyle.Render(t.TableName), stat)
		}
		lines = append(lines, line)
	}

	// Truncate to height
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// --- Mouse handling ---

// handleMouse processes mouse events for click-to-focus, item selection,
// and scroll wheel support.
func (a App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Ignore motion-only events (no button pressed)
	if msg.Action == tea.MouseActionMotion {
		return a, nil
	}

	// Only process presses (including wheel events)
	if msg.Action != tea.MouseActionPress {
		return a, nil
	}

	// Determine which panel the mouse is in and handle accordingly
	panel, innerRow := a.mousePanelHit(msg.X, msg.Y)
	if panel < 0 {
		return a, nil
	}

	// Wheel events: scroll the targeted panel
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return a.handleMouseWheel(panel, msg)
	}

	// Left click: focus panel and optionally select item
	if msg.Button == tea.MouseButtonLeft {
		return a.handleMouseClick(panel, innerRow)
	}

	return a, nil
}

// mousePanelHit determines which panel a mouse coordinate hits.
// Returns the panel and the row offset within the panel's inner content area.
// Returns panel=-1 if the click is not in any panel.
func (a App) mousePanelHit(x, y int) (panel components.Panel, innerRow int) {
	if a.screenMode == ScreenHalf {
		return a.mousePanelHitHalf(x, y)
	}
	if a.screenMode == ScreenFullscreen {
		return a.mousePanelHitFullscreen(x, y)
	}
	return a.mousePanelHitNormal(x, y)
}

// mousePanelHitNormal handles the default two-column layout.
func (a App) mousePanelHitNormal(x, y int) (components.Panel, int) {
	leftW := a.leftColumnWidth()
	const borderH = 2

	statusInnerH := a.statusBar.Lines()

	// Compute panel heights (same logic as View)
	availForPanels := a.height - 1 - (statusInnerH + borderH) - 3*borderH
	var tablesH, branchesH, commitsH int
	if a.focused == components.PanelMain {
		each := availForPanels / 3
		remainder := availForPanels - 3*each
		tablesH = each + remainder
		branchesH = each
		commitsH = each
	} else {
		focusedH, unfocusedH := a.panelHeights(availForPanels)
		heightFor := func(p components.Panel) int {
			if p == a.focused {
				return focusedH
			}
			return unfocusedH
		}
		tablesH = heightFor(components.PanelTables)
		branchesH = heightFor(components.PanelBranches)
		commitsH = heightFor(components.PanelCommits)
	}

	if x < leftW {
		// Left column: status, tables, branches, commits stacked vertically
		statusEnd := statusInnerH + borderH // outer height of status box
		tablesEnd := statusEnd + tablesH + borderH
		branchesEnd := tablesEnd + branchesH + borderH
		commitsEnd := branchesEnd + commitsH + borderH
		_ = commitsEnd

		switch {
		case y < statusEnd:
			return -1, 0 // Status box, not a focusable panel
		case y < tablesEnd:
			return components.PanelTables, y - statusEnd - 1 // -1 for top border
		case y < branchesEnd:
			return components.PanelBranches, y - tablesEnd - 1
		case y < commitsEnd:
			return components.PanelCommits, y - branchesEnd - 1
		default:
			return -1, 0 // hints bar or beyond
		}
	}

	// Right column
	return components.PanelMain, y - 1 // -1 for top border
}

// mousePanelHitHalf handles the ScreenHalf vertical split layout.
func (a App) mousePanelHitHalf(x, y int) (components.Panel, int) {
	topH := (a.height - 1) / 3 // same as View() calculation
	if y < topH {
		return a.focused, y - 1
	}
	return components.PanelMain, y - topH - 1
}

// mousePanelHitFullscreen handles the ScreenFullscreen layout.
func (a App) mousePanelHitFullscreen(x, y int) (components.Panel, int) {
	// In fullscreen, only left panels are visible
	return a.focused, y - 1
}

// handleMouseWheel handles scroll wheel events on a given panel.
func (a App) handleMouseWheel(panel components.Panel, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	delta := 3 // lines per wheel tick
	if msg.Button == tea.MouseButtonWheelUp {
		delta = -delta
	}

	switch panel {
	case components.PanelTables:
		for range abs(delta) {
			if delta < 0 {
				a.tables.MoveUp()
			} else {
				a.tables.MoveDown()
			}
		}
		a.schemaDiff = false
		a.diffStat = false
		a.blameMode = false
		return a, a.autoPreview()

	case components.PanelBranches:
		for range abs(delta) {
			if delta < 0 {
				a.branches.MoveUp()
			} else {
				a.branches.MoveDown()
			}
		}
		return a, nil

	case components.PanelCommits:
		for range abs(delta) {
			if delta < 0 {
				a.commits.MoveUp()
			} else {
				a.commits.MoveDown()
			}
		}
		a.schemaDiff = false
		a.diffStat = false
		return a, a.autoPreview()

	case components.PanelMain:
		// Forward to the active viewport
		var cmd tea.Cmd
		switch a.mainView {
		case MainViewDiff:
			a.diffView, cmd = a.diffView.Update(msg)
		case MainViewSchema:
			a.schemaView, cmd = a.schemaView.Update(msg)
		case MainViewBrowser:
			a.browserView, cmd = a.browserView.Update(msg)
		}
		return a, cmd
	}
	return a, nil
}

// handleMouseClick handles left-click for focus and item selection.
func (a App) handleMouseClick(panel components.Panel, innerRow int) (tea.Model, tea.Cmd) {
	// Focus the clicked panel
	prevFocused := a.focused
	a.setFocus(panel)

	var cmd tea.Cmd
	switch panel {
	case components.PanelTables:
		if innerRow >= 0 {
			a.tables.ClickRow(innerRow)
		}
		if prevFocused != panel {
			a.schemaDiff = false
			a.diffStat = false
			a.blameMode = false
			cmd = a.autoPreview()
		}
	case components.PanelBranches:
		if innerRow >= 0 {
			a.branches.ClickRow(innerRow)
		}
	case components.PanelCommits:
		if innerRow >= 0 {
			a.commits.ClickRow(innerRow)
		}
		if prevFocused != panel {
			a.schemaDiff = false
			a.diffStat = false
			cmd = a.autoPreview()
		}
	case components.PanelMain:
		// Click on main panel just focuses it
	}
	return a, cmd
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
	{"Global", "z", "Undo last operation"},
	{"Global", "Z", "Redo undone operation"},
	{"Global", "y", "Copy selected item to clipboard"},
	{"Global", "@", "View dolt configuration"},
	{"Global", "Ctrl+L", "Redraw screen"},
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
	{"Tables Panel", "w", "Toggle diff statistics"},
	{"Tables Panel", "b", "View blame"},
	{"Tables Panel", "o", "Table operations (rename, copy, drop, export)"},
	{"Tables Panel", "Enter", "Browse table data"},
	{"Branches Panel", "j/k", "Navigate"},
	{"Branches Panel", "Enter", "View branch commits"},
	{"Branches Panel", "Space", "Checkout branch"},
	{"Branches Panel", "W", "Diff against current branch"},
	{"Branches Panel", "m", "Merge into current"},
	{"Branches Panel", "e", "Rebase onto branch"},
	{"Branches Panel", "n", "New branch"},
	{"Branches Panel", "r", "Rename branch"},
	{"Branches Panel", "s", "Sort branches"},
	{"Branches Panel", "D", "Delete branch/tag/remote"},
	{"Branches Panel", "a", "Add remote"},
	{"Commits Panel", "j/k", "Navigate"},
	{"Commits Panel", "Enter", "View commit details"},
	{"Commits Panel", "s", "Sort commits"},
	{"Commits Panel", "A", "Amend last commit"},
	{"Commits Panel", "g", "Reset to commit"},
	{"Commits Panel", "C", "Cherry-pick commit"},
	{"Commits Panel", "t", "Revert commit"},
	{"Commits Panel", "T", "Create tag at commit"},
	{"Commits Panel", "l", "View reflog"},
	{"Main Panel", "j/k", "Scroll up/down"},
	{"Main Panel", "PgUp/PgDn", "Page up/down"},
	{"Main Panel", "u/d", "Half page up/down"},
	{"Main Panel", "H/L", "Scroll left/right"},
	{"Main Panel", "s", "Toggle schema diff"},
	{"Main Panel", "w", "Toggle diff statistics"},
}

// helpSection groups filtered bindings by section name.
type helpSection struct {
	Name     string
	Bindings []struct{ Key, Desc string }
}

// filteredHelpSections returns help bindings grouped by section,
// filtered by the current help filter string.
func filteredHelpSections(filter string) ([]helpSection, int) {
	var sections []helpSection
	matchCount := 0
	sectionMap := make(map[string]int) // section name → index in sections

	for _, b := range helpBindings {
		if filter != "" {
			combined := strings.ToLower(b.Key + " " + b.Desc + " " + b.Section)
			if !strings.Contains(combined, filter) {
				continue
			}
		}
		matchCount++

		idx, ok := sectionMap[b.Section]
		if !ok {
			idx = len(sections)
			sectionMap[b.Section] = idx
			sections = append(sections, helpSection{Name: b.Section})
		}
		sections[idx].Bindings = append(sections[idx].Bindings,
			struct{ Key, Desc string }{b.Key, b.Desc})
	}
	return sections, matchCount
}

// renderHelpSection renders one section block as a string.
func renderHelpSection(s helpSection) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(s.Name))
	sb.WriteString("\n")
	for _, b := range s.Bindings {
		sb.WriteString(fmt.Sprintf("  %-14s%s\n", b.Key, b.Desc))
	}
	return sb.String()
}

// sectionHeight returns the number of lines a section takes (header + bindings).
func sectionHeight(s helpSection) int {
	return 1 + len(s.Bindings) // header line + one line per binding
}

// renderHelpColumns lays out sections across columns, balancing height.
func renderHelpColumns(sections []helpSection, colCount, colWidth int) string {
	if len(sections) == 0 || colCount <= 0 {
		return ""
	}

	// Calculate total height across all sections (including gaps between them).
	totalH := 0
	for _, s := range sections {
		totalH += sectionHeight(s) + 1 // +1 for blank line between sections
	}
	targetH := (totalH + colCount - 1) / colCount

	// Distribute sections across columns greedily.
	columns := make([][]helpSection, colCount)
	colHeights := make([]int, colCount)
	col := 0
	for _, s := range sections {
		sh := sectionHeight(s) + 1
		if col < colCount-1 && colHeights[col] > 0 && colHeights[col]+sh > targetH+2 {
			col++
		}
		columns[col] = append(columns[col], s)
		colHeights[col] += sh
	}

	// Render each column.
	colStyle := lipgloss.NewStyle().Width(colWidth)
	rendered := make([]string, colCount)
	for i, colSections := range columns {
		var parts []string
		for _, s := range colSections {
			parts = append(parts, renderHelpSection(s))
		}
		rendered[i] = colStyle.Render(strings.Join(parts, "\n"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// helpLayout computes dialog and column dimensions for the help overlay.
func (a App) helpLayout() (dialogW, innerW, colCount, colWidth int) {
	dialogW = a.width - 6
	if dialogW < 40 {
		dialogW = 40
	}
	if dialogW > a.width-4 {
		dialogW = a.width - 4
	}
	// Inner width after box padding (2 chars each side) and border (1 each side).
	innerW = dialogW - 6

	colCount = 1
	if innerW >= 110 {
		colCount = 3
	} else if innerW >= 60 {
		colCount = 2
	}
	colWidth = innerW / colCount
	return
}

// helpViewportHeight returns the available height for the help viewport.
func (a App) helpViewportHeight() int {
	// header: title + blank + filter + blank = 4 lines
	// footer: blank + hint = 2 lines
	// box: border=2 rows, padding=2 rows
	overhead := 2 + 2 + 4 + 2
	vpH := (a.height - 4) - overhead
	if vpH < 3 {
		vpH = 3
	}
	return vpH
}

// syncHelpViewport rebuilds the help viewport content from current filter.
// Must be called from Update (pointer receiver context).
func (a *App) syncHelpViewport() {
	filter := strings.ToLower(strings.TrimSpace(a.helpFilter.Value()))
	_, innerW, colCount, colWidth := a.helpLayout()
	sections, matchCount := filteredHelpSections(filter)

	var body string
	if filter != "" && matchCount == 0 {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("  No matches") + "\n"
	} else {
		body = renderHelpColumns(sections, colCount, colWidth)
	}

	a.helpViewport.Width = innerW
	a.helpViewport.Height = a.helpViewportHeight()
	a.helpViewport.SetContent(body)
}

func (a App) renderHelp() string {
	dialogW, _, _, _ := a.helpLayout()

	// Header: title + filter input.
	header := titleStyle.Render("lazydolt - Keyboard Shortcuts") + "\n\n" +
		a.helpFilter.View() + "\n\n"

	// Footer.
	scrollHint := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).
		Render("j/k scroll · PgUp/PgDn page · ? or Esc to close")

	// Assemble: header + viewport + footer.
	content := header + a.helpViewport.View() + "\n\n" + scrollHint

	box := commitBoxStyle.Width(dialogW).Render(content)
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
type viewBranchMsg = components.ViewBranchMsg
type deleteBranchMsg = components.DeleteBranchMsg
type deleteTagMsg = components.DeleteTagMsg
type newBranchPromptMsg = components.NewBranchPromptMsg
type addRemotePromptMsg = components.AddRemotePromptMsg
type deleteRemoteMsg = components.DeleteRemoteMsg
type browserPageMsg = components.BrowserPageMsg
