package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeroblack/yagura/internal/git"
	"github.com/zeroblack/yagura/internal/model"
)

func newestChangedFile(files []model.FileChange) *model.FileChange {
	if len(files) == 0 {
		return nil
	}
	newest := &files[0]
	for i := 1; i < len(files); i++ {
		if files[i].ModTime.After(newest.ModTime) {
			newest = &files[i]
		}
	}
	return newest
}

func diffContentFor(svc *git.Service, wtPath string, f model.FileChange) (string, error) {
	if f.Binary() {
		return "(binary file)", nil
	}
	if f.Status == model.StatusUntracked {
		return syntheticDiff(wtPath, f.Path)
	}
	return svc.Diff(context.Background(), wtPath, f.Path, f.Staged)
}

func syntheticDiff(wtPath, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(wtPath, path))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i, l := range lines {
		lines[i] = "+" + l
	}
	return strings.Join(lines, "\n"), nil
}
