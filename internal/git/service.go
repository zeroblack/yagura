package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Service struct {
	sem     chan struct{}
	timeout time.Duration
}

func NewService(maxProcs int, timeout time.Duration) *Service {
	if maxProcs <= 0 {
		maxProcs = min(max(runtime.NumCPU()*2, 4), 32)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Service{sem: make(chan struct{}, maxProcs), timeout: timeout}
}

func (s *Service) RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// The deadline starts after a semaphore slot is acquired, so queue wait
	// under load never eats into the command budget; it bounds the command
	// itself (hung network filesystems, credential prompts, stale locks).
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
