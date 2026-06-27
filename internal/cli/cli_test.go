package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/scan"
)

func testSnapshot(t *testing.T) scan.Snapshot {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "r")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "t@t.io"}, {"config", "user.name", "t"}, {"commit", "--allow-empty", "-qm", "i"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Agents.ClaudeRoot = filepath.Join(t.TempDir(), "none")

	snap, err := scan.NewScanner(cfg).Take(context.Background())
	require.NoError(t, err)
	return snap
}

func TestRenderJSON(t *testing.T) {
	out, err := renderJSON(testSnapshot(t))
	require.NoError(t, err)

	var parsed []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Len(t, parsed, 1)
	require.Equal(t, "r", parsed[0]["name"])
}

func TestRenderTable(t *testing.T) {
	out := renderTable(testSnapshot(t))
	lines := strings.Split(out, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	require.Contains(t, lines[0], "REPO")
	require.Contains(t, lines[0], "BRANCH")
	require.Contains(t, lines[1], "r")
	require.Contains(t, lines[1], "main")
}

func TestDoctorReportsToolchain(t *testing.T) {
	cfg := config.Default()
	cfg.Roots = []string{t.TempDir()}

	var b strings.Builder
	require.NoError(t, doctor(cfg, &b))
	out := b.String()
	require.Contains(t, out, "git")
	require.Contains(t, out, "theme")
	require.Contains(t, out, "root")
}
