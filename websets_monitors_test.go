package exa

import (
	"context"
	"net/http"
	"testing"
)

const monitorRunFixture = `{"id":"mrun_1","object":"monitor_run","monitorId":"mon_1","status":"completed","type":"search",
	"completedAt":"2026-10-02T00:10:00Z","failedAt":null,"failedReason":null,"canceledAt":null,
	"createdAt":"2026-10-02T00:00:00Z","updatedAt":"2026-10-02T00:10:00Z"}`

const monitorFixture = `{"id":"mon_1","object":"monitor","status":"enabled","websetId":"ws_1",
	"cadence":{"cron":"0 9 * * 1","timezone":"America/Chicago"},
	"behavior":{"type":"search","config":{"query":"new AI startups","criteria":[{"description":"Seed stage"}],
		"entity":{"type":"company"},"count":10,"behavior":"append"}},
	"lastRun":` + monitorRunFixture + `,"nextRunAt":"2026-10-09T14:00:00Z","metadata":{"k":"v"},
	"createdAt":"2026-10-01T00:00:00Z","updatedAt":"2026-10-01T00:00:00Z"}`

func TestWebsetMonitors(t *testing.T) {
	c, rec := recordingClient(t, http.StatusCreated, monitorFixture)
	ctx := context.Background()
	behavior := MonitorBehavior{Type: MonitorBehaviorSearch, Config: MonitorBehaviorConfig{
		Query:    "new AI startups",
		Criteria: []CreateCriterionParameters{{Description: "Seed stage"}},
		Entity:   &Entity{Type: EntityCompany},
		Count:    10,
		Behavior: WebsetSearchBehaviorAppend,
	}}
	m, err := c.Websets.Monitors.Create(ctx, &CreateMonitorParameters{
		WebsetID: "ws_1",
		Cadence:  MonitorCadence{Cron: "0 9 * * 1", Timezone: "America/Chicago"},
		Behavior: behavior,
		Metadata: map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/monitors", `{"websetId":"ws_1","cadence":{"cron":"0 9 * * 1","timezone":"America/Chicago"},
		"behavior":{"type":"search","config":{"query":"new AI startups","criteria":[{"description":"Seed stage"}],"entity":{"type":"company"},"count":10,"behavior":"append"}},
		"metadata":{"k":"v"}}`)
	if m.ID != "mon_1" || m.Status != MonitorStatusEnabled || m.Cadence.Timezone != "America/Chicago" ||
		m.Behavior.Config.Count != 10 || m.LastRun == nil || m.LastRun.Type != MonitorRunTypeSearch ||
		m.LastRun.CompletedAt == nil || m.NextRunAt == nil || !m.NextRunAt.Equal(mustTime(t, "2026-10-09T14:00:00Z")) {
		t.Errorf("monitor = %+v", m)
	}

	if _, err := c.Websets.Monitors.Get(ctx, "mon_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/monitors/mon_1", "")

	if _, err := c.Websets.Monitors.Update(ctx, "mon_1", &UpdateMonitor{Status: MonitorStatusDisabled, Cadence: &MonitorCadence{Cron: "0 0 * * *"}}); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPatch, "/websets/v0/monitors/mon_1", `{"status":"disabled","cadence":{"cron":"0 0 * * *"}}`)

	if _, err := c.Websets.Monitors.Delete(ctx, "mon_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodDelete, "/websets/v0/monitors/mon_1", "")
}

func TestWebsetMonitorsList(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[`+monitorFixture+`],"hasMore":false,"nextCursor":null}`)
	page, err := c.Websets.Monitors.List(context.Background(), &ListWebsetMonitorsParams{ListParams: ListParams{Limit: 5}, WebsetID: "ws_1"})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/monitors", "")
	if rec.Query.Encode() != "limit=5&websetId=ws_1" || len(page.Data) != 1 {
		t.Errorf("query = %s, page = %+v", rec.Query.Encode(), page)
	}
}

func TestWebsetMonitorRuns(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[`+monitorRunFixture+`],"hasMore":true,"nextCursor":"c2"}`)
	ctx := context.Background()
	page, err := c.Websets.Monitors.ListRuns(ctx, "mon_1", &ListParams{Cursor: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/monitors/mon_1/runs", "")
	if rec.Query.Get("cursor") != "c1" || len(page.Data) != 1 || page.Data[0].Status != MonitorRunStatusCompleted || *page.NextCursor != "c2" {
		t.Errorf("query = %v, page = %+v", rec.Query, page)
	}

	c2, rec2 := recordingClient(t, 200, monitorRunFixture)
	run, err := c2.Websets.Monitors.GetRun(ctx, "mon_1", "mrun_1")
	if err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodGet, "/websets/v0/monitors/mon_1/runs/mrun_1", "")
	if run.MonitorID != "mon_1" || run.FailedAt != nil {
		t.Errorf("run = %+v", run)
	}
}
