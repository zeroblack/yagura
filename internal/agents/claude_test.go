package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

func TestMapToWorktreeLongestPrefix(t *testing.T) {
	wts := []string{"/a/repo", "/a/repo/.wt/feat"}
	require.Equal(t, "/a/repo/.wt/feat", MatchWorktree("/a/repo/.wt/feat/src", wts))
	require.Equal(t, "/a/repo", MatchWorktree("/a/repo/cmd", wts))
	require.Equal(t, "", MatchWorktree("/other", wts))
}

func TestClassify(t *testing.T) {
	now := time.Now()
	tt, win := 30*time.Second, 10*time.Minute
	require.Equal(t, model.LiveActive, classify(now.Add(-5*time.Second), now, tt, win))
	require.Equal(t, model.LiveRecent, classify(now.Add(-3*time.Minute), now, tt, win))
	require.Equal(t, model.LiveIdle, classify(now.Add(-30*time.Minute), now, tt, win))
}

func TestDecayState(t *testing.T) {
	now := time.Now()
	require.Equal(t, model.StateRunning, DecayState(model.StateRunning, model.LiveActive, now.Add(-5*time.Second), now, 30*time.Second))
	require.Equal(t, model.StateWaiting, DecayState(model.StateRunning, model.LiveActive, now.Add(-2*time.Minute), now, 30*time.Second))
	require.Equal(t, model.StateIdle, DecayState(model.StateRunning, model.LiveRecent, now.Add(-5*time.Second), now, 30*time.Second))
}

func paddedLine(json string, width int) string {
	return json + strings.Repeat(" ", width-len(json)-1) + "\n"
}

func writeTranscript(t *testing.T, path, line string, mtime time.Time) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(line), 0o644))
	require.NoError(t, os.Chtimes(path, mtime, mtime))
}

func TestCollectFindsFreshSessions(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	running := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test"}}]}}`
	writeTranscript(t, filepath.Join(root, "proj", "s1.jsonl"), paddedLine(running, 400), now)
	writeTranscript(t, filepath.Join(root, "proj", "stale.jsonl"), paddedLine(running, 400), now.Add(-time.Hour))
	writeTranscript(t, filepath.Join(root, "proj", "subagents", "sub.jsonl"), paddedLine(running, 400), now)

	src := NewClaudeSource(root, 10*time.Minute, 30*time.Second)
	out := src.Collect()
	require.Len(t, out, 1, "stale transcripts and subagents must be excluded")
	require.Equal(t, "/repo/wt", out[0].Cwd)
	require.Equal(t, model.StateRunning, out[0].State)
	require.Equal(t, model.LiveActive, out[0].Liveness)
}

func TestCollectReusesCacheWhileMtimeUnchanged(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "proj", "s1.jsonl")
	mtime := time.Now().Add(-time.Minute)
	running := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test"}}]}}`
	editing := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/repo/wt/a.go"}}]}}`
	writeTranscript(t, path, paddedLine(running, 400), mtime)

	src := NewClaudeSource(root, 10*time.Minute, 30*time.Second)
	first := src.Collect()
	require.Len(t, first, 1)
	require.Equal(t, model.StateRunning, first[0].State)

	writeTranscript(t, path, paddedLine(editing, 400), mtime)
	cached := src.Collect()
	require.Len(t, cached, 1)
	require.Equal(t, model.StateRunning, cached[0].State, "same mtime and size must serve the cached session")

	writeTranscript(t, path, paddedLine(editing, 400), mtime.Add(time.Second))
	refreshed := src.Collect()
	require.Len(t, refreshed, 1)
	require.Equal(t, model.StateEditing, refreshed[0].State, "a new mtime must invalidate the cache")
}

func TestCollectKeepsNewestSessionPerCwd(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	older := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go vet"}}]}}`
	newer := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/repo/wt/b.go"}}]}}`
	writeTranscript(t, filepath.Join(root, "proj", "old.jsonl"), paddedLine(older, 400), now.Add(-5*time.Minute))
	writeTranscript(t, filepath.Join(root, "proj", "new.jsonl"), paddedLine(newer, 400), now)

	src := NewClaudeSource(root, 10*time.Minute, 30*time.Second)
	out := src.Collect()
	require.Len(t, out, 1)
	require.Equal(t, model.StateEditing, out[0].State)
}
