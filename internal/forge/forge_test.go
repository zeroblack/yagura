package forge

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

const ghFixture = `[
  {"number":42,"headRefName":"feat/landing","state":"OPEN","title":"Hero","url":"https://github.com/o/r/pull/42","reviewDecision":"REVIEW_REQUIRED","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}]},
  {"number":41,"headRefName":"fix/quiz","state":"OPEN","title":"Latex","url":"https://github.com/o/r/pull/41","reviewDecision":"APPROVED","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"},{"status":"IN_PROGRESS","conclusion":""}]},
  {"number":40,"headRefName":"exp/x","state":"MERGED","title":"X","url":"https://github.com/o/r/pull/40","reviewDecision":"CHANGES_REQUESTED","statusCheckRollup":[{"status":"COMPLETED","conclusion":"FAILURE"}]}
]`

func TestParsePRList(t *testing.T) {
	prs, err := parsePRList([]byte(ghFixture))
	require.NoError(t, err)
	require.Len(t, prs, 3)

	byBranch := map[string]model.PRInfo{}
	for _, pr := range prs {
		byBranch[pr.Branch] = pr
	}

	require.Equal(t, 42, byBranch["feat/landing"].Number)
	require.Equal(t, "open", byBranch["feat/landing"].State)
	require.Equal(t, model.ReviewRequired, byBranch["feat/landing"].Review)
	require.Equal(t, model.CIPassing, byBranch["feat/landing"].CI)

	require.Equal(t, model.ReviewApproved, byBranch["fix/quiz"].Review)
	require.Equal(t, model.CIPending, byBranch["fix/quiz"].CI)

	require.Equal(t, "merged", byBranch["exp/x"].State)
	require.Equal(t, model.ReviewChangesRequested, byBranch["exp/x"].Review)
	require.Equal(t, model.CIFailing, byBranch["exp/x"].CI)
}

func TestManagerCachesAndDegrades(t *testing.T) {
	calls := 0
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return []byte("https://github.com/o/r.git\n"), nil
		}
		if name == "gh" && len(args) > 0 && args[0] == "auth" {
			return nil, nil
		}
		if name == "gh" && len(args) > 0 && args[0] == "pr" {
			calls++
			return []byte(ghFixture), nil
		}
		return nil, errors.New("unexpected")
	}
	man := newManager("auto", time.Minute, run)
	ctx := context.Background()

	prs := man.PRs(ctx, "/repo")
	require.Len(t, prs, 3)
	man.PRs(ctx, "/repo")
	require.Equal(t, 1, calls, "second call within TTL must hit cache")
}

func TestManagerNonGithubReturnsNil(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return []byte("https://bitbucket.org/o/r.git\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	man := newManager("auto", time.Minute, run)
	require.Nil(t, man.PRs(context.Background(), "/repo"))
}

func TestGithubSlug(t *testing.T) {
	require.Equal(t, "o/r", githubSlug("https://github.com/o/r.git"))
	require.Equal(t, "o/r", githubSlug("git@github.com:o/r.git"))
	require.Equal(t, "o/r", githubSlug("https://github.com/o/r"))
	require.Equal(t, "zeroblack/yagura", githubSlug("git@github.com:zeroblack/yagura.git"))
}

func TestManagerDisabled(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("should not be called")
	}
	man := newManager("off", time.Minute, run)
	require.Nil(t, man.PRs(context.Background(), "/repo"))
}

func TestManagerCachesFailuresWithShorterTTL(t *testing.T) {
	attempts := 0
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return []byte("https://github.com/o/r.git\n"), nil
		}
		attempts++
		return nil, errors.New("gh down")
	}
	man := newManager("on", time.Minute, run)
	now := time.Unix(1000, 0)
	man.now = func() time.Time { return now }

	require.Nil(t, man.PRs(context.Background(), "/repo"))
	first := attempts
	require.Nil(t, man.PRs(context.Background(), "/repo"))
	require.Equal(t, first, attempts, "a failed fetch must not retry before its error TTL")

	now = now.Add(errTTL + time.Second)
	require.Nil(t, man.PRs(context.Background(), "/repo"))
	require.Greater(t, attempts, first, "after the error TTL the fetch retries")
}

func TestManagerSkipsWhenFetchInFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	prCalls := 0
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return []byte("https://github.com/o/r.git\n"), nil
		}
		mu.Lock()
		prCalls++
		first := prCalls == 1
		mu.Unlock()
		if first {
			close(started)
			<-release
		}
		return []byte(ghFixture), nil
	}
	man := newManager("on", time.Minute, run)

	done := make(chan struct{})
	go func() {
		man.PRs(context.Background(), "/repo")
		close(done)
	}()
	<-started

	require.Nil(t, man.PRs(context.Background(), "/repo"), "a second call while one is in flight must not spawn")
	mu.Lock()
	require.Equal(t, 1, prCalls)
	mu.Unlock()

	close(release)
	<-done
	require.Len(t, man.PRs(context.Background(), "/repo"), 3)
}

func TestManagerBoundsEverySubprocessWithADeadline(t *testing.T) {
	missing := 0
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if _, ok := ctx.Deadline(); !ok {
			missing++
		}
		if name == "git" {
			return []byte("https://github.com/o/r.git\n"), nil
		}
		return []byte(ghFixture), nil
	}
	man := newManager("auto", time.Minute, run)
	man.PRs(context.Background(), "/repo")
	require.Zero(t, missing, "every forge subprocess must run under a deadline")
}

func TestManagerCachesRemoteDetection(t *testing.T) {
	gitCalls := 0
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			gitCalls++
			return []byte("https://github.com/o/r.git\n"), nil
		}
		return []byte(ghFixture), nil
	}
	man := newManager("on", time.Minute, run)
	now := time.Unix(1000, 0)
	man.now = func() time.Time { return now }

	man.PRs(context.Background(), "/repo")
	now = now.Add(2 * time.Minute)
	man.PRs(context.Background(), "/repo")
	require.Equal(t, 1, gitCalls, "the remote URL must be resolved once, not per refresh")
}
