package model

import "time"

type FileStatus int

const (
	StatusModified FileStatus = iota
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusUntracked
	StatusConflicted
)

func (s FileStatus) String() string {
	switch s {
	case StatusAdded:
		return "added"
	case StatusDeleted:
		return "deleted"
	case StatusRenamed:
		return "renamed"
	case StatusUntracked:
		return "untracked"
	case StatusConflicted:
		return "conflicted"
	default:
		return "modified"
	}
}

func (s FileStatus) Code() string {
	switch s {
	case StatusAdded:
		return "A"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	case StatusUntracked:
		return "?"
	case StatusConflicted:
		return "U"
	default:
		return "M"
	}
}

type FileChange struct {
	Path    string
	Orig    string
	Status  FileStatus
	Staged  bool
	Added   int
	Deleted int
	ModTime time.Time
}

func (c FileChange) Binary() bool { return c.Added < 0 || c.Deleted < 0 }

type StatusResult struct {
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Files    []FileChange
}

type Divergence struct {
	Ahead  int
	Behind int
}

type RepoStats struct {
	Commits       int
	Branches      int
	DefaultBranch string
}

type PRReview int

const (
	ReviewNone PRReview = iota
	ReviewRequired
	ReviewChangesRequested
	ReviewApproved
)

type CIState int

const (
	CINone CIState = iota
	CIPending
	CIPassing
	CIFailing
)

type PRInfo struct {
	Number    int
	Branch    string
	Title     string
	URL       string
	State     string
	Review    PRReview
	CI        CIState
	Draft     bool
	CreatedAt time.Time
}

type Attention struct {
	Needs  bool
	Reason string
}

type GraphLine struct {
	Graph     string
	HasCommit bool
	Hash      string
	When      time.Time
	Author    string
	Parents   []string
	Refs      []string
	Subject   string
}

func (g GraphLine) IsMerge() bool { return len(g.Parents) > 1 }
