package exa

import (
	"context"
	"net/http"
	"time"
)

// batchesBeta is the Exa-Beta token every Batches call must send.
const batchesBeta = "batches-2026-06-06"

// CreateBatchRequest submits requests to run asynchronously.
type CreateBatchRequest struct {
	Requests []BatchRequestItem `json:"requests"`
	Metadata map[string]string  `json:"metadata,omitempty"`
}

// BatchRequestItem is one request in a batch.
type BatchRequestItem struct {
	// CustomID identifies the request's line in the results (1-64 chars).
	CustomID string `json:"customId"`
	// Method must be "POST".
	Method string `json:"method"`
	// URL is BatchURLSearch or BatchURLAgentRuns.
	URL string `json:"url"`
	// Body is the request body for URL, e.g. a *CreateAgentRunRequest or a
	// map. It must encode as a JSON object.
	Body any `json:"body"`
}

// Endpoints a BatchRequestItem can target.
const (
	BatchURLSearch    = "/search"
	BatchURLAgentRuns = "/agent/runs"
)

// Batch is a submitted batch.
type Batch struct {
	ID            string             `json:"id"`
	Object        string             `json:"object"`
	Status        BatchStatus        `json:"status"`
	RequestCounts BatchRequestCounts `json:"requestCounts"`
	CreatedAt     time.Time          `json:"createdAt"`
	ExpiresAt     *time.Time         `json:"expiresAt"`
	EndedAt       *time.Time         `json:"endedAt"`
	// ResultsURL is a presigned URL to the JSONL results once the batch has
	// ended; fetch it with a plain GET.
	ResultsURL *string           `json:"resultsUrl"`
	Metadata   map[string]string `json:"metadata"`
}

// BatchStatus is a batch's lifecycle state.
type BatchStatus string

// BatchStatus values.
const (
	BatchStatusInProgress BatchStatus = "in_progress"
	BatchStatusCompleted  BatchStatus = "completed"
	BatchStatusCancelling BatchStatus = "cancelling"
	BatchStatusCancelled  BatchStatus = "cancelled"
	BatchStatusExpired    BatchStatus = "expired"
)

// BatchRequestCounts tallies a batch's requests.
type BatchRequestCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// DeletedBatch confirms a deleted batch.
type DeletedBatch struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// ListBatchesParams filter Batches.List.
type ListBatchesParams struct {
	ListParams
	// Status filters by state; the API accepts only BatchStatusCompleted.
	Status BatchStatus
}

// Create submits a batch.
func (s *BatchesService) Create(ctx context.Context, req *CreateBatchRequest) (*Batch, error) {
	return s.batch(ctx, http.MethodPost, "/batches", req)
}

// List returns batches.
func (s *BatchesService) List(ctx context.Context, p *ListBatchesParams) (*Page[Batch], error) {
	var lp *ListParams
	if p != nil {
		lp = &p.ListParams
	}
	q := lp.values()
	if p != nil && p.Status != "" {
		q.Set("status", string(p.Status))
	}
	var out Page[Batch]
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/batches", q, nil, &out, withHeader("Exa-Beta", batchesBeta)); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns a batch.
func (s *BatchesService) Get(ctx context.Context, id string) (*Batch, error) {
	return s.batch(ctx, http.MethodGet, "/batches/"+pathEscape(id), nil)
}

// Delete deletes a batch.
func (s *BatchesService) Delete(ctx context.Context, id string) (*DeletedBatch, error) {
	var out DeletedBatch
	if err := s.c.do(ctx, http.MethodDelete, s.c.baseURL, "/batches/"+pathEscape(id), nil, nil, &out, withHeader("Exa-Beta", batchesBeta)); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel stops a batch; requests already finished keep their results.
func (s *BatchesService) Cancel(ctx context.Context, id string) (*Batch, error) {
	return s.batch(ctx, http.MethodPost, "/batches/"+pathEscape(id)+"/cancel", nil)
}

func (s *BatchesService) batch(ctx context.Context, method, path string, body any) (*Batch, error) {
	var out Batch
	if err := s.c.do(ctx, method, s.c.baseURL, path, nil, body, &out, withHeader("Exa-Beta", batchesBeta)); err != nil {
		return nil, err
	}
	return &out, nil
}
