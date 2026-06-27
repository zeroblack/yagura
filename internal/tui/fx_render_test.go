package tui

import (
	"strings"
	"testing"

	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/theme"
)

func fxModel() *appModel {
	m := New(config.Default())
	m.fx = newFxSet(theme.Evangelion(), allOn(), true)
	m.width, m.height = 100, 40
	m.fx.resize(m.width)
	return m
}

func TestDivergenceBarReachesFocusCard(t *testing.T) {
	out := fxModel().divergenceBadgeFX(12, 3)
	if !strings.Contains(out, "█") {
		t.Fatalf("focus-card divergence has no magnitude bar: %q", stripANSI(out))
	}
}

// On a remote-less repo git status reports no ahead/behind (no upstream), so the
// focus card must fall back to the locally computed divergence or the bar never
// shows.
func TestFocusCardUsesLocalDivergenceWithoutUpstream(t *testing.T) {
	m := fxModel()
	repo := model.Repo{Path: "/p/app", Name: "app", Worktrees: []model.Worktree{{Path: "/p/wt", Branch: "feat/x"}}}
	m.inspect = inspectState{
		repo: &repo,
		wt:   &repo.Worktrees[0],
		div:  map[string]model.Divergence{"feat/x": {Ahead: 7}},
	}
	frame := stripANSI(strings.Join(m.focusCardLines(), "\n"))
	if !strings.Contains(frame, "↑7") || !strings.Contains(frame, "█") {
		t.Fatalf("focus card did not surface local divergence bar:\n%s", frame)
	}
}

func TestLiveChipPulsesAcrossBeats(t *testing.T) {
	m := fxModel()
	m.beat = 0
	lo := m.chip(theme.RoleLive, " 02 LIVE ")
	m.beat = pulseSteps - 1
	hi := m.chip(theme.RoleLive, " 02 LIVE ")
	if lo == hi {
		t.Fatal("LIVE chip did not change across pulse beats")
	}
}

func TestDepthDimsIdleWorktree(t *testing.T) {
	m := fxModel()
	r := model.Repo{Path: "/p/app", Name: "app"}
	idle := m.renderWorktree(0, r, model.Worktree{Path: "/p/wt", Branch: "feat/x"}, false)
	active := m.renderWorktree(0, r, model.Worktree{
		Path:   "/p/wt",
		Branch: "feat/x",
		Agents: []model.AgentSession{{Liveness: model.LiveActive}},
	}, false)
	if idle == active {
		t.Fatal("idle and active worktree rows render identically; depth not applied")
	}
}

func TestHeaderRuleSpansWidth(t *testing.T) {
	m := fxModel()
	if m.fx.headerRule == "" {
		t.Fatal("header rule empty after resize")
	}
	if !strings.Contains(m.headerRule(), "━") {
		t.Fatal("header rule lost its glyph")
	}
}
