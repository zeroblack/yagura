package forge

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/zeroblack/yagura/internal/execx"
	"github.com/zeroblack/yagura/internal/model"
)

// errTTL is the retry backoff after a failed fetch. Without it a repo whose gh
// call fails (offline, unauthenticated, rate-limited) would re-spawn the whole
// git+gh chain on every inspector refresh, a few seconds apart, forever.
const errTTL = 30 * time.Second

const DefaultTimeout = 30 * time.Second

type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func hardenedRunner(waitDelay time.Duration) runner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		execx.Harden(cmd, waitDelay)
		return cmd.Output()
	}
}

type Provider interface {
	Available(ctx context.Context) bool
	PRs(ctx context.Context) ([]model.PRInfo, error)
}

type Manager struct {
	enabled string
	ttl     time.Duration
	timeout time.Duration
	run     runner
	limit   int
	now     func() time.Time

	mu       sync.Mutex
	cache    map[string]entry
	remotes  map[string]string
	inflight map[string]bool
}

type entry struct {
	prs    []model.PRInfo
	at     time.Time
	failed bool
}

func NewManager(enabled string, ttl, timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	man := newManager(enabled, ttl, hardenedRunner(timeout))
	man.timeout = timeout
	return man
}

func newManager(enabled string, ttl time.Duration, run runner) *Manager {
	man := &Manager{
		enabled:  enabled,
		ttl:      ttl,
		timeout:  DefaultTimeout,
		limit:    100,
		now:      time.Now,
		cache:    map[string]entry{},
		remotes:  map[string]string{},
		inflight: map[string]bool{},
	}
	man.run = man.bound(run)
	return man
}

// bound gives every subprocess its own deadline; without one a hung gh call
// (network stall, credential prompt) would live for the process lifetime.
func (man *Manager) bound(run runner) runner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(ctx, man.timeout)
		defer cancel()
		return run(ctx, name, args...)
	}
}

// PRs returns the cached pull requests for the repo, refreshing them when the
// cache entry expired. While a refresh is in flight concurrent callers get the
// last known value instead of stacking additional subprocesses.
func (man *Manager) PRs(ctx context.Context, repoPath string) []model.PRInfo {
	if man == nil || man.enabled == "off" {
		return nil
	}
	man.mu.Lock()
	e, ok := man.cache[repoPath]
	if ok && man.now().Sub(e.at) < man.entryTTL(e) {
		man.mu.Unlock()
		return e.prs
	}
	if man.inflight[repoPath] {
		man.mu.Unlock()
		return e.prs
	}
	man.inflight[repoPath] = true
	man.mu.Unlock()
	defer func() {
		man.mu.Lock()
		delete(man.inflight, repoPath)
		man.mu.Unlock()
	}()

	prov := man.detect(ctx, repoPath)
	if prov == nil {
		return nil
	}
	if man.enabled != "on" && !prov.Available(ctx) {
		man.store(repoPath, nil, true)
		return nil
	}
	prs, err := prov.PRs(ctx)
	if err != nil {
		man.store(repoPath, nil, true)
		return nil
	}
	man.store(repoPath, prs, false)
	return prs
}

func (man *Manager) entryTTL(e entry) time.Duration {
	if e.failed {
		return errTTL
	}
	return man.ttl
}

func (man *Manager) store(repoPath string, prs []model.PRInfo, failed bool) {
	man.mu.Lock()
	man.cache[repoPath] = entry{prs: prs, at: man.now(), failed: failed}
	man.mu.Unlock()
}

// detect resolves the repo's forge once per process: origin URLs do not change
// under a running monitor, and re-running git remote per refresh was pure churn.
func (man *Manager) detect(ctx context.Context, repoPath string) Provider {
	man.mu.Lock()
	url, ok := man.remotes[repoPath]
	man.mu.Unlock()
	if !ok {
		url = remoteURL(ctx, man.run, repoPath)
		man.mu.Lock()
		man.remotes[repoPath] = url
		man.mu.Unlock()
	}
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
