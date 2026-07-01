package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/theme"
)

func (m *appModel) toggleHelp() {
	if m.showHelp {
		m.showHelp = false
		return
	}
	if m.detail != detailNone {
		m.closeDetail()
	}
	m.showHelp = true
}

type helpRow struct {
	keys  []string
	label string
}

func helpRows(k config.KeysConfig) []helpRow {
	return []helpRow{
		{[]string{k.Inspect}, "inspect worktree"},
		{k.FocusNext, "focus next pane"},
		{k.FocusPrev, "focus prev pane"},
		{joinKeys(k.PaneList, k.PaneChange, k.PaneGraph, k.PanePR), "jump to pane"},
		{k.Up, "scroll up"},
		{k.Down, "scroll down"},
		{k.PageUp, "page up"},
		{k.PageDown, "page down"},
		{k.Home, "jump to top"},
		{k.End, "jump to bottom"},
		{k.Pin, "pin worktree"},
		{k.Group, "toggle grouping"},
		{k.Diff, "diff --stat"},
		{k.Commits, "recent commits"},
		{k.Status, "git status"},
		{k.AgentLog, "agent log"},
		{k.Refresh, "force sync"},
		{k.Close, "close panel"},
		{k.Help, "toggle this help"},
		{k.Quit, "quit"},
	}
}

func joinKeys(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func renderKeys(keys []string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		parts = append(parts, keyGlyph(key))
	}
	return strings.Join(parts, " / ")
}

func (m *appModel) helpPanel() string {
	rows := helpRows(m.cfg.Keys)
	rendered := make([]string, len(rows))
	keyCol := 0
	for i, r := range rows {
		rendered[i] = renderKeys(r.keys)
		if w := lipgloss.Width(rendered[i]); w > keyCol {
			keyCol = w
		}
	}
	var body strings.Builder
	for i, r := range rows {
		if i > 0 {
			body.WriteByte('\n')
		}
		keys := m.fgBold(theme.RoleAmber).Render(cellPad(rendered[i], keyCol))
		body.WriteString(keys + "  " + m.fg(theme.RoleTextMuted).Render(r.label))
	}
	return m.consoleBox("keybindings", body.String())
}
