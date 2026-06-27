package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoStats(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, dir, "a.go", "package main\n")
	gitRun(t, dir, "add", "a.go")
	gitRun(t, dir, "commit", "-qm", "second")
	gitRun(t, dir, "branch", "feat/x")

	s := NewService(4, 0)
	st, err := s.RepoStats(context.Background(), dir)
	require.NoError(t, err)
	require.Equal(t, 2, st.Commits)
	require.Equal(t, 2, st.Branches)
	require.Equal(t, "main", st.DefaultBranch)
}
