package exa

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// agentCapture records the last request a test server saw. Shared by the
// Agent, Batches, Monitors and API key tests.
type agentCapture struct {
	Method, Path string
	Query        url.Values
	Header       http.Header
	Body         string
}

// newAgentCaptureClient answers every request with resp (as SSE when the
// request asks for it) and records the request.
func newAgentCaptureClient(t *testing.T, resp string) (*Client, *agentCapture) {
	t.Helper()
	got := &agentCapture{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = agentCapture{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Header: r.Header.Clone(), Body: string(b)}
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		_, _ = io.WriteString(w, resp)
	})
	return c, got
}

// expect checks method, path and body. body "" means no body; otherwise the
// sent JSON must equal it (key order aside).
func (a *agentCapture) expect(t *testing.T, method, path, body string) {
	t.Helper()
	if a.Method != method || a.Path != path {
		t.Errorf("request = %s %s, want %s %s", a.Method, a.Path, method, path)
	}
	if body == "" {
		if a.Body != "" {
			t.Errorf("body = %s, want none", a.Body)
		}
		return
	}
	var got, want any
	if err := json.Unmarshal([]byte(a.Body), &got); err != nil {
		t.Fatalf("body %q: %v", a.Body, err)
	}
	if err := json.Unmarshal([]byte(body), &want); err != nil {
		t.Fatalf("want %q: %v", body, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %s\nwant   %s", a.Body, body)
	}
}

const agentRunFixture = `{
  "id": "agent_run_01",
  "object": "agent_run",
  "status": "completed",
  "stopReason": "schema_satisfied",
  "createdAt": "2026-05-07T21:21:52.051Z",
  "completedAt": "2026-05-07T21:23:10Z",
  "request": {
    "query": "Find AI eval tools",
    "effort": "low",
    "dataSources": [{"provider": "fiber"}],
    "budget": {"maxCostDollars": 2.5}
  },
  "output": {
    "text": "Here are five tools.",
    "structured": {"tools": ["a", "b"]},
    "grounding": [{"field": "tools", "citations": [{"url": "https://a.example", "title": "A"}], "confidence": "high"}]
  },
  "usage": {"agentComputeUnits": 3, "searches": 12, "emails": 0, "phoneNumbers": 0, "dataSources": {"fiber": 2}},
  "costDollars": {"total": 0.42, "agentCompute": 0.3, "search": 0.12, "emails": 0, "phoneNumbers": 0, "dataSources": {"fiber": 0.01}}
}`

func checkAgentRun(t *testing.T, run *AgentRun) {
	t.Helper()
	if run.ID != "agent_run_01" || run.Object != "agent_run" || run.Status != AgentRunStatusCompleted || run.StopReason != AgentStopReasonSchemaSatisfied {
		t.Errorf("run = %+v", run)
	}
	if run.CreatedAt.IsZero() || run.CompletedAt == nil || run.CompletedAt.Year() != 2026 {
		t.Errorf("times = %v %v", run.CreatedAt, run.CompletedAt)
	}
	if run.Request == nil || run.Request.Query != "Find AI eval tools" || run.Request.Effort != AgentEffortLow ||
		run.Request.DataSources[0].Provider != "fiber" || *run.Request.Budget.MaxCostDollars != 2.5 {
		t.Errorf("request = %+v", run.Request)
	}
	if run.Output.Text != "Here are five tools." || strings.ReplaceAll(string(run.Output.Structured), " ", "") != `{"tools":["a","b"]}` {
		t.Errorf("output = %+v", run.Output)
	}
	g := run.Output.Grounding[0]
	if g.Field != "tools" || g.Confidence != "high" || g.Citations[0].URL != "https://a.example" || g.Citations[0].Title != "A" {
		t.Errorf("grounding = %+v", g)
	}
	if run.Usage.Searches != 12 || run.Usage.DataSources["fiber"] != 2 || run.CostDollars.Total != 0.42 || run.CostDollars.DataSources["fiber"] != 0.01 {
		t.Errorf("usage/cost = %+v %+v", run.Usage, run.CostDollars)
	}
}

func TestAgentCreate(t *testing.T) {
	c, got := newAgentCaptureClient(t, agentRunFixture)
	run, err := c.Agent.Create(context.Background(), &CreateAgentRunRequest{
		Query:        "Find AI eval tools",
		SystemPrompt: "Be brief.",
		Effort:       AgentEffortLow,
		Input: &AgentRunInput{
			Data:      []map[string]any{{"name": "Acme"}},
			Exclusion: []map[string]any{{"name": "Initech"}},
		},
		OutputSchema:   map[string]any{"type": "object"},
		PreviousRunID:  "agent_run_00",
		Metadata:       map[string]string{"team": "x"},
		DataSources:    []AgentDataSource{{Provider: "fiber"}},
		Budget:         &AgentBudget{MaxCostDollars: Ptr(2.5), MaxDurationSeconds: Ptr(0)},
		UserIdentifier: "user-1",
		Beta:           "agent-beta-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/agent/runs", `{
		"query": "Find AI eval tools", "systemPrompt": "Be brief.", "effort": "low",
		"input": {"data": [{"name": "Acme"}], "exclusion": [{"name": "Initech"}]},
		"outputSchema": {"type": "object"}, "previousRunId": "agent_run_00",
		"metadata": {"team": "x"}, "dataSources": [{"provider": "fiber"}],
		"budget": {"maxCostDollars": 2.5, "maxDurationSeconds": 0}, "userIdentifier": "user-1"}`)
	if got.Header.Get("Exa-Beta") != "agent-beta-1" {
		t.Errorf("Exa-Beta = %q", got.Header.Get("Exa-Beta"))
	}
	checkAgentRun(t, run)
}

func TestAgentCreateMinimalSendsNoBeta(t *testing.T) {
	c, got := newAgentCaptureClient(t, agentRunFixture)
	if _, err := c.Agent.Create(context.Background(), &CreateAgentRunRequest{Query: "q"}); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/agent/runs", `{"query":"q"}`)
	if _, ok := got.Header["Exa-Beta"]; ok {
		t.Error("Exa-Beta sent without Beta")
	}
}

const agentSSEFixture = "id: 1\nevent: agent_run.created\ndata: {\"id\":\"agent_run_01\",\"status\":\"queued\",\"createdAt\":\"2026-05-07T21:21:52.051Z\"}\n\n" +
	": keep-alive\n\n" +
	"id: 2\nevent: agent_run.source.added\ndata: {\"url\":\"https://a.example\"}\n\n"

func TestAgentCreateStream(t *testing.T) {
	run, _ := json.Marshal(json.RawMessage(agentRunFixture)) // compact to one data line
	c, got := newAgentCaptureClient(t, agentSSEFixture+"id: 3\nevent: agent_run.completed\ndata: "+string(run)+"\n\n")
	var evs []AgentRunEvent
	for ev, err := range c.Agent.CreateStream(context.Background(), &CreateAgentRunRequest{Query: "q", Beta: "b1"}) {
		if err != nil {
			t.Fatal(err)
		}
		evs = append(evs, ev)
	}
	got.expect(t, http.MethodPost, "/agent/runs", `{"query":"q"}`)
	if got.Header.Get("Accept") != "text/event-stream" || got.Header.Get("Exa-Beta") != "b1" {
		t.Errorf("headers = %v", got.Header)
	}
	if len(evs) != 3 {
		t.Fatalf("got %d events", len(evs))
	}
	if evs[0].ID != "1" || evs[0].Event != AgentRunEventCreated || !strings.Contains(string(evs[0].Data), `"queued"`) {
		t.Errorf("ev0 = %+v", evs[0])
	}
	if evs[1].Event != "agent_run.source.added" {
		t.Errorf("progress event not passed through: %+v", evs[1])
	}
	if evs[2].Event != AgentRunEventCompleted {
		t.Fatalf("ev2 = %+v", evs[2])
	}
	r, err := evs[2].Run()
	if err != nil {
		t.Fatal(err)
	}
	checkAgentRun(t, r)
}

func TestAgentRunEventRunRejectsBadData(t *testing.T) {
	if _, err := (AgentRunEvent{Data: json.RawMessage(`"x"`)}).Run(); err == nil {
		t.Fatal("want error")
	}
}

func TestAgentList(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"object":"list","data":[`+agentRunFixture+`],"hasMore":true,"nextCursor":"agent_run_02"}`)
	page, err := c.Agent.List(context.Background(), &ListParams{Cursor: "agent_run_00", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/agent/runs", "")
	if got.Query.Get("cursor") != "agent_run_00" || got.Query.Get("limit") != "5" {
		t.Errorf("query = %v", got.Query)
	}
	if page.Object != "list" || !page.HasMore || *page.NextCursor != "agent_run_02" || len(page.Data) != 1 {
		t.Fatalf("page = %+v", page)
	}
	checkAgentRun(t, &page.Data[0])
}

func TestAgentGetCancelStop(t *testing.T) {
	c, got := newAgentCaptureClient(t, agentRunFixture)
	ctx := context.Background()
	for _, tc := range []struct {
		method, path string
		call         func() (*AgentRun, error)
	}{
		{http.MethodGet, "/agent/runs/agent_run_01", func() (*AgentRun, error) { return c.Agent.Get(ctx, "agent_run_01") }},
		{http.MethodPost, "/agent/runs/agent_run_01/cancel", func() (*AgentRun, error) { return c.Agent.Cancel(ctx, "agent_run_01") }},
		{http.MethodPost, "/agent/runs/agent_run_01/stop", func() (*AgentRun, error) { return c.Agent.Stop(ctx, "agent_run_01") }},
	} {
		run, err := tc.call()
		if err != nil {
			t.Fatal(err)
		}
		got.expect(t, tc.method, tc.path, "")
		checkAgentRun(t, run)
	}
}

func TestAgentDelete(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"id":"agent_run_01","object":"agent_run.deleted","deleted":true}`)
	res, err := c.Agent.Delete(context.Background(), "agent_run_01")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/agent/runs/agent_run_01", "")
	if res.ID != "agent_run_01" || res.Object != "agent_run.deleted" || !res.Deleted {
		t.Errorf("res = %+v", res)
	}
}

