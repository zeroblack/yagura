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
// semaphore, the agent source transcript cache, the worktree skeleton from the
// last full scan (so agent-only refreshes skip discovery/git/file walks) and a
// per-worktree last-commit cache keyed by HEAD.
type Scanner struct {
	cfg config.Config
	git *git.Service
	src agents.Source

	mu      sync.Mutex
	cached  []model.Repo
	commits map[string]commitCache
}

type commitCache struct {
	head   string
	commit time.Time
}

func NewScanner(cfg config.Config) *Scanner {
	return &Scanner{
		cfg:     cfg,
		git:     git.NewService(cfg.Git.MaxProcs, cfg.Git.Timeout),
		src:     agents.NewClaudeSource(cfg.Agents.ClaudeRoot, cfg.Agents.ActiveWindow, cfg.Agents.ToolTimeout),
		commits: map[string]commitCache{},
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

	s.mu.Lock()
	s.cached = cloneRepos(repos)
	s.mu.Unlock()

	attachAgents(repos, s.src, s.cfg.Agents.ToolTimeout)
	activity.SortRepos(repos)
	return Snapshot{Repos: repos}, nil
}

// RefreshAgents re-attaches live agent sessions to the worktree skeleton from
// the last full Take, skipping discovery, git execs and the file-mod walk. It
// keeps agent liveness responsive on the fast tick while the expensive repo
// scan only runs on the slow full tick. It falls back to a full Take before the
// first snapshot exists.
func (s *Scanner) RefreshAgents(ctx context.Context) (Snapshot, error) {
	s.mu.Lock()
	cached := s.cached
	s.mu.Unlock()
	if len(cached) == 0 {
		return s.Take(ctx)
	}
	repos := cloneRepos(cached)
	attachAgents(repos, s.src, s.cfg.Agents.ToolTimeout)
	activity.SortRepos(repos)
	return Snapshot{Repos: repos}, nil
}

// cloneRepos copies the repo/worktree structure so agent attachment on a refresh
// never mutates the cached skeleton or a snapshot the UI still holds.
func cloneRepos(src []model.Repo) []model.Repo {
	out := make([]model.Repo, len(src))
	for i, r := range src {
		out[i] = r
		out[i].Worktrees = make([]model.Worktree, len(r.Worktrees))
		copy(out[i].Worktrees, r.Worktrees)
		for j := range out[i].Worktrees {
			out[i].Worktrees[j].Agents = nil
		}
	}
	return out
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
		w.LastCommit = s.lastCommit(ctx, w.Path, w.Head)
		w.LastFileMod = worktree.LatestFileMod(w.Path, ignore, s.cfg.MaxDepth)
	}
	return r
}

// lastCommit reuses the cached commit time when HEAD hasn't moved since the
// previous scan, avoiding a git log exec per worktree on every full scan.
func (s *Scanner) lastCommit(ctx context.Context, wtPath, head string) time.Time {
	if head != "" {
		s.mu.Lock()
		c, ok := s.commits[wtPath]
		s.mu.Unlock()
		if ok && c.head == head {
			return c.commit
		}
	}
	t := worktree.LastCommit(ctx, s.git, wtPath)
	if head != "" {
		s.mu.Lock()
		s.commits[wtPath] = commitCache{head: head, commit: t}
		s.mu.Unlock()
	}
	return t
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
