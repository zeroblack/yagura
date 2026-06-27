package worktree

import (
	"strings"

	"github.com/zeroblack/yagura/internal/model"
)

func ParsePorcelain(out string) []model.Worktree {
	var res []model.Worktree
	var cur *model.Worktree
	flush := func() {
		if cur != nil {
			switch {
			case cur.Branch == "" && cur.Detached:
				cur.Branch = "(detached)"
			case cur.Branch == "" && cur.Bare:
				cur.Branch = "(bare)"
			}
			res = append(res, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			// git lists the main worktree (or the bare repo) first.
			cur = &model.Worktree{Path: strings.TrimPrefix(line, "worktree "), IsMain: len(res) == 0}
		case cur == nil:
			continue
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = shortSHA(strings.TrimPrefix(line, "HEAD "))
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "detached":
			cur.Detached = true
		case line == "bare":
			cur.IsMain = true
			cur.Bare = true
		case line == "locked" || strings.HasPrefix(line, "locked "):
			cur.Locked = true
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			cur.Prunable = true
		}
	}
	flush()
	return res
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
