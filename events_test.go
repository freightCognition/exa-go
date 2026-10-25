package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWebsetEventsList(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[
		{"id":"ev_1","object":"event","type":"webset.item.created","data":`+companyItemFixture+`,"createdAt":"2026-10-01T12:01:00Z"},
		{"id":"ev_2","object":"event","type":"monitor.run.completed","data":`+monitorRunFixture+`,"createdAt":"2026-10-02T00:10:00Z"}
	],"hasMore":false,"nextCursor":null}`)
	after := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 10, 3, 0, 0, 0, 0, time.FixedZone("CDT", -5*3600))
	page, err := c.Websets.Events.List(context.Background(), &ListEventsParams{
		ListParams:    ListParams{Limit: 2},
		Types:         []EventType{EventWebsetItemCreated, EventMonitorRunCompleted},
		CreatedAfter:  &after,
		CreatedBefore: &before,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/events", "")
	want := "createdAfter=2026-10-01T00%3A00%3A00Z&createdBefore=2026-10-03T00%3A00%3A00-05%3A00&limit=2&types=webset.item.created&types=monitor.run.completed"
	if rec.Query.Encode() != want {
		t.Errorf("query =\n  %s\nwant\n  %s", rec.Query.Encode(), want)
	}
	if len(page.Data) != 2 {
		t.Fatalf("page = %+v", page)
	}

	item, err := page.Data[0].Item()
	if err != nil || item.ID != "witem_1" || item.Properties.Company.Name != "Acme" {
		t.Errorf("Item() = %+v, %v", item, err)
	}
	if _, err := page.Data[0].Webset(); err == nil {
		t.Error("Webset() on an item event should fail")
	}
	run, err := page.Data[1].MonitorRun()
	if err != nil || run.ID != "mrun_1" {
		t.Errorf("MonitorRun() = %+v, %v", run, err)
	}
	if _, err := page.Data[1].Monitor(); err == nil {
		t.Error("Monitor() on a monitor.run event should fail")
	}
}

func TestWebsetEventsGet(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"id":"ev_3","object":"event","type":"webset.idle","data":`+websetFixture+`,"createdAt":"2026-10-01T13:00:00Z"}`)
	ev, err := c.Websets.Events.Get(context.Background(), "ev_3")
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/events/ev_3", "")
	ws, err := ev.Webset()
	if err != nil || ws.ID != "ws_1" || ev.Type != EventWebsetIdle || !ev.CreatedAt.Equal(mustTime(t, "2026-10-01T13:00:00Z")) {
		t.Errorf("Webset() = %+v, %v (event %+v)", ws, err, ev)
	}
}

func TestEventAccessors(t *testing.T) {
	cases := []struct {
		typ  EventType
		data string
		ok   func(Event) error
	}{
		{EventWebsetSearchCompleted, websetSearchFixture, func(e Event) error { _, err := e.Search(); return err }},
		{EventImportCompleted, importFixture + `}`, func(e Event) error { _, err := e.Import(); return err }},
		{EventMonitorUpdated, monitorFixture, func(e Event) error { _, err := e.Monitor(); return err }},
	}
	for _, tc := range cases {
		e := Event{Type: tc.typ, Data: json.RawMessage(tc.data)}
		if err := tc.ok(e); err != nil {
			t.Errorf("%s: %v", tc.typ, err)
		}
	}

	// Export events have no typed payload: every accessor refuses, Data stays raw.
	exp := Event{Type: EventWebsetExportCompleted, Data: json.RawMessage(`{"id":"exp_1"}`)}
	for name, f := range map[string]func() error{
		"Webset":     func() error { _, err := exp.Webset(); return err },
		"Item":       func() error { _, err := exp.Item(); return err },
		"Search":     func() error { _, err := exp.Search(); return err },
		"Import":     func() error { _, err := exp.Import(); return err },
		"Monitor":    func() error { _, err := exp.Monitor(); return err },
		"MonitorRun": func() error { _, err := exp.MonitorRun(); return err },
	} {
		if f() == nil {
			t.Errorf("%s() on an export event should fail", name)
		}
	}
	if string(exp.Data) != `{"id":"exp_1"}` {
		t.Errorf("Data = %s", exp.Data)
	}
}
