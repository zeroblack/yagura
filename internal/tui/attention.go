package tui

import "github.com/zeroblack/yagura/internal/model"

type attnItem struct {
	branch string
	reason string
}

func computeAttention(pr *model.PRInfo, div model.Divergence, conflicts bool) model.Attention {
	switch {
	case pr != nil && pr.CI == model.CIFailing:
		return model.Attention{Needs: true, Reason: "ci failing"}
	case pr != nil && pr.Review == model.ReviewChangesRequested:
		return model.Attention{Needs: true, Reason: "changes requested"}
	case conflicts:
		return model.Attention{Needs: true, Reason: "conflict"}
	case div.Behind > 0:
		return model.Attention{Needs: true, Reason: "behind base"}
	}
	return model.Attention{}
}

func (m *appModel) branchAttention(branch string) model.Attention {
	var pr *model.PRInfo
	if p, ok := m.inspect.prByBranch[branch]; ok {
		pr = &p
	}
	conflicts := false
	if m.inspect.wt != nil && m.inspect.wt.Branch == branch {
		conflicts = hasConflicts(m.inspect.status)
	}
	return computeAttention(pr, m.inspect.div[branch], conflicts)
}

func hasConflicts(st model.StatusResult) bool {
	for _, f := range st.Files {
		if f.Status == model.StatusConflicted {
			return true
		}
	}
	return false
}

func (m *appModel) attentionItems() []attnItem {
	if m.inspect.repo == nil {
		return nil
	}
	var out []attnItem
	for _, w := range m.inspect.repo.Worktrees {
		if a := m.branchAttention(w.Branch); a.Needs {
			out = append(out, attnItem{branch: w.Branch, reason: a.Reason})
		}
	}
	return out
}
