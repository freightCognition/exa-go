package exa

import (
	"context"
	"net/http"
	"testing"
)

const batchFixture = `{
  "id": "batch_01",
  "object": "batch",
  "status": "completed",
  "requestCounts": {"total": 2, "completed": 1, "failed": 1},
  "createdAt": "2026-06-06T10:00:00Z",
  "expiresAt": "2026-06-07T10:00:00Z",
  "endedAt": "2026-06-06T10:05:00Z",
  "resultsUrl": "https://files.example/results.jsonl?sig=1",
  "metadata": {"job": "nightly"}
}`

func checkBatch(t *testing.T, b *Batch) {
	t.Helper()
	if b.ID != "batch_01" || b.Object != "batch" || b.Status != BatchStatusCompleted {
		t.Errorf("batch = %+v", b)
	}
	if b.RequestCounts != (BatchRequestCounts{Total: 2, Completed: 1, Failed: 1}) {
		t.Errorf("counts = %+v", b.RequestCounts)
	}
	if b.CreatedAt.IsZero() || b.ExpiresAt == nil || b.EndedAt == nil || b.EndedAt.Minute() != 5 {
		t.Errorf("times = %v %v %v", b.CreatedAt, b.ExpiresAt, b.EndedAt)
	}
	if b.ResultsURL == nil || *b.ResultsURL != "https://files.example/results.jsonl?sig=1" || b.Metadata["job"] != "nightly" {
		t.Errorf("results/metadata = %v %v", b.ResultsURL, b.Metadata)
	}
}

func checkBatchBeta(t *testing.T, got *agentCapture) {
	t.Helper()
	if got.Header.Get("Exa-Beta") != "batches-2026-06-06" {
		t.Errorf("Exa-Beta = %q", got.Header.Get("Exa-Beta"))
	}
}

func TestBatchesCreate(t *testing.T) {
	c, got := newAgentCaptureClient(t, batchFixture)
	b, err := c.Batches.Create(context.Background(), &CreateBatchRequest{
		Requests: []BatchRequestItem{
			{CustomID: "s1", Method: http.MethodPost, URL: BatchURLSearch, Body: map[string]any{"query": "eval tools"}},
			{CustomID: "a1", Method: http.MethodPost, URL: BatchURLAgentRuns, Body: &CreateAgentRunRequest{Query: "q", Beta: "ignored"}},
		},
		Metadata: map[string]string{"job": "nightly"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/batches", `{
		"requests": [
			{"customId": "s1", "method": "POST", "url": "/search", "body": {"query": "eval tools"}},
			{"customId": "a1", "method": "POST", "url": "/agent/runs", "body": {"query": "q"}}
		],
		"metadata": {"job": "nightly"}}`)
	checkBatchBeta(t, got)
	checkBatch(t, b)
}

func TestBatchesList(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"object":"list","data":[`+batchFixture+`],"hasMore":false,"nextCursor":null}`)
	page, err := c.Batches.List(context.Background(), &ListBatchesParams{ListParams: ListParams{Cursor: "c1", Limit: 3}, Status: BatchStatusCompleted})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/batches", "")
	checkBatchBeta(t, got)
	if got.Query.Get("cursor") != "c1" || got.Query.Get("limit") != "3" || got.Query.Get("status") != "completed" {
		t.Errorf("query = %v", got.Query)
	}
	if page.Object != "list" || len(page.Data) != 1 {
		t.Fatalf("page = %+v", page)
	}
	checkBatch(t, &page.Data[0])

	if _, err := c.Batches.List(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(got.Query) != 0 {
		t.Errorf("nil params sent query %v", got.Query)
	}
}

func TestBatchesGetCancel(t *testing.T) {
	c, got := newAgentCaptureClient(t, batchFixture)
	ctx := context.Background()
	b, err := c.Batches.Get(ctx, "batch_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/batches/batch_01", "")
	checkBatchBeta(t, got)
	checkBatch(t, b)

	b, err = c.Batches.Cancel(ctx, "batch_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/batches/batch_01/cancel", "")
	checkBatchBeta(t, got)
	checkBatch(t, b)
}

func TestBatchesDelete(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"id":"batch_01","object":"batch.deleted","deleted":true}`)
	res, err := c.Batches.Delete(context.Background(), "batch_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/batches/batch_01", "")
	checkBatchBeta(t, got)
	if *res != (DeletedBatch{ID: "batch_01", Object: "batch.deleted", Deleted: true}) {
		t.Errorf("res = %+v", res)
	}
}

func TestBatchInProgressHasNullTimes(t *testing.T) {
	c, _ := newAgentCaptureClient(t, `{"id":"b","object":"batch","status":"in_progress","requestCounts":{"total":1,"completed":0,"failed":0},"createdAt":"2026-06-06T10:00:00Z","expiresAt":null,"endedAt":null,"resultsUrl":null,"metadata":{}}`)
	b, err := c.Batches.Get(context.Background(), "b")
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != BatchStatusInProgress || b.ExpiresAt != nil || b.EndedAt != nil || b.ResultsURL != nil {
		t.Errorf("batch = %+v", b)
	}
}
