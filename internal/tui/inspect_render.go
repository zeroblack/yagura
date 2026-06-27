package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/theme"
)

func (m *appModel) focusCardLines() []string {
	wt := m.inspect.wt
	repo := m.inspect.repo
	if wt == nil || repo == nil {
		return nil
	}
	st := m.inspect.status

	marker := m.fg(theme.RoleLine).Render("▸")
	if m.pinnedPath == wt.Path {
		marker = m.fgBold(theme.RolePin).Render("◉")
	}
	ident := marker + " " +
		m.fgBold(theme.RoleAmber).Render(m.repoProject(repo)) + " " +
		m.fg(theme.RoleLine).Render("›") + " " +
		m.branchLabel(wt.Branch)

	var badges []string
	ahead, behind := st.Ahead, st.Behind
	if ahead == 0 && behind == 0 {
		if d, ok := m.inspect.div[wt.Branch]; ok {
			ahead, behind = d.Ahead, d.Behind
		}
	}
	if b := m.divergenceBadgeFX(ahead, behind); b != "" {
		badges = append(badges, b)
	}
	if pr := m.prBadge(wt.Branch); pr != "" {
		badges = append(badges, pr)
	}
	lines := []string{m.rowLine(" "+ident, strings.Join(badges, "  "))}

	if agent := m.focusAgentLine(wt); agent != "" {
		lines = append(lines, agent)
	}

	tag := "primary"
	if !isPrimary(*repo, *wt) {
		tag = "⟨" + filepath.Base(wt.Path) + "⟩"
	}
	sep := m.fg(theme.RoleLine).Render(" · ")
	left := "   " + m.fg(theme.RoleMain).Render(tag) + sep + m.projectStats()
	budget := max(8, m.width-lipgloss.Width(left)-4)
	path := m.fg(theme.RoleTextMuted).Render(cellTrunc(m.prettyPath(wt.Path), budget))
	return append(lines, m.rowLine(left, path))
}

func (m *appModel) repoProject(r *model.Repo) string {
	if m.cfg.Display.ProjectFrom == "repo" {
		return r.Name
	}
	base := filepath.Base(r.Path)
	for _, c := range m.cfg.Display.ProjectContainers {
		if base != c {
			continue
		}
		if parent := filepath.Base(filepath.Dir(r.Path)); parent != "" && parent != "." && parent != string(filepath.Separator) {
			return parent
		}
	}
	return r.Name
}

func (m *appModel) focusAgentLine(w *model.Worktree) string {
	a := latestAgent(w)
	if a == nil {
		return ""
	}
	prefix := "   " + m.stateChip(a.State) + " "
	live := m.fg(theme.RoleTextMuted).Render("idle")
	switch a.Liveness {
	case model.LiveActive:
		live = m.fg(theme.RoleLive).Render("live")
	case model.LiveRecent:
		live = m.fg(theme.RoleAccent).Render("recent")
	}
	right := live + " " + m.fg(theme.RoleTextMuted).Render(relTime(a.UpdatedAt))
	budget := max(12, m.width-lipgloss.Width(prefix)-lipgloss.Width(right)-3)
	task := m.fg(theme.RoleTextPrimary).Render(cellTrunc(displayTask(*a), budget))
	return m.rowLine(prefix+task, right)
}

func latestAgent(w *model.Worktree) *model.AgentSession {
	if len(w.Agents) == 0 {
		return nil
	}
	idx := 0
	for i := 1; i < len(w.Agents); i++ {
		if w.Agents[i].UpdatedAt.After(w.Agents[idx].UpdatedAt) {
			idx = i
		}
	}
	return &w.Agents[idx]
}

func isPrimary(r model.Repo, w model.Worktree) bool {
	return w.IsMain && filepath.Base(w.Path) == filepath.Base(r.Path)
}

func (m *appModel) projectStats() string {
	var parts []string
	if m.inspect.statsLoaded {
		parts = append(parts, m.statNum(m.inspect.stats.Commits, "commit"))
		parts = append(parts, m.statNum(m.inspect.stats.Branches, "branch"))
	}
	if m.inspect.repo != nil {
		parts = append(parts, m.statNum(len(m.inspect.repo.Worktrees), "worktree"))
	}
	if len(m.inspect.graph) > 0 && m.inspect.graph[0].HasCommit {
		g := m.inspect.graph[0]
		parts = append(parts, m.fg(theme.RoleTextMuted).Render("last "+relTime(g.When)+" by "+g.Author))
	}
	return strings.Join(parts, m.fg(theme.RoleLine).Render(" · "))
}

