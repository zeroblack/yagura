package tui

import (
	"context"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/forge"
	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/scan"
	"github.com/zeroblack/yagura/internal/theme"
)

type rowKind int

const (
	rowGroup rowKind = iota
	rowCombined
	rowWorktree
	rowAgent
)

type row struct {
	kind     rowKind
	repo     *model.Repo
	worktree *model.Worktree
	agentIdx int
	lastSub  bool
	nested   bool
}

type appModel struct {
	cfg           config.Config
	styles        styleSet
	fx            fxSet
	keymap        map[string]keyAction
	footerLine    string
	scanner       *scan.Scanner
	gitSvc        *git.Service
	home          string
	snap          scan.Snapshot
	rows          []row
	cursor        int
	cursorPath    string
	pinnedPath    string
	grouped       bool
	width         int
	height        int
	detail        detailMode
	detailContent string
	detailLoading bool
	inspect       inspectState
	inspectGen    int
	focus         paneFocus
	changesOff    int
	graphOff      int
	prsOff        int
	listOff       int
	forge         *forge.Manager
	scanInFlight  bool
	lastSync      time.Time
	lastFullSync  time.Time
	beat          int
	err           error
}

func New(cfg config.Config) *appModel {
	t := theme.ByName(cfg.Theme)
	styles := newStyleSet(t)
	fx := newFxSet(t, fxConfigFrom(cfg.FX), richColorTerminal())
	scanner := scan.NewScanner(cfg)
	home, _ := os.UserHomeDir()
	return &appModel{
		cfg:        cfg,
		styles:     styles,
		fx:         fx,
		keymap:     buildKeymap(cfg.Keys),
		footerLine: buildFooter(styles, cfg.Keys),
		scanner:    scanner,
		gitSvc:     scanner.Git(),
		home:       home,
		grouped:    cfg.Sort.GroupByRepo,
		forge:      forge.NewManager(cfg.Forge.Enabled, cfg.Forge.TTL),
	}
}

const uiTickInterval = 250 * time.Millisecond

