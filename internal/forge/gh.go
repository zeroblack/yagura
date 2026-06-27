package forge

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

type ghProvider struct {
	repo  string
	run   runner
	limit int
}

func (p *ghProvider) Available(ctx context.Context) bool {
	_, err := p.run(ctx, "gh", "auth", "status")
	return err == nil
}

func (p *ghProvider) PRs(ctx context.Context) ([]model.PRInfo, error) {
	out, err := p.run(ctx, "gh", "pr", "list",
		"-R", p.repo,
		"--json", "number,headRefName,state,title,url,reviewDecision,statusCheckRollup,isDraft,createdAt",
		"--limit", strconv.Itoa(p.limit))
	if err != nil {
		return nil, err
	}
	return parsePRList(out)
}

type ghPR struct {
	Number         int       `json:"number"`
	HeadRefName    string    `json:"headRefName"`
	State          string    `json:"state"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	ReviewDecision string    `json:"reviewDecision"`
	IsDraft        bool      `json:"isDraft"`
	CreatedAt      time.Time `json:"createdAt"`
	Rollup         []struct {
		State      string `json:"state"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	} `json:"statusCheckRollup"`
}

func parsePRList(data []byte) ([]model.PRInfo, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var raw []ghPR
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	prs := make([]model.PRInfo, 0, len(raw))
	for _, r := range raw {
		prs = append(prs, model.PRInfo{
			Number:    r.Number,
			Branch:    r.HeadRefName,
			Title:     r.Title,
			URL:       r.URL,
			State:     strings.ToLower(r.State),
			Review:    mapReview(r.ReviewDecision),
			CI:        deriveCI(r),
			Draft:     r.IsDraft,
			CreatedAt: r.CreatedAt,
		})
	}
	return prs, nil
}

func mapReview(d string) model.PRReview {
	switch d {
	case "APPROVED":
		return model.ReviewApproved
	case "CHANGES_REQUESTED":
		return model.ReviewChangesRequested
	case "REVIEW_REQUIRED":
		return model.ReviewRequired
	default:
		return model.ReviewNone
	}
}

func deriveCI(r ghPR) model.CIState {
	if len(r.Rollup) == 0 {
		return model.CINone
	}
	var failing, pending, passing bool
	for _, it := range r.Rollup {
		switch effectiveCheckState(it.State, it.Status, it.Conclusion) {
		case "FAILURE", "ERROR", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE":
			failing = true
		case "PENDING", "IN_PROGRESS", "QUEUED", "EXPECTED", "WAITING", "REQUESTED":
			pending = true
		case "SUCCESS", "NEUTRAL", "SKIPPED":
			passing = true
		}
	}
	switch {
	case failing:
		return model.CIFailing
	case pending:
		return model.CIPending
	case passing:
		return model.CIPassing
	default:
		return model.CINone
	}
}

func effectiveCheckState(state, status, conclusion string) string {
	if status != "" && status != "COMPLETED" {
		return status
	}
	if conclusion != "" {
		return conclusion
	}
	return state
}