func (m *appModel) statNum(n int, singular string) string {
	return m.fgBold(theme.RoleAmber).Render(strconv.Itoa(n)) + " " + m.fg(theme.RoleTextMuted).Render(plural(n, singular))
}

func plural(n int, singular string) string {
	if n == 1 {
		return singular
	}
	if strings.HasSuffix(singular, "ch") || strings.HasSuffix(singular, "s") || strings.HasSuffix(singular, "x") {
		return singular + "es"
	}
	return singular + "s"
}

// divergenceBadgeFX renders ahead/behind with a magnitude bar appended to each
// side so divergence reads at a glance.
func (m *appModel) divergenceBadgeFX(ahead, behind int) string {
	var out []string
	if ahead > 0 {
		seg := m.fg(theme.RoleAdd).Render("↑" + strconv.Itoa(ahead))
		if bar := m.fx.bar(ahead); bar != "" {
			seg += " " + bar
		}
		out = append(out, seg)
	}
	if behind > 0 {
		seg := m.fg(theme.RoleWarn).Render("↓" + strconv.Itoa(behind))
		if bar := m.fx.bar(behind); bar != "" {
			seg += " " + bar
		}
		out = append(out, seg)
	}
	return strings.Join(out, "  ")
}

func (m *appModel) attentionBanner() string {
	if !m.cfg.Inspect.AttentionFirst {
		return ""
	}
	items := m.attentionItems()
	if len(items) == 0 {
		return ""
	}
	var parts []string
	for _, it := range items {
		parts = append(parts, m.fgBold(theme.RoleHazard).Render(it.branch)+" "+m.fg(theme.RoleTextMuted).Render("("+it.reason+")"))
	}
	return m.chip(theme.RoleHazard, " ATTENTION "+strconv.Itoa(len(items))+" ") + " " + strings.Join(parts, m.fg(theme.RoleLine).Render(" · "))
}

func (m *appModel) changesContent() []string {
	if !m.inspect.statusLoaded {
		return []string{m.fg(theme.RoleTextMuted).Render("   loading…")}
	}
	if m.inspect.statusErr != nil {
		return []string{m.fg(theme.RoleError).Render("   " + m.inspect.statusErr.Error())}
	}
	return m.changesLines()
}

func (m *appModel) changesLines() []string {
	files := m.inspect.sorted
	limit := m.cfg.Inspect.FileLimit
	if limit <= 0 {
		limit = len(files)
	}
	var lines []string
	for i, f := range files {
		if i >= limit {
			lines = append(lines, m.fg(theme.RoleTextMuted).Render(fmt.Sprintf("   … +%d more", len(files)-limit)))
			break
		}
		lines = append(lines, m.changeLine(f))
	}
	if len(files) == 0 {
		lines = append(lines, m.fg(theme.RoleTextMuted).Render("   working tree clean"))
	}
	return lines
}

func sortChanges(files []model.FileChange, mode string) []model.FileChange {
	out := slices.Clone(files)
	switch mode {
	case "path":
		slices.SortFunc(out, func(a, b model.FileChange) int { return strings.Compare(a.Path, b.Path) })
	case "status":
		slices.SortStableFunc(out, func(a, b model.FileChange) int { return int(a.Status) - int(b.Status) })
	default:
		slices.SortStableFunc(out, func(a, b model.FileChange) int { return b.ModTime.Compare(a.ModTime) })
	}
	return out
}

func (m *appModel) changeLine(f model.FileChange) string {
	code := m.fgBold(m.changeRole(f)).Render(cellPad(f.Status.Code(), 1))
	dir, base := filepath.Split(f.Path)
	path := m.fg(theme.RoleTextMuted).Render(dir) + m.fg(theme.RoleTextPrimary).Render(base)
	left := "  " + code + "  " + path
	age := ""
	if !f.ModTime.IsZero() {
		age = m.fg(theme.RoleTextMuted).Render(relTime(f.ModTime))
	}
	right := m.changeDelta(f) + "  " + age
	return m.rowLine(left, right)
}

func (m *appModel) changeRole(f model.FileChange) theme.Role {
	switch f.Status {
	case model.StatusConflicted:
		return theme.RoleHazard
	case model.StatusUntracked:
		return theme.RoleUntracked
	}
	if f.Staged {
		return theme.RoleStaged
	}
	return theme.RoleUnstaged
}

