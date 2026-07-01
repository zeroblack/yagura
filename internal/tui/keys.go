package tui

import tea "charm.land/bubbletea/v2"

const pageScrollLines = 10

func (m *appModel) handleKey(s string) tea.Cmd {
	act := m.keymap[s]
	if act == actQuit {
		return tea.Quit
	}
	if act == actHelp {
		m.toggleHelp()
		return nil
	}
	if m.showHelp {
		if act == actClose {
			m.showHelp = false
		}
		return nil
	}
	if m.detail != detailNone {
		if act == actClose {
			m.closeDetail()
		}
		return nil
	}
	switch act {
	case actInspect, actFocusNext:
		m.cycleFocus(1)
	case actFocusPrev:
		m.cycleFocus(-1)
	case actPaneList:
		m.focus = focusList
	case actPaneChanges:
		m.focus = focusChanges
	case actPaneGraph:
		m.focus = focusGraph
	case actPanePR:
		if m.prsVisible() {
			m.focus = focusPRs
		}
	case actUp:
		return m.scrollUp()
	case actDown:
		return m.scrollDown()
	case actPageUp:
		m.scrollFocused(-pageScrollLines)
	case actPageDown:
		m.scrollFocused(pageScrollLines)
	case actHome:
		return m.scrollHome()
	case actEnd:
		return m.scrollEnd()
	case actPin:
		return m.togglePin()
	case actGroup:
		m.toggleGrouping()
		return m.refreshInspect()
	case actRefresh:
		return m.forceRefresh()
	case actDiff:
		return m.openDetail(detailDiff)
	case actCommits:
		return m.openDetail(detailCommits)
	case actStatus:
		return m.openDetail(detailStatus)
	case actAgentLog:
		return m.openDetail(detailAgentLog)
	}
	return nil
}
