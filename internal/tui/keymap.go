package tui

import "github.com/zeroblack/yagura/internal/config"

type keyAction int

const (
	actNone keyAction = iota
	actQuit
	actClose
	actInspect
	actFocusNext
	actFocusPrev
	actPaneList
	actPaneChanges
	actPaneGraph
	actPanePR
	actUp
	actDown
	actPageUp
	actPageDown
	actHome
	actEnd
	actPin
	actGroup
	actRefresh
	actDiff
	actCommits
	actStatus
	actAgentLog
)

func buildKeymap(k config.KeysConfig) map[string]keyAction {
	bindings := []struct {
		act  keyAction
		keys []string
	}{
		{actInspect, []string{k.Inspect}},
		{actQuit, k.Quit},
		{actClose, k.Close},
		{actFocusNext, k.FocusNext},
		{actFocusPrev, k.FocusPrev},
		{actPaneList, k.PaneList},
		{actPaneChanges, k.PaneChange},
		{actPaneGraph, k.PaneGraph},
		{actPanePR, k.PanePR},
		{actUp, k.Up},
		{actDown, k.Down},
		{actPageUp, k.PageUp},
		{actPageDown, k.PageDown},
		{actHome, k.Home},
		{actEnd, k.End},
		{actPin, k.Pin},
		{actGroup, k.Group},
		{actRefresh, k.Refresh},
		{actDiff, k.Diff},
		{actCommits, k.Commits},
		{actStatus, k.Status},
		{actAgentLog, k.AgentLog},
	}
	m := make(map[string]keyAction)
	for _, b := range bindings {
		for _, key := range b.keys {
			if key == "" {
				continue
			}
			m[key] = b.act
			if alias := keyAlias(key); alias != "" {
				m[alias] = b.act
			}
		}
	}
	return m
}

// keyAlias bridges the two spellings the spacebar can arrive as across Charm
// versions, so a `space` binding matches whether the runtime emits "space" or " ".
func keyAlias(key string) string {
	switch key {
	case "space":
		return " "
	case " ":
		return "space"
	}
	return ""
}

func primaryKey(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return keyGlyph(keys[0])
}

func keyGlyph(key string) string {
	switch key {
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "enter":
		return "⏎"
	default:
		return key
	}
}
