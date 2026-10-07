package pipeline

import (
	"encoding/json"
	"testing"
	"time"
)

// TestJobRowToJobPreservesJSON guards the jsonb → json.RawMessage conversion:
// params/progress must serialise as JSON objects, not base64 strings.
func TestJobRowToJobPreservesJSON(t *testing.T) {
	row := jobRow{
		ID: 7, Kind: "run_pipeline",
		Params:   []byte(`{"sources":["a"]}`),
		Status:   "running",
		Progress: []byte(`{"current":"map","stages":{"crawl":{}}}`),
		Attempts: 1, MaxAttempts: 3, CreatedAt: time.Now(),
	}
	job := row.toJob()
	if job.Kind != "run_pipeline" || job.Status != "running" {
		t.Fatalf("unexpected job: %+v", job)
	}
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if _, ok := back["params"].(map[string]any); !ok {
		t.Fatalf("params should marshal as an object, got %T", back["params"])
	}
	if _, ok := back["progress"].(map[string]any); !ok {
		t.Fatalf("progress should marshal as an object, got %T", back["progress"])
	}
}

func TestAllowLists(t *testing.T) {
	for _, kind := range []string{"crawl", "run_pipeline", "set_source_status", "watch"} {
		if !jobKinds[kind] {
			t.Fatalf("job kind %q should be allowed", kind)
		}
	}
	if jobKinds["bogus"] {
		t.Fatal("bogus job kind must be rejected")
	}
	if !sourceStatuses["active"] || !crawlFrequencies["daily"] {
		t.Fatal("allow-list is missing an expected value")
	}
}

func TestWorkerStatusSerialization(t *testing.T) {
	now := time.Now()
	jobID := int64(42)
	info := WorkerInfo{
		Status:          "active",
		Label:           "Aktif",
		LastHeartbeatAt: &now,
		CurrentJobID:    &jobID,
		RunningJobs:     1,
		QueuedJobs:      2,
		Message:         "Worker sedang memproses",
	}
	metrics := PipelineMetrics{
		CrawledItems:       15,
		ExtractedChunks:    45,
		MaterialCandidates: 10,
		QuestionCandidates: 30,
		ApprovedQuestions:  5,
	}

	b, err := json.Marshal(map[string]any{"worker": info, "metrics": metrics})
	if err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		Worker  WorkerInfo      `json:"worker"`
		Metrics PipelineMetrics `json:"metrics"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Worker.Status != "active" || *parsed.Worker.CurrentJobID != 42 {
		t.Fatalf("unexpected worker: %+v", parsed.Worker)
	}
	if parsed.Metrics.QuestionCandidates != 30 || parsed.Metrics.ApprovedQuestions != 5 {
		t.Fatalf("unexpected metrics: %+v", parsed.Metrics)
	}
}

func TestResolveWorkerStatus(t *testing.T) {
	now := time.Now()
	recent := now.Add(-30 * time.Second)
	stale := now.Add(-6 * time.Minute)
	jobID := int64(101)

	// Case 1: Healthy active heartbeat
	w1 := ResolveWorkerStatus(now, 1, 0, nil, &jobID, &recent, "w-1", "active", &jobID, &recent, nil)
	if w1.Status != "active" || w1.Label != "Aktif" || *w1.CurrentJobID != 101 {
		t.Fatalf("expected active worker, got: %+v", w1)
	}

	// Case 2: Healthy idle heartbeat
	w2 := ResolveWorkerStatus(now, 0, 0, nil, nil, nil, "w-1", "idle", nil, &recent, nil)
	if w2.Status != "idle" || w2.Label != "Siaga" {
		t.Fatalf("expected idle worker, got: %+v", w2)
	}

	// Case 3: Explicitly stopped worker heartbeat
	w3 := ResolveWorkerStatus(now, 0, 0, nil, nil, nil, "w-1", "stopped", nil, &recent, nil)
	if w3.Status != "stopped" || w3.Label != "Offline" {
		t.Fatalf("expected stopped worker, got: %+v", w3)
	}

	// Case 4: Heartbeat stale (> 2 min), no running jobs
	w4 := ResolveWorkerStatus(now, 0, 0, nil, nil, nil, "w-1", "idle", nil, &stale, nil)
	if w4.Status != "stopped" || w4.Label != "Terhenti" {
		t.Fatalf("expected stalled/stopped worker for stale heartbeat, got: %+v", w4)
	}

	// Case 5: No heartbeat table, but job running recently
	w5 := ResolveWorkerStatus(now, 1, 0, nil, &jobID, &recent, "", "", nil, nil, nil)
	if w5.Status != "active" || w5.Label != "Aktif" || *w5.CurrentJobID != 101 {
		t.Fatalf("expected active fallback worker, got: %+v", w5)
	}

	// Case 6: No heartbeat table, running job stuck for > 5 min
	w6 := ResolveWorkerStatus(now, 1, 0, nil, &jobID, &stale, "", "", nil, nil, nil)
	if w6.Status != "stalled" || w6.Label != "Terhenti" {
		t.Fatalf("expected stalled fallback worker, got: %+v", w6)
	}

	// Case 7: No heartbeat table, queued job waiting > 2 min
	w7 := ResolveWorkerStatus(now, 0, 1, &stale, nil, nil, "", "", nil, nil, nil)
	if w7.Status != "stopped" || w7.Label != "Terhenti" {
		t.Fatalf("expected stopped queue fallback, got: %+v", w7)
	}

	// Case 8: No heartbeats ever and no recent jobs -> Offline
	w8 := ResolveWorkerStatus(now, 0, 0, nil, nil, nil, "", "", nil, nil, nil)
	if w8.Status != "stopped" || w8.Label != "Offline" {
		t.Fatalf("expected offline worker when never seen, got: %+v", w8)
	}
}
