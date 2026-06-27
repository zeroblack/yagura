package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLatestFileMod(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "p"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "p", "big.txt"), []byte("y"), 0o644))

	future := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "node_modules", "p", "big.txt"), future, future))

	got := LatestFileMod(dir, map[string]bool{"node_modules": true, ".git": true}, 4)
	require.WithinDuration(t, time.Now(), got, 5*time.Second)
}