func TestAgentEvents(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"object":"list","data":[{"id":"1","event":"agent_run.created","data":{"id":"agent_run_01","status":"queued"},"createdAt":"2026-05-07T21:21:52Z"}],"hasMore":false,"nextCursor":null}`)
	page, err := c.Agent.Events(context.Background(), "agent_run_01", &ListParams{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/agent/runs/agent_run_01/events", "")
	if got.Query.Get("limit") != "10" || got.Header.Get("Accept") != "application/json" {
		t.Errorf("query = %v accept = %q", got.Query, got.Header.Get("Accept"))
	}
	ev := page.Data[0]
	if page.HasMore || page.NextCursor != nil || ev.ID != "1" || ev.Event != AgentRunEventCreated || !ev.CreatedAt.Equal(time.Date(2026, 5, 7, 21, 21, 52, 0, time.UTC)) {
		t.Errorf("page = %+v", page)
	}
	if r, err := ev.Run(); err != nil || r.Status != AgentRunStatusQueued {
		t.Errorf("Run() = %+v, %v", r, err)
	}
}

func TestAgentStreamEvents(t *testing.T) {
	c, got := newAgentCaptureClient(t, agentSSEFixture)
	var ids []string
	for ev, err := range c.Agent.StreamEvents(context.Background(), "agent_run_01", "12") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, ev.ID)
	}
	got.expect(t, http.MethodGet, "/agent/runs/agent_run_01/events", "")
	if got.Header.Get("Accept") != "text/event-stream" || got.Header.Get("Last-Event-ID") != "12" {
		t.Errorf("headers = %v", got.Header)
	}
	if strings.Join(ids, ",") != "1,2" {
		t.Errorf("ids = %v", ids)
	}
}

func TestAgentStreamEventsWithoutLastEventID(t *testing.T) {
	c, got := newAgentCaptureClient(t, agentSSEFixture)
	for _, err := range c.Agent.StreamEvents(context.Background(), "agent_run_01", "") {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := got.Header["Last-Event-Id"]; ok {
		t.Error("Last-Event-ID sent when empty")
	}
}

func TestAgentPathEscapesID(t *testing.T) {
	var path string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		_, _ = io.WriteString(w, agentRunFixture)
	})
	if _, err := c.Agent.Get(context.Background(), "a/b"); err != nil {
		t.Fatal(err)
	}
	if path != "/agent/runs/a%2Fb" {
		t.Errorf("path = %q", path)
	}
}

func TestAgentErrorDecoded(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"type":"NOT_FOUND","code":"RUN_NOT_FOUND","message":"no such run"}}`)
	})
	_, err := c.Agent.Get(context.Background(), "x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Tag != "RUN_NOT_FOUND" {
		t.Fatalf("err = %v", err)
	}
}
