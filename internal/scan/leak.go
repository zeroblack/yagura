package scan

import (
	"sync"
	"time"
)

// leakTracker debounces cross-worktree edit attribution across scans. A leak is
// only raised once the same agent keeps editing into the same foreign worktree
// for at least the confirm window; a transient edit that is superseded before
// the window elapses never reaches the UI. The tracker survives between scans on
// the Scanner, since a single snapshot has no memory of the previous tick.
type leakTracker struct {
	mu      sync.Mutex
	pending map[string]leakObservation
}

type leakObservation struct {
	target    string
	firstSeen time.Time
}

func newLeakTracker() *leakTracker {
	return &leakTracker{pending: map[string]leakObservation{}}
}

// confirm reports the leak target only after the observation has aged past the
// window. A window of zero disables the debounce and confirms on first sight.
// Changing target restarts the clock so an agent hopping between worktrees does
// not inherit a stale confirmation.
func (lt *leakTracker) confirm(session, target string, now time.Time, window time.Duration) string {
	if window <= 0 {
		return target
	}
	lt.mu.Lock()
	defer lt.mu.Unlock()
	obs, ok := lt.pending[session]
	if !ok || obs.target != target {
		lt.pending[session] = leakObservation{target: target, firstSeen: now}
		return ""
	}
	if now.Sub(obs.firstSeen) >= window {
		return target
	}
	return ""
}

func (lt *leakTracker) clear(session string) {
	lt.mu.Lock()
	delete(lt.pending, session)
	lt.mu.Unlock()
}

func (lt *leakTracker) prune(live map[string]struct{}) {
	lt.mu.Lock()
	for session := range lt.pending {
		if _, ok := live[session]; !ok {
			delete(lt.pending, session)
		}
	}
	lt.mu.Unlock()
}
