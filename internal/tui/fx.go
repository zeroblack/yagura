package tui

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
	"github.com/zeroblack/yagura/internal/theme"
)

// Visual effects share the precompute discipline of styleSet: every animated or
// gradient surface is built once into a lookup table here, and the render path
// only indexes it by m.beat or by a data value. Nothing in fxSet allocates a
// lipgloss.Style per frame; the View is reassembled at the UI tick rate.

const pulseSteps = 8

var pulseRoles = []theme.Role{theme.RoleLive, theme.RoleHazard}

type fxConfig struct {
	enabled   bool
	bars      bool
	pulse     bool
	depth     bool
	gradients bool
	barWidth  int
	barCap    int
}

func fxConfigFrom(c config.FXConfig) fxConfig {
	return fxConfig{
		enabled:   c.Enabled,
		bars:      c.Bars,
		pulse:     c.Pulse,
		depth:     c.Depth,
		gradients: c.Gradients,
		barWidth:  c.BarWidth,
		barCap:    c.BarCap,
	}
}

// richColorTerminal reports whether the terminal has at least 256 colors. We
// blend in truecolor and let bubbletea downsample to the real profile; a 256+
// palette renders a gradient cleanly, so gating on the stricter TrueColor
// profile (which many terminals under-report when COLORTERM is unset) would
// needlessly collapse every gradient to a flat color.
func richColorTerminal() bool {
	return richColor(colorprofile.Detect(os.Stdout, os.Environ()))
}

func richColor(p colorprofile.Profile) bool {
	return p >= colorprofile.ANSI256
}

type fxSet struct {
	cfg  fxConfig
	rich bool

	barStyles []lipgloss.Style
	barTrack  lipgloss.Style

	pulseChip map[theme.Role][]lipgloss.Style
	pulseRun  []string

	dim []lipgloss.Style

	amber    lipgloss.Style
	rule     lipgloss.Style
	ruleStop color.Color
	amberCol color.Color
	shaCol   color.Color

	headerRule string
	cardRule   string
}

func newFxSet(t theme.Theme, cfg fxConfig, rich bool) fxSet {
	f := fxSet{cfg: cfg, rich: rich}
	if cfg.barWidth < 1 {
		f.cfg.barWidth = 1
	}
	if cfg.barCap < 1 {
		f.cfg.barCap = 1
	}

	f.amberCol = lipgloss.Color(t.Color(theme.RoleAmber))
	f.ruleStop = lipgloss.Color(t.Color(theme.RoleRule))
	f.shaCol = lipgloss.Color(t.Color(theme.RoleSha))
	f.amber = lipgloss.NewStyle().Foreground(f.amberCol)
	f.rule = lipgloss.NewStyle().Foreground(f.ruleStop)

	f.buildBars(t)
	f.buildPulse(t)
	f.buildDim(t)
	return f
}

func (f *fxSet) buildBars(t theme.Theme) {
	w := f.cfg.barWidth
	f.barStyles = make([]lipgloss.Style, w)
	f.barTrack = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Color(theme.RoleLine)))
	if !f.rich || !f.cfg.gradients {
		solid := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Color(theme.RoleAccent)))
		for i := range f.barStyles {
			f.barStyles[i] = solid
		}
		return
	}
	ramp := lipgloss.Blend1D(w, lipgloss.Color(t.Color(theme.RoleAccent)), lipgloss.Color(t.Color(theme.RoleHazard)))
	for i := 0; i < w; i++ {
		f.barStyles[i] = lipgloss.NewStyle().Foreground(ramp[i])
	}
}

func (f *fxSet) buildPulse(t theme.Theme) {
	ink := lipgloss.Color(t.Color(theme.RoleInk))
	f.pulseChip = make(map[theme.Role][]lipgloss.Style, len(pulseRoles))
	for _, r := range pulseRoles {
		base := lipgloss.Color(t.Color(r))
		ramp := lipgloss.Blend1D(pulseSteps, base, brighten(base, 0.45))
		styles := make([]lipgloss.Style, pulseSteps)
		for i, c := range ramp {
			styles[i] = lipgloss.NewStyle().Background(c).Foreground(ink).Bold(true)
		}
		f.pulseChip[r] = styles
	}

	runBase := lipgloss.Color(t.StateColor(model.StateRunning))
	runRamp := lipgloss.Blend1D(pulseSteps, runBase, brighten(runBase, 0.45))
	label := " " + cellPad(model.StateRunning.String(), 8) + " "
	f.pulseRun = make([]string, pulseSteps)
	for i, c := range runRamp {
		f.pulseRun[i] = lipgloss.NewStyle().Background(c).Foreground(ink).Bold(true).Render(label)
	}
}

