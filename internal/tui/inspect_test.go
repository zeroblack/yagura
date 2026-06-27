package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/scan"
)

func inspectSnap() scan.Snapshot {
	return scan.Snapshot{Repos: []model.Repo{{
		Name: "reddit-brain", Path: "/p/reddit-brain",
		Worktrees: []model.Worktree{{
			Path: "/p/reddit-brain", Branch: "feat/landing", IsMain: true,
			Agents: []model.AgentSession{{
				Tool: "claude", State: model.StateEditing, Liveness: model.LiveActive, UpdatedAt: time.Now(),
			}},
		}},
	}}}
}

func TestDefaultLayoutShowsInspector(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	require.NotNil(t, m.inspect.wt, "selected worktree must be auto-targeted without Enter")
	out := stripANSI(m.render())
	require.Contains(t, out, "CHANGES")
	require.Contains(t, out, "GRAPH")
}

func TestFocusCyclesWorktreesChangesGraph(t *testing.T) {
	m := New(config.Default())
	require.Equal(t, focusList, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusChanges, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusGraph, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusList, m.focus)
	m.cycleFocus(-1)
	require.Equal(t, focusGraph, m.focus)
}

func TestBracketKeysCycleFocus(t *testing.T) {
	m := New(config.Default())
	m.handleKey("]")
	require.Equal(t, focusChanges, m.focus)
	m.handleKey("]")
	require.Equal(t, focusGraph, m.focus)
	m.handleKey("[")
	require.Equal(t, focusChanges, m.focus)
}

func TestScrollFocusedPaneMovesOffset(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.focus = focusGraph
	m.scrollDown()
	m.scrollDown()
	require.Equal(t, 2, m.graphOff)
	require.Equal(t, 0, m.changesOff, "scrolling graph must not move changes offset")
	m.scrollUp()
	require.Equal(t, 1, m.graphOff)
	m.focus = focusChanges
	m.scrollDown()
	require.Equal(t, 1, m.changesOff)
}

func TestPaneTitlesShowAllGroups(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	out := stripANSI(m.render())
	require.Contains(t, out, "WORKTREES")
	require.Contains(t, out, "CHANGES")
	require.Contains(t, out, "GRAPH")
}

func TestProjectHeaderShowsProjectAndStats(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.stats = model.RepoStats{Commits: 142, Branches: 5}
	m.inspect.statsLoaded = true
	out := stripANSI(strings.Join(m.focusCardLines(), "\n"))
	require.Contains(t, out, "reddit-brain")
	require.Contains(t, out, "feat/landing")
	require.Contains(t, out, "142 commits")
	require.Contains(t, out, "5 branches")
	require.Contains(t, out, "1 worktree")
}

func TestRefreshInspectTargetsSelection(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 100, 40
	m.applySnapshot(inspectSnap())
	cmd := m.refreshInspect()
	require.NotNil(t, m.inspect.wt)
	require.Equal(t, "feat/landing", m.inspect.wt.Branch)
	require.NotNil(t, cmd)
}

func TestProjectHeaderShowsBranchDivergence(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.status = model.StatusResult{Branch: "feat/landing", Upstream: "origin/main", Ahead: 3, Behind: 1}
	m.inspect.statusLoaded = true
	out := stripANSI(strings.Join(m.focusCardLines(), "\n"))
	require.Contains(t, out, "feat/landing")
	require.Contains(t, out, "↑3")
	require.Contains(t, out, "↓1")
	require.Contains(t, out, "EDITING", "the focus card surfaces the active agent of the focused worktree")
}

