package theme

import "github.com/zeroblack/yagura/internal/model"

type Role int

const (
	RoleBackground Role = iota
	RoleLine
	RoleLive
	RoleSelection
	RoleTextPrimary
	RoleTextMuted
	RoleError
	RoleWarn
	RoleBranch
	RoleWorktree
	RoleMain
	RoleSha
	RoleAdd
	RoleDel
	RoleAccent
	RoleHazard
	RoleAmber
	RoleInk
	RoleStaged
	RoleUnstaged
	RoleUntracked
	RolePR
	RoleRule
	RolePin

	NumRoles int = iota
)

type Theme struct {
	Name   string
	roles  map[Role]string
	states map[model.AgentState]string
}

func (t Theme) Color(r Role) string { return t.roles[r] }

func (t Theme) StateColor(s model.AgentState) string {
	if c, ok := t.states[s]; ok {
		return c
	}
	return t.roles[RoleTextMuted]
}

func Evangelion() Theme {
	return Theme{
		Name: "evangelion",
		roles: map[Role]string{
			RoleBackground:  "#07080A",
			RoleLine:        "#232A2C",
			RoleLive:        "#4DFF8F",
			RoleSelection:   "#B98CFF",
			RoleTextPrimary: "#DDE6E1",
			RoleTextMuted:   "#79857E",
			RoleError:       "#FF3030",
			RoleWarn:        "#FFC400",
			RoleBranch:      "#2BE8FF",
			RoleWorktree:    "#B98CFF",
			RoleMain:        "#79857E",
			RoleSha:         "#FFA033",
			RoleAdd:         "#9EFF2E",
			RoleDel:         "#FF3030",
			RoleAccent:      "#36B6FF",
			RoleHazard:      "#FF3030",
			RoleAmber:       "#FFA033",
			RoleInk:         "#07080A",
			RoleStaged:      "#4DFF8F",
			RoleUnstaged:    "#FFC400",
			RoleUntracked:   "#79857E",
			RolePR:          "#20F0FF",
			RoleRule:        "#7A5C28",
			RolePin:         "#FFB454",
		},
		states: map[model.AgentState]string{
			model.StateRunning:  "#2BFF85",
			model.StateEditing:  "#FFA033",
			model.StateBrowsing: "#2BE8FF",
			model.StateWaiting:  "#FFC400",
			model.StateThinking: "#B98CFF",
			model.StateIdle:     "#79857E",
		},
	}
}

func ByName(name string) Theme {
	switch name {
	default:
		return Evangelion()
	}
}
