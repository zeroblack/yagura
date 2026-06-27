package scan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"}, {"config", "user.email", "t@t.io"}, {"config", "user.name", "t"},
		{"commit", "--allow-empty", "-qm", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
}

func TestSnapshotFindsReposAndWorktrees(t *testing.T) {
	root := t.TempDir()
	gitInit(t, filepath.Join(root, "a"))
	gitInit(t, filepath.Join(root, "b"))

	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Agents.ClaudeRoot = filepath.Join(t.TempDir(), "no-sessions")

	snap, err := NewScanner(cfg).Take(context.Background())
	require.NoError(t, err)
	require.Len(t, snap.Repos, 2)
	for _, r := range snap.Repos {
		require.GreaterOrEqual(t, len(r.Worktrees), 1)
		require.Equal(t, "main", r.Worktrees[0].Branch)
	}
}
