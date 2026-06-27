package tui

import (
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/theme"
)

func syncCfgFrom(c config.Config) syncCfg {
	return syncCfg{
		motion:        c.Indicator.Motion,
		frame:         c.Indicator.Frame,
		fullRotations: c.Indicator.FullRotations,
		blip:          150 * time.Millisecond,
		settle:        160 * time.Millisecond,
		agentBlip:     c.Indicator.AgentBlip,
		fullInterval:  c.Refresh.FullTick,
	}
}

type scanKind int

const (
	scanAgent scanKind = iota
	scanFull
)

// Confirm-bright green for the settle bloom and cyan (the data color) for the
// agent blip — neither is a steady theme role, so they live here.
var (
	syncBloomColor = lipgloss.Color("#94FFBC")
	syncBlipColor  = lipgloss.Color("#2BE8FF")
)

type syncCfg struct {
	motion        string // full | reduced | off
	frame         time.Duration
	fullRotations int
	blip          time.Duration
	settle        time.Duration
	agentBlip     bool
	fullInterval  time.Duration
}

// syncIndicator is the state machine behind the top-right SYNC readout. The spin
// is reserved for the heavy full scan and runs for a minimum visible duration so
// a fast scan still reads as a deliberate sync; the cheap agent refresh gets a
// one-beat color blip. At rest the dot carries a health color and the age ticks.
type syncIndicator struct {
	cfg syncCfg
	th  theme.Theme

	spinning    bool
	kind        scanKind
	frame       int
	spinUntil   time.Time
	settleUntil time.Time
	blipUntil   time.Time
	errStreak   int
}

func newSyncIndicator(cfg syncCfg, th theme.Theme) *syncIndicator {
	return &syncIndicator{cfg: cfg, th: th}
}

func (s *syncIndicator) start(kind scanKind, now time.Time) {
	s.kind = kind
	switch s.cfg.motion {
	case "off":
		return
	case "reduced":
		s.blipUntil = now.Add(s.cfg.blip)
		return
	}
	if kind == scanFull || !s.cfg.agentBlip {
		rotations := 1
		if kind == scanFull {
			rotations = s.cfg.fullRotations
		}
		s.spinning = true
		s.frame = 0
		s.spinUntil = now.Add(time.Duration(rotations*len(brailleFrames)) * s.cfg.frame)
		return
	}
	s.blipUntil = now.Add(s.cfg.blip)
}

func (s *syncIndicator) finish(errored bool, _ time.Time) {
	if errored {
		s.errStreak++
		return
	}
	s.errStreak = 0
}

// advance steps the spinner once per tick. The spin ends only once the minimum
// duration has elapsed, the scan is done, and the frame is back at the top of
// the arc — never mid-rotation — then blooms into the resting dot.
func (s *syncIndicator) advance(now time.Time, scanInFlight bool) {
	if !s.spinning {
		return
	}
	s.frame = (s.frame + 1) % len(brailleFrames)
	if now.After(s.spinUntil) && !scanInFlight && s.frame == 0 {
		s.spinning = false
		if s.errStreak == 0 {
			s.settleUntil = now.Add(s.cfg.settle)
		}
	}
}

func (s *syncIndicator) animating(now time.Time) bool {
	return s.spinning || now.Before(s.blipUntil) || now.Before(s.settleUntil)
}

func (s *syncIndicator) frameInterval() time.Duration { return s.cfg.frame }

type syncVisual struct {
	glyph string
	fg    color.Color
	chip  bool
}

func (s *syncIndicator) visual(now, lastFull time.Time) syncVisual {
	switch {
	case s.spinning:
		fg := lipgloss.Color(s.th.Color(theme.RoleAmber))
		if s.kind == scanAgent {
			fg = syncBlipColor
		}
		return syncVisual{glyph: string(brailleFrames[s.frame]), fg: fg}
	case now.Before(s.settleUntil):
		return syncVisual{glyph: "●", fg: syncBloomColor}
	case s.errStreak >= 2:
		return syncVisual{glyph: " SYNC ✕ ", fg: lipgloss.Color(s.th.Color(theme.RoleError)), chip: true}
	case s.errStreak == 1:
		return syncVisual{glyph: "●", fg: lipgloss.Color(s.th.Color(theme.RoleError))}
	default:
		fg := s.restColor(now, lastFull)
		if now.Before(s.blipUntil) {
			fg = syncBlipColor
		}
		return syncVisual{glyph: "●", fg: fg}
	}
}

// restColor escalates the dot from healthy green through amber to red as the
// time since the last full scan crosses multiples of the configured cadence.
func (s *syncIndicator) restColor(now, lastFull time.Time) color.Color {
	role := theme.RoleLive
	if !lastFull.IsZero() && s.cfg.fullInterval > 0 {
		switch age := now.Sub(lastFull); {
		case age >= 4*s.cfg.fullInterval:
			role = theme.RoleError
		case age >= 3*s.cfg.fullInterval/2:
			role = theme.RoleAmber
		}
	}
	return lipgloss.Color(s.th.Color(role))
}
