package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGraphLineCommit(t *testing.T) {
	line := "* \x1fabc1234\x1f1700000000\x1fDioni\x1f9991111\x1fHEAD -> main, origin/main, tag: v1\x1fchore: bump version"
	gl := parseGraphLine(line)
	require.True(t, gl.HasCommit)
	require.Equal(t, "abc1234", gl.Hash)
	require.Equal(t, "Dioni", gl.Author)
	require.Equal(t, "chore: bump version", gl.Subject)
	require.Equal(t, "* ", gl.Graph)
	require.False(t, gl.When.IsZero())
	require.Equal(t, []string{"9991111"}, gl.Parents)
	require.False(t, gl.IsMerge())
	require.Contains(t, gl.Refs, "main")
	require.Contains(t, gl.Refs, "origin/main")
	require.Contains(t, gl.Refs, "v1")
}

func TestParseGraphLineMerge(t *testing.T) {
	line := "*   \x1ffff0001\x1f1700000200\x1fDioni\x1faaa1 bbb2\x1f\x1fMerge branch 'feat/x'"
	gl := parseGraphLine(line)
	require.True(t, gl.IsMerge())
	require.Equal(t, []string{"aaa1", "bbb2"}, gl.Parents)
	require.Equal(t, "Merge branch 'feat/x'", gl.Subject)
}

func TestParseGraphLineConnector(t *testing.T) {
	gl := parseGraphLine("│ ╲")
	require.False(t, gl.HasCommit)
	require.Equal(t, "│ ╲", gl.Graph)
	require.Empty(t, gl.Refs)
}

func TestParseGraphLineMergePrefix(t *testing.T) {
	line := "│ * \x1fdef5678\x1f1700000100\x1fAna\x1fp1\x1f feat/x\x1ffeat: thing"
	gl := parseGraphLine(line)
	require.True(t, gl.HasCommit)
	require.Equal(t, "│ * ", gl.Graph)
	require.Equal(t, "def5678", gl.Hash)
	require.Contains(t, gl.Refs, "feat/x")
}

func TestGraphIntegration(t *testing.T) {
	dir := initRepo(t)
	gitRun(t, dir, "checkout", "-qb", "feat/x")
	writeFile(t, dir, "a.go", "package main\n")
	gitRun(t, dir, "add", "a.go")
	gitRun(t, dir, "commit", "-qm", "feat: add a")

	s := NewService(4, 0)
	lines, err := s.Graph(context.Background(), dir, []string{"main", "feat/x"}, 50)
	require.NoError(t, err)

	var commits int
	foundFeat := false
	for _, gl := range lines {
		if gl.HasCommit {
			commits++
		}
		for _, r := range gl.Refs {
			if r == "feat/x" {
				foundFeat = true
			}
		}
	}
	require.GreaterOrEqual(t, commits, 2)
	require.True(t, foundFeat, "expected a commit decorated with feat/x")
}
