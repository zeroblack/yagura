package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/scan"
)

func TestForceRefreshGuardsConcurrentScans(t *testing.T) {
	m := New(config.Default())
	require.NotNil(t, m.forceRefresh())
	require.True(t, m.scanInFlight)
	require.Nil(t, m.forceRefresh(), "a second refresh while one is in flight must be a no-op")

	m.Update(snapshotMsg{snap: scan.Snapshot{}})
	require.False(t, m.scanInFlight, "an arriving snapshot clears the in-flight flag")
	require.NotNil(t, m.forceRefresh())
}

func TestSnapshotErrorIsShownAndCleared(t *testing.T) {
	m := New(config.Default())
	m.Update(snapshotMsg{err: errFake})
	require.Equal(t, errFake, m.err)
	m.Update(snapshotMsg{snap: sampleSnap()})
	require.Nil(t, m.err, "a successful snapshot clears a previous error")
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }

const errFake = fakeErr("scan failed")

func TestOpenDetailLoadsAsynchronously(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(inspectSnap())

	cmd := m.handleKey("d")
	require.Equal(t, detailDiff, m.detail)
	require.True(t, m.detailLoading)
	require.Empty(t, m.detailContent)
	require.NotNil(t, cmd, "opening a detail must dispatch a load command, never block Update")

	m.Update(detailMsg{mode: detailDiff, wtPath: "/somewhere/else", content: "stale"})
	require.Empty(t, m.detailContent, "results for another worktree are dropped")

	m.Update(detailMsg{mode: detailDiff, wtPath: m.selectedWorktreePath(), content: "diff body"})
	require.Equal(t, "diff body", m.detailContent)
	require.False(t, m.detailLoading)
}

func TestStatusAndAgentLogKeysOpenDetail(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(inspectSnap())

	m.handleKey("s")
	require.Equal(t, detailStatus, m.detail)
	m.handleKey("esc")
	require.Equal(t, detailNone, m.detail)

	m.handleKey("l")
	require.Equal(t, detailAgentLog, m.detail)
	m.handleKey("esc")
	require.Equal(t, detailNone, m.detail)
}

func TestKeymapHonorsConfigOverrides(t *testing.T) {
	cfg := config.Default()
	cfg.Keys.Quit = []string{"x"}
	cfg.Keys.Diff = []string{"D"}
	m := New(cfg)
	m.applySnapshot(inspectSnap())

	cmd := m.handleKey("x")
	require.NotNil(t, cmd)
	require.IsType(t, tea.QuitMsg{}, cmd())

	require.Nil(t, m.handleKey("q"), "default binding is gone once overridden")

	m.handleKey("D")
	require.Equal(t, detailDiff, m.detail)
}

func TestFooterReflectsConfiguredKeys(t *testing.T) {
	m := New(config.Default())
	out := stripANSI(m.footerLine)
	for _, want := range []string{"d diff", "c commits", "s status", "l log", "r sync", "q quit", "o group"} {
		require.Contains(t, out, want)
	}
}

func TestCellTruncRespectsWideGlyphs(t *testing.T) {
	require.Equal(t, "hello", cellTrunc("hello", 10))
	require.Equal(t, "he…", cellTrunc("hello", 3))
	require.Equal(t, "日…", cellTrunc("日本語", 4), "a kanji spans two cells")
	require.Equal(t, "", cellTrunc("abc", 0))
}

func TestRenderWorktreeShowsLockedAndPrunable(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	snap := inspectSnap()
	snap.Repos[0].Worktrees[0].Locked = true
	snap.Repos[0].Worktrees[0].Prunable = true
	m.applySnapshot(snap)
	out := stripANSI(m.render())
	require.Contains(t, out, "[locked]")
	require.Contains(t, out, "[prunable]")
}
