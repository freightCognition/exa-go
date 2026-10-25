package exa

// Regression tests for bugs found in review of the first full build.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A monitor saved with subpageTarget "about" (string form) must still decode.
func TestSubpageTargetAcceptsStringOrList(t *testing.T) {
	var m SearchMonitorContents
	if err := json.Unmarshal([]byte(`{"subpageTarget":"about"}`), &m); err != nil {
		t.Fatal(err)
	}
	var o ContentsOptions
	if err := json.Unmarshal([]byte(`{"subpageTarget":["a","b"]}`), &o); err != nil {
		t.Fatal(err)
	}
	if len(m.SubpageTarget) != 1 || m.SubpageTarget[0] != "about" || len(o.SubpageTarget) != 2 {
		t.Fatalf("monitor %v, options %v", m.SubpageTarget, o.SubpageTarget)
	}
	if err := json.Unmarshal([]byte(`{"subpageTarget":3}`), &o); err == nil {
		t.Error("a number should not decode")
	}
}

func TestUpdateSearchMonitorCanClearFields(t *testing.T) {
	got := marshal(t, UpdateSearchMonitorParameters{ClearName: true, ClearTrigger: true, ClearOutputSchema: true, ClearMetadata: true, Trigger: &SearchMonitorTrigger{Type: "interval", Period: "1h"}})
	want := `{"metadata":null,"name":null,"outputSchema":null,"trigger":null}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if got := marshal(t, UpdateSearchMonitorParameters{Name: "x"}); got != `{"name":"x"}` {
		t.Fatalf("plain update = %s", got)
	}
}

// A server or proxy that ignores Accept: text/event-stream must not look like
// an empty, successful stream.
func TestStreamRejectsNonEventStream(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"answer":"hi"}`))
	})
	var gotErr error
	n := 0
	for _, err := range c.AnswerStream(context.Background(), &AnswerRequest{Query: "q"}) {
		n++
		gotErr = err
	}
	if n != 1 || gotErr == nil || !strings.Contains(gotErr.Error(), "event stream") {
		t.Fatalf("yields=%d err=%v", n, gotErr)
	}
}

func TestUploadLeavesCallerFileOpen(t *testing.T) {
	var gotLen int64
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLen = r.ContentLength
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "rows.csv")
	if err := os.WriteFile(path, []byte("url\nhttps://a.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, _ := New(WithAPIKey("k"))
	if err := c.Websets.Imports.Upload(context.Background(), srv.URL+"/up", f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat(); err != nil {
		t.Fatalf("caller's file was closed: %v", err)
	}
	if gotLen != int64(len("url\nhttps://a.com\n")) || gotBody != "url\nhttps://a.com\n" {
		t.Fatalf("len=%d body=%q", gotLen, gotBody)
	}
}

func TestEmptyIDNeverHitsTheCollection(t *testing.T) {
	hit := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{"data":[]}`))
	})
	if _, err := c.Agent.Get(context.Background(), ""); err == nil {
		t.Error("Agent.Get(\"\") succeeded")
	}
	if _, err := c.Websets.Items.List(context.Background(), "", nil); err == nil {
		t.Error("Items.List with empty webset ID succeeded")
	}
	if hit {
		t.Error("request reached the server")
	}
}

func TestUploadValidUntilIsKeptVerbatim(t *testing.T) {
	var r CreateImportResponse
	if err := json.Unmarshal([]byte(`{"id":"i","uploadUrl":"u","uploadValidUntil":"in one hour"}`), &r); err != nil {
		t.Fatalf("a non-RFC3339 expiry lost the upload URL: %v", err)
	}
	if r.UploadURL != "u" || r.UploadValidUntil != "in one hour" {
		t.Fatalf("got %+v", r)
	}
}

// Search Monitor webhooks reuse monitor.run.* names; their payloads are not
// Websets monitor runs.
func TestEventAccessorChecksPayloadObject(t *testing.T) {
	searchRun := Event{Type: EventMonitorRunCompleted, Data: json.RawMessage(`{"id":"r","monitorId":"m","status":"completed","output":{}}`)}
	if _, err := searchRun.MonitorRun(); err == nil {
		t.Error("a Search Monitor run decoded as a Websets monitor run")
	}
	websetRun := Event{Type: EventMonitorRunCompleted, Data: json.RawMessage(`{"id":"r","object":"monitor_run","monitorId":"m","status":"completed"}`)}
	if _, err := websetRun.MonitorRun(); err != nil {
		t.Errorf("websets monitor run: %v", err)
	}
}

func TestTypedNilRequestSendsNoBody(t *testing.T) {
	var body []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{}`))
	})
	_, _ = c.Agent.Create(context.Background(), nil)
	if string(body) == "null" {
		t.Fatal("sent a JSON null body")
	}
}

func TestNegativeToleranceStillRejectsOldTimestamps(t *testing.T) {
	payload := []byte(`{"id":"evt"}`)
	ts := time.Now().Add(-time.Hour).Unix()
	header := "t=" + strconv.FormatInt(ts, 10) + ",v1=" + sign("whsec", ts, string(payload))
	if err := VerifyWebhook(payload, header, "whsec", -5*time.Minute); !errors.Is(err, ErrWebhookTimestampOutOfTolerance) {
		t.Fatalf("err = %v", err)
	}
}

func TestTransportErrorHidesPresignedQuery(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	u := srv.URL
	srv.Close()
	c, _ := New(WithAPIKey("k"))
	err := c.Websets.Imports.Upload(context.Background(), u+"/up?X-Amz-Signature=topsecret", strings.NewReader("x"))
	if err == nil || strings.Contains(err.Error(), "topsecret") {
		t.Fatalf("err = %v", err)
	}
}

func TestAgentRunEventFailure(t *testing.T) {
	e := AgentRunEvent{Event: AgentRunEventFailed, Data: json.RawMessage(`{"id":"r","status":"failed","error":{"type":"SERVER_ERROR","code":"TIMEOUT","message":"took too long"}}`)}
	f := e.Failure()
	if f == nil || f.Code != "TIMEOUT" || f.Message != "took too long" {
		t.Fatalf("failure = %+v", f)
	}
	str := AgentRunEvent{Event: AgentRunEventFailed, Data: json.RawMessage(`{"id":"r","status":"failed","error":"boom"}`)}
	if f := str.Failure(); f == nil || f.Message != "boom" {
		t.Fatalf("string failure = %+v", f)
	}
	ok := AgentRunEvent{Event: AgentRunEventCompleted, Data: json.RawMessage(`{"id":"r","status":"completed"}`)}
	if ok.Failure() != nil {
		t.Fatal("completed run reported a failure")
	}
}
