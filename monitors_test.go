package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

const searchMonitorFields = `
  "id": "mon_01",
  "name": "AI evals",
  "status": "active",
  "search": {
    "query": "new AI eval tools",
    "numResults": 10,
    "includeDomains": ["github.com"],
    "contents": {"text": true, "summary": {"query": "what is it"}, "highlights": {"maxCharacters": 500}, "filterEmptyResults": true}
  },
  "trigger": {"type": "interval", "period": "1d"},
  "outputSchema": {"type": "text"},
  "metadata": {"owner": "ops"},
  "webhook": {"url": "https://hooks.example/exa", "events": ["monitor.run.completed"]},
  "nextRunAt": "2026-06-07T14:30:00Z",
  "createdAt": "2026-06-06T14:30:00Z",
  "updatedAt": "2026-06-06T14:30:00Z"`

const searchMonitorFixture = `{` + searchMonitorFields + `}`

func checkSearchMonitor(t *testing.T, m *SearchMonitor) {
	t.Helper()
	if m.ID != "mon_01" || m.Name != "AI evals" || m.Status != SearchMonitorStatusActive {
		t.Errorf("monitor = %+v", m)
	}
	s := m.Search
	if s.Query != "new AI eval tools" || *s.NumResults != 10 || s.IncludeDomains[0] != "github.com" || s.Contents == nil {
		t.Fatalf("search = %+v", s)
	}
	if s.Contents.Text == nil || s.Contents.Summary == nil || s.Contents.Summary.Query != "what is it" ||
		*s.Contents.Highlights.MaxCharacters != 500 || !*s.Contents.FilterEmptyResults {
		t.Errorf("contents = %+v", s.Contents)
	}
	if m.Trigger == nil || *m.Trigger != (SearchMonitorTrigger{Type: "interval", Period: "1d"}) {
		t.Errorf("trigger = %+v", m.Trigger)
	}
	if m.OutputSchema == nil || m.OutputSchema.Type != OutputSchemaTypeText || m.Metadata["owner"] != "ops" {
		t.Errorf("outputSchema/metadata = %+v %v", m.OutputSchema, m.Metadata)
	}
	if m.Webhook.URL != "https://hooks.example/exa" || m.Webhook.Events[0] != SearchMonitorEventRunCompleted {
		t.Errorf("webhook = %+v", m.Webhook)
	}
	if m.NextRunAt == nil || m.NextRunAt.Day() != 7 || m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		t.Errorf("times = %v %v %v", m.NextRunAt, m.CreatedAt, m.UpdatedAt)
	}
}

func TestMonitorsCreate(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{`+searchMonitorFields+`, "webhookSecret": "whsec_1"}`)
	res, err := c.Monitors.Create(context.Background(), &CreateSearchMonitorParameters{
		Name: "AI evals",
		Search: SearchMonitorSearch{
			Query:          "new AI eval tools",
			NumResults:     Ptr(10),
			IncludeDomains: []string{"github.com"},
			ExcludeDomains: []string{"spam.example"},
			Contents: &SearchMonitorContents{
				Text:               &TextOptions{},
				Summary:            &SummaryWithMaxTokensOptions{Query: "what is it"},
				Highlights:         &HighlightsOptions{MaxCharacters: Ptr(500)},
				MaxAgeHours:        Ptr(0),
				FilterEmptyResults: Ptr(true),
			},
		},
		Trigger:      &SearchMonitorTrigger{Type: "interval", Period: "1d"},
		OutputSchema: &OutputSchema{Type: OutputSchemaTypeText},
		Metadata:     map[string]string{"owner": "ops"},
		Webhook:      SearchMonitorWebhook{URL: "https://hooks.example/exa", Events: []SearchMonitorEvent{SearchMonitorEventRunCompleted}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/monitors", `{
		"name": "AI evals",
		"search": {
			"query": "new AI eval tools", "numResults": 10,
			"includeDomains": ["github.com"], "excludeDomains": ["spam.example"],
			"contents": {"text": true, "summary": {"query": "what is it"}, "highlights": {"maxCharacters": 500}, "maxAgeHours": 0, "filterEmptyResults": true}
		},
		"trigger": {"type": "interval", "period": "1d"},
		"outputSchema": {"type": "text"},
		"metadata": {"owner": "ops"},
		"webhook": {"url": "https://hooks.example/exa", "events": ["monitor.run.completed"]}}`)
	if res.WebhookSecret != "whsec_1" {
		t.Errorf("webhookSecret = %q", res.WebhookSecret)
	}
	checkSearchMonitor(t, &res.SearchMonitor)
}

func TestSearchMonitorContentsBooleans(t *testing.T) {
	var got SearchMonitorContents
	in := `{"text":true,"highlights":false,"summary":true,"context":true,"livecrawl":"never"}`
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.Text == nil || got.Highlights != nil || got.Summary == nil || got.Context == nil || got.Livecrawl != LivecrawlNever {
		t.Fatalf("got = %+v", got)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"text":true,"summary":true,"context":true,"livecrawl":"never"}` {
		t.Errorf("round trip = %s", out)
	}
	var empty SearchMonitorContents
	if err := json.Unmarshal([]byte(`null`), &empty); err != nil {
		t.Errorf("null: %v", err)
	}
	if err := json.Unmarshal([]byte(`[]`), &empty); err == nil {
		t.Error("want error for non-object")
	}
}

