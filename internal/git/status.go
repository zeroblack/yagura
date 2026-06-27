package git

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zeroblack/yagura/internal/model"
)

type lineDelta struct{ added, deleted int }

func (s *Service) Status(ctx context.Context, dir string) (model.StatusResult, error) {
	out, err := s.RunGit(ctx, dir, "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return model.StatusResult{}, err
	}
	res := parsePorcelainV2(out)

	// Line deltas are decoration on top of the file list; a failed numstat
	// (timeout, repo mid-rewrite) must not hide the files, so counts simply
	// stay at zero.
	unstaged, _ := s.numstat(ctx, dir, false)
	staged, _ := s.numstat(ctx, dir, true)
	for i := range res.Files {
		f := &res.Files[i]
		delta, ok := staged[f.Path]
		if !ok {
			delta, ok = unstaged[f.Path]
		}
		if ok {
			f.Added, f.Deleted = delta.added, delta.deleted
		}
		if fi, err := os.Stat(filepath.Join(dir, f.Path)); err == nil {
			f.ModTime = fi.ModTime()
		}
	}
	return res, nil
}

func (s *Service) numstat(ctx context.Context, dir string, staged bool) (map[string]lineDelta, error) {
	args := []string{"diff", "--numstat"}
	if staged {
		args = append(args, "--cached")
	}
	out, err := s.RunGit(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	return parseNumstat(out), nil
}

func (s *Service) Divergence(ctx context.Context, dir, base, branch string) (model.Divergence, error) {
	out, err := s.RunGit(ctx, dir, "rev-list", "--left-right", "--count", base+"..."+branch)
	if err != nil {
		return model.Divergence{}, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return model.Divergence{}, nil
	}
	behind, _ := strconv.Atoi(fields[0])
	ahead, _ := strconv.Atoi(fields[1])
	return model.Divergence{Ahead: ahead, Behind: behind}, nil
}

func parsePorcelainV2(raw string) model.StatusResult {
	var res model.StatusResult
	tokens := strings.Split(raw, "\x00")
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "" {
			continue
		}
		switch tok[0] {
		case '#':
			applyHeader(&res, tok)
		case '1':
			parts := strings.SplitN(tok, " ", 9)
			if len(parts) == 9 {
				res.Files = append(res.Files, entryFromXY(parts[1], parts[8], ""))
			}
		case '2':
			parts := strings.SplitN(tok, " ", 10)
			orig := ""
			if i+1 < len(tokens) {
				orig = tokens[i+1]
				i++
			}
			if len(parts) == 10 {
				res.Files = append(res.Files, entryFromXY(parts[1], parts[9], orig))
			}
		case 'u':
			parts := strings.SplitN(tok, " ", 11)
			if len(parts) == 11 {
				res.Files = append(res.Files, model.FileChange{Path: parts[10], Status: model.StatusConflicted})
			}
		case '?':
			res.Files = append(res.Files, model.FileChange{Path: strings.TrimPrefix(tok, "? "), Status: model.StatusUntracked})
		}
	}
	return res
}

func applyHeader(res *model.StatusResult, tok string) {
	switch {
	case strings.HasPrefix(tok, "# branch.head "):
		res.Branch = strings.TrimPrefix(tok, "# branch.head ")
	case strings.HasPrefix(tok, "# branch.upstream "):
		res.Upstream = strings.TrimPrefix(tok, "# branch.upstream ")
	case strings.HasPrefix(tok, "# branch.ab "):
		for _, f := range strings.Fields(strings.TrimPrefix(tok, "# branch.ab ")) {
			n, _ := strconv.Atoi(f[1:])
			if strings.HasPrefix(f, "+") {
				res.Ahead = n
			} else if strings.HasPrefix(f, "-") {
				res.Behind = n
			}
		}
	}
}

func entryFromXY(xy, path, orig string) model.FileChange {
	staged := xy[0] != '.'
	letter := xy[0]
	if !staged {
		letter = xy[1]
	}
	return model.FileChange{
		Path:   path,
		Orig:   orig,
		Status: statusFromLetter(letter),
		Staged: staged,
	}
}

func statusFromLetter(c byte) model.FileStatus {
	switch c {
	case 'A':
		return model.StatusAdded
	case 'D':
		return model.StatusDeleted
	case 'R', 'C':
		return model.StatusRenamed
	default:
		return model.StatusModified
	}
}

func parseNumstat(raw string) map[string]lineDelta {
	res := map[string]lineDelta{}
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		var d lineDelta
		if parts[0] == "-" {
			d.added = -1
		} else {
			d.added, _ = strconv.Atoi(parts[0])
		}
		if parts[1] == "-" {
			d.deleted = -1
		} else {
			d.deleted, _ = strconv.Atoi(parts[1])
		}
		res[numstatPath(parts[2])] = d
	}
	return res
}

func numstatPath(p string) string {
	if !strings.Contains(p, "=>") {
		return p
	}
	if l := strings.IndexByte(p, '{'); l >= 0 {
		if r := strings.IndexByte(p, '}'); r > l {
			arrow := strings.TrimSpace(p[l+1 : r])
			_, after, _ := strings.Cut(arrow, "=>")
			return filepath.Clean(p[:l] + strings.TrimSpace(after) + p[r+1:])
		}
	}
	_, after, _ := strings.Cut(p, "=>")
	return strings.TrimSpace(after)
}
