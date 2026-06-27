package agents

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/zeroblack/yagura/internal/model"
)

// Transcripts grow to hundreds of MB while the session state lives in the
// most recent events, so only the tail window is parsed. The head probe
// covers the rare transcript whose envelope metadata (cwd) never repeats
// inside the tail.
const (
	tailWindowBytes = 256 * 1024
	headProbeBytes  = 64 * 1024
	maxLineBytes    = 8 * 1024 * 1024
)

type envelope struct {
	Cwd       string `json:"cwd"`
	GitBranch string `json:"gitBranch"`
	Message   struct {
		Role    string          `json:"role"`
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type toolInput struct {
	Command      string `json:"command"`
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
	Path         string `json:"path"`
	Pattern      string `json:"pattern"`
	URL          string `json:"url"`
	Query        string `json:"query"`
}

func ParseTranscript(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	s := Session{Path: path, State: model.StateIdle}
	offset, err := seekTail(f, tailWindowBytes)
	if err != nil {
		return Session{}, err
	}
	sc := newLineScanner(f)
	if offset > 0 {
		sc.Scan()
	}
	for sc.Scan() {
		applyLine(&s, sc.Bytes())
	}
	if err := sc.Err(); err != nil {
		return s, err
	}
	if s.Cwd == "" && offset > 0 {
		probeHead(f, &s)
	}
	return s, nil
}

func TailLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	offset, err := seekTail(f, tailWindowBytes)
	if err != nil {
		return nil, err
	}
	sc := newLineScanner(f)
	if offset > 0 {
		sc.Scan()
	}
	var lines []string
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, sc.Err()
}

func seekTail(f *os.File, window int64) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	offset := fi.Size() - window
	if offset <= 0 {
		return 0, nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return offset, nil
}

func newLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	return sc
}

func applyLine(s *Session, line []byte) {
	var e envelope
	if json.Unmarshal(line, &e) != nil {
		return
	}
	if e.Cwd != "" {
		s.Cwd = e.Cwd
	}
	if e.GitBranch != "" {
		s.Branch = e.GitBranch
	}
	if e.Message.Model != "" {
		s.Model = e.Message.Model
	}
	var blocks []contentBlock
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return
	}
	for _, b := range blocks {
		switch b.Type {
		case "tool_use":
			s.State = stateForTool(b.Name)
			in := decodeInput(b.Input)
			s.Task = taskLabel(b.Name, in)
			if p := in.lastPath(); p != "" {
				s.LastPath = p
				if isEditTool(b.Name) {
					s.LastEditPath = p
				}
			}
		case "text":
			if e.Message.Role == "user" && b.Text != "" {
				s.Task = truncate(b.Text, 60)
			}
		}
	}
}

func probeHead(f *os.File, s *Session) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return
	}
	sc := newLineScanner(io.LimitReader(f, headProbeBytes))
	for sc.Scan() {
		var e envelope
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if s.Cwd == "" && e.Cwd != "" {
			s.Cwd = e.Cwd
		}
		if s.Branch == "" && e.GitBranch != "" {
			s.Branch = e.GitBranch
		}
		if s.Cwd != "" && s.Branch != "" {
			return
		}
	}
}

func decodeInput(raw json.RawMessage) toolInput {
	var in toolInput
	if json.Unmarshal(raw, &in) != nil {
		return toolInput{}
	}
	return in
}

func (in toolInput) lastPath() string {
	for _, p := range []string{in.FilePath, in.NotebookPath, in.Path} {
		if strings.HasPrefix(p, "/") {
			return p
		}
	}
	return ""
}

func isEditTool(name string) bool {
	switch name {
	case "Edit", "Write", "MultiEdit", "NotebookEdit":
		return true
	}
	return false
}

func stateForTool(name string) model.AgentState {
	switch name {
	case "Bash", "BashOutput":
		return model.StateRunning
	case "Edit", "Write", "NotebookEdit", "MultiEdit":
		return model.StateEditing
	case "WebFetch", "WebSearch", "Read", "Grep", "Glob":
		return model.StateBrowsing
	default:
		return model.StateThinking
	}
}

func taskLabel(name string, in toolInput) string {
	switch name {
	case "Bash":
		return "running " + truncate(in.Command, 80)
	case "Edit", "Write", "MultiEdit":
		return "editing " + truncate(baseName(in.FilePath), 40)
	case "WebFetch":
		return "fetching " + truncate(in.URL, 50)
	case "WebSearch":
		return "searching " + truncate(in.Query, 50)
	case "Read", "Grep", "Glob":
		return "reading " + truncate(baseName(cmp.Or(in.FilePath, in.Pattern, in.Path)), 40)
	default:
		return "working on " + name
	}
}

func baseName(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 && i < len(p)-1 {
		return p[i+1:]
	}
	return p
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
