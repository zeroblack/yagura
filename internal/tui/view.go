package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zeroblack/yagura/internal/activity"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/theme"
	"github.com/zeroblack/yagura/internal/version"
)

const (
	colBranch = 22
	colTag    = 12

	commitMetaWidth = 30
	graphMetaWidth  = 58
)

var brailleFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧")

func (m *appModel) fg(r theme.Role) lipgloss.Style     { return m.styles.fg[r] }
func (m *appModel) fgBold(r theme.Role) lipgloss.Style { return m.styles.fgBold[r] }

func (m *appModel) chip(r theme.Role, text string) string {
	if st, ok := m.fx.pulseChipStyle(r, m.beat); ok {
		return st.Render(text)
	}
	return m.styles.chip[r].Render(text)
}

func (m *appModel) stateChip(s model.AgentState) string {
	if s == model.StateRunning {
		if c, ok := m.fx.runStateChip(m.beat); ok {
			return c
		}
	}
	return m.styles.stateChip[s]
}

func (m *appModel) dimIdle(r theme.Role, active bool) (lipgloss.Style, bool) {
	if active {
		return lipgloss.Style{}, false
	}
	return m.fx.dimStyle(r)
}

func (m *appModel) headerRule() string {
	if m.fx.headerRule != "" {
		return m.fx.headerRule
	}
	return m.fg(theme.RoleAmber).Render(strings.Repeat("━", m.width))
}

func (m *appModel) cardRule() string {
	if m.fx.cardRule != "" {
		return m.fx.cardRule
	}
	return m.fg(theme.RoleRule).Render(strings.Repeat("─", max(0, m.width-2)))
}

func (m *appModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.BackgroundColor = m.styles.background
	return v
}

func (m *appModel) render() string {
	if m.width == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(m.width * 256)
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(m.headerRule())
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(" " + m.chip(theme.RoleError, " ERROR ") + " " + m.fg(theme.RoleError).Render(m.err.Error()) + "\n")
	}
	if m.detail != detailNone {
		for i, r := range m.rows {
			b.WriteString(m.renderRow(i, r))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(m.detailPanel())
	} else {
		b.WriteString(m.renderInspectorBody())
	}
	b.WriteString("\n")
	b.WriteString(m.footerLine)
	return b.String()
}

