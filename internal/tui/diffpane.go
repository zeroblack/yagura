package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
)

func newestChangedFile(files []model.FileChange) *model.FileChange {
	if len(files) == 0 {
		return nil
	}
	newest := &files[0]
	for i := 1; i < len(files); i++ {
		if files[i].ModTime.After(newest.ModTime) {
			newest = &files[i]
		}
	}
	return newest
}

func diffContentFor(svc *git.Service, wtPath string, f model.FileChange) (string, error) {
	if f.Binary() {
		return "(binary file)", nil
	}
	if f.Status == model.StatusUntracked {
		return syntheticDiff(wtPath, f.Path)
	}
	return svc.Diff(context.Background(), wtPath, f.Path, f.Staged)
}

func syntheticDiff(wtPath, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(wtPath, path))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i, l := range lines {
		lines[i] = "+" + l
	}
	return strings.Join(lines, "\n"), nil
}

type inspectDiffMsg struct {
	wtPath  string
	path    string
	content string
	err     error
}

func loadInspectDiff(svc *git.Service, wtPath string, f model.FileChange) tea.Cmd {
	return func() tea.Msg {
		content, err := diffContentFor(svc, wtPath, f)
		return inspectDiffMsg{wtPath: wtPath, path: f.Path, content: content, err: err}
	}
}

func (m *appModel) followNewestDiff() tea.Cmd {
	if m.inspect.wt == nil {
		return nil
	}
	f := newestChangedFile(m.inspect.status.Files)
	if f == nil {
		m.inspect.diffTarget = ""
		m.inspect.diffContent = ""
		m.inspect.diffErr = nil
		m.inspect.diffLoaded = true
		m.inspect.diffModTime = time.Time{}
		return nil
	}
	m.inspect.diffTarget = f.Path
	m.inspect.diffLoaded = false
	m.inspect.diffModTime = f.ModTime
	return loadInspectDiff(m.gitSvc, m.inspect.wt.Path, *f)
}

func (m *appModel) diffIsStale() bool {
	if m.inspect.diffModTime.IsZero() {
		return false
	}
	if m.inspect.wt != nil {
		for _, a := range m.inspect.wt.Agents {
			if a.Liveness.IsLive() {
				return false
			}
		}
	}
	return time.Since(m.inspect.diffModTime) > m.cfg.Inspect.DiffStaleAfter
}
