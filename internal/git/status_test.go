package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

func TestParsePorcelainV2Branch(t *testing.T) {
	raw := join(
		"# branch.oid abc123",
		"# branch.head feat/landing",
		"# branch.upstream origin/feat/landing",
		"# branch.ab +3 -1",
	)
	res := parsePorcelainV2(raw)
	require.Equal(t, "feat/landing", res.Branch)
	require.Equal(t, "origin/feat/landing", res.Upstream)
	require.Equal(t, 3, res.Ahead)
	require.Equal(t, 1, res.Behind)
}

func TestParsePorcelainV2Entries(t *testing.T) {
	raw := join(
		"# branch.head main",
		"1 MM N... 100644 100644 100644 hH hI src/session/share.go",
		"1 A. N... 000000 100644 100644 0000000 hI src/config/sentry.go",
		"1 .D N... 100644 100644 000000 hH hI old/gone.go",
		"1 .M N... 100644 100644 100644 hH hI src/with space.go",
		"? notes/scratch.md",
		"u UU N... 100644 100644 100644 100644 h1 h2 h3 src/conflict.go",
	) + "2 R. N... 100644 100644 100644 hH hI R100 dst/name.go\x00src/name.go\x00"

	res := parsePorcelainV2(raw)
	byPath := map[string]model.FileChange{}
	for _, f := range res.Files {
		byPath[f.Path] = f
	}

	require.Equal(t, model.StatusModified, byPath["src/session/share.go"].Status)
	require.True(t, byPath["src/session/share.go"].Staged)

	require.Equal(t, model.StatusAdded, byPath["src/config/sentry.go"].Status)
	require.True(t, byPath["src/config/sentry.go"].Staged)

	require.Equal(t, model.StatusDeleted, byPath["old/gone.go"].Status)
	require.False(t, byPath["old/gone.go"].Staged)

	require.Equal(t, model.StatusModified, byPath["src/with space.go"].Status)
	require.False(t, byPath["src/with space.go"].Staged)

	require.Equal(t, model.StatusUntracked, byPath["notes/scratch.md"].Status)

	require.Equal(t, model.StatusConflicted, byPath["src/conflict.go"].Status)

	rn := byPath["dst/name.go"]
	require.Equal(t, model.StatusRenamed, rn.Status)
	require.Equal(t, "src/name.go", rn.Orig)
	require.True(t, rn.Staged)
}

func TestParseNumstat(t *testing.T) {
	raw := "42\t8\tsrc/session/share.go\n-\t-\tassets/logo.png\n3\t0\tsrc/with space.go\n"
	d := parseNumstat(raw)
	require.Equal(t, lineDelta{42, 8}, d["src/session/share.go"])
	require.Equal(t, lineDelta{-1, -1}, d["assets/logo.png"])
	require.Equal(t, lineDelta{3, 0}, d["src/with space.go"])
}

func TestParseNumstatRename(t *testing.T) {
	raw := "5\t2\tsrc/{old => new}/file.go\n1\t1\tdst/name.go => moved/name.go\n"
	d := parseNumstat(raw)
	require.Equal(t, lineDelta{5, 2}, d["src/new/file.go"])
	require.Equal(t, lineDelta{1, 1}, d["moved/name.go"])
}

func TestFileStatusCode(t *testing.T) {
	require.Equal(t, "M", model.StatusModified.Code())
	require.Equal(t, "A", model.StatusAdded.Code())
	require.Equal(t, "D", model.StatusDeleted.Code())
	require.Equal(t, "R", model.StatusRenamed.Code())
	require.Equal(t, "?", model.StatusUntracked.Code())
	require.Equal(t, "U", model.StatusConflicted.Code())
}

func TestStatusIntegration(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, dir, "tracked.go", "package main\n")
	gitRun(t, dir, "add", "tracked.go")
	gitRun(t, dir, "commit", "-qm", "add tracked")

	writeFile(t, dir, "tracked.go", "package main\n\nvar x = 1\n")
	writeFile(t, dir, "staged.go", "package main\n")
	gitRun(t, dir, "add", "staged.go")
	writeFile(t, dir, "untracked.md", "notes\n")

	s := NewService(4, 0)
	res, err := s.Status(context.Background(), dir)
	require.NoError(t, err)

	byPath := map[string]model.FileChange{}
	for _, f := range res.Files {
		byPath[f.Path] = f
	}
	require.Equal(t, model.StatusModified, byPath["tracked.go"].Status)
	require.False(t, byPath["tracked.go"].Staged)
	require.Equal(t, 2, byPath["tracked.go"].Added)
	require.True(t, byPath["staged.go"].Staged)
	require.Equal(t, model.StatusUntracked, byPath["untracked.md"].Status)
	require.False(t, byPath["tracked.go"].ModTime.IsZero())
}

func TestDivergenceIntegration(t *testing.T) {
	dir := initRepo(t)
	gitRun(t, dir, "checkout", "-qb", "feat")
	writeFile(t, dir, "a.go", "package main\n")
	gitRun(t, dir, "add", "a.go")
	gitRun(t, dir, "commit", "-qm", "feat commit")

	s := NewService(4, 0)
	div, err := s.Divergence(context.Background(), dir, "main", "feat")
	require.NoError(t, err)
	require.Equal(t, 1, div.Ahead)
	require.Equal(t, 0, div.Behind)
}

func join(lines ...string) string {
	out := ""
	for _, l := range lines {
		out += l + "\x00"
	}
	return out
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "git %v", args)
}