func (m *appModel) header() string {
	left := " " + m.headerBrand() + m.fg(theme.RoleLine).Render("   ") + m.headerCounts()
	right := m.syncBadge() + " "
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func (m *appModel) headerBrand() string {
	return m.chip(theme.RoleAmber, " YAGURA ") +
		m.fg(theme.RoleSha).Render(" v"+version.Version) +
		m.fg(theme.RoleTextMuted).Render("  worktree monitor")
}

func (m *appModel) headerCounts() string {
	live, agents := 0, 0
	for _, r := range m.snap.Repos {
		for _, w := range r.Worktrees {
			live += w.LiveAgents()
			agents += len(w.Agents)
		}
	}
	sep := m.fg(theme.RoleLine).Render("   ")
	return m.fgBold(theme.RoleAmber).Render(fmt.Sprintf("%02d", m.worktreeCount())) + m.fg(theme.RoleTextMuted).Render(" TREES") +
		sep + m.fgBold(theme.RoleAmber).Render(fmt.Sprintf("%02d", m.repoCount())) + m.fg(theme.RoleTextMuted).Render(" REPOS") +
		sep + m.liveCount(live, agents)
}

func (m *appModel) liveCount(live, agents int) string {
	if live > 0 {
		return m.chip(theme.RoleLive, fmt.Sprintf(" %02d LIVE ", live))
	}
	if agents > 0 {
		return m.fg(theme.RoleTextMuted).Render(fmt.Sprintf("%02d idle", agents))
	}
	return m.fg(theme.RoleTextMuted).Render("00 LIVE")
}

func (m *appModel) syncBadge() string {
	spin := string(brailleFrames[m.beat%len(brailleFrames)])
	age := "—"
	if !m.lastSync.IsZero() {
		age = relTime(m.lastSync)
	}
	col := theme.RoleAccent
	if time.Since(m.lastSync) < uiTickInterval*3 {
		col = theme.RoleLive
	}
	return m.fg(col).Render(spin) + m.fg(theme.RoleTextMuted).Render(" SYNC ") + m.fg(theme.RoleAmber).Render(age)
}

func buildFooter(s styleSet, k config.KeysConfig) string {
	entries := []struct {
		key   string
		label string
	}{
		{primaryKey(k.FocusNext), "focus"},
		{primaryKey(k.PaneList) + "·" + primaryKey(k.PaneChange) + "·" + primaryKey(k.PaneGraph) + "·" + primaryKey(k.PanePR), "panes"},
		{primaryKey(k.Up) + primaryKey(k.Down), "scroll"},
		{primaryKey(k.Pin), "pin"},
		{primaryKey(k.Group), "group"},
		{primaryKey(k.Diff), "diff"},
		{primaryKey(k.Commits), "commits"},
		{primaryKey(k.Status), "status"},
		{primaryKey(k.AgentLog), "log"},
		{primaryKey(k.Refresh), "sync"},
		{primaryKey(k.Quit), "quit"},
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.key == "" {
			continue
		}
		parts = append(parts, s.fg[theme.RoleAmber].Render(e.key)+s.fg[theme.RoleTextMuted].Render(" "+e.label))
	}
	return " " + strings.Join(parts, s.fg[theme.RoleLine].Render(" · "))
}

func (m *appModel) rail(i int) string {
	if i == m.cursor {
		return m.fgBold(theme.RoleSelection).Render("▌")
	}
	return m.fg(theme.RoleLine).Render("│")
}

func (m *appModel) rowLine(left, right string) string {
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)-1)
	return left + strings.Repeat(" ", gap) + right + " "
}

func (m *appModel) renderRow(i int, r row) string {
	switch r.kind {
	case rowGroup:
		return m.renderGroup(i, *r.repo)
	case rowAgent:
		return m.renderAgent(i, *r.worktree, r.agentIdx, r.lastSub, r.nested)
	default:
		return m.renderWorktree(i, *r.repo, *r.worktree, r.nested)
	}
}

func (m *appModel) renderGroup(i int, r model.Repo) string {
	active := activity.RepoActive(r)
	bullet := m.fg(theme.RoleTextMuted).Render("⬡")
	if active {
		bullet = m.fgBold(theme.RoleLive).Render("⬢")
	}
	nameStyle := m.fgBold(theme.RoleAmber)
	if st, ok := m.dimIdle(theme.RoleAmber, active); ok {
		nameStyle = st
	}
	name := nameStyle.Render(m.repoProject(&r))
	meta := m.fg(theme.RoleTextMuted).Render(fmt.Sprintf("  %d trees", len(r.Worktrees)))
	if r.Err != nil {
		meta = "  " + m.chip(theme.RoleError, " ERROR ") + m.fg(theme.RoleTextMuted).Render(" git")
	}
	return m.rail(i) + " " + bullet + " " + name + meta
}

func (m *appModel) worktreeTag(r model.Repo, w model.Worktree, active bool) string {
	if isPrimary(r, w) {
		return m.fg(theme.RoleMain).Render(cellPad("primary", colTag))
	}
	style := m.fgBold(theme.RoleWorktree)
	if st, ok := m.dimIdle(theme.RoleWorktree, active); ok {
		style = st
	}
	base := filepath.Base(w.Path)
	return style.Render(cellPad(cellTrunc("⟨"+base+"⟩", colTag), colTag))
}

func (m *appModel) worktreeBullet(w model.Worktree) string {
	if m.pinnedPath == w.Path {
		return m.fgBold(theme.RolePin).Render("◉")
	}
	return m.dot(w.LiveAgents() > 0)
}

