package tui

import tea "charm.land/bubbletea/v2"

func (m *appModel) visibleFocus() []paneFocus {
	order := []paneFocus{focusList}
	if m.prsVisible() {
		order = append(order, focusPRs)
	}
	return append(order, focusChanges, focusGraph)
}

func (m *appModel) cycleFocus(dir int) {
	order := m.visibleFocus()
	idx := 0
	for i, f := range order {
		if f == m.focus {
			idx = i
			break
		}
	}
	n := len(order)
	m.focus = order[((idx+dir)%n+n)%n]
}

func (m *appModel) scrollUp() tea.Cmd {
	if m.focus == focusList {
		m.stepCursor(-1)
		m.syncCursorPath()
		return m.refreshInspect()
	}
	m.scrollFocused(-1)
	return nil
}

func (m *appModel) scrollDown() tea.Cmd {
	if m.focus == focusList {
		m.stepCursor(1)
		m.syncCursorPath()
		return m.refreshInspect()
	}
	m.scrollFocused(1)
	return nil
}

func (m *appModel) scrollFocused(delta int) {
	switch m.focus {
	case focusChanges:
		m.changesOff = max(0, m.changesOff+delta)
	case focusGraph:
		m.graphOff = max(0, m.graphOff+delta)
	case focusPRs:
		m.prsOff = max(0, m.prsOff+delta)
	}
}

func (m *appModel) scrollHome() tea.Cmd {
	switch m.focus {
	case focusChanges:
		m.changesOff = 0
	case focusGraph:
		m.graphOff = 0
	case focusPRs:
		m.prsOff = 0
	default:
		m.cursor = m.firstFocusableRow()
		m.syncCursorPath()
		return m.refreshInspect()
	}
	return nil
}

func (m *appModel) scrollEnd() tea.Cmd {
	switch m.focus {
	case focusChanges:
		m.changesOff = 1 << 30
	case focusGraph:
		m.graphOff = 1 << 30
	case focusPRs:
		m.prsOff = 1 << 30
	default:
		m.cursor = max(0, len(m.rows)-1)
		for m.cursor > 0 && !m.rows[m.cursor].focusable() {
			m.cursor--
		}
		m.syncCursorPath()
		return m.refreshInspect()
	}
	return nil
}