func (m *appModel) changeDelta(f model.FileChange) string {
	if f.Status == model.StatusUntracked {
		return m.fg(theme.RoleTextMuted).Render("—")
	}
	if f.Binary() {
		return m.fg(theme.RoleTextMuted).Render("bin")
	}
	var parts []string
	if f.Added > 0 {
		parts = append(parts, m.fg(theme.RoleAdd).Render("+"+strconv.Itoa(f.Added)))
	}
	if f.Deleted > 0 {
		parts = append(parts, m.fg(theme.RoleDel).Render("-"+strconv.Itoa(f.Deleted)))
	}
	return strings.Join(parts, " ")
}

func (m *appModel) graphContent() []string {
	if !m.inspect.graphLoaded {
		return []string{m.fg(theme.RoleTextMuted).Render("   loading…")}
	}
	return m.graphLines()
}

func (m *appModel) graphLines() []string {
	if len(m.inspect.graph) == 0 {
		return []string{m.fg(theme.RoleTextMuted).Render("   (no history)")}
	}
	tips := map[string]bool{}
	if m.inspect.repo != nil {
		for _, w := range m.inspect.repo.Worktrees {
			tips[w.Branch] = true
		}
	}
	lines := make([]string, 0, len(m.inspect.graph))
	for _, gl := range m.inspect.graph {
		lines = append(lines, m.graphLine(gl, tips))
	}
	return lines
}

func (m *appModel) graphLine(gl model.GraphLine, tips map[string]bool) string {
	graph := m.colorGraphPrefix(gl.Graph)
	if !gl.HasCommit {
		return " " + graph
	}
	age := m.fg(theme.RoleTextMuted).Render(relTime(gl.When))
	tip := tipBranch(gl.Refs, tips)
	merge := m.mergeLabel(gl)
	budget := max(12, m.width-graphMetaWidth)
	if tip != "" {
		left := " " + graph + m.branchLabel(tip)
		if d, ok := m.inspect.div[tip]; ok {
			if b := m.divergenceBadgeFX(d.Ahead, d.Behind); b != "" {
				left += "  " + b
			}
		}
		if pr := m.prBadge(tip); pr != "" {
			left += "  " + pr
		}
		left += "  " + merge + m.fg(theme.RoleTextMuted).Render(cellTrunc(gl.Subject, budget))
		return m.rowLine(left, age)
	}
	left := " " + graph + m.fg(theme.RoleSha).Render(gl.Hash) + " " + merge +
		m.fg(theme.RoleTextMuted).Render(cellTrunc(gl.Subject, budget)) + "  " +
		m.fg(theme.RoleTextMuted).Render(gl.Author)
	return m.rowLine(left, age)
}

func (m *appModel) mergeLabel(gl model.GraphLine) string {
	if !gl.IsMerge() {
		return ""
	}
	name := mergeBranchName(gl.Subject)
	if name == "" {
		return ""
	}
	return m.fg(theme.RoleBranch).Render("⌥ "+name) + " "
}

var (
	reMergePR     = regexp.MustCompile(`Merge pull request #\d+ from [^/\s]+/(\S+)`)
	reMergeBranch = regexp.MustCompile(`Merge (?:remote-tracking )?branch '([^']+)'`)
)

func mergeBranchName(subject string) string {
	if mm := reMergePR.FindStringSubmatch(subject); mm != nil {
		return mm[1]
	}
	if mm := reMergeBranch.FindStringSubmatch(subject); mm != nil {
		return strings.TrimPrefix(mm[1], "origin/")
	}
	return ""
}

func (m *appModel) branchLabel(branch string) string {
	if m.branchAttention(branch).Needs {
		return m.fgBold(theme.RoleHazard).Render(branch + " !")
	}
	return m.fgBold(theme.RoleBranch).Render(branch)
}

func tipBranch(refs []string, tips map[string]bool) string {
	for _, r := range refs {
		if tips[r] {
			return r
		}
	}
	return ""
}

func (m *appModel) prBadge(branch string) string {
	pr, ok := m.inspect.prByBranch[branch]
	if !ok {
		return ""
	}
	return m.fg(theme.RolePR).Render("PR #"+strconv.Itoa(pr.Number)) + " " + m.prMetaBadge(pr)
}

