package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func mkGitRepo(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(path, ".git"), 0o755))
}

func TestFindReposNestedDepths(t *testing.T) {
	root := t.TempDir()
	mkGitRepo(t, filepath.Join(root, "a"))
	mkGitRepo(t, filepath.Join(root, "grp", "b"))
	mkGitRepo(t, filepath.Join(root, "grp", "b", "sub"))
	mkGitRepo(t, filepath.Join(root, "x", "y", "z"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "grp", "b", "node_modules", "pkg"), 0o755))

	got, err := Find([]string{root}, Options{MaxDepth: 4, Ignore: []string{"node_modules"}})
	require.NoError(t, err)

	var paths []string
	for _, p := range got {
		rel, _ := filepath.Rel(root, p)
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	require.Equal(t, []string{"a", "grp/b", "x/y/z"}, paths)
}

func TestFindRootIsRepo(t *testing.T) {
	root := t.TempDir()
	mkGitRepo(t, root)
	got, err := Find([]string{root}, Options{MaxDepth: 4})
	require.NoError(t, err)
	require.Len(t, got, 1)
	resolved, _ := filepath.EvalSymlinks(got[0])
	wantResolved, _ := filepath.EvalSymlinks(root)
	require.Equal(t, wantResolved, resolved)
}
