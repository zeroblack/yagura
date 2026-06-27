package scan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/agents"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
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

func TestRefreshAgentsReusesSkeleton(t *testing.T) {
	root := t.TempDir()
	gitInit(t, filepath.Join(root, "a"))
	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Agents.ClaudeRoot = filepath.Join(t.TempDir(), "none")

	sc := NewScanner(cfg)
	full, err := sc.Take(context.Background())
	require.NoError(t, err)
	require.Len(t, full.Repos, 1)
	wtPath := full.Repos[0].Worktrees[0].Path

	sc.src = fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: wtPath, State: model.StateRunning,
		Liveness: model.LiveActive, ModTime: time.Now(),
	}}}
	refreshed, err := sc.RefreshAgents(context.Background())
	require.NoError(t, err)
	require.Len(t, refreshed.Repos, 1, "reuses the cached skeleton")
	require.Equal(t, wtPath, refreshed.Repos[0].Worktrees[0].Path)
	require.Len(t, refreshed.Repos[0].Worktrees[0].Agents, 1, "fresh agents attached on refresh")
	require.Empty(t, full.Repos[0].Worktrees[0].Agents, "the prior snapshot is not mutated by a refresh")
}

func TestRefreshAgentsFallsBackToTakeWithoutCache(t *testing.T) {
	root := t.TempDir()
	gitInit(t, filepath.Join(root, "a"))
	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Agents.ClaudeRoot = filepath.Join(t.TempDir(), "none")

	snap, err := NewScanner(cfg).RefreshAgents(context.Background())
	require.NoError(t, err)
	require.Len(t, snap.Repos, 1, "falls back to a full scan before the first snapshot")
}
