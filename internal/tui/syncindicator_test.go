package tui

import (
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/theme"
)

func testIndicator() *syncIndicator {
	return newSyncIndicator(syncCfg{
		motion: "full", frame: 80 * time.Millisecond, fullRotations: 2,
		blip: 150 * time.Millisecond, settle: 160 * time.Millisecond,
		agentBlip: true, fullInterval: 30 * time.Second,
	}, theme.ByName("evangelion"))
}

func TestSyncAgeText(t *testing.T) {
	s := testIndicator()
	now := time.Now()
	require.Equal(t, "—", s.ageText(now, time.Time{}))
	require.Equal(t, "3s", s.ageText(now, now.Add(-3*time.Second)))
	require.Equal(t, "2m", s.ageText(now, now.Add(-2*time.Minute)))
	require.Equal(t, "99m+", s.ageText(now, now.Add(-200*time.Minute)))
}

func TestSyncRestColorEscalates(t *testing.T) {
	s := testIndicator()
	th := theme.ByName("evangelion")
	now := time.Now()
	require.Equal(t, lipgloss.Color(th.Color(theme.RoleLive)), s.restColor(now, now.Add(-10*time.Second)), "fresh")
	require.Equal(t, lipgloss.Color(th.Color(theme.RoleAmber)), s.restColor(now, now.Add(-50*time.Second)), "aging >=1.5x")
	require.Equal(t, lipgloss.Color(th.Color(theme.RoleError)), s.restColor(now, now.Add(-130*time.Second)), "stale >=4x")
}

func TestSyncFullSpinFinishesArcThenBlooms(t *testing.T) {
	s := testIndicator()
	now := time.Now()
	s.start(scanFull, now)
	require.True(t, s.spinning)
	require.True(t, s.animating(now))

	s.advance(now.Add(80*time.Millisecond), false)
	require.True(t, s.spinning, "stays spinning before the minimum duration elapses")

	late := now.Add(2 * time.Second) // past spinUntil (2 rotations = 1.28s)
	for i := 0; i < len(brailleFrames)+1 && s.spinning; i++ {
		s.advance(late, false)
	}
	require.False(t, s.spinning, "spin ends only after the latch, at the top of the arc")
	require.Equal(t, 0, s.frame, "settled at frame 0, never mid-rotation")
	require.True(t, late.Before(s.settleUntil), "completion bloom scheduled")
}

func TestSyncFullSpinKeepsSpinningWhileScanInFlight(t *testing.T) {
	s := testIndicator()
	now := time.Now()
	s.start(scanFull, now)
	late := now.Add(2 * time.Second)
	for i := 0; i < 3*len(brailleFrames); i++ {
		s.advance(late, true) // scan still running
	}
	require.True(t, s.spinning, "never settles while the scan is in flight")
}

func TestSyncAgentBlipsNoSpin(t *testing.T) {
	s := testIndicator()
	now := time.Now()
	s.start(scanAgent, now)
	require.False(t, s.spinning, "agent refresh blips, never spins")
	require.True(t, s.animating(now))
	v := s.visual(now, now.Add(-2*time.Second))
	require.Equal(t, "●", v.glyph)
	require.Equal(t, syncBlipColor, v.fg)
}

func TestSyncErrorEscalatesToChip(t *testing.T) {
	s := testIndicator()
	now := time.Now()
	s.finish(true, now)
	v := s.visual(now, now)
	require.Equal(t, "●", v.glyph)
	require.Equal(t, "err", v.age)
	require.False(t, v.chip)

	s.finish(true, now)
	require.True(t, s.visual(now, now).chip, "two consecutive errors escalate to the filled chip")

	s.finish(false, now)
	require.False(t, s.visual(now, now).chip, "a clean scan clears the error")
}

func TestSyncMotionOff(t *testing.T) {
	s := testIndicator()
	s.cfg.motion = "off"
	now := time.Now()
	s.start(scanFull, now)
	require.False(t, s.spinning)
	require.False(t, s.animating(now))
}