func TestMonitorsList(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"data":[`+searchMonitorFixture+`],"hasMore":true,"nextCursor":"cur2"}`)
	page, err := c.Monitors.List(context.Background(), &ListSearchMonitorsParams{
		ListParams: ListParams{Cursor: "cur1", Limit: 20},
		Status:     SearchMonitorStatusPaused,
		Name:       "AI",
		Metadata:   map[string]string{"owner": "ops", "env": "prod"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/monitors", "")
	q := got.Query
	if q.Get("cursor") != "cur1" || q.Get("limit") != "20" || q.Get("status") != "paused" || q.Get("name") != "AI" ||
		q.Get("metadata[owner]") != "ops" || q.Get("metadata[env]") != "prod" {
		t.Errorf("query = %v", q)
	}
	if !page.HasMore || *page.NextCursor != "cur2" || len(page.Data) != 1 {
		t.Fatalf("page = %+v", page)
	}
	checkSearchMonitor(t, &page.Data[0])
}

func TestMonitorsGetDelete(t *testing.T) {
	c, got := newAgentCaptureClient(t, searchMonitorFixture)
	ctx := context.Background()
	m, err := c.Monitors.Get(ctx, "mon_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/monitors/mon_01", "")
	checkSearchMonitor(t, m)

	m, err = c.Monitors.Delete(ctx, "mon_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/monitors/mon_01", "")
	checkSearchMonitor(t, m)
}

func TestMonitorsUpdate(t *testing.T) {
	c, got := newAgentCaptureClient(t, searchMonitorFixture)
	m, err := c.Monitors.Update(context.Background(), "mon_01", &UpdateSearchMonitorParameters{
		Name:    "renamed",
		Status:  SearchMonitorStatusPaused,
		Search:  &SearchMonitorSearch{NumResults: Ptr(5)},
		Webhook: &SearchMonitorWebhook{Events: []SearchMonitorEvent{SearchMonitorEventCreated}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPatch, "/monitors/mon_01", `{"name":"renamed","status":"paused","search":{"numResults":5},"webhook":{"events":["monitor.created"]}}`)
	checkSearchMonitor(t, m)
}

func TestMonitorsBatch(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"action":"pause","affected":2,"ids":["mon_01","mon_02"],"dry_run":false,"has_more":true}`)
	res, err := c.Monitors.Batch(context.Background(), &BatchMonitorsRequest{
		Action: BatchMonitorsActionPause,
		Filter: BatchMonitorsFilter{Name: "AI", Status: SearchMonitorStatusActive, Metadata: map[string]string{"owner": "ops"}},
		DryRun: Ptr(false),
		Limit:  100,
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/monitors/batch", `{"action":"pause","filter":{"name":"AI","status":"active","metadata":{"owner":"ops"}},"dry_run":false,"limit":100}`)
	if res.Action != BatchMonitorsActionPause || res.Affected != 2 || len(res.IDs) != 2 || res.DryRun || !res.HasMore {
		t.Errorf("res = %+v", res)
	}

	// dry_run defaults to true on the server, so leaving DryRun nil must omit it.
	if _, err := c.Monitors.Batch(context.Background(), &BatchMonitorsRequest{Action: BatchMonitorsActionDelete}); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/monitors/batch", `{"action":"delete","filter":{}}`)
}

func TestMonitorsTrigger(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"triggered":true}`)
	res, err := c.Monitors.Trigger(context.Background(), "mon_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/monitors/mon_01/trigger", "")
	if !res.Triggered {
		t.Error("triggered = false")
	}
}

const searchMonitorRunFixture = `{
  "id": "run_01",
  "monitorId": "mon_01",
  "status": "completed",
  "output": {
    "results": [{"url": "https://a.example", "title": "A"}],
    "content": "Two new tools.",
    "grounding": [{"field": "content", "citations": [{"url": "https://a.example", "title": "A"}], "confidence": "medium"}]
  },
  "failReason": null,
  "startedAt": "2026-06-06T14:30:01Z",
  "completedAt": "2026-06-06T14:30:09Z",
  "failedAt": null,
  "cancelledAt": null,
  "durationMs": 8000,
  "createdAt": "2026-06-06T14:30:00Z",
  "updatedAt": "2026-06-06T14:30:09Z"
}`

func checkSearchMonitorRun(t *testing.T, r *SearchMonitorRun) {
	t.Helper()
	if r.ID != "run_01" || r.MonitorID != "mon_01" || r.Status != SearchMonitorRunStatusCompleted || r.FailReason != "" {
		t.Errorf("run = %+v", r)
	}
	if r.StartedAt == nil || r.CompletedAt == nil || r.FailedAt != nil || r.CancelledAt != nil || *r.DurationMs != 8000 || r.CreatedAt.IsZero() {
		t.Errorf("times = %+v", r)
	}
	if r.Output == nil || r.Output.Results[0]["url"] != "https://a.example" || string(r.Output.Content) != `"Two new tools."` ||
		r.Output.Grounding[0].Citations[0].Title != "A" || r.Output.Grounding[0].Confidence != "medium" {
		t.Errorf("output = %+v", r.Output)
	}
}

func TestMonitorsListRuns(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"data":[`+searchMonitorRunFixture+`],"hasMore":false,"nextCursor":null}`)
	page, err := c.Monitors.ListRuns(context.Background(), "mon_01", &ListParams{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/monitors/mon_01/runs", "")
	if got.Query.Get("limit") != "2" || len(page.Data) != 1 {
		t.Fatalf("query = %v page = %+v", got.Query, page)
	}
	checkSearchMonitorRun(t, &page.Data[0])
}

func TestMonitorsGetRun(t *testing.T) {
	c, got := newAgentCaptureClient(t, searchMonitorRunFixture)
	r, err := c.Monitors.GetRun(context.Background(), "mon_01", "run_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/monitors/mon_01/runs/run_01", "")
	checkSearchMonitorRun(t, r)
}
