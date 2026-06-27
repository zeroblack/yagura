package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/zeroblack/yagura/internal/forge"
	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
)

// A short settle delay coalesces rapid cursor movement into a single round
// of git subprocess loads instead of one burst per keypress.
const inspectSettleDelay = 100 * time.Millisecond

type paneFocus int

const (
	focusList paneFocus = iota
	focusChanges
	focusGraph
	focusPRs
)

func (p paneFocus) String() string {
	switch p {
	case focusChanges:
		return "CHANGES"
	case focusGraph:
		return "GRAPH"
	case focusPRs:
		return "PRS"
	default:
		return "WORKTREES"
	}
}

type inspectState struct {
	repo         *model.Repo
	wt           *model.Worktree
	status       model.StatusResult
	statusErr    error
	statusLoaded bool
	sorted       []model.FileChange
	graph        []model.GraphLine
	graphLoaded  bool
	div          map[string]model.Divergence
	prList       []model.PRInfo
	prByBranch   map[string]model.PRInfo
	stats        model.RepoStats
	statsLoaded  bool
}

type inspectStatsMsg struct {
	repoPath string
	stats    model.RepoStats
}

func loadInspectStats(svc *git.Service, repoPath string) tea.Cmd {
	return func() tea.Msg {
		st, _ := svc.RepoStats(context.Background(), repoPath)
		return inspectStatsMsg{repoPath: repoPath, stats: st}
	}
}

type inspectStatusMsg struct {
	wtPath string
	status model.StatusResult
	err    error
}

func loadInspectStatus(svc *git.Service, wtPath string) tea.Cmd {
	return func() tea.Msg {
		st, err := svc.Status(context.Background(), wtPath)
		return inspectStatusMsg{wtPath: wtPath, status: st, err: err}
	}
}

type inspectGraphMsg struct {
	wtPath string
	lines  []model.GraphLine
	div    map[string]model.Divergence
}

type inspectPRMsg struct {
	wtPath string
	prs    []model.PRInfo
}

func loadInspectPRs(man *forge.Manager, wtPath, repoPath string) tea.Cmd {
	return func() tea.Msg {
		return inspectPRMsg{wtPath: wtPath, prs: man.PRs(context.Background(), repoPath)}
	}
}

func prsByBranch(prs []model.PRInfo) map[string]model.PRInfo {
	out := make(map[string]model.PRInfo, len(prs))
	for _, pr := range prs {
		out[pr.Branch] = pr
	}
	return out
}

func loadInspectGraph(svc *git.Service, wtPath, repoPath string, branches []string, max int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		lines, _ := svc.Graph(ctx, repoPath, branches, max)
		base := defaultBase(branches)
		div := map[string]model.Divergence{}
		for _, b := range branches {
			if b == base {
				continue
			}
			if d, err := svc.Divergence(ctx, repoPath, base, b); err == nil {
				div[b] = d
			}
		}
		return inspectGraphMsg{wtPath: wtPath, lines: lines, div: div}
	}
}

func branchesForGraph(repo *model.Repo) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range repo.Worktrees {
		if w.Branch == "" || w.Detached || w.Branch == "(detached)" || seen[w.Branch] {
			continue
		}
		seen[w.Branch] = true
		out = append(out, w.Branch)
	}
	return out
}

func defaultBase(branches []string) string {
	for _, b := range branches {
		if b == "main" {
			return b
		}
	}
	for _, b := range branches {
		if b == "master" {
			return b
		}
	}
	if len(branches) > 0 {
		return branches[0]
	}
	return "main"
}

func (m *appModel) selectedRepo() *model.Repo {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].repo
}

func (m *appModel) refreshInspect() tea.Cmd {
	wt := m.selectedWorktree()
	if wt == nil {
		m.inspect = inspectState{}
		return nil
	}
	if m.inspect.wt != nil && m.inspect.wt.Path == wt.Path {
		m.inspect.repo = m.selectedRepo()
		m.inspect.wt = wt
		return tea.Batch(m.loadInspectData()...)
	}
	m.inspect = inspectState{repo: m.selectedRepo(), wt: wt}
	m.changesOff, m.graphOff = 0, 0
	m.inspectGen++
	gen := m.inspectGen
	return tea.Tick(inspectSettleDelay, func(time.Time) tea.Msg { return inspectSettleMsg{gen: gen} })
}

func (m *appModel) loadInspectData() []tea.Cmd {
	cmds := []tea.Cmd{loadInspectStatus(m.gitSvc, m.inspect.wt.Path)}
	if m.inspect.repo != nil {
		cmds = append(cmds, loadInspectGraph(m.gitSvc, m.inspect.wt.Path, m.inspect.repo.Path, branchesForGraph(m.inspect.repo), m.cfg.Inspect.GraphMax))
		cmds = append(cmds, loadInspectStats(m.gitSvc, m.inspect.repo.Path))
		if m.forge != nil {
			cmds = append(cmds, loadInspectPRs(m.forge, m.inspect.wt.Path, m.inspect.repo.Path))
		}
	}
	return cmds
}

func (m *appModel) prsVisible() bool {
	return len(m.inspect.prList) > 0
}

// visibleFocus lists the panes top-to-bottom, omitting PRS when there are no
// pull requests, so tab cycling and the focus guard share one source of truth.
