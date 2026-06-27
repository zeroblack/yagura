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

func TestComputeAttention(t *testing.T) {
	a := computeAttention(&model.PRInfo{CI: model.CIFailing}, model.Divergence{}, false)
	require.True(t, a.Needs)
	require.Equal(t, "ci failing", a.Reason)

	a = computeAttention(&model.PRInfo{Review: model.ReviewChangesRequested}, model.Divergence{}, false)
	require.Equal(t, "changes requested", a.Reason)

	a = computeAttention(nil, model.Divergence{}, true)
	require.Equal(t, "conflict", a.Reason)

	a = computeAttention(nil, model.Divergence{Behind: 2}, false)
	require.Equal(t, "behind base", a.Reason)

	a = computeAttention(&model.PRInfo{CI: model.CIPassing, Review: model.ReviewApproved}, model.Divergence{}, false)
	require.False(t, a.Needs)
}

func TestGraphAttentionMarker(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	snap := scan.Snapshot{Repos: []model.Repo{{
		Name: "r", Path: "/p/r",
		Worktrees: []model.Worktree{
			{Path: "/p/r", Branch: "main", IsMain: true},
			{Path: "/p/wt", Branch: "feat/landing"},
		},
	}}}
	m.applySnapshot(snap)
	m.cursor = 1
	m.refreshInspect()
	m.inspect.prByBranch = map[string]model.PRInfo{"feat/landing": {Number: 42, CI: model.CIFailing}}
	m.inspect.graph = []model.GraphLine{
		{Graph: "* ", HasCommit: true, Hash: "abc", Subject: "x", Refs: []string{"feat/landing"}, When: time.Now()},
	}
	m.inspect.graphLoaded = true
	out := stripANSI(strings.Join(m.graphLines(), "\n"))
	require.Contains(t, out, "feat/landing !")
}

func TestAttentionSectionWhenEnabled(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 120, 40
	snap := scan.Snapshot{Repos: []model.Repo{{
		Name: "r", Path: "/p/r",
		Worktrees: []model.Worktree{
			{Path: "/p/r", Branch: "main", IsMain: true},
			{Path: "/p/wt", Branch: "fix/quiz"},
		},
	}}}
	m.applySnapshot(snap)
	m.cursor = 1
	m.refreshInspect()
	m.inspect.statusLoaded = true
	m.inspect.prByBranch = map[string]model.PRInfo{"fix/quiz": {Number: 41, Review: model.ReviewChangesRequested}}
	out := stripANSI(m.attentionBanner())
	require.Contains(t, out, "ATTENTION")
	require.Contains(t, out, "fix/quiz")
	require.Contains(t, out, "changes requested")
}
