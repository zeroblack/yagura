package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	c := Default()
	require.Equal(t, 4, c.MaxDepth)
	require.Contains(t, c.Ignore, "node_modules")
	require.Equal(t, "evangelion", c.Theme)
	require.Equal(t, 5*time.Second, c.Refresh.Tick)
	require.Equal(t, 10*time.Second, c.Git.Timeout)
	require.Equal(t, 0, c.Git.MaxProcs)
	require.Contains(t, c.Agents.ClaudeRoot, ".claude")
	require.Equal(t, []string{"q", "ctrl+c"}, c.Keys.Quit)
	require.Equal(t, []string{"down", "j"}, c.Keys.Down)
}

func TestLoadGitSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "git:\n  max_procs: 8\n  timeout: 3s\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, 8, c.Git.MaxProcs)
	require.Equal(t, 3*time.Second, c.Git.Timeout)
}

func TestLoadKeyOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "keys:\n  quit: [x]\n  diff: [D, f]\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, []string{"x"}, c.Keys.Quit)
	require.Equal(t, []string{"D", "f"}, c.Keys.Diff)
	require.Equal(t, []string{"down", "j"}, c.Keys.Down, "unset keys keep their defaults")
}

func TestLoadExpandsHomeInRoots(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("roots:\n  - ~/code\nagents:\n  claude_root: ~/.claude/projects\n"), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	home, herr := os.UserHomeDir()
	require.NoError(t, herr)
	require.Equal(t, []string{filepath.Join(home, "code")}, c.Roots)
	require.Equal(t, filepath.Join(home, ".claude", "projects"), c.Agents.ClaudeRoot)
}

func TestLoadMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("max_depth: 2\ntheme: magi\nroots:\n  - /tmp/x\n"), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, 2, c.MaxDepth)
	require.Equal(t, "magi", c.Theme)
	require.Equal(t, []string{"/tmp/x"}, c.Roots)
	require.Contains(t, c.Ignore, "node_modules")
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	require.Equal(t, "evangelion", c.Theme)
}

func TestInspectForgeDefaults(t *testing.T) {
	c := Default()
	require.Equal(t, "mtime", c.Inspect.ChangesSort)
	require.Equal(t, 40, c.Inspect.FileLimit)
	require.Equal(t, 200, c.Inspect.GraphMax)
	require.True(t, c.Inspect.AttentionFirst)
	require.Equal(t, "auto", c.Forge.Enabled)
	require.Equal(t, 60*time.Second, c.Forge.TTL)
	require.Equal(t, "enter", c.Keys.Inspect)
}

func TestLoadInspectAndForge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "inspect:\n  changes_sort: status\n  graph_max: 50\n  file_limit: 10\n  attention_first: false\nforge:\n  enabled: \"off\"\n  ttl: 2m\nkeys:\n  inspect: space\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "status", c.Inspect.ChangesSort)
	require.Equal(t, 50, c.Inspect.GraphMax)
	require.Equal(t, 10, c.Inspect.FileLimit)
	require.False(t, c.Inspect.AttentionFirst)
	require.Equal(t, "off", c.Forge.Enabled)
	require.Equal(t, 2*time.Minute, c.Forge.TTL)
	require.Equal(t, "space", c.Keys.Inspect)
}

func TestLoadParsesDurations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "refresh:\n  tick: 3s\nagents:\n  active_window: 5m\n  tool_timeout: 45s\nsort:\n  group_by_repo: false\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	c, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, 3*time.Second, c.Refresh.Tick)
	require.Equal(t, 5*time.Minute, c.Agents.ActiveWindow)
	require.Equal(t, 45*time.Second, c.Agents.ToolTimeout)
	require.False(t, c.Sort.GroupByRepo)
}