func TestGraphTipAnnotation(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	snap := scan.Snapshot{Repos: []model.Repo{{
		Name: "reddit-brain", Path: "/p/reddit-brain",
		Worktrees: []model.Worktree{
			{Path: "/p/reddit-brain", Branch: "main", IsMain: true},
			{Path: "/p/wt-landing", Branch: "feat/landing", Agents: []model.AgentSession{{
				Tool: "claude", State: model.StateEditing, Liveness: model.LiveActive, UpdatedAt: time.Now(),
			}}},
		},
	}}}
	m.applySnapshot(snap)
	m.cursor = 1
	m.refreshInspect()
	m.inspect.graph = []model.GraphLine{
		{Graph: "* ", HasCommit: true, Hash: "abc1234", Author: "Dioni", Subject: "hero gradient", Refs: []string{"feat/landing"}, When: time.Now().Add(-2 * time.Minute)},
		{Graph: "* ", HasCommit: true, Hash: "def5678", Author: "Dioni", Subject: "bump version", Refs: []string{"main"}, When: time.Now().Add(-3 * time.Hour)},
	}
	m.inspect.graphLoaded = true
	m.inspect.div = map[string]model.Divergence{"feat/landing": {Ahead: 3}}
	out := stripANSI(strings.Join(m.graphLines(), "\n"))
	require.Contains(t, out, "feat/landing")
	require.NotContains(t, out, "EDITING", "agent state belongs in the WORKTREES list, not the graph")
	require.Contains(t, out, "↑3")
	require.Contains(t, out, "hero gradient")
	require.Contains(t, out, "bump version")
}

func TestMergeBranchName(t *testing.T) {
	require.Equal(t, "feat/landing", mergeBranchName("Merge pull request #42 from owner/feat/landing"))
	require.Equal(t, "fix-x", mergeBranchName("Merge branch 'fix-x'"))
	require.Equal(t, "foo", mergeBranchName("Merge branch 'foo' into main"))
	require.Equal(t, "bar", mergeBranchName("Merge remote-tracking branch 'origin/bar'"))
	require.Equal(t, "", mergeBranchName("merge: unifica settings overhaul"))
}

func TestGraphMergeShowsSourceBranch(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.graph = []model.GraphLine{
		{Graph: "*   ", HasCommit: true, Hash: "fff0001", Author: "Dioni", Parents: []string{"a", "b"}, Subject: "Merge branch 'feat/sidebar'", When: time.Now()},
	}
	m.inspect.graphLoaded = true
	out := stripANSI(strings.Join(m.graphLines(), "\n"))
	require.Contains(t, out, "feat/sidebar")
}

func TestInspectHeaderShowsPR(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.status = model.StatusResult{Branch: "feat/landing"}
	m.inspect.statusLoaded = true
	m.inspect.prByBranch = map[string]model.PRInfo{
		"feat/landing": {Number: 42, State: "open", Review: model.ReviewRequired, CI: model.CIPassing},
	}
	out := stripANSI(strings.Join(m.focusCardLines(), "\n"))
	require.Contains(t, out, "PR #42")
	require.Contains(t, out, "CI ✓")
}

func TestPrsByBranch(t *testing.T) {
	prs := []model.PRInfo{{Number: 1, Branch: "a"}, {Number: 2, Branch: "b"}}
	m := prsByBranch(prs)
	require.Equal(t, 1, m["a"].Number)
	require.Equal(t, 2, m["b"].Number)
}

func TestChangesSortByMtime(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.Update(inspectStatusMsg{wtPath: m.inspect.wt.Path, status: model.StatusResult{Files: []model.FileChange{
		{Path: "old.go", Status: model.StatusModified, ModTime: time.Now().Add(-time.Hour)},
		{Path: "new.go", Status: model.StatusModified, ModTime: time.Now()},
	}}})
	out := stripANSI(strings.Join(m.changesLines(), "\n"))
	require.Less(t, strings.Index(out, "new.go"), strings.Index(out, "old.go"))
}

func TestInspectChangesRender(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.Update(inspectStatusMsg{wtPath: m.inspect.wt.Path, status: model.StatusResult{Files: []model.FileChange{
		{Path: "src/session/share.go", Status: model.StatusModified, Staged: true, Added: 42, Deleted: 8, ModTime: time.Now().Add(-2 * time.Minute)},
		{Path: "notes/scratch.md", Status: model.StatusUntracked},
	}}})
	out := stripANSI(strings.Join(m.changesLines(), "\n"))
	require.Contains(t, out, "src/session/share.go")
	require.Contains(t, out, "+42")
	require.Contains(t, out, "-8")
	require.Contains(t, out, "M")
	require.Contains(t, out, "notes/scratch.md")
	require.Contains(t, out, "?")
}
