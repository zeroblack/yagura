package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
)

func prModel() *appModel {
	m := New(config.Default())
	m.width, m.height = 120, 44
	return m
}

func TestPRPaneHiddenWhenNoPRs(t *testing.T) {
	m := prModel()
	frame := stripANSI(m.renderInspectorBody())
	if strings.Contains(frame, "PRS") {
		t.Fatalf("PRS pane should be hidden with no PRs:\n%s", frame)
	}
}

func TestPRPaneShownWithPRs(t *testing.T) {
	m := prModel()
	repo := model.Repo{Path: "/p/app", Name: "app", Worktrees: []model.Worktree{{Path: "/p/wt", Branch: "feat/login"}}}
	m.inspect = inspectState{
		repo:   &repo,
		wt:     &repo.Worktrees[0],
		prList: []model.PRInfo{{Number: 42, Branch: "feat/login", Title: "add login", CreatedAt: time.Now().Add(-2 * time.Hour)}},
	}
	frame := stripANSI(m.renderInspectorBody())
	if !strings.Contains(frame, "PRS") || !strings.Contains(frame, "#42") {
		t.Fatalf("PRS pane missing with PRs present:\n%s", frame)
	}
}

func TestPRsSortedWorktreeMappedFirst(t *testing.T) {
	m := prModel()
	repo := model.Repo{Path: "/p/app", Worktrees: []model.Worktree{{Branch: "feat/mine"}}}
	m.inspect = inspectState{
		repo: &repo,
		prList: []model.PRInfo{
			{Number: 1, Branch: "other/old", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Number: 2, Branch: "feat/mine", CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	lines := m.prsContent()
	first := stripANSI(lines[0])
	if !strings.Contains(first, "#2") || !strings.Contains(first, "◉") {
		t.Fatalf("worktree-mapped PR should be first and marked, got: %q", first)
	}
}

func TestPRLineShowsDraftAndAge(t *testing.T) {
	m := prModel()
	line := stripANSI(m.prLine(model.PRInfo{
		Number:    7,
		Branch:    "chore/deps",
		Title:     "bump deps",
		Draft:     true,
		CreatedAt: time.Now().Add(-3 * time.Hour),
	}, false))
	for _, want := range []string{"#7", "chore/deps", "DRAFT", "3h", "bump deps"} {
		if !strings.Contains(line, want) {
			t.Fatalf("pr line missing %q: %q", want, line)
		}
	}
}

func TestCycleFocusSkipsAndIncludesPRs(t *testing.T) {
	m := prModel()
	m.focus = focusList
	m.cycleFocus(1)
	if m.focus != focusChanges {
		t.Fatalf("with no PRs, cycle from list should reach changes, got %v", m.focus)
	}

	m.inspect.prList = []model.PRInfo{{Number: 1, Branch: "b"}}
	m.focus = focusList
	m.cycleFocus(1)
	if m.focus != focusPRs {
		t.Fatalf("with PRs, cycle from list should reach PRS, got %v", m.focus)
	}
}

func TestFocusGuardResetsWhenPRsVanish(t *testing.T) {
	m := prModel()
	m.focus = focusPRs
	m.renderInspectorBody()
	if m.focus != focusList {
		t.Fatalf("focus should reset off PRS when none present, got %v", m.focus)
	}
}
