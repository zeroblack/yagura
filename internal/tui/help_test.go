package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
)

func TestHelpToggle(t *testing.T) {
	m := New(config.Default())

	m.handleKey("?")
	require.True(t, m.showHelp, "? opens the help overlay")

	m.handleKey("?")
	require.False(t, m.showHelp, "? again closes it")

	m.handleKey("?")
	m.handleKey("esc")
	require.False(t, m.showHelp, "close key dismisses the overlay")
}

func TestHelpClosesOpenDetail(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(inspectSnap())
	m.detail = detailDiff

	m.handleKey("?")
	require.True(t, m.showHelp)
	require.Equal(t, detailNone, m.detail, "opening help dismisses any detail panel")
}

func TestHelpPanelListsEveryBinding(t *testing.T) {
	m := New(config.Default())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	out := stripANSI(m.helpPanel())
	for _, want := range []string{"inspect worktree", "q / ctrl+c", "quit", "toggle this help", "force sync", "jump to pane"} {
		require.Contains(t, out, want)
	}
}
