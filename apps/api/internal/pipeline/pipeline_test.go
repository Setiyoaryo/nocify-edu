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
