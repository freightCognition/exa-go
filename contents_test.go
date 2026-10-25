package exa

import (
	"context"
	"net/http"
	"testing"
)

func TestContentsFlattensOptionsAndReportsStatuses(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{
	  "requestId": "req_c",
	  "results": [{"title": "Acme", "url": "https://acme.example", "id": "https://acme.example", "text": "Acme builds rockets."}],
	  "statuses": [
	    {"id": "https://acme.example", "status": "success", "source": "cached"},
	    {"id": "https://gone.example", "status": "error", "error": {"tag": "CRAWL_NOT_FOUND", "httpStatusCode": 404}}
	  ],
	  "costDollars": {"total": 0.001, "contents": {"text": 0.001}},
	  "searchTime": 33.1
	}`)
	resp, err := c.Contents(context.Background(), &ContentsRequest{
		URLs:           []string{"https://acme.example", "https://gone.example"},
		Compliance:     ComplianceHIPAA,
		UserIdentifier: "user_1",
		ContentsOptions: ContentsOptions{
			Text:        &TextOptions{},
			Summary:     &SummaryOptions{Query: "what do they sell"},
			MaxAgeHours: Ptr(0),
			Extras:      &ExtrasOptions{Links: Ptr(3)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, got, http.MethodPost, "/contents")
	assertJSON(t, got.body, `{
	  "urls": ["https://acme.example", "https://gone.example"],
	  "compliance": "hipaa", "userIdentifier": "user_1",
	  "text": true, "summary": {"query": "what do they sell"}, "maxAgeHours": 0, "extras": {"links": 3}
	}`)
	if resp.RequestID != "req_c" || resp.SearchTime != 33.1 || resp.CostDollars.Total != 0.001 || len(resp.Results) != 1 || resp.Results[0].Text == "" {
		t.Errorf("resp = %+v", resp)
	}
	if len(resp.Statuses) != 2 || resp.Statuses[0].Source != "cached" {
		t.Fatalf("statuses = %+v", resp.Statuses)
	}
	failed := resp.Failed()
	if len(failed) != 1 || failed[0].ID != "https://gone.example" || failed[0].Error == nil ||
		failed[0].Error.Tag != "CRAWL_NOT_FOUND" || failed[0].Error.HTTPStatusCode == nil || *failed[0].Error.HTTPStatusCode != 404 {
		t.Errorf("failed = %+v", failed)
	}
}

func TestContentsByIDs(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{"results":[]}`)
	resp, err := c.Contents(context.Background(), &ContentsRequest{IDs: []string{"id1"}})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got.body, `{"ids":["id1"]}`)
	if resp.Failed() != nil {
		t.Errorf("failed = %+v", resp.Failed())
	}
}

func TestContentsNilRequest(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{}`)
	if _, err := c.Contents(context.Background(), nil); err == nil || got.calls != 0 {
		t.Fatalf("err = %v, calls = %d", err, got.calls)
	}
}
