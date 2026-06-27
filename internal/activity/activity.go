package activity

import (
	"sort"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

func Score(w model.Worktree) (live int, recency time.Time) {
	for _, a := range w.Agents {
		if a.Liveness == model.LiveActive && live < 2 {
			live = 2
		} else if a.Liveness == model.LiveRecent && live < 1 {
			live = 1
		}
	}
	recency = w.LastFileMod
	if w.LastCommit.After(recency) {
		recency = w.LastCommit
	}
	return live, recency
}

func SortWorktrees(wts []model.Worktree) {
	sort.SliceStable(wts, func(i, j int) bool {
		li, ri := Score(wts[i])
		lj, rj := Score(wts[j])
		if li != lj {
			return li > lj
		}
		return ri.After(rj)
	})
}

func RepoActive(r model.Repo) bool {
	for _, w := range r.Worktrees {
		if w.LiveAgents() > 0 {
			return true
		}
	}
	return false
}

func SortRepos(repos []model.Repo) {
	for i := range repos {
		SortWorktrees(repos[i].Worktrees)
	}
	sort.SliceStable(repos, func(i, j int) bool {
		ai, aj := RepoActive(repos[i]), RepoActive(repos[j])
		if ai != aj {
			return ai
		}
		return repos[i].Name < repos[j].Name
	})
}
