package tui

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/scan"
)

func sampleSnap() scan.Snapshot {
	return scan.Snapshot{Repos: []model.Repo{
		{Name: "a", Worktrees: []model.Worktree{{Branch: "main"}, {Branch: "feat/x"}}},
		{Name: "b", Worktrees: []model.Worktree{{Branch: "main"}}},
	}}
}

func TestModelBuildsFlatRows(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(sampleSnap())
	require.Equal(t, 2, m.repoCount())
	require.Equal(t, 3, m.worktreeCount())
}

func TestToggleGrouping(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(sampleSnap())
	require.True(t, m.grouped)
	groupedRows := len(m.rows)
	m.toggleGrouping()
	require.False(t, m.grouped)
	require.Less(t, len(m.rows), groupedRows)
}