func (m *appModel) prsContent() []string {
	prs := m.sortedPRs()
	if len(prs) == 0 {
		return []string{m.fg(theme.RoleTextMuted).Render("   no open PRs")}
	}
	lines := make([]string, 0, len(prs))
	for _, pr := range prs {
		lines = append(lines, m.prLine(pr, m.prHasWorktree(pr.Branch)))
	}
	return lines
}

// sortedPRs puts PRs whose branch is checked out as a worktree first, then the
// rest, each group newest-first.
func (m *appModel) sortedPRs() []model.PRInfo {
	prs := slices.Clone(m.inspect.prList)
	slices.SortStableFunc(prs, func(a, b model.PRInfo) int {
		am, bm := m.prHasWorktree(a.Branch), m.prHasWorktree(b.Branch)
		if am != bm {
			if am {
				return -1
			}
			return 1
		}
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return prs
}

func (m *appModel) prHasWorktree(branch string) bool {
	if m.inspect.repo == nil {
		return false
	}
	for i := range m.inspect.repo.Worktrees {
		if m.inspect.repo.Worktrees[i].Branch == branch {
			return true
		}
	}
	return false
}

func (m *appModel) prLine(pr model.PRInfo, mapped bool) string {
	marker := m.fg(theme.RoleLine).Render(" ")
	if mapped {
		marker = m.fgBold(theme.RoleWorktree).Render("◉")
	}
	num := m.fg(theme.RolePR).Render("#" + strconv.Itoa(pr.Number))
	branch := m.fgBold(theme.RoleBranch).Render(cellPad(cellTrunc(pr.Branch, colBranch), colBranch))
	left := "  " + marker + " " + num + " " + branch
	if meta := m.prRowMeta(pr); meta != "" {
		left += "  " + meta
	}
	right := ""
	if !pr.CreatedAt.IsZero() {
		right = m.fg(theme.RoleTextMuted).Render(relTime(pr.CreatedAt))
	}
	budget := max(8, m.width-lipgloss.Width(left)-lipgloss.Width(right)-4)
	left += "  " + m.fg(theme.RoleTextPrimary).Render(cellTrunc(pr.Title, budget))
	return m.rowLine(left, right)
}

func (m *appModel) prRowMeta(pr model.PRInfo) string {
	if pr.Draft {
		return m.chip(theme.RoleTextMuted, " DRAFT ")
	}
	return m.prMetaBadge(pr)
}

func (m *appModel) prMetaBadge(pr model.PRInfo) string {
	var parts []string
	switch pr.Review {
	case model.ReviewApproved:
		parts = append(parts, m.fg(theme.RoleAdd).Render("✓ approved"))
	case model.ReviewChangesRequested:
		parts = append(parts, m.fg(theme.RoleHazard).Render("✗ changes"))
	case model.ReviewRequired:
		parts = append(parts, m.fg(theme.RoleWarn).Render("◆ review"))
	}
	switch pr.CI {
	case model.CIPassing:
		parts = append(parts, m.fg(theme.RoleLive).Render("CI ✓"))
	case model.CIFailing:
		parts = append(parts, m.fg(theme.RoleHazard).Render("CI ✗"))
	case model.CIPending:
		parts = append(parts, m.fg(theme.RoleWarn).Render("CI ●"))
	}
	return strings.Join(parts, " ")
}

func (m *appModel) colorGraphPrefix(g string) string {
	var b strings.Builder
	for _, r := range g {
		switch r {
		case '*':
			b.WriteString(m.fg(theme.RoleSha).Render("●"))
		case ' ':
			b.WriteByte(' ')
		default:
			b.WriteString(m.fg(theme.RoleLine).Render(string(r)))
		}
	}
	return b.String()
}

func (m *appModel) renderInspectorBody() string {
	if m.focus == focusPRs && !m.prsVisible() {
		m.focus = focusList
	}
	var b strings.Builder

	chrome := 3 // header(2) + footer(1)
	for _, ln := range m.focusCardLines() {
		b.WriteString(ln + "\n")
		chrome++
	}
	if banner := m.attentionBanner(); banner != "" {
		b.WriteString(" " + banner + "\n")
		chrome++
	}
	b.WriteString(" " + m.cardRule() + "\n")
	chrome++

	avail := m.height - chrome
	if avail < 6 {
		avail = 6
	}
	showPRs := m.prsVisible()
	panes := 3
	if showPRs {
		panes = 4
	}
	bodyAvail := avail - (2*panes - 1) // one title per pane + a gap between groups
	if bodyAvail < 3 {
		bodyAvail = 3
	}

	listLen := len(m.rows)
	listH := clampInt(listLen, 1, 6)
	if m.focus == focusList {
		listH = clampInt(listLen, 1, max(1, bodyAvail/2))
	}

	var prs []string
	prsH := 0
	if showPRs {
		prs = m.prsContent()
		prsH = clampInt(len(prs), 1, 6)
		if m.focus == focusPRs {
			prsH = clampInt(len(prs), 1, max(1, bodyAvail/2))
		}
		m.prsOff = clampOffset(m.prsOff, len(prs), prsH)
	}

	rem := max(2, bodyAvail-listH-prsH)
	chH, grH := splitPanes(rem, m.focus)

	changes := m.changesContent()
	graph := m.graphContent()
	m.changesOff = clampOffset(m.changesOff, len(changes), chH)
	m.graphOff = clampOffset(m.graphOff, len(graph), grH)

	b.WriteString(m.paneTitle("WORKTREES", m.worktreeCount(), m.focus == focusList, m.listOff, listH, listLen) + "\n")
	b.WriteString(m.listPane(listH))
	b.WriteString("\n")
	if showPRs {
		b.WriteString(m.paneTitle("PRS", len(m.inspect.prList), m.focus == focusPRs, m.prsOff, prsH, len(prs)) + "\n")
		b.WriteString(windowLines(prs, m.prsOff, prsH))
		b.WriteString("\n")
	}
	b.WriteString(m.paneTitle("CHANGES", len(m.inspect.status.Files), m.focus == focusChanges, m.changesOff, chH, len(changes)) + "\n")
	b.WriteString(windowLines(changes, m.changesOff, chH))
	b.WriteString("\n")
	b.WriteString(m.paneTitle("GRAPH", 0, m.focus == focusGraph, m.graphOff, grH, len(graph)) + "\n")
	b.WriteString(windowLines(graph, m.graphOff, grH))
	return strings.TrimRight(b.String(), "\n")
}

func splitPanes(rem int, focus paneFocus) (int, int) {
	switch focus {
	case focusChanges:
		ch := max(1, (rem*3+2)/5)
		return ch, max(1, rem-ch)
	case focusGraph:
		gr := max(1, (rem*3+2)/5)
		return max(1, rem-gr), gr
	default:
		ch := rem / 2
		return max(1, ch), max(1, rem-ch)
	}
}

func (m *appModel) paneTitle(title string, count int, focused bool, off, height, total int) string {
	label := title
	if count > 0 {
		label += " " + strconv.Itoa(count)
	}
	var head string
	if focused {
		head = m.chip(theme.RoleAmber, " ▸ "+label+" ")
	} else {
		head = m.fg(theme.RoleRule).Render("▸ ") + m.fgBold(theme.RoleAmber).Render(label)
	}
	scroll := ""
	if off > 0 {
		scroll += "↑"
	}
	if off+height < total {
		scroll += "↓"
	}
	if scroll != "" {
		scroll = " " + m.fg(theme.RoleAccent).Render(scroll)
	}
	rule := m.fg(theme.RoleRule).Render(strings.Repeat("─", max(0, m.width-lipgloss.Width(head)-lipgloss.Width(scroll)-3)))
	return " " + head + " " + rule + scroll
}

func (m *appModel) listPane(h int) string {
	if h < 1 {
		h = 1
	}
	if m.cursor < m.listOff {
		m.listOff = m.cursor
	}
	if m.cursor >= m.listOff+h {
		m.listOff = m.cursor - h + 1
	}
	m.listOff = clampInt(m.listOff, 0, max(0, len(m.rows)-h))
	var b strings.Builder
	for i := 0; i < h; i++ {
		idx := m.listOff + i
		if idx < len(m.rows) {
			b.WriteString(m.renderRow(idx, m.rows[idx]))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func windowLines(lines []string, off, h int) string {
	var b strings.Builder
	for i := 0; i < h; i++ {
		idx := off + i
		if idx >= 0 && idx < len(lines) {
			b.WriteString(lines[idx])
		}
		b.WriteString("\n")
	}
	return b.String()
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampOffset(off, total, height int) int {
	maxOff := max(0, total-height)
	if off > maxOff {
		off = maxOff
	}
	if off < 0 {
		off = 0
	}
	return off
}

func (m *appModel) prettyPath(p string) string {
	if m.home != "" && strings.HasPrefix(p, m.home) {
		return "~" + strings.TrimPrefix(p, m.home)
	}
	return p
}
