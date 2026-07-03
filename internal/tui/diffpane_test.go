package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
	"github.com/zeroblack/yagura/internal/model"
)

func TestNewestChangedFileEmptyList(t *testing.T) {
	require.Nil(t, newestChangedFile(nil))
}

func TestNewestChangedFilePicksMostRecentMtime(t *testing.T) {
	files := []model.FileChange{
		{Path: "old.go", ModTime: time.Now().Add(-time.Hour)},
		{Path: "new.go", ModTime: time.Now()},
		{Path: "middle.go", ModTime: time.Now().Add(-time.Minute)},
	}
	f := newestChangedFile(files)
	require.Equal(t, "new.go", f.Path)
}

func TestNewestChangedFileIgnoresDisplaySortOrder(t *testing.T) {
	files := []model.FileChange{
		{Path: "a.go", ModTime: time.Now()},
		{Path: "b.go", ModTime: time.Now().Add(-time.Hour)},
	}
	f := newestChangedFile(files)
	require.Equal(t, "a.go", f.Path, "auto-follow always picks by mtime regardless of how the caller ordered the slice")
}

func TestNewestChangedFileTieBreaksToFirstFound(t *testing.T) {
	same := time.Now()
	files := []model.FileChange{
		{Path: "first.go", ModTime: same},
		{Path: "second.go", ModTime: same},
	}
	f := newestChangedFile(files)
	require.Equal(t, "first.go", f.Path, "a tied mtime falls back to the first file in the slice, no secondary sort key")
}

func TestSyntheticDiffMarksEveryLineAdded(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.md"), []byte("line one\nline two\n"), 0o644))
	out, err := syntheticDiff(dir, "notes.md")
	require.NoError(t, err)
	require.Equal(t, "+line one\n+line two", out)
}

func TestSyntheticDiffEmptyFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.md"), []byte(""), 0o644))
	out, err := syntheticDiff(dir, "empty.md")
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestDiffContentForBinarySkipsGit(t *testing.T) {
	out, err := diffContentFor(nil, "/does/not/matter", model.FileChange{Path: "logo.png", Added: -1, Deleted: -1})
	require.NoError(t, err)
	require.Equal(t, "(binary file)", out)
}

func TestDiffContentForUntrackedReadsFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644))
	out, err := diffContentFor(nil, dir, model.FileChange{Path: "new.go", Status: model.StatusUntracked})
	require.NoError(t, err)
	require.Equal(t, "+package main", out)
}

func TestDiffContentForUntrackedBinarySkipsGit(t *testing.T) {
	out, err := diffContentFor(nil, "/does/not/matter", model.FileChange{Path: "logo.png", Status: model.StatusUntracked, Added: -1, Deleted: -1})
	require.NoError(t, err)
	require.Equal(t, "(binary file)", out)
}

func TestFollowNewestDiffQueuesLoadForNewestFile(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.status = model.StatusResult{Files: []model.FileChange{
		{Path: "old.go", ModTime: time.Now().Add(-time.Hour)},
		{Path: "new.go", ModTime: time.Now()},
	}}
	cmd := m.followNewestDiff()
	require.Equal(t, "new.go", m.inspect.diffTarget)
	require.False(t, m.inspect.diffLoaded)
	require.NotNil(t, cmd)
}

func TestFollowNewestDiffClearsWhenTreeIsClean(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.diffTarget = "stale.go"
	cmd := m.followNewestDiff()
	require.Equal(t, "", m.inspect.diffTarget)
	require.True(t, m.inspect.diffLoaded)
	require.Nil(t, cmd)
}

func TestInspectDiffMsgDropsStaleResults(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.diffTarget = "current.go"

	m.Update(inspectDiffMsg{wtPath: m.inspect.wt.Path, path: "other.go", content: "stale"})
	require.Empty(t, m.inspect.diffContent, "a result for a file that is no longer the target is dropped")

	m.Update(inspectDiffMsg{wtPath: m.inspect.wt.Path, path: "current.go", content: "+line"})
	require.Equal(t, "+line", m.inspect.diffContent)
	require.True(t, m.inspect.diffLoaded)
}

func TestRefreshInspectFastNoopsWithoutSelection(t *testing.T) {
	m := New(config.Default())
	require.Nil(t, m.refreshInspectFast())
}

func TestDiffIsStaleZeroModTimeIsNotStale(t *testing.T) {
	m := New(config.Default())
	require.False(t, m.diffIsStale(), "no target selected yet, nothing to call stale")
}

func TestDiffIsStaleFreshModTimeIsNotStale(t *testing.T) {
	m := New(config.Default())
	m.inspect.wt = &model.Worktree{}
	m.inspect.diffModTime = time.Now()
	require.False(t, m.diffIsStale())
}

func TestDiffIsStaleOldModTimeWithoutLiveAgentIsStale(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.inspect.wt = &model.Worktree{}
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)
	require.True(t, m.diffIsStale())
}

func TestDiffIsStaleOldModTimeWithLiveAgentIsNotStale(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.inspect.wt = &model.Worktree{Agents: []model.AgentSession{{Liveness: model.LiveActive}}}
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)
	require.False(t, m.diffIsStale(), "a live agent overrides the time threshold")
}

func TestDiffIsStaleOldModTimeWithRecentAgentIsNotStale(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.inspect.wt = &model.Worktree{Agents: []model.AgentSession{{Liveness: model.LiveRecent}}}
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)
	require.False(t, m.diffIsStale())
}

func TestDiffIsStaleOldModTimeWithIdleAgentIsStale(t *testing.T) {
	m := New(config.Default())
	m.cfg.Inspect.DiffStaleAfter = time.Minute
	m.inspect.wt = &model.Worktree{Agents: []model.AgentSession{{Liveness: model.LiveIdle}}}
	m.inspect.diffModTime = time.Now().Add(-2 * time.Minute)
	require.True(t, m.diffIsStale(), "an idle agent does not count as live")
}

func TestFollowNewestDiffStampsModTime(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	newest := time.Now()
	m.inspect.status = model.StatusResult{Files: []model.FileChange{
		{Path: "old.go", ModTime: newest.Add(-time.Hour)},
		{Path: "new.go", ModTime: newest},
	}}
	m.followNewestDiff()
	require.Equal(t, newest, m.inspect.diffModTime)
}

func TestFollowNewestDiffClearsModTimeWhenTreeIsClean(t *testing.T) {
	m := New(config.Default())
	m.width, m.height = 110, 40
	m.applySnapshot(inspectSnap())
	m.refreshInspect()
	m.inspect.diffTarget = "stale.go"
	m.inspect.diffModTime = time.Now()
	m.followNewestDiff()
	require.True(t, m.inspect.diffModTime.IsZero())
}
