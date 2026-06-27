package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentStateString(t *testing.T) {
	assert.Equal(t, "RUNNING", StateRunning.String())
	assert.Equal(t, "EDITING", StateEditing.String())
	assert.Equal(t, "BROWSING", StateBrowsing.String())
	assert.Equal(t, "WAITING", StateWaiting.String())
	assert.Equal(t, "THINKING", StateThinking.String())
	assert.Equal(t, "IDLE", StateIdle.String())
}

func TestLivenessActive(t *testing.T) {
	assert.True(t, LiveActive.IsLive())
	assert.True(t, LiveRecent.IsLive())
	assert.False(t, LiveIdle.IsLive())
}
