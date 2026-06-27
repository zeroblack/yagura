package worktree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePorcelain(t *testing.T) {
	in := "worktree /repo\n" +
		"HEAD abc123\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/.wt/feat\n" +
		"HEAD def456\n" +
		"branch refs/heads/feat/login\n" +
		"\n" +
		"worktree /repo/.wt/detached\n" +
		"HEAD 999aaa\n" +
		"detached\n"

	got := ParsePorcelain(in)
	require.Len(t, got, 3)

	require.True(t, got[0].IsMain)
	require.Equal(t, "/repo", got[0].Path)
	require.Equal(t, "main", got[0].Branch)
	require.Equal(t, "abc123", got[0].Head)

	require.False(t, got[1].IsMain)
	require.Equal(t, "feat/login", got[1].Branch)

	require.True(t, got[2].Detached)
	require.Equal(t, "(detached)", got[2].Branch)
}

func TestParsePorcelainBare(t *testing.T) {
	in := "worktree /srv/repo.git\n" +
		"bare\n" +
		"\n" +
		"worktree /srv/wt/feat\n" +
		"HEAD def456\n" +
		"branch refs/heads/feat/x\n"

	got := ParsePorcelain(in)
	require.Len(t, got, 2)
	require.True(t, got[0].IsMain)
	require.True(t, got[0].Bare)
	require.Equal(t, "(bare)", got[0].Branch)
	require.False(t, got[1].Bare)
}

func TestParsePorcelainLockedAndPrunable(t *testing.T) {
	in := "worktree /repo\n" +
		"HEAD abc123\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/.wt/locked\n" +
		"HEAD bbb222\n" +
		"branch refs/heads/feat/locked\n" +
		"locked reason with spaces\n" +
		"\n" +
		"worktree /repo/.wt/gone\n" +
		"HEAD ccc333\n" +
		"branch refs/heads/feat/gone\n" +
		"prunable gitdir file points to non-existent location\n"

	got := ParsePorcelain(in)
	require.Len(t, got, 3)
	require.False(t, got[0].Locked)
	require.True(t, got[1].Locked)
	require.False(t, got[1].Prunable)
	require.True(t, got[2].Prunable)
}