func (f *fxSet) buildDim(t theme.Theme) {
	bg := lipgloss.Color(t.Color(theme.RoleBackground))
	f.dim = make([]lipgloss.Style, theme.NumRoles)
	for r := range theme.NumRoles {
		base := lipgloss.Color(t.Color(theme.Role(r)))
		f.dim[r] = lipgloss.NewStyle().Foreground(darken(base, bg, 0.5))
	}
}

func (f *fxSet) resize(width int) {
	if width <= 0 {
		f.headerRule, f.cardRule = "", ""
		return
	}
	cardW := max(0, width-2)
	if !f.cfg.enabled || !f.cfg.gradients || !f.rich {
		f.headerRule = f.amber.Render(strings.Repeat("━", width))
		f.cardRule = f.rule.Render(strings.Repeat("─", cardW))
		return
	}
	f.headerRule = gradientLine("━", width, f.ruleStop, f.amberCol, f.ruleStop)
	f.cardRule = gradientLine("─", cardW, f.ruleStop, f.shaCol, f.ruleStop)
}

func (f fxSet) bar(value int) string {
	if !f.cfg.enabled || !f.cfg.bars {
		return ""
	}
	w := f.cfg.barWidth
	ratio := float64(value) / float64(f.cfg.barCap)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio*float64(w) + 0.5)
	var b strings.Builder
	for i := 0; i < w; i++ {
		if i < filled {
			b.WriteString(f.barStyles[i].Render("█"))
		} else {
			b.WriteString(f.barTrack.Render("░"))
		}
	}
	return b.String()
}

func (f fxSet) pulseChipStyle(r theme.Role, beat int) (lipgloss.Style, bool) {
	if !f.cfg.enabled || !f.cfg.pulse {
		return lipgloss.Style{}, false
	}
	ramp, ok := f.pulseChip[r]
	if !ok {
		return lipgloss.Style{}, false
	}
	return ramp[triIndex(beat, len(ramp))], true
}

func (f fxSet) runStateChip(beat int) (string, bool) {
	if !f.cfg.enabled || !f.cfg.pulse || len(f.pulseRun) == 0 {
		return "", false
	}
	return f.pulseRun[triIndex(beat, len(f.pulseRun))], true
}

func (f fxSet) dimStyle(r theme.Role) (lipgloss.Style, bool) {
	if !f.cfg.enabled || !f.cfg.depth || int(r) >= len(f.dim) {
		return lipgloss.Style{}, false
	}
	return f.dim[r], true
}

func gradientLine(glyph string, width int, stops ...color.Color) string {
	if width <= 0 {
		return ""
	}
	ramp := lipgloss.Blend1D(width, stops...)
	var b strings.Builder
	for i := 0; i < width; i++ {
		b.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Render(glyph))
	}
	return b.String()
}

// triIndex maps a monotonically increasing beat to a ping-pong index over
// [0, n-1], so a pulse rises to its peak frame and falls back instead of
// snapping from the brightest frame to the dimmest.
func triIndex(beat, n int) int {
	if n <= 1 {
		return 0
	}
	period := 2 * (n - 1)
	p := ((beat % period) + period) % period
	if p < n {
		return p
	}
	return period - p
}

func brighten(base color.Color, amt float64) color.Color {
	return sample(lipgloss.Blend1D(64, base, lipgloss.Color("#FFFFFF")), amt)
}

func darken(base, bg color.Color, amt float64) color.Color {
	return sample(lipgloss.Blend1D(64, base, bg), amt)
}

func sample(ramp []color.Color, amt float64) color.Color {
	if len(ramp) == 0 {
		return lipgloss.Color("#000000")
	}
	if amt < 0 {
		amt = 0
	}
	if amt > 1 {
		amt = 1
	}
	return ramp[int(amt*float64(len(ramp)-1)+0.5)]
}
