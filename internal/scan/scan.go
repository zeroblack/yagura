package scan

import (
	"cmp"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zeroblack/yagura/internal/activity"
	"github.com/zeroblack/yagura/internal/agents"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/discovery"
	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/worktree"
)

type Snapshot struct {
	Repos []model.Repo
}

// Scanner holds the state that must survive between snapshots: the git
// semaphore and the agent source transcript cache.
type Scanner struct {
	cfg config.Config
	git *git.Service
	src agents.Source
}

func NewScanner(cfg config.Config) *Scanner {
	return &Scanner{
		cfg: cfg,
		git: git.NewService(cfg.Git.MaxProcs, cfg.Git.Timeout),
		src: agents.NewClaudeSource(cfg.Agents.ClaudeRoot, cfg.Agents.ActiveWindow, cfg.Agents.ToolTimeout),
	}
}

func (s *Scanner) Git() *git.Service { return s.git }

func (s *Scanner) Take(ctx context.Context) (Snapshot, error) {
	roots := s.cfg.Roots
	if len(roots) == 0 {
		if wd, err := filepath.Abs("."); err == nil {
			roots = []string{wd}
		}
	}

	repoPaths, err := discovery.Find(roots, discovery.Options{MaxDepth: s.cfg.MaxDepth, Ignore: s.cfg.Ignore})
	if err != nil {
		return Snapshot{}, err
	}

	ignore := discovery.IgnoreSet(s.cfg.Ignore)
	repos := make([]model.Repo, len(repoPaths))
	var wg sync.WaitGroup
	for i, p := range repoPaths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			repos[i] = s.buildRepo(ctx, path, roots, ignore)
		}(i, p)
	}
	wg.Wait()

	attachAgents(repos, s.src, s.cfg.Agents.ToolTimeout)
	activity.SortRepos(repos)
	return Snapshot{Repos: repos}, nil
}

func (s *Scanner) buildRepo(ctx context.Context, path string, roots []string, ignore map[string]bool) model.Repo {
	r := model.Repo{Name: repoName(path, roots), Path: path}
	out, err := s.git.RunGit(ctx, path, "worktree", "list", "--porcelain")
	if err != nil {
		r.Err = err
		return r
	}
	r.Worktrees = worktree.ParsePorcelain(out)
	for j := range r.Worktrees {
		w := &r.Worktrees[j]
		w.LastCommit = worktree.LastCommit(ctx, s.git, w.Path)
		w.LastFileMod = worktree.LatestFileMod(w.Path, ignore, s.cfg.MaxDepth)
	}
	return r
}

type worktreeRef struct {
	repoPath string
	branch   string
}

func worktreeIndex(repos []model.Repo) ([]string, map[string]worktreeRef) {
	var paths []string
	refs := map[string]worktreeRef{}
	for _, r := range repos {
		for _, w := range r.Worktrees {
			paths = append(paths, w.Path)
			refs[w.Path] = worktreeRef{repoPath: r.Path, branch: cmp.Or(w.Branch, filepath.Base(w.Path))}
		}
	}
	return paths, refs
}

func attachAgents(repos []model.Repo, src agents.Source, toolTimeout time.Duration) {
	sessions := src.Collect()
	paths, refs := worktreeIndex(repos)
	now := time.Now()

	byWt := map[string][]model.AgentSession{}
	for _, s := range sessions {
		home := agents.MatchWorktree(s.Cwd, paths)
		if home == "" {
			home = agents.MatchWorktree(s.LastEditPath, paths)
		}
		if home == "" {
			home = agents.MatchWorktree(s.LastPath, paths)
		}
		if home == "" {
			continue
		}
		leak := ""
		if edit := agents.MatchWorktree(s.LastEditPath, paths); edit != "" && edit != home && refs[edit].repoPath == refs[home].repoPath {
			leak = refs[edit].branch
		}
		byWt[home] = append(byWt[home], model.AgentSession{
			Tool:       "claude",
			Model:      s.Model,
			State:      agents.DecayState(s.State, s.Liveness, s.ModTime, now, toolTimeout),
			Task:       s.Task,
			Liveness:   s.Liveness,
			SessionID:  s.Path,
			UpdatedAt:  s.ModTime,
			LeakTarget: leak,
		})
	}
	for i := range repos {
		for j := range repos[i].Worktrees {
			repos[i].Worktrees[j].Agents = byWt[repos[i].Worktrees[j].Path]
		}
	}
}

func repoName(path string, roots []string) string {
	for _, root := range roots {
		if rel, err := filepath.Rel(root, path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return filepath.Base(path)
}
