package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/theme"
)

// Styles are immutable per theme, so they are built once here instead of on
// every render call; the View is reassembled at the UI tick rate and per-call
// style construction dominated its allocations.
type styleSet struct {
	background color.Color
	fg         []lipgloss.Style
	fgBold     []lipgloss.Style
	chip       []lipgloss.Style
	stateChip  []string
}

func newStyleSet(t theme.Theme) styleSet {
	s := styleSet{
		background: lipgloss.Color(t.Color(theme.RoleBackground)),
		fg:         make([]lipgloss.Style, theme.NumRoles),
		fgBold:     make([]lipgloss.Style, theme.NumRoles),
		chip:       make([]lipgloss.Style, theme.NumRoles),
		stateChip:  make([]string, model.NumAgentStates),
	}
	ink := lipgloss.Color(t.Color(theme.RoleInk))
	for r := range theme.NumRoles {
		color := lipgloss.Color(t.Color(theme.Role(r)))
		s.fg[r] = lipgloss.NewStyle().Foreground(color)
		s.fgBold[r] = lipgloss.NewStyle().Foreground(color).Bold(true)
		s.chip[r] = lipgloss.NewStyle().Background(color).Foreground(ink).Bold(true)
	}
	for i := range model.NumAgentStates {
		state := model.AgentState(i)
		chip := lipgloss.NewStyle().
			Background(lipgloss.Color(t.StateColor(state))).
			Foreground(ink).
			Bold(true)
		s.stateChip[i] = chip.Render(" " + cellPad(state.String(), 8) + " ")
	}
	return s
}
