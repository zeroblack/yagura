package forge

import (
	"testing"
	"time"

	"github.com/zeroblack/yagura/internal/model"
)

func TestParsePRListDraftAndCreatedAt(t *testing.T) {
	data := []byte(`[
		{"number":42,"headRefName":"feat/login","state":"OPEN","title":"add login","url":"u","reviewDecision":"APPROVED","isDraft":false,"createdAt":"2026-06-16T10:00:00Z","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}]},
		{"number":38,"headRefName":"chore/deps","state":"OPEN","title":"bump","url":"u","reviewDecision":"","isDraft":true,"createdAt":"2026-06-11T08:00:00Z","statusCheckRollup":[]}
	]`)
	prs, err := parsePRList(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("want 2 PRs, got %d", len(prs))
	}
	if prs[0].Draft {
		t.Fatal("PR 42 should not be draft")
	}
	if !prs[1].Draft {
		t.Fatal("PR 38 should be draft")
	}
	if prs[0].Review != model.ReviewApproved || prs[0].CI != model.CIPassing {
		t.Fatalf("PR 42 review/CI mismapped: %v %v", prs[0].Review, prs[0].CI)
	}
	want := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	if !prs[0].CreatedAt.Equal(want) {
		t.Fatalf("PR 42 createdAt = %v, want %v", prs[0].CreatedAt, want)
	}
}
