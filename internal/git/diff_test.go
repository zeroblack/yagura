package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffUnstagedShowsChangedLines(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, dir, "tracked.go", "package main\n")
	gitRun(t, dir, "add", "tracked.go")
	gitRun(t, dir, "commit", "-qm", "add tracked")
	writeFile(t, dir, "tracked.go", "package main\n\nvar x = 1\n")

	s := NewService(4, 0)
	out, err := s.Diff(context.Background(), dir, "tracked.go", false)
	require.NoError(t, err)
	require.Contains(t, out, "+var x = 1")
}

func TestDiffStagedOnlyShowsIndexChanges(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, dir, "tracked.go", "package main\n")
	gitRun(t, dir, "add", "tracked.go")
	gitRun(t, dir, "commit", "-qm", "add tracked")
	writeFile(t, dir, "tracked.go", "package main\n\nvar x = 1\n")
	gitRun(t, dir, "add", "tracked.go")

	s := NewService(4, 0)
	unstaged, err := s.Diff(context.Background(), dir, "tracked.go", false)
	require.NoError(t, err)
	require.Empty(t, unstaged, "the change is staged, so the unstaged diff must be empty")

	staged, err := s.Diff(context.Background(), dir, "tracked.go", true)
	require.NoError(t, err)
	require.Contains(t, staged, "+var x = 1")
}

func TestDiffUnchangedFileIsEmpty(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, dir, "tracked.go", "package main\n")
	gitRun(t, dir, "add", "tracked.go")
	gitRun(t, dir, "commit", "-qm", "add tracked")

	s := NewService(4, 0)
	out, err := s.Diff(context.Background(), dir, "tracked.go", false)
	require.NoError(t, err)
	require.Empty(t, out)
}