func (m *appModel) Init() tea.Cmd {
	return tea.Batch(m.forceRefresh(), tick(uiTickInterval))
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.fx.resize(m.width)
	case tea.KeyPressMsg:
		return m, m.handleKey(msg.String())
	case snapshotMsg:
		m.scanInFlight = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if msg.full {
			m.lastFullSync = time.Now()
		}
		return m, m.applySnapshot(msg.snap)
	case detailMsg:
		if m.detail == msg.mode && m.selectedWorktreePath() == msg.wtPath {
			m.detailContent = msg.content
			m.detailLoading = false
		}
	case inspectSettleMsg:
		if msg.gen == m.inspectGen && m.inspect.wt != nil {
			return m, tea.Batch(m.loadInspectData()...)
		}
	case inspectStatusMsg:
		if m.inspect.wt != nil && msg.wtPath == m.inspect.wt.Path {
			m.inspect.status = msg.status
			m.inspect.statusErr = msg.err
			m.inspect.statusLoaded = true
			m.inspect.sorted = sortChanges(msg.status.Files, m.cfg.Inspect.ChangesSort)
		}
	case inspectGraphMsg:
		if m.inspect.wt != nil && msg.wtPath == m.inspect.wt.Path {
			m.inspect.graph = msg.lines
			m.inspect.div = msg.div
			m.inspect.graphLoaded = true
		}
	case inspectPRMsg:
		if m.inspect.wt != nil && msg.wtPath == m.inspect.wt.Path {
			m.inspect.prList = msg.prs
			m.inspect.prByBranch = prsByBranch(msg.prs)
			m.prsOff = 0
		}
	case inspectStatsMsg:
		if m.inspect.repo != nil && msg.repoPath == m.inspect.repo.Path {
			m.inspect.stats = msg.stats
			m.inspect.statsLoaded = true
		}
	case tickMsg:
		m.beat++
		cmds := []tea.Cmd{tick(uiTickInterval)}
		if !m.scanInFlight {
			switch {
			case time.Since(m.lastFullSync) >= m.cfg.Refresh.FullTick:
				m.scanInFlight = true
				cmds = append(cmds, m.loadSnapshot())
			case time.Since(m.lastSync) >= m.cfg.Refresh.Tick:
				m.scanInFlight = true
				cmds = append(cmds, m.loadAgents())
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m *appModel) forceRefresh() tea.Cmd {
	if m.scanInFlight {
		return nil
	}
	m.scanInFlight = true
	return m.loadSnapshot()
}

func (m *appModel) applySnapshot(snap scan.Snapshot) tea.Cmd {
	m.snap = snap
	m.lastSync = time.Now()
	m.prunePin()
	m.rebuildRows()
	m.rebindCursor()
	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
	if m.cursorPath == "" {
		m.cursor = m.firstFocusableRow()
		m.syncCursorPath()
	}
	cmds := []tea.Cmd{m.refreshInspect()}
	if m.detail != detailNone {
		cmds = append(cmds, m.loadDetailCmd(m.detail))
	}
	return tea.Batch(cmds...)
}

func (m *appModel) rebuildRows() {
	m.rows = m.rows[:0]
	for _, ri := range m.orderedRepoIndices() {
		r := &m.snap.Repos[ri]
		switch {
		case len(r.Worktrees) == 0:
			m.rows = append(m.rows, row{kind: rowGroup, repo: r})
		case m.grouped && len(r.Worktrees) >= 2:
			m.rows = append(m.rows, row{kind: rowGroup, repo: r})
			for _, j := range m.orderedWorktreeIndices(r) {
				w := &r.Worktrees[j]
				m.rows = append(m.rows, row{kind: rowWorktree, repo: r, worktree: w, nested: true})
				m.appendAgentRows(r, w, true)
			}
		case m.grouped:
			w := &r.Worktrees[0]
			m.rows = append(m.rows, row{kind: rowCombined, repo: r, worktree: w})
			m.appendAgentRows(r, w, false)
		default:
			for _, j := range m.orderedWorktreeIndices(r) {
				w := &r.Worktrees[j]
				m.rows = append(m.rows, row{kind: rowCombined, repo: r, worktree: w})
				m.appendAgentRows(r, w, false)
			}
		}
	}
}

// orderedRepoIndices returns repo indices with the pinned worktree's repo first,
// so a pinned focus stays anchored at the top of the list regardless of the
// activity sort applied to the snapshot.
func (m *appModel) orderedRepoIndices() []int {
	idx := make([]int, len(m.snap.Repos))
	for i := range idx {
		idx[i] = i
	}
	if m.pinnedPath == "" {
		return idx
	}
	pinned := -1
	for i := range m.snap.Repos {
		if m.repoHasPath(&m.snap.Repos[i], m.pinnedPath) {
			pinned = i
			break
		}
	}
	if pinned <= 0 {
		return idx
	}
	out := make([]int, 0, len(idx))
	out = append(out, pinned)
	for _, i := range idx {
		if i != pinned {
			out = append(out, i)
		}
	}
	return out
}

func (m *appModel) orderedWorktreeIndices(r *model.Repo) []int {
	idx := make([]int, len(r.Worktrees))
	for i := range idx {
		idx[i] = i
	}
	if m.pinnedPath == "" {
		return idx
	}
	for k := range r.Worktrees {
		if r.Worktrees[k].Path == m.pinnedPath {
			if k == 0 {
				return idx
			}
			out := make([]int, 0, len(idx))
			out = append(out, k)
			for _, i := range idx {
				if i != k {
					out = append(out, i)
				}
			}
			return out
		}
	}
	return idx
}

func (m *appModel) repoHasPath(r *model.Repo, path string) bool {
	for j := range r.Worktrees {
		if r.Worktrees[j].Path == path {
			return true
		}
	}
	return false
}

func (r row) focusable() bool {
	return r.worktree != nil && (r.kind == rowWorktree || r.kind == rowCombined)
}

func (m *appModel) firstFocusableRow() int {
	for i, r := range m.rows {
		if r.focusable() {
			return i
		}
	}
	return 0
}

// stepCursor moves the selection to the next focusable worktree row in the given
// direction, skipping group headers and agent sub-rows so the focus jumps cleanly
// worktree-to-worktree.
func (m *appModel) stepCursor(dir int) {
	for i := m.cursor + dir; i >= 0 && i < len(m.rows); i += dir {
		if m.rows[i].focusable() {
			m.cursor = i
			return
		}
	}
}

// rebindCursor keeps the focus on the same worktree by identity after a rebuild,
// so the activity sort never drags the cursor onto a different worktree.
func (m *appModel) rebindCursor() {
	if m.cursorPath == "" {
		return
	}
	for i, r := range m.rows {
		if r.worktree != nil && r.worktree.Path == m.cursorPath {
			m.cursor = i
			return
		}
	}
}

func (m *appModel) syncCursorPath() {
	if w := m.selectedWorktree(); w != nil {
		m.cursorPath = w.Path
	}
}

func (m *appModel) prunePin() {
	if m.pinnedPath == "" {
		return
	}
	for i := range m.snap.Repos {
		if m.repoHasPath(&m.snap.Repos[i], m.pinnedPath) {
			return
		}
	}
	m.pinnedPath = ""
}

func (m *appModel) togglePin() tea.Cmd {
	w := m.selectedWorktree()
	if w == nil {
		return nil
	}
	if m.pinnedPath == w.Path {
		m.pinnedPath = ""
	} else {
		m.pinnedPath = w.Path
	}
	m.cursorPath = w.Path
	m.rebuildRows()
	m.rebindCursor()
	return m.refreshInspect()
}

func (m *appModel) appendAgentRows(r *model.Repo, w *model.Worktree, nested bool) {
	for k := range w.Agents {
		m.rows = append(m.rows, row{kind: rowAgent, repo: r, worktree: w, agentIdx: k, lastSub: k == len(w.Agents)-1, nested: nested})
	}
}

func (m *appModel) toggleGrouping() {
	m.grouped = !m.grouped
	m.rebuildRows()
}

func (m *appModel) repoCount() int     { return len(m.snap.Repos) }
func (m *appModel) worktreeCount() int { return countWorktrees(m.snap.Repos) }

func countWorktrees(repos []model.Repo) int {
	n := 0
	for _, r := range repos {
		n += len(r.Worktrees)
	}
	return n
}

func Run(ctx context.Context, cfg config.Config) error {
	p := tea.NewProgram(New(cfg), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
