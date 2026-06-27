package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/zeroblack/yagura/internal/scan"
)

type snapshotMsg struct {
	snap scan.Snapshot
	full bool
	err  error
}

type tickMsg time.Time

type detailMsg struct {
	mode    detailMode
	wtPath  string
	content string
}

type inspectSettleMsg struct {
	gen int
}

func (m *appModel) loadSnapshot() tea.Cmd {
	scanner := m.scanner
	return func() tea.Msg {
		snap, err := scanner.Take(context.Background())
		return snapshotMsg{snap: snap, full: true, err: err}
	}
}

func (m *appModel) loadAgents() tea.Cmd {
	scanner := m.scanner
	return func() tea.Msg {
		snap, err := scanner.RefreshAgents(context.Background())
		return snapshotMsg{snap: snap, full: false, err: err}
	}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}
