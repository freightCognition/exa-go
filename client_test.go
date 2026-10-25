package exa

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := New(WithAPIKey("test-key"), WithBaseURL(srv.URL), WithAdminBaseURL(srv.URL+"/team-management"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Setenv("EXA_API_KEY", "")
	if _, err := New(); err == nil {
		t.Fatal("expected error without an API key")
	}
}

func TestNewReadsKeyFromEnv(t *testing.T) {
	t.Setenv("EXA_API_KEY", "env-key")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("x-api-key")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.do(context.Background(), http.MethodGet, c.baseURL, "/x", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got != "env-key" {
		t.Fatalf("x-api-key = %q, want env-key", got)
	}
}

func TestExplicitKeyBeatsEnv(t *testing.T) {
	t.Setenv("EXA_API_KEY", "env-key")
	c, err := New(WithAPIKey("explicit"))
	if err != nil {
		t.Fatal(err)
	}
	if c.apiKey != "explicit" {
		t.Fatalf("apiKey = %q", c.apiKey)
	}
}

func TestDefaults(t *testing.T) {
	c, err := New(WithAPIKey("k"))
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "https://api.exa.ai" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.adminBaseURL != "https://admin-api.exa.ai/team-management" {
		t.Errorf("adminBaseURL = %q", c.adminBaseURL)
	}
}

// The community client replaced the caller's http.Client and lost its Timeout.
func TestCustomHTTPClientIsKept(t *testing.T) {
	hc := &http.Client{Timeout: 7 * time.Second}
	c, err := New(WithAPIKey("k"), WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient != hc {
		t.Fatal("custom http.Client was replaced")
	}
}

func TestBaseURLTrailingSlashTrimmed(t *testing.T) {
	c, err := New(WithAPIKey("k"), WithBaseURL("https://example.com/"))
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "https://example.com" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
}

func TestDoSendsHeadersBodyAndQuery(t *testing.T) {
	var r0 *http.Request
	var body []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		r0 = r
		body, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"ok":true}`))
	})
	var out struct {
		OK bool `json:"ok"`
	}
	q := url.Values{"limit": {"5"}}
	err := c.do(context.Background(), http.MethodPost, c.baseURL, "/thing", q, map[string]string{"a": "b"}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Error("response not decoded")
	}
	if r0.Method != http.MethodPost || r0.URL.Path != "/thing" || r0.URL.Query().Get("limit") != "5" {
		t.Errorf("got %s %s", r0.Method, r0.URL)
	}
	if r0.Header.Get("x-api-key") != "test-key" {
		t.Error("missing x-api-key")
	}
	if r0.Header.Get("Content-Type") != "application/json" {
		t.Error("missing content type")
	}
	if r0.Header.Get("User-Agent") == "" {
		t.Error("missing user agent")
	}
	if string(body) != `{"a":"b"}` {
		t.Errorf("body = %s", body)
	}
}

func TestDoWithoutBodySendsNoContentType(t *testing.T) {
	var ct string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.do(context.Background(), http.MethodDelete, c.baseURL, "/x", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if ct != "" {
		t.Errorf("Content-Type = %q on a bodyless request", ct)
	}
}

func TestDoDecodesAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"requestId":"req_1","error":"query is required","tag":"INVALID_REQUEST_BODY"}`))
	})
	err := c.do(context.Background(), http.MethodPost, c.baseURL, "/search", nil, struct{}{}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T %v, want *APIError", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Message != "query is required" || apiErr.Tag != "INVALID_REQUEST_BODY" || apiErr.RequestID != "req_1" {
		t.Errorf("apiErr = %+v", apiErr)
	}
	if apiErr.Error() == "" {
		t.Error("empty Error()")
	}
}

// Agent endpoints nest the error: {"error":{"type","code","message"}}.
func TestDoDecodesAgentErrorObject(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"type":"NOT_FOUND","code":"RUN_NOT_FOUND","message":"no such run"}}`))
	})
	err := c.do(context.Background(), http.MethodGet, c.baseURL, "/agent/runs/x", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
	if apiErr.Type != "NOT_FOUND" || apiErr.Tag != "RUN_NOT_FOUND" || apiErr.Message != "no such run" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestDoAppliesRequestEdits(t *testing.T) {
	var beta string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		beta = r.Header.Get("Exa-Beta")
		w.Write([]byte(`{}`))
	})
	if err := c.do(context.Background(), http.MethodGet, c.baseURL, "/batches", nil, nil, nil, withHeader("Exa-Beta", "batches-2026-06-06")); err != nil {
		t.Fatal(err)
	}
	if beta != "batches-2026-06-06" {
		t.Errorf("Exa-Beta = %q", beta)
	}
}

func TestStreamYieldsEventsWithQueryAndHeaders(t *testing.T) {
	var r0 *http.Request
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		r0 = r
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("id: 1\nevent: a\ndata: {\"n\":1}\n\nid: 2\ndata: {\"n\":2}\n\ndata: [DONE]\n\ndata: {\"n\":3}\n\n"))
	})
	type msg struct {
		N int `json:"n"`
	}
	var got []int
	seq := stream(context.Background(), c, http.MethodGet, c.baseURL, "/agent/runs/r1/events", url.Values{"limit": {"5"}}, nil,
		decodeJSONEvent[msg], withHeader("Last-Event-ID", "0"))
	for m, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, m.N)
	}
	if len(got) != 2 || got[1] != 2 {
		t.Errorf("got %v, want [1 2] (stop at [DONE])", got)
	}
	if r0.Method != http.MethodGet || r0.URL.Query().Get("limit") != "5" || r0.Header.Get("Last-Event-ID") != "0" || r0.Header.Get("Accept") != "text/event-stream" {
		t.Errorf("request = %s %s %v", r0.Method, r0.URL, r0.Header)
	}
}

func TestStreamReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"bad key","tag":"INVALID_API_KEY"}`))
	})
	for _, err := range stream(context.Background(), c, http.MethodPost, c.baseURL, "/search", nil, struct{}{}, decodeJSONEvent[map[string]any]) {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 401 {
			t.Fatalf("err = %v", err)
		}
		return
	}
	t.Fatal("stream yielded nothing")
}

func TestDoKeepsNonJSONErrorBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`upstream down`))
	})
	err := c.do(context.Background(), http.MethodGet, c.baseURL, "/x", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
	if apiErr.StatusCode != 502 || string(apiErr.Body) != "upstream down" || apiErr.Message != "upstream down" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestDoKeepsPaymentRequiredBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"x402Version":1,"error":"payment required","accepts":[]}`))
	})
	err := c.do(context.Background(), http.MethodPost, c.baseURL, "/search", nil, struct{}{}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 402 {
		t.Fatalf("err = %v", err)
	}
	var raw map[string]any
	if json.Unmarshal(apiErr.Body, &raw) != nil || raw["x402Version"] == nil {
		t.Errorf("402 body not kept: %s", apiErr.Body)
	}
}

func TestDoHonoursContext(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.do(ctx, http.MethodGet, c.baseURL, "/x", nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestPathEscape(t *testing.T) {
	var got string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.Write([]byte(`{}`))
	})
	if err := c.do(context.Background(), http.MethodGet, c.baseURL, "/v0/websets/"+pathEscape("a/b c"), nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got != "/v0/websets/a%2Fb%20c" {
		t.Fatalf("path = %q", got)
	}
}

func TestPtr(t *testing.T) {
	if *Ptr(3) != 3 || *Ptr("x") != "x" {
		t.Fatal("Ptr")
	}
}
