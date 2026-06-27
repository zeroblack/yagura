package discovery

import (
	"os"
	"path/filepath"
)

type Options struct {
	MaxDepth int
	Ignore   []string
}

func IgnoreSet(names []string) map[string]bool {
	set := map[string]bool{".git": true}
	for _, n := range names {
		set[n] = true
	}
	return set
}

// Find walks each root looking for git repositories. Roots are made absolute
// but tilde expansion is the caller's job (config normalizes paths on load).
func Find(roots []string, opts Options) ([]string, error) {
	ignore := IgnoreSet(opts.Ignore)
	seen := map[string]bool{}
	var repos []string
	for _, root := range roots {
		abs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		walk(abs, 0, opts.MaxDepth, ignore, seen, &repos)
	}
	return repos, nil
}

func walk(dir string, depth, maxDepth int, ignore, seen map[string]bool, out *[]string) {
	if isRepo(dir) {
		if !seen[dir] {
			seen[dir] = true
			*out = append(*out, dir)
		}
		return
	}
	if depth >= maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		// DirEntry uses lstat semantics: symlinked directories report
		// IsDir() == false, so cycles via symlinks are never followed.
		if !e.IsDir() || ignore[e.Name()] {
			continue
		}
		walk(filepath.Join(dir, e.Name()), depth+1, maxDepth, ignore, seen, out)
	}
}

func isRepo(dir string) bool {
	if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return fi.IsDir() || fi.Mode().IsRegular()
	}
	return false
}
