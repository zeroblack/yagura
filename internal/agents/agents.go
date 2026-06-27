package agents

import (
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

type Session struct {
	Path         string
	Cwd          string
	LastPath     string
	LastEditPath string
	Branch       string
	Model        string
	State        model.AgentState
	Task         string
	ModTime      time.Time
	Liveness     model.Liveness
}

type Source interface {
	Collect() []Session
}

func DecayState(raw model.AgentState, liveness model.Liveness, mtime, now time.Time, toolTimeout time.Duration) model.AgentState {
	if liveness == model.LiveActive {
		if !mtime.IsZero() && now.Sub(mtime) < toolTimeout {
			return raw
		}
		return model.StateWaiting
	}
	return model.StateIdle
}
