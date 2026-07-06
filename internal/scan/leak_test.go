package scan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/agents"
	"github.com/zeroblack/yagura/internal/model"
)

const leakWindow = 8 * time.Second

func tick(leaks *leakTracker, editPath string, now time.Time) string {
	repos := repoWithWorktrees()
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "/repo/.wt/feat", LastEditPath: editPath,
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: now,
	}}}
	attachAgents(repos, src, time.Minute, leaks, leakWindow, now)
	return agentByWorktree(repos)["/repo/.wt/feat"][0].LeakTarget
}

func TestLeakDebouncedAcrossContextSwitch(t *testing.T) {
	leaks := newLeakTracker()
	t0 := time.Unix(1_700_000_000, 0)

	require.Empty(t, tick(leaks, "/repo/src/app.ts", t0),
		"a single cross-worktree edit must not raise the chip on first sight")
	require.Empty(t, tick(leaks, "/repo/.wt/feat/src/app.ts", t0.Add(5*time.Second)),
		"agent returned home before the window elapsed: the transient edit was not a leak")
	require.Empty(t, tick(leaks, "/repo/.wt/feat/src/app.ts", t0.Add(20*time.Second)),
		"home edits keep the chip clear even after the window has passed")
}

func TestLeakConfirmedWhenSustained(t *testing.T) {
	leaks := newLeakTracker()
	t0 := time.Unix(1_700_000_000, 0)

	require.Empty(t, tick(leaks, "/repo/src/app.ts", t0))
	require.Empty(t, tick(leaks, "/repo/src/app.ts", t0.Add(5*time.Second)),
		"still inside the debounce window")
	require.Equal(t, "main", tick(leaks, "/repo/src/app.ts", t0.Add(leakWindow)),
		"a leak sustained past the window is real and must surface")
}
