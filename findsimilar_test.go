package exa

import (
	"context"
	"net/http"
	"testing"
)

func TestFindSimilar(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{
	  "requestId": "req_f",
	  "results": [{"title": "Rival", "url": "https://rival.example", "id": "https://rival.example", "score": 0.91, "text": "Rival builds rockets."}],
	  "costDollars": {"total": 0.005},
	  "searchTime": 120
	}`)
	resp, err := c.FindSimilar(context.Background(), &FindSimilarRequest{
		URL:                 "https://acme.example",
		NumResults:          Ptr(3),
		IncludeDomains:      []string{"rival.example"},
		StartPublishedDate:  "2025-01-01T00:00:00Z",
		Category:            CategoryCompany,
		ExcludeSourceDomain: Ptr(true),
		Contents:            &ContentsOptions{Text: &TextOptions{MaxCharacters: Ptr(1000)}},
		IncludeText:         []string{"rocket"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, got, http.MethodPost, "/findSimilar")
	assertJSON(t, got.body, `{
	  "url": "https://acme.example", "numResults": 3, "includeDomains": ["rival.example"],
	  "startPublishedDate": "2025-01-01T00:00:00Z", "category": "company", "excludeSourceDomain": true,
	  "contents": {"text": {"maxCharacters": 1000}}, "includeText": ["rocket"]
	}`)
	if resp.RequestID != "req_f" || resp.SearchTime != 120 || resp.CostDollars.Total != 0.005 || len(resp.Results) != 1 {
		t.Fatalf("resp = %+v", resp)
	}
	if r := resp.Results[0]; r.Score != 0.91 || r.Title != "Rival" || r.Text != "Rival builds rockets." {
		t.Errorf("result = %+v", r)
	}
}

func TestFindSimilarNilRequest(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{}`)
	if _, err := c.FindSimilar(context.Background(), nil); err == nil || got.calls != 0 {
		t.Fatalf("err = %v, calls = %d", err, got.calls)
	}
}
