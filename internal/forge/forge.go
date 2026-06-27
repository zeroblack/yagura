package forge

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type Provider interface {
	Available(ctx context.Context) bool
	PRs(ctx context.Context) ([]model.PRInfo, error)
}

type Manager struct {
	enabled string
	ttl     time.Duration
	run     runner
	limit   int

	mu    sync.Mutex
	cache map[string]entry
}

type entry struct {
	prs []model.PRInfo
	at  time.Time
}

func NewManager(enabled string, ttl time.Duration) *Manager {
	return newManager(enabled, ttl, execRunner)
}

func newManager(enabled string, ttl time.Duration, run runner) *Manager {
	return &Manager{enabled: enabled, ttl: ttl, run: run, limit: 100, cache: map[string]entry{}}
}

func (man *Manager) PRs(ctx context.Context, repoPath string) []model.PRInfo {
	if man == nil || man.enabled == "off" {
		return nil
	}
	man.mu.Lock()
	if e, ok := man.cache[repoPath]; ok && time.Since(e.at) < man.ttl {
		man.mu.Unlock()
		return e.prs
	}
	man.mu.Unlock()

	prov := man.detect(ctx, repoPath)
	if prov == nil {
		return nil
	}
	if man.enabled != "on" && !prov.Available(ctx) {
		return nil
	}
	prs, err := prov.PRs(ctx)
	if err != nil {
		return nil
	}
	man.mu.Lock()
	man.cache[repoPath] = entry{prs: prs, at: time.Now()}
	man.mu.Unlock()
	return prs
}

func (man *Manager) detect(ctx context.Context, repoPath string) Provider {
	url := remoteURL(ctx, man.run, repoPath)
	switch {
	case strings.Contains(url, "github.com"):
		return &ghProvider{repo: githubSlug(url), run: man.run, limit: man.limit}
	default:
		return nil
	}
}

func remoteURL(ctx context.Context, run runner, repoPath string) string {
	out, err := run(ctx, "git", "-C", repoPath, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func githubSlug(url string) string {
	s := url
	if i := strings.Index(s, "github.com"); i >= 0 {
		s = s[i+len("github.com"):]
	}
	s = strings.TrimLeft(s, ":/")
	s = strings.TrimSuffix(s, ".git")
	return s
}
