package pipeline

import (
	"fmt"
	"net/http"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/candidates"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// Publish moves approved question candidates into canonical questions — either a
// single candidate ({kind,id}) or a batch ({all,limit,dry_run}). This is the UI
// replacement for the `eduplay publish-candidates` CLI. Reviewer-only.
func (h Handler) Publish(w http.ResponseWriter, r *http.Request) {
	if !h.readyJobs(r.Context()) {
		httpx.Error(w, http.StatusConflict, "pipeline_data_unavailable", "Pipeline belum siap.")
		return
	}
	var in struct {
		Kind   string `json:"kind"`
		ID     int64  `json:"id"`
		All    bool   `json:"all"`
		Limit  int    `json:"limit"`
		DryRun bool   `json:"dry_run"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := auth.MustUser(r.Context())
	ctx := r.Context()

	if in.ID > 0 {
		kind := in.Kind
		if kind == "" {
			kind = candidates.KindQuestion
		}
		outcome, err := candidates.PublishCandidate(ctx, h.DB, kind, in.ID)
		if err != nil {
			httpx.Error(w, http.StatusConflict, "publish_failed", err.Error())
			return
		}
		audit.Log(ctx, h.DB, u.ID, "pipeline.publish", "candidate",
			fmt.Sprintf("%s:%d", kind, in.ID), map[string]any{"outcome": outcome})
		httpx.JSON(w, http.StatusOK, map[string]any{"outcome": outcome})
		return
	}

	report, err := candidates.Publish(ctx, h.DB, candidates.PublishOptions{
		Limit: in.Limit, All: in.All, DryRun: in.DryRun,
	})
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, h.DB, u.ID, "pipeline.publish", "candidate_batch", "all",
		map[string]any{"published": report.Published, "linked": report.Linked,
			"failed": report.Failed, "skipped": report.Skipped, "dry_run": report.DryRun})
	httpx.JSON(w, http.StatusOK, map[string]any{"report": report})
}
