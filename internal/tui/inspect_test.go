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

func primedModel(snap scan.Snapshot) *appModel {
	m := New(config.Default())
	m.applySnapshot(snap, true)
	m.inspect = inspectState{repo: m.selectedRepo(), wt: m.selectedWorktree()}
	return m
}

func TestAgentSnapshotSkipsInspectorReloadWhenIdle(t *testing.T) {
	snap := inspectSnap()
	snap.Repos[0].Worktrees[0].Agents = nil
	m := primedModel(snap)
	require.Nil(t, m.refreshInspect(false), "an idle worktree must not respawn git on the fast tick")
}

func TestAgentSnapshotReloadsStatusWhileWorktreeIsLive(t *testing.T) {
	m := primedModel(inspectSnap())
	require.NotNil(t, m.refreshInspect(false), "a live agent keeps the status and diff panes hot")
}

func TestAgentSnapshotReloadsStatusWhileFilesAreFresh(t *testing.T) {
	snap := inspectSnap()
	snap.Repos[0].Worktrees[0].Agents = nil
	snap.Repos[0].Worktrees[0].LastFileMod = time.Now()
	m := primedModel(snap)
	require.NotNil(t, m.refreshInspect(false), "fresh file activity keeps the fast reload alive")
}

func TestFullSnapshotReloadsInspector(t *testing.T) {
	snap := inspectSnap()
	snap.Repos[0].Worktrees[0].Agents = nil
	m := primedModel(snap)
	require.NotNil(t, m.refreshInspect(true), "the full scan always refreshes the inspector")
}

func TestDefaultLayoutShowsInspector(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	require.NotNil(t, m.inspect.wt, "selected worktree must be auto-targeted without Enter")
	out := stripANSI(m.render())
	require.Contains(t, out, "CHANGES")
	require.Contains(t, out, "DIFF")
	require.Contains(t, out, "GRAPH")
}

func TestFocusCyclesWorktreesChangesDiffGraph(t *testing.T) {
	m := New(config.Default())
	require.Equal(t, focusList, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusChanges, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusDiff, m.focus)
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
	require.Equal(t, focusDiff, m.focus)
	m.handleKey("[")
	require.Equal(t, focusChanges, m.focus)
}

func TestPaneDiffKeyJumpsToDiffFocus(t *testing.T) {
	m := New(config.Default())
	m.handleKey("4")
	require.Equal(t, focusDiff, m.focus)
}

func TestScrollFocusedPaneMovesDiffOffset(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.focus = focusDiff
	m.scrollDown()
	m.scrollDown()
	require.Equal(t, 2, m.diffOff)
	require.Equal(t, 0, m.graphOff, "scrolling diff must not move the graph offset")
}

func TestScrollFocusedPaneMovesOffset(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	out := stripANSI(m.render())
	require.Contains(t, out, "WORKTREES")
	require.Contains(t, out, "CHANGES")
	require.Contains(t, out, "DIFF")
	require.Contains(t, out, "GRAPH")
}

func TestDiffPaneTitleShowsRealtimeLabel(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	out := stripANSI(m.render())
	require.Contains(t, out, "DIFF · realtime")
}

func TestChangesPaneHeightMatchesContentWhenRoomIsPlentiful(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 50
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.status = model.StatusResult{Files: []model.FileChange{{Path: "a.go", ModTime: time.Now()}}}
	m.inspect.sorted = m.inspect.status.Files
	m.inspect.statusLoaded = true
	m.inspect.graphLoaded = true
	m.inspect.diffLoaded = true

	out := m.renderInspectorBody()
	lines := strings.Split(out, "\n")
	changesTitle, diffTitle := -1, -1
	for i, l := range lines {
		s := stripANSI(l)
		if strings.Contains(s, "▸ CHANGES") {
			changesTitle = i
		}
		if changesTitle >= 0 && diffTitle == -1 && strings.Contains(s, "▸ DIFF") {
			diffTitle = i
		}
	}
	require.NotEqual(t, -1, changesTitle)
	require.NotEqual(t, -1, diffTitle)
	require.Equal(t, 3, diffTitle-changesTitle, "CHANGES pane must render exactly its 1 line of content plus its blank separator, no padding")
}

func TestProjectHeaderShowsProjectAndStats(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	cmd := m.refreshInspect(true)
	require.NotNil(t, m.inspect.wt)
	require.Equal(t, "feat/landing", m.inspect.wt.Branch)
	require.NotNil(t, cmd)
}

