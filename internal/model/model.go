package model

import "time"

type AgentState int

const (
	StateIdle AgentState = iota
	StateThinking
	StateRunning
	StateEditing
	StateBrowsing
	StateWaiting

	NumAgentStates int = iota
)

func (s AgentState) String() string {
	switch s {
	case StateRunning:
		return "RUNNING"
	case StateEditing:
		return "EDITING"
	case StateBrowsing:
		return "BROWSING"
	case StateWaiting:
		return "WAITING"
	case StateThinking:
		return "THINKING"
	default:
		return "IDLE"
	}
}

type Liveness int

const (
	LiveIdle Liveness = iota
	LiveRecent
	LiveActive
)

func (l Liveness) IsLive() bool { return l == LiveActive || l == LiveRecent }

type AgentSession struct {
	Tool       string
	Model      string
	State      AgentState
	Task       string
	Liveness   Liveness
	PID        int
	SessionID  string
	UpdatedAt  time.Time
	LeakTarget string
}

type Worktree struct {
	Path        string
	Branch      string
	Head        string
	IsMain      bool
	Detached    bool
	Bare        bool
	Locked      bool
	Prunable    bool
	LastCommit  time.Time
	LastFileMod time.Time
	Agents      []AgentSession
}

type Repo struct {
	Name      string
	Path      string
	Worktrees []Worktree
	Err       error
}

func (w Worktree) LiveAgents() int {
	n := 0
	for _, a := range w.Agents {
		if a.Liveness == LiveActive {
			n++
		}
	}
	return n
}

func (w Worktree) RecentAgents() int {
	n := 0
	for _, a := range w.Agents {
		if a.Liveness == LiveRecent {
			n++
		}
	}
	return n
}
