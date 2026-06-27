package git

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@t.io"},
		{"config", "user.name", "t"},
		{"commit", "--allow-empty", "-qm", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "git %v", args)
	}
	return dir
}

func TestServiceRunGit(t *testing.T) {
	dir := initRepo(t)
	s := NewService(4, 0)
	out, err := s.RunGit(context.Background(), dir, "rev-parse", "--abbrev-ref", "HEAD")
	require.NoError(t, err)
	require.Contains(t, out, "main")
}

func TestServiceConcurrencyBounded(t *testing.T) {
	s := NewService(2, 0)
	require.Equal(t, 2, cap(s.sem))
}

func TestServiceDefaultTimeout(t *testing.T) {
	s := NewService(2, 0)
	require.Equal(t, DefaultTimeout, s.timeout)
}

func TestServiceCommandTimeout(t *testing.T) {
	dir := initRepo(t)
	s := NewService(1, time.Nanosecond)
	_, err := s.RunGit(context.Background(), dir, "rev-parse", "HEAD")
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestServiceHonorsCanceledContext(t *testing.T) {
	dir := initRepo(t)
	s := NewService(1, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.RunGit(ctx, dir, "rev-parse", "HEAD")
	require.ErrorIs(t, err, context.Canceled)
}
