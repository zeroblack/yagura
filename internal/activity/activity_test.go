package activity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

func TestSortWorktreesLiveFirst(t *testing.T) {
	now := time.Now()
	wts := []model.Worktree{
		{Branch: "old", LastCommit: now.Add(-48 * time.Hour)},
		{Branch: "live", Agents: []model.AgentSession{{Liveness: model.LiveActive}}},
		{Branch: "recent-file", LastFileMod: now.Add(-2 * time.Minute)},
	}
	SortWorktrees(wts)
	require.Equal(t, "live", wts[0].Branch)
	require.Equal(t, "recent-file", wts[1].Branch)
	require.Equal(t, "old", wts[2].Branch)
}

func TestRepoIsActive(t *testing.T) {
	r := model.Repo{Worktrees: []model.Worktree{
		{Agents: []model.AgentSession{{Liveness: model.LiveActive}}},
	}}
	require.True(t, RepoActive(r))
	require.False(t, RepoActive(model.Repo{Worktrees: []model.Worktree{{Branch: "x"}}}))
}

func TestScoreRanksLivenessAndRecency(t *testing.T) {
	now := time.Now()

	live, recency := Score(model.Worktree{
		Agents:     []model.AgentSession{{Liveness: model.LiveActive}, {Liveness: model.LiveRecent}},
		LastCommit: now.Add(-time.Hour),
	})
	require.Equal(t, 2, live)
	require.Equal(t, now.Add(-time.Hour), recency)

	live, _ = Score(model.Worktree{Agents: []model.AgentSession{{Liveness: model.LiveRecent}}})
	require.Equal(t, 1, live)

	live, recency = Score(model.Worktree{LastFileMod: now, LastCommit: now.Add(-time.Minute)})
	require.Equal(t, 0, live)
	require.Equal(t, now, recency, "recency is the most recent of file mod and commit")
}

func TestSortReposActiveFirstThenName(t *testing.T) {
	repos := []model.Repo{
		{Name: "zeta"},
		{Name: "beta", Worktrees: []model.Worktree{
			{Branch: "idle", LastCommit: time.Now().Add(-time.Hour)},
			{Branch: "hot", Agents: []model.AgentSession{{Liveness: model.LiveActive}}},
		}},
		{Name: "alpha"},
	}
	SortRepos(repos)
	require.Equal(t, "beta", repos[0].Name, "repos with live agents sort first")
	require.Equal(t, "alpha", repos[1].Name)
	require.Equal(t, "zeta", repos[2].Name)
	require.Equal(t, "hot", repos[0].Worktrees[0].Branch, "worktrees inside a repo are sorted too")
}
