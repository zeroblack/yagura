package tui

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/config"
)

func TestDetailModeTransitions(t *testing.T) {
	m := New(config.Default())
	m.applySnapshot(sampleSnap())
	require.Equal(t, detailNone, m.detail)
	m.openDetail(detailStatus)
	require.Equal(t, detailStatus, m.detail)
	m.closeDetail()
	require.Equal(t, detailNone, m.detail)
}
