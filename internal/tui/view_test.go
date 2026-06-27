package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/scan"
)

func TestViewShowsBranchAndState(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 100, 30
	m.applySnapshot(scan.Snapshot{Repos: []model.Repo{
		{Name: "togi", Worktrees: []model.Worktree{
			{Branch: "feat/togi-mvp", Agents: []model.AgentSession{
				{Tool: "claude", State: model.StateRunning, Task: "running go test", Liveness: model.LiveActive},
			}},
		}},
	}})
	out := stripANSI(m.render())
	require.Contains(t, out, "feat/togi-mvp")
	require.Contains(t, out, "RUNNING")
	require.Contains(t, out, "togi")
}

func TestViewShowsLeakChipWhenAgentEditsOtherWorktree(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 100, 30
	m.applySnapshot(scan.Snapshot{Repos: []model.Repo{
		{Name: "shirei", Worktrees: []model.Worktree{
			{Branch: "feat/new-window", Agents: []model.AgentSession{
				{Tool: "claude", State: model.StateEditing, Task: "editing lib.rs", Liveness: model.LiveActive, LeakTarget: "main"},
			}},
		}},
	}})
	out := stripANSI(m.render())
	require.Contains(t, out, "LEAK → main")
}

func TestViewOmitsLeakChipWithoutLeak(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 100, 30
	m.applySnapshot(scan.Snapshot{Repos: []model.Repo{
		{Name: "shirei", Worktrees: []model.Worktree{
			{Branch: "feat/new-window", Agents: []model.AgentSession{
				{Tool: "claude", State: model.StateEditing, Task: "editing lib.rs", Liveness: model.LiveActive},
			}},
		}},
	}})
	out := stripANSI(m.render())
	require.NotContains(t, out, "LEAK")
}

func TestHeaderIsSingleLine(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 100, 30
	out := stripANSI(m.header())
	require.NotContains(t, out, "\n", "header is a single line")
	require.Contains(t, out, "YAGURA")
}

func TestColorCommitShowsAgeAndAuthor(t *testing.T) {
	m := New(config.Default())
	m.width = 100
	now := strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10)
	line := strings.Join([]string{"4dcf98c", now, "Dioni", "chore(dealer-mobile): import inicial"}, "\x1f")
	out := stripANSI(m.colorCommit(line))
	require.Contains(t, out, "4dcf98c")
	require.Contains(t, out, "2h")
	require.Contains(t, out, "Dioni")
	require.Contains(t, out, "chore(dealer-mobile): import inicial")
}

func stripANSI(s string) string {
	var b strings.Builder
	skip := false
	for _, r := range s {
		if r == 0x1b {
			skip = true
			continue
		}
		if skip {
			if r == 'm' {
				skip = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