func (m *appModel) renderWorktree(i int, r model.Repo, w model.Worktree, nested bool) string {
	active := w.LiveAgents() > 0
	branchStyle := m.fgBold(theme.RoleBranch)
	if st, ok := m.dimIdle(theme.RoleBranch, active); ok {
		branchStyle = st
	}
	branch := branchStyle.Render(cellPad(cellTrunc(w.Branch, colBranch), colBranch))
	head := branch + " " + m.worktreeTag(r, w, active)
	if w.Locked {
		head += " " + m.fg(theme.RoleWarn).Render("[locked]")
	}
	if w.Prunable {
		head += " " + m.fg(theme.RoleTextMuted).Render("[prunable]")
	}
	var prefix string
	if nested {
		prefix = m.rail(i) + "   " + m.worktreeBullet(w) + " "
	} else {
		prefix = m.rail(i) + " " + m.worktreeBullet(w) + " " +
			m.fgBold(theme.RoleAmber).Render(m.repoProject(&r)) + " " + m.fg(theme.RoleLine).Render("›") + " "
	}
	left := prefix + head
	if n := len(w.Agents); n > 1 {
		left += "  " + m.fg(theme.RoleAmber).Render(fmt.Sprintf("%d agents", n))
	}
	right := m.fg(theme.RoleTextMuted).Render(relTime(latest(w)))
	return m.rowLine(left, right)
}

func (m *appModel) renderAgent(i int, w model.Worktree, idx int, last, nested bool) string {
	if idx >= len(w.Agents) {
		return ""
	}
	a := w.Agents[idx]
	guide := "├─"
	if last {
		guide = "╰─"
	}
	indent := "   "
	if nested {
		indent = "     "
	}
	prefix := m.rail(i) + m.fg(theme.RoleLine).Render(indent+guide+" ") + m.fg(theme.RoleTextMuted).Render("Agent ") + m.stateChip(a.State) + " "
	right := m.fg(theme.RoleTextMuted).Render(relTime(a.UpdatedAt))
	leak := ""
	if a.LeakTarget != "" {
		leak = "  " + m.chip(theme.RoleHazard, " LEAK → "+a.LeakTarget+" ")
	}
	budget := max(12, m.width-lipgloss.Width(prefix)-lipgloss.Width(right)-lipgloss.Width(leak)-2)
	taskStyle := m.fg(theme.RoleTextPrimary)
	if st, ok := m.dimIdle(theme.RoleTextPrimary, a.Liveness == model.LiveActive); ok {
		taskStyle = st
	}
	task := taskStyle.Render(cellTrunc(displayTask(a), budget))
	return m.rowLine(prefix+task+leak, right)
}

func (m *appModel) dot(active bool) string {
	if active {
		return m.fgBold(theme.RoleLive).Render("●")
	}
	return m.fg(theme.RoleTextMuted).Render("○")
}

func displayTask(a model.AgentSession) string {
	switch a.State {
	case model.StateIdle:
		return "idle"
	case model.StateWaiting:
		return "awaiting input"
	default:
		if a.Task == "" {
			return strings.ToLower(a.State.String())
		}
		return a.Task
	}
}

func (m *appModel) detailPanel() string {
	body := strings.TrimRight(m.colorizeDetail(m.detailContent), "\n")
	if strings.TrimSpace(body) == "" {
		placeholder := "(empty)"
		if m.detailLoading {
			placeholder = "loading…"
		}
		body = m.fg(theme.RoleTextMuted).Render(placeholder)
	}
	return m.consoleBox(detailTitle(m.detail), body)
}

