package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/zeroblack/yagura/internal/theme"
)

func TestRichColorThreshold(t *testing.T) {
	rich := []colorprofile.Profile{colorprofile.ANSI256, colorprofile.TrueColor}
	flat := []colorprofile.Profile{colorprofile.ANSI, colorprofile.ASCII, colorprofile.NoTTY}
	for _, p := range rich {
		if !richColor(p) {
			t.Fatalf("profile %v should be treated as rich", p)
		}
	}
	for _, p := range flat {
		if richColor(p) {
			t.Fatalf("profile %v should not be treated as rich", p)
		}
	}
}

func allOn() fxConfig {
	return fxConfig{enabled: true, bars: true, pulse: true, depth: true, gradients: true, barWidth: 7, barCap: 20}
}

func TestBarWidthIsExact(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), true)
	for _, v := range []int{0, 1, 5, 20, 100, -3} {
		got := lipgloss.Width(f.bar(v))
		if got != f.cfg.barWidth {
			t.Fatalf("bar(%d) width = %d, want %d", v, got, f.cfg.barWidth)
		}
	}
}

func TestBarFillScalesWithValue(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), true)
	empty := countFilled(f.bar(0))
	mid := countFilled(f.bar(10))
	full := countFilled(f.bar(20))
	over := countFilled(f.bar(1000))
	if empty != 0 {
		t.Fatalf("bar(0) filled = %d, want 0", empty)
	}
	if full != f.cfg.barWidth {
		t.Fatalf("bar(cap) filled = %d, want %d", full, f.cfg.barWidth)
	}
	if over != f.cfg.barWidth {
		t.Fatalf("bar(over cap) filled = %d, want clamp to %d", over, f.cfg.barWidth)
	}
	if !(empty < mid && mid < full) {
		t.Fatalf("fill not monotonic: empty=%d mid=%d full=%d", empty, mid, full)
	}
}

func TestBarDisabledIsEmpty(t *testing.T) {
	cfg := allOn()
	cfg.bars = false
	f := newFxSet(theme.Evangelion(), cfg, true)
	if got := f.bar(10); got != "" {
		t.Fatalf("bar with bars off = %q, want empty", got)
	}
}

func TestBarRampSolidWithoutRichColor(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), false)
	first := f.barStyles[0].Render("x")
	for i := 1; i < len(f.barStyles); i++ {
		if f.barStyles[i].Render("x") != first {
			t.Fatalf("non-truecolor bar ramp not solid at cell %d", i)
		}
	}
	g := newFxSet(theme.Evangelion(), allOn(), true)
	if g.barStyles[0].Render("x") == g.barStyles[len(g.barStyles)-1].Render("x") {
		t.Fatalf("truecolor bar ramp endpoints should differ")
	}
}

func TestPulseTriIndexPingPong(t *testing.T) {
	want := []int{0, 1, 2, 3, 2, 1, 0, 1, 2, 3}
	for beat, exp := range want {
		if got := triIndex(beat, 4); got != exp {
			t.Fatalf("triIndex(%d,4) = %d, want %d", beat, got, exp)
		}
	}
	if got := triIndex(99, 1); got != 0 {
		t.Fatalf("triIndex with n=1 = %d, want 0", got)
	}
}

func TestPulseChipOnlyForActiveRoles(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), true)
	if _, ok := f.pulseChipStyle(theme.RoleLive, 0); !ok {
		t.Fatal("RoleLive should pulse")
	}
	if _, ok := f.pulseChipStyle(theme.RoleHazard, 0); !ok {
		t.Fatal("RoleHazard should pulse")
	}
	if _, ok := f.pulseChipStyle(theme.RoleAmber, 0); ok {
		t.Fatal("RoleAmber should not pulse")
	}
	base, _ := f.pulseChipStyle(theme.RoleLive, 0)
	peak, _ := f.pulseChipStyle(theme.RoleLive, pulseSteps-1)
	if base.Render(" x ") == peak.Render(" x ") {
		t.Fatal("pulse frame 0 and peak should differ")
	}
}

func TestPulseDisabled(t *testing.T) {
	cfg := allOn()
	cfg.pulse = false
	f := newFxSet(theme.Evangelion(), cfg, true)
	if _, ok := f.pulseChipStyle(theme.RoleLive, 0); ok {
		t.Fatal("pulse off should report not-ok")
	}
	if _, ok := f.runStateChip(0); ok {
		t.Fatal("run state chip off should report not-ok")
	}
}

func TestDimGatedByDepth(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), true)
	if _, ok := f.dimStyle(theme.RoleBranch); !ok {
		t.Fatal("depth on should yield dim style")
	}
	cfg := allOn()
	cfg.depth = false
	g := newFxSet(theme.Evangelion(), cfg, true)
	if _, ok := g.dimStyle(theme.RoleBranch); ok {
		t.Fatal("depth off should report not-ok")
	}
}

func TestResizeRuleWidths(t *testing.T) {
	f := newFxSet(theme.Evangelion(), allOn(), true)
	f.resize(80)
	if got := lipgloss.Width(f.headerRule); got != 80 {
		t.Fatalf("headerRule width = %d, want 80", got)
	}
	if got := lipgloss.Width(f.cardRule); got != 78 {
		t.Fatalf("cardRule width = %d, want 78", got)
	}
	f.resize(0)
	if f.headerRule != "" || f.cardRule != "" {
		t.Fatal("resize(0) should clear rules")
	}
}

func TestResizeSolidWhenDisabled(t *testing.T) {
	cfg := allOn()
	cfg.gradients = false
	f := newFxSet(theme.Evangelion(), cfg, true)
	f.resize(40)
	if got := lipgloss.Width(f.headerRule); got != 40 {
		t.Fatalf("solid headerRule width = %d, want 40", got)
	}
}

func countFilled(bar string) int {
	n := 0
	for _, r := range bar {
		if r == '█' {
			n++
		}
	}
	return n
}
