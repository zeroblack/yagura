package git

import "context"

func (s *Service) Diff(ctx context.Context, dir, path string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	return s.RunGit(ctx, dir, args...)
}