func (m *appModel) consoleBox(title, body string) string {
	w := max(20, m.width-2)
	line := m.fg(theme.RoleLine)
	head := line.Render("┌╴ ") + m.fgBold(theme.RoleAmber).Render(title) + " " +
		line.Render("╶"+strings.Repeat("─", max(0, w-lipgloss.Width(title)-6))+"┐")
	var out strings.Builder
	out.WriteString(" " + head + "\n")
	for _, ln := range strings.Split(body, "\n") {
		out.WriteString(" " + line.Render("│") + " " + ln + "\n")
	}
	out.WriteString(" " + line.Render("└"+strings.Repeat("─", w)+"┘"))
	return out.String()
}

func (m *appModel) colorizeDetail(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		switch m.detail {
		case detailCommits:
			lines[i] = m.colorCommit(line)
		case detailDiff:
			lines[i] = m.colorDiff(line)
		case detailStatus:
			lines[i] = m.colorStatus(line)
		default:
			lines[i] = m.fg(theme.RoleTextPrimary).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

func (m *appModel) colorCommit(line string) string {
	f := strings.Split(line, "\x1f")
	if len(f) != 4 {
		return m.fg(theme.RoleTextPrimary).Render(line)
	}
	sha := m.fg(theme.RoleSha).Render(f[0])
	age := m.fg(theme.RoleAccent).Render(cellPad(commitAge(f[1]), 5))
	author := m.fg(theme.RoleTextMuted).Render(cellPad(cellTrunc(f[2], 14), 14))
	subject := f[3]
	if budget := m.width - commitMetaWidth; budget >= 12 {
		subject = cellTrunc(subject, budget)
	}
	return sha + " " + age + " " + author + " " + m.fg(theme.RoleTextPrimary).Render(subject)
}

func commitAge(unix string) string {
	sec, err := strconv.ParseInt(strings.TrimSpace(unix), 10, 64)
	if err != nil {
		return ""
	}
	return relTime(time.Unix(sec, 0))
}

func (m *appModel) colorDiff(line string) string {
	if i := strings.Index(line, "|"); i >= 0 {
		var stat strings.Builder
		for _, r := range line[i:] {
			switch r {
			case '+':
				stat.WriteString(m.fg(theme.RoleAdd).Render("+"))
			case '-':
				stat.WriteString(m.fg(theme.RoleDel).Render("-"))
			default:
				stat.WriteString(m.fg(theme.RoleTextMuted).Render(string(r)))
			}
		}
		return m.fg(theme.RoleTextPrimary).Render(line[:i]) + stat.String()
	}
	return m.fg(theme.RoleTextMuted).Render(line)
}

func (m *appModel) colorStatus(line string) string {
	if strings.HasPrefix(line, "##") {
		return m.fg(theme.RoleBranch).Render(line)
	}
	if len(line) < 2 {
		return m.fg(theme.RoleTextPrimary).Render(line)
	}
	// git status --short guarantees two ASCII status bytes before the path,
	// so byte indexing is safe here even for multibyte paths.
	code := line[:2]
	var col theme.Role
	switch {
	case strings.Contains(code, "?"):
		col = theme.RoleTextMuted
	case strings.Contains(code, "A"):
		col = theme.RoleAdd
	case strings.Contains(code, "D"):
		col = theme.RoleDel
	default:
		col = theme.RoleWarn
	}
	return m.fg(col).Render(code) + m.fg(theme.RoleTextPrimary).Render(line[2:])
}

func detailTitle(d detailMode) string {
	switch d {
	case detailStatus:
		return "git status"
	case detailDiff:
		return "diff --stat"
	case detailCommits:
		return "recent commits"
	case detailAgentLog:
		return "agent log"
	default:
		return ""
	}
}

func latest(w model.Worktree) time.Time {
	t := w.LastFileMod
	if w.LastCommit.After(t) {
		t = w.LastCommit
	}
	for _, a := range w.Agents {
		if a.UpdatedAt.After(t) {
			t = a.UpdatedAt
		}
	}
	return t
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < 5*time.Second:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func cellPad(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func cellTrunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	width := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > n-1 {
			break
		}
		width += rw
		b.WriteRune(r)
	}
	return b.String() + "…"
}
