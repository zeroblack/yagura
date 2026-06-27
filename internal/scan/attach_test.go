package scan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/agents"
	"github.com/zeroblack/yagura/internal/model"
)

type fakeSource struct{ sessions []agents.Session }

func (f fakeSource) Collect() []agents.Session { return f.sessions }

func repoWithWorktrees() []model.Repo {
	return []model.Repo{{
		Name: "shirei", Path: "/repo",
		Worktrees: []model.Worktree{
			{Path: "/repo", Branch: "main", IsMain: true},
			{Path: "/repo/.wt/feat", Branch: "feat/x"},
		},
	}}
}

func agentByWorktree(repos []model.Repo) map[string][]model.AgentSession {
	out := map[string][]model.AgentSession{}
	for _, r := range repos {
		for _, w := range r.Worktrees {
			out[w.Path] = w.Agents
		}
	}
	return out
}

func TestAttachAgentsPlacesByHomeWithLeak(t *testing.T) {
	repos := repoWithWorktrees()
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "/repo/.wt/feat", LastEditPath: "/repo/src/app.ts",
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: time.Now(),
	}}}

	attachAgents(repos, src, time.Minute)

	byWt := agentByWorktree(repos)
	require.Len(t, byWt["/repo/.wt/feat"], 1, "agent belongs to its home worktree (cwd)")
	require.Empty(t, byWt["/repo"], "agent must not be placed under the edited tree")
	require.Equal(t, "main", byWt["/repo/.wt/feat"][0].LeakTarget)
}

func TestAttachAgentsNoLeakWhenEditingHome(t *testing.T) {
	repos := repoWithWorktrees()
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "/repo/.wt/feat", LastEditPath: "/repo/.wt/feat/src/app.ts",
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: time.Now(),
	}}}

	attachAgents(repos, src, time.Minute)

	byWt := agentByWorktree(repos)
	require.Len(t, byWt["/repo/.wt/feat"], 1)
	require.Empty(t, byWt["/repo/.wt/feat"][0].LeakTarget)
}

func TestAttachAgentsNoLeakForExternalPath(t *testing.T) {
	repos := repoWithWorktrees()
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "/repo/.wt/feat", LastEditPath: "/tmp/scratch.ts",
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: time.Now(),
	}}}

	attachAgents(repos, src, time.Minute)

	byWt := agentByWorktree(repos)
	require.Len(t, byWt["/repo/.wt/feat"], 1)
	require.Empty(t, byWt["/repo/.wt/feat"][0].LeakTarget)
}

func TestAttachAgentsNoLeakAcrossRepos(t *testing.T) {
	repos := []model.Repo{
		{Name: "a", Path: "/a", Worktrees: []model.Worktree{{Path: "/a", Branch: "main", IsMain: true}}},
		{Name: "b", Path: "/b", Worktrees: []model.Worktree{{Path: "/b", Branch: "main", IsMain: true}}},
	}
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "/a", LastEditPath: "/b/src/app.ts",
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: time.Now(),
	}}}

	attachAgents(repos, src, time.Minute)

	byWt := agentByWorktree(repos)
	require.Len(t, byWt["/a"], 1)
	require.Empty(t, byWt["/a"][0].LeakTarget, "edits landing in a different repo are not a worktree leak")
}

func TestAttachAgentsFallsBackToEditPathWithoutCwd(t *testing.T) {
	repos := repoWithWorktrees()
	src := fakeSource{sessions: []agents.Session{{
		Path: "s1", Cwd: "", LastEditPath: "/repo/.wt/feat/src/app.ts",
		State: model.StateEditing, Liveness: model.LiveActive, ModTime: time.Now(),
	}}}

	attachAgents(repos, src, time.Minute)

	byWt := agentByWorktree(repos)
	require.Len(t, byWt["/repo/.wt/feat"], 1)
	require.Empty(t, byWt["/repo/.wt/feat"][0].LeakTarget)
}