func TestProjectHeaderShowsBranchDivergence(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(snap, true)
	m.cursor = 1
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
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

func TestDiffPaneRendersUnifiedDiffWithColorRoles(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.diffTarget = "src/app.go"
	m.inspect.diffContent = "@@ -1,2 +1,3 @@\n-old line\n+new line\n context line"
	m.inspect.diffLoaded = true
	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "src/app.go")
	require.Contains(t, out, "@@ -1,2 +1,3 @@")
	require.Contains(t, out, "-old line")
	require.Contains(t, out, "+new line")
}

func TestDiffPaneShowsCleanPlaceholder(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.diffLoaded = true
	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "working tree clean")
}

func TestDiffPaneTruncatesAtLineLimit(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffLineLimit = 2
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.diffTarget = "big.go"
	m.inspect.diffContent = "+a\n+b\n+c\n+d"
	m.inspect.diffLoaded = true
	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "+2 more")
	require.NotContains(t, out, "+c")
}

func TestSplitPanesThreeWay(t *testing.T) {
	ch, df, gr := splitPanes(30, focusList, 20, 20, 20)
	require.Equal(t, 10, ch)
	require.Equal(t, 10, df)
	require.Equal(t, 10, gr)

	ch, df, gr = splitPanes(30, focusChanges, 20, 20, 20)
	require.Greater(t, ch, df)
	require.Greater(t, ch, gr)
	require.Equal(t, 30, ch+df+gr)

	ch, df, gr = splitPanes(30, focusDiff, 20, 20, 20)
	require.Greater(t, df, ch)
	require.Greater(t, df, gr)
	require.Equal(t, 30, ch+df+gr)

	ch, df, gr = splitPanes(30, focusGraph, 20, 20, 20)
	require.Greater(t, gr, ch)
	require.Greater(t, gr, df)
	require.Equal(t, 30, ch+df+gr)
}

func TestSplitPanesThreeWaySumsAtMinimumRem(t *testing.T) {
	for _, focus := range []paneFocus{focusList, focusChanges, focusDiff, focusGraph} {
		ch, df, gr := splitPanes(3, focus, 20, 20, 20)
		require.Equal(t, 3, ch+df+gr, "focus %v must not exceed the rem=3 budget", focus)
		require.GreaterOrEqual(t, ch, 1)
		require.GreaterOrEqual(t, df, 1)
		require.GreaterOrEqual(t, gr, 1)
	}
}

func TestSplitPanesFitsNaturallyGivesLeftoverToDiff(t *testing.T) {
	ch, df, gr := splitPanes(30, focusList, 3, 3, 3)
	require.Equal(t, 3, ch)
	require.Equal(t, 3, gr)
	require.Equal(t, 24, df, "all leftover room goes to DIFF once CHANGES/GRAPH/DIFF fit their natural content")
}

func TestSplitPanesCapsChangesAndGraphAtHalfRem(t *testing.T) {
	ch, df, gr := splitPanes(30, focusList, 3, 3, 25)
	require.Equal(t, 15, gr, "GRAPH cannot exceed half of rem even if it has more natural content than that")
	require.Equal(t, 3, ch)
	require.Equal(t, 12, df)
	require.Equal(t, 30, ch+df+gr)
}

func TestSplitPanesIgnoresFocusWhenContentFits(t *testing.T) {
	ch1, df1, gr1 := splitPanes(30, focusList, 3, 3, 3)
	ch2, df2, gr2 := splitPanes(30, focusChanges, 3, 3, 3)
	require.Equal(t, []int{ch1, df1, gr1}, []int{ch2, df2, gr2}, "once everything fits naturally, which pane has focus doesn't change the split")
}

func TestDiffPaneShowsWaitingPlaceholderWhenStale(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.wt.Agents = nil
	m.inspect.diffTarget = "src/app.go"
	m.inspect.diffContent = "+new line"
	m.inspect.diffLoaded = true
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)

	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "waiting for changes")
	require.NotContains(t, out, "src/app.go")
	require.NotContains(t, out, "+new line")
}

func TestDiffPaneKeepsRenderingWhenAgentIsLive(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.diffTarget = "src/app.go"
	m.inspect.diffContent = "+new line"
	m.inspect.diffLoaded = true
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)

	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "src/app.go")
	require.Contains(t, out, "+new line")
	require.NotContains(t, out, "waiting for changes")
}

func TestDiffPaneKeepsRenderingWhenFresh(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.width, m.height = 120, 40
	m.applySnapshot(inspectSnap(), true)
	m.refreshInspect(true)
	m.inspect.wt.Agents = nil
	m.inspect.diffTarget = "src/app.go"
	m.inspect.diffContent = "+new line"
	m.inspect.diffLoaded = true
	m.inspect.diffModTime = time.Now()

	out := stripANSI(strings.Join(m.diffContent(), "\n"))
	require.Contains(t, out, "src/app.go")
	require.Contains(t, out, "+new line")
	require.NotContains(t, out, "waiting for changes")
}
