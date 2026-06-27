package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/scan"
)

func twoWorktreeSnap(order ...string) scan.Snapshot {
	wts := map[string]model.Worktree{
		"main": {Path: "/p/proj/app", Branch: "main", IsMain: true},
		"feat": {Path: "/p/proj/wt-feat", Branch: "feat/x"},
	}
	if len(order) == 0 {
		order = []string{"main", "feat"}
	}
	var list []model.Worktree
	for _, k := range order {
		list = append(list, wts[k])
	}
	return scan.Snapshot{Repos: []model.Repo{{Name: "app", Path: "/p/proj/app", Worktrees: list}}}
}

func rowIndexForPath(m *appModel, path string) int {
	for i, r := range m.rows {
		if r.worktree != nil && r.worktree.Path == path {
			return i
		}
	}
	return -1
}

func TestFocusTracksIdentityAcrossResort(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(twoWorktreeSnap())
	m.cursor = rowIndexForPath(m, "/p/proj/wt-feat")
	m.syncCursorPath()
	require.Equal(t, "/p/proj/wt-feat", m.cursorPath)

	m.applySnapshot(twoWorktreeSnap("feat", "main"))
	require.NotNil(t, m.selectedWorktree())
	require.Equal(t, "/p/proj/wt-feat", m.selectedWorktree().Path,
		"the activity re-sort must not drag the focus onto a different worktree")
}

func TestTogglePinAnchorsAboveSiblings(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(twoWorktreeSnap())
	m.cursor = rowIndexForPath(m, "/p/proj/wt-feat")
	m.syncCursorPath()
	m.togglePin()
	require.Equal(t, "/p/proj/wt-feat", m.pinnedPath)
	require.Less(t, rowIndexForPath(m, "/p/proj/wt-feat"), rowIndexForPath(m, "/p/proj/app"),
		"pinned worktree anchors above its siblings")

	m.togglePin()
	require.Equal(t, "", m.pinnedPath)
}

func TestPinPrunedWhenWorktreeDisappears(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(twoWorktreeSnap())
	m.cursor = rowIndexForPath(m, "/p/proj/wt-feat")
	m.syncCursorPath()
	m.togglePin()
	require.Equal(t, "/p/proj/wt-feat", m.pinnedPath)

	m.applySnapshot(twoWorktreeSnap("main"))
	require.Equal(t, "", m.pinnedPath, "a removed worktree must not stay pinned")
}

func TestPinKeyTogglesPinViaAlias(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	require.Equal(t, "", m.pinnedPath)
	m.handleKey("space")
	require.Equal(t, "/p/reddit-brain", m.pinnedPath)
	m.handleKey(" ")
	require.Equal(t, "", m.pinnedPath, "the space alias must reach the pin action too")
}

func TestWorktreeTagUsesPrimaryNotMain(t *testing.T) {
	m := New(config.Default())
	primary := stripANSI(m.worktreeTag(model.Repo{Path: "/p/app"}, model.Worktree{Path: "/p/app", IsMain: true}, true))
	require.Contains(t, primary, "primary")
	require.NotContains(t, primary, "main")
	linked := stripANSI(m.worktreeTag(model.Repo{Path: "/p/app"}, model.Worktree{Path: "/p/wt-x", Branch: "feat/x"}, true))
	require.Contains(t, linked, "⟨wt-x⟩")
}

func TestRepoProjectResolution(t *testing.T) {
	m := New(config.Default())
	require.Equal(t, "yagura", m.repoProject(&model.Repo{Name: "app", Path: "/Users/x/yagura/app"}))
	require.Equal(t, "reddit-brain", m.repoProject(&model.Repo{Name: "reddit-brain", Path: "/p/reddit-brain"}))
	m.cfg.Display.ProjectFrom = "repo"
	require.Equal(t, "app", m.repoProject(&model.Repo{Name: "app", Path: "/Users/x/yagura/app"}))
}

func multiRepoSnap() scan.Snapshot {
	return scan.Snapshot{Repos: []model.Repo{
		{Name: "simce/mvp", Path: "/p/simce/mvp", Worktrees: []model.Worktree{
			{Path: "/p/simce/mvp", Branch: "main", IsMain: true},
			{Path: "/p/simce/.wt/audit", Branch: "perf/audit"},
		}},
		{Name: "cockpit", Path: "/p/cockpit", Worktrees: []model.Worktree{
			{Path: "/p/cockpit", Branch: "feat/cockpit-mvp", IsMain: true},
		}},
	}}
}

func TestSingleWorktreeRepoShowsProjectLabel(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 90, 40
	m.applySnapshot(multiRepoSnap())
	out := stripANSI(strings.Join(rowStrings(m), "\n"))
	require.Contains(t, out, "simce/mvp", "the multi-worktree repo keeps its group header")
	require.Contains(t, out, "cockpit › feat/cockpit-mvp",
		"a single-worktree repo must show its project, not float unlabeled under another project")
}

func TestNavigationSkipsHeadersAndAgents(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 90, 40
	snap := scan.Snapshot{Repos: []model.Repo{{
		Name: "app", Path: "/p/app", Worktrees: []model.Worktree{
			{Path: "/p/app", Branch: "main", IsMain: true, Agents: []model.AgentSession{{Tool: "claude", State: model.StateIdle}}},
			{Path: "/p/app/.wt/x", Branch: "feat/x"},
		},
	}}}
	m.applySnapshot(snap)
	require.True(t, m.rows[m.cursor].focusable(), "launch focus lands on a worktree, not a header")
	require.Equal(t, "/p/app", m.selectedWorktree().Path)

	m.focus = focusList
	m.scrollDown()
	require.Equal(t, "/p/app/.wt/x", m.selectedWorktree().Path,
		"down skips the group header and the agent sub-row to reach the next worktree")
	m.scrollUp()
	require.Equal(t, "/p/app", m.selectedWorktree().Path)
}

func rowStrings(m *appModel) []string {
	out := make([]string, len(m.rows))
	for i, r := range m.rows {
		out[i] = m.renderRow(i, r)
	}
	return out
}

func TestFocusCardShowsPinMarkerAndAgent(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.togglePin()
	out := stripANSI(strings.Join(m.focusCardLines(), "\n"))
	require.Contains(t, out, "◉")
	require.Contains(t, out, "reddit-brain")
	require.Contains(t, out, "EDITING")
	require.Contains(t, out, "primary")
}
