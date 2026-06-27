package git

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

func (s *Service) Graph(ctx context.Context, dir string, refs []string, max int) ([]model.GraphLine, error) {
	args := []string{
		"log", "--graph", "--no-color", "--decorate=short", "-n", strconv.Itoa(max),
		"--format=%x1f%h%x1f%ct%x1f%an%x1f%p%x1f%D%x1f%s",
	}
	args = append(args, refs...)
	out, err := s.RunGit(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	var lines []model.GraphLine
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimRight(ln, " ") == "" {
			continue
		}
		lines = append(lines, parseGraphLine(ln))
	}
	return lines, nil
}

func parseGraphLine(line string) model.GraphLine {
	idx := strings.IndexByte(line, '\x1f')
	if idx < 0 {
		return model.GraphLine{Graph: strings.TrimRight(line, " ")}
	}
	fields := strings.Split(line[idx+1:], "\x1f")
	gl := model.GraphLine{Graph: line[:idx], HasCommit: true}
	if len(fields) > 0 {
		gl.Hash = fields[0]
	}
	if len(fields) > 1 {
		if secs, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64); err == nil {
			gl.When = time.Unix(secs, 0)
		}
	}
	if len(fields) > 2 {
		gl.Author = fields[2]
	}
	if len(fields) > 3 {
		gl.Parents = strings.Fields(fields[3])
	}
	if len(fields) > 4 {
		gl.Refs = parseRefs(fields[4])
	}
	if len(fields) > 5 {
		gl.Subject = fields[5]
	}
	return gl
}

func parseRefs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var refs []string
	for _, p := range strings.Split(s, ", ") {
		p = strings.TrimSpace(p)
		if i := strings.Index(p, "-> "); i >= 0 {
			p = p[i+3:]
		}
		p = strings.TrimPrefix(p, "tag: ")
		if p != "" {
			refs = append(refs, p)
		}
	}
	return refs
}
