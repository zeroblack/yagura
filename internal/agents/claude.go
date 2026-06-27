package agents

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

type ClaudeSource struct {
	Root         string
	ActiveWindow time.Duration
	ToolTimeout  time.Duration

	mu    sync.Mutex
	cache map[string]cachedTranscript
}

type cachedTranscript struct {
	modTime time.Time
	size    int64
	session Session
}

func NewClaudeSource(root string, activeWindow, toolTimeout time.Duration) *ClaudeSource {
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".claude", "projects")
	}
	if activeWindow <= 0 {
		activeWindow = 10 * time.Minute
	}
	if toolTimeout <= 0 {
		toolTimeout = 30 * time.Second
	}
	return &ClaudeSource{
		Root:         root,
		ActiveWindow: activeWindow,
		ToolTimeout:  toolTimeout,
		cache:        map[string]cachedTranscript{},
	}
}

func (c *ClaudeSource) Collect() []Session {
	now := time.Now()
	bySession := map[string]Session{}
	modBySession := map[string]time.Time{}
	next := map[string]cachedTranscript{}
	_ = filepath.WalkDir(c.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if strings.Contains(path, string(os.PathSeparator)+"subagents"+string(os.PathSeparator)) {
			return nil
		}
		info, serr := os.Stat(path)
		if serr != nil || now.Sub(info.ModTime()) >= c.ActiveWindow {
			return nil
		}
		s, ok := c.lookup(path, info)
		if !ok {
			var perr error
			if s, perr = ParseTranscript(path); perr != nil {
				return nil
			}
		}
		next[path] = cachedTranscript{modTime: info.ModTime(), size: info.Size(), session: s}
		if s.Cwd == "" {
			return nil
		}
		if cur, seen := modBySession[s.Cwd]; seen && !info.ModTime().After(cur) {
			return nil
		}
		s.ModTime = info.ModTime()
		bySession[s.Cwd] = s
		modBySession[s.Cwd] = info.ModTime()
		return nil
	})
	c.mu.Lock()
	c.cache = next
	c.mu.Unlock()

	var out []Session
	for _, s := range bySession {
		s.Liveness = classify(s.ModTime, now, c.ToolTimeout, c.ActiveWindow)
		if s.Liveness == model.LiveIdle {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (c *ClaudeSource) lookup(path string, info os.FileInfo) (Session, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[path]
	if !ok || !e.modTime.Equal(info.ModTime()) || e.size != info.Size() {
		return Session{}, false
	}
	return e.session, true
}

func classify(mtime, now time.Time, toolTimeout, window time.Duration) model.Liveness {
	if mtime.IsZero() {
		return model.LiveIdle
	}
	age := now.Sub(mtime)
	switch {
	case age < toolTimeout:
		return model.LiveActive
	case age < window:
		return model.LiveRecent
	default:
		return model.LiveIdle
	}
}

func MatchWorktree(cwd string, worktrees []string) string {
	best := ""
	for _, w := range worktrees {
		if w == cwd || hasPathPrefix(cwd, w) {
			if len(w) > len(best) {
				best = w
			}
		}
	}
	return best
}

func hasPathPrefix(p, prefix string) bool {
	if len(p) <= len(prefix) {
		return false
	}
	return p[:len(prefix)] == prefix && p[len(prefix)] == os.PathSeparator
}
