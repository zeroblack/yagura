package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/zeroblack/yagura/internal/agents"
	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
)

type detailMode int

const (
	detailNone detailMode = iota
	detailStatus
	detailDiff
	detailCommits
	detailAgentLog
)

const agentLogLines = 20

func (m *appModel) openDetail(mode detailMode) tea.Cmd {
	m.detail = mode
	m.detailContent = ""
	m.detailLoading = true
	return m.loadDetailCmd(mode)
}

func (m *appModel) closeDetail() {
	m.detail = detailNone
	m.detailContent = ""
	m.detailLoading = false
}

func (m *appModel) selectedWorktree() *model.Worktree {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].worktree
}

func (m *appModel) selectedWorktreePath() string {
	if w := m.selectedWorktree(); w != nil {
		return w.Path
	}
	return ""
}

func (m *appModel) loadDetailCmd(mode detailMode) tea.Cmd {
	w := m.selectedWorktree()
	if w == nil {
		return func() tea.Msg { return detailMsg{mode: mode, content: "(select a worktree)"} }
	}
	svc := m.gitSvc
	wtPath := w.Path
	sessionID := ""
	if len(w.Agents) > 0 {
		sessionID = w.Agents[0].SessionID
	}
	return func() tea.Msg {
		return detailMsg{mode: mode, wtPath: wtPath, content: fetchDetail(svc, mode, wtPath, sessionID)}
	}
}

func fetchDetail(svc *git.Service, mode detailMode, wtPath, sessionID string) string {
	ctx := context.Background()
	switch mode {
	case detailStatus:
		return textOrError(svc.RunGit(ctx, wtPath, "status", "--short", "--branch"))
	case detailDiff:
		return textOrError(svc.RunGit(ctx, wtPath, "diff", "--stat"))
	case detailCommits:
		return textOrError(svc.RunGit(ctx, wtPath, "log", "-10", "--format=%h%x1f%ct%x1f%an%x1f%s"))
	case detailAgentLog:
		if sessionID == "" {
			return "(no agent session)"
		}
		lines, err := agents.TailLines(sessionID, agentLogLines)
		if err != nil {
			return err.Error()
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

func textOrError(out string, err error) string {
	if err != nil {
		return err.Error()
	}
	return out
}
