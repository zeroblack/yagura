package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

func TestParseTranscriptDerivesStateFromTool(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "session.jsonl")
	lines := `{"cwd":"/repo/wt","gitBranch":"feat/x","message":{"role":"user","content":[{"type":"text","text":"add login"}]}}
{"message":{"role":"assistant","model":"claude-opus","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}
`
	require.NoError(t, os.WriteFile(f, []byte(lines), 0o644))

	s, err := ParseTranscript(f)
	require.NoError(t, err)
	require.Equal(t, "/repo/wt", s.Cwd)
	require.Equal(t, "feat/x", s.Branch)
	require.Equal(t, "claude-opus", s.Model)
	require.Equal(t, model.StateRunning, s.State)
	require.Contains(t, s.Task, "go test")
}

func TestParseTranscriptBrowsing(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "s.jsonl")
	lines := `{"cwd":"/r","message":{"role":"assistant","content":[{"type":"tool_use","name":"WebSearch","input":{"query":"golang"}}]}}
`
	require.NoError(t, os.WriteFile(f, []byte(lines), 0o644))
	s, err := ParseTranscript(f)
	require.NoError(t, err)
	require.Equal(t, model.StateBrowsing, s.State)
}

func TestParseTranscriptLargeFileUsesTailAndProbesHead(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.jsonl")

	var b strings.Builder
	b.WriteString(`{"cwd":"/repo/wt","gitBranch":"feat/big","message":{"role":"assistant","content":[]}}` + "\n")
	filler := `{"message":{"role":"assistant","content":[{"type":"text","text":"` + strings.Repeat("x", 1024) + `"}]}}` + "\n"
	for b.Len() < tailWindowBytes+64*1024 {
		b.WriteString(filler)
	}
	b.WriteString(`{"message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/repo/wt/main.go"}}]}}` + "\n")
	require.NoError(t, os.WriteFile(f, []byte(b.String()), 0o644))

	s, err := ParseTranscript(f)
	require.NoError(t, err)
	require.Equal(t, "/repo/wt", s.Cwd, "cwd outside the tail window must come from the head probe")
	require.Equal(t, "feat/big", s.Branch)
	require.Equal(t, model.StateEditing, s.State)
	require.Equal(t, "/repo/wt/main.go", s.LastPath)
}

func TestParseTranscriptTracksEditPathSeparateFromReads(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "s.jsonl")
	lines := `{"cwd":"/repo/wt","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/repo/wt/a.go"}}]}}
{"message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/other/b.go"}}]}}
`
	require.NoError(t, os.WriteFile(f, []byte(lines), 0o644))

	s, err := ParseTranscript(f)
	require.NoError(t, err)
	require.Equal(t, "/other/b.go", s.LastPath, "any tool path updates LastPath")
	require.Equal(t, "/repo/wt/a.go", s.LastEditPath, "reads must not pollute the edit signal")
}

func TestTailLinesReturnsLastN(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "s.jsonl")
	var b strings.Builder
	for i := range 30 {
		fmt.Fprintf(&b, "line-%02d\n", i)
	}
	require.NoError(t, os.WriteFile(f, []byte(b.String()), 0o644))

	lines, err := TailLines(f, 5)
	require.NoError(t, err)
	require.Equal(t, []string{"line-25", "line-26", "line-27", "line-28", "line-29"}, lines)
}

func TestTruncateIsRuneSafe(t *testing.T) {
	in := strings.Repeat("é", 80)
	out := truncate(in, 20)
	require.True(t, utf8.ValidString(out))
	require.Equal(t, 20, len([]rune(out)))
	require.Equal(t, "…", string([]rune(out)[19]))
	require.Equal(t, "short", truncate("short", 20))
}

func TestTaskLabelPicksToolFields(t *testing.T) {
	require.Equal(t, "running go build", taskLabel("Bash", toolInput{Command: "go build"}))
	require.Equal(t, "editing main.go", taskLabel("Edit", toolInput{FilePath: "/x/main.go"}))
	require.Equal(t, "fetching https://go.dev", taskLabel("WebFetch", toolInput{URL: "https://go.dev"}))
	require.Equal(t, "searching generics", taskLabel("WebSearch", toolInput{Query: "generics"}))
	require.Equal(t, "reading parse.go", taskLabel("Read", toolInput{FilePath: "/y/parse.go"}))
	require.Equal(t, "working on Skill", taskLabel("Skill", toolInput{}))
}
