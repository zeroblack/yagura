package git

import (
	"context"
	"strconv"
	"strings"

	"github.com/zeroblack/yagura/internal/model"
)

func (s *Service) RepoStats(ctx context.Context, dir string) (model.RepoStats, error) {
	var st model.RepoStats
	if out, err := s.RunGit(ctx, dir, "rev-list", "--count", "HEAD"); err == nil {
		st.Commits, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	if out, err := s.RunGit(ctx, dir, "for-each-ref", "--format=%(refname)", "refs/heads"); err == nil {
		st.Branches = countNonEmptyLines(out)
	}
	st.DefaultBranch = s.defaultBranch(ctx, dir)
	return st, nil
}

func (s *Service) defaultBranch(ctx context.Context, dir string) string {
	if out, err := s.RunGit(ctx, dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(strings.TrimSpace(out), "origin/")
	}
	for _, b := range []string{"main", "master"} {
		if _, err := s.RunGit(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+b); err == nil {
			return b
		}
	}
	return ""
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
