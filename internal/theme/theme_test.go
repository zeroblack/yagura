package theme

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeroblack/yagura/internal/model"
)

func TestEvangelionRoles(t *testing.T) {
	th := Evangelion()
	require.Equal(t, "#4DFF8F", th.Color(RoleLive))
	require.Equal(t, "#FFA033", th.Color(RoleAmber))
}

func TestStateColor(t *testing.T) {
	th := Evangelion()
	require.Equal(t, "#2BFF85", th.StateColor(model.StateRunning))
	require.Equal(t, "#FFA033", th.StateColor(model.StateEditing))
	require.Equal(t, "#2BE8FF", th.StateColor(model.StateBrowsing))
	require.Equal(t, "#FFC400", th.StateColor(model.StateWaiting))
}

func TestEveryRoleHasColor(t *testing.T) {
	th := Evangelion()
	for r := range NumRoles {
		require.NotEmpty(t, th.Color(Role(r)), "role %d must define a color", r)
	}
}
