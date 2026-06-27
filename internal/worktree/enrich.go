package worktree

import (
	"context"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeroblack/yagura/internal/git"
)

func LatestFileMod(dir string, ignore map[string]bool, maxDepth int) time.Time {
	var latest time.Time
	base := strings.Count(dir, string(filepath.Separator))
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignore[d.Name()] {
				return filepath.SkipDir
			}
			if strings.Count(path, string(filepath.Separator))-base > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if fi, err := d.Info(); err == nil && fi.ModTime().After(latest) {
			latest = fi.ModTime()
		}
		return nil
	})
	return latest
}

func LastCommit(ctx context.Context, svc *git.Service, dir string) time.Time {
	out, err := svc.RunGit(ctx, dir, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(secs, 0)
}
