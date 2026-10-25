package exa

// Live tests call the real Exa API. They are skipped unless EXA_LIVE=1 and
// EXA_API_KEY are set, and they spend a few cents of credits:
//
//	EXA_LIVE=1 EXA_API_KEY=... go test -run Live -v
//
// Each response is decoded strictly, so a field Exa sends that the SDK (and
// its spec) doesn't know about fails the test.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("EXA_LIVE") != "1" || os.Getenv("EXA_API_KEY") == "" {
		t.Skip("set EXA_LIVE=1 and EXA_API_KEY to run live tests")
	}
	c, err := New(WithHTTPClient(&http.Client{Timeout: 60 * time.Second}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// liveCall sends one request and decodes the body strictly into out.
func liveCall(t *testing.T, c *Client, method, base, path string, body, out any, edits ...func(*http.Request)) {
	t.Helper()
	var raw json.RawMessage
	if err := c.do(context.Background(), method, base, path, nil, body, &raw, edits...); err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		t.Errorf("%s %s: response doesn't match the Go type: %v\nbody: %.2000s", method, path, err, raw)
	}
}

func TestLiveSearch(t *testing.T) {
	c := liveClient(t)
	var resp SearchResponse
	liveCall(t, c, http.MethodPost, c.baseURL, "/search", &SearchRequest{
		Query:      "freight broker transportation management system",
		NumResults: Ptr(2),
		Contents:   &ContentsOptions{Highlights: &HighlightsOptions{}},
	}, &resp)
	if len(resp.Results) == 0 || resp.RequestID == "" {
		t.Fatalf("empty search response: %+v", resp)
	}
	if resp.CostDollars == nil || resp.CostDollars.Total <= 0 {
		t.Errorf("cost not decoded: %+v", resp.CostDollars)
	}
}

func TestLiveContents(t *testing.T) {
	c := liveClient(t)
	var resp ContentsResponse
	liveCall(t, c, http.MethodPost, c.baseURL, "/contents", &ContentsRequest{
		URLs:            []string{"https://exa.ai", "https://this-domain-does-not-exist-exa-go.invalid"},
		ContentsOptions: ContentsOptions{Text: &TextOptions{MaxCharacters: Ptr(500)}},
	}, &resp)
	if len(resp.Statuses) != 2 {
		t.Fatalf("statuses = %+v", resp.Statuses)
	}
	if len(resp.Failed()) != 1 {
		t.Errorf("want exactly one failed URL, got %+v", resp.Failed())
	}
}

func TestLiveAnswer(t *testing.T) {
	c := liveClient(t)
	var resp AnswerResponse
	liveCall(t, c, http.MethodPost, c.baseURL, "/answer", &AnswerRequest{Query: "What is a freight broker? One sentence."}, &resp)
	if s, ok := resp.Text(); !ok || s == "" {
		t.Fatalf("answer = %s", resp.Answer)
	}
}

func TestLiveAnswerStream(t *testing.T) {
	c := liveClient(t)
	var text string
	for chunk, err := range c.AnswerStream(context.Background(), &AnswerRequest{Query: "Name one US interstate highway."}) {
		if err != nil {
			t.Fatal(err)
		}
		text += chunk.Content()
	}
	if text == "" {
		t.Fatal("stream produced no text")
	}
}

// The keyless tests below cost nothing; they only need EXA_LIVE=1.
func liveOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("EXA_LIVE") != "1" {
		t.Skip("set EXA_LIVE=1 to run live tests")
	}
}

// With no key at all, Exa answers 402 with an x402 payment challenge.
func TestLivePaymentChallenge(t *testing.T) {
	liveOnly(t)
	c, _ := New(WithAPIKey("unused"))
	dropKey := func(r *http.Request) { r.Header.Del("x-api-key") }
	err := c.do(context.Background(), http.MethodPost, c.baseURL, "/search", nil, &SearchRequest{Query: "x"}, nil, dropKey)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusPaymentRequired || apiErr.Tag != "X402_PAYMENT_REQUIRED" {
		t.Fatalf("err = %v, want a 402 X402_PAYMENT_REQUIRED", err)
	}
	ch, ok := apiErr.PaymentChallenge()
	if !ok || ch.X402Version == 0 || len(ch.Accepts) == 0 {
		t.Fatalf("challenge = %+v ok=%v", ch, ok)
	}
	dec := json.NewDecoder(bytes.NewReader(apiErr.Body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&X402PaymentChallenge{}); err != nil {
		t.Errorf("402 body doesn't match X402PaymentChallenge: %v\n%s", err, apiErr.Body)
	}
}

func TestLiveBadKey(t *testing.T) {
	liveOnly(t)
	c, _ := New(WithAPIKey("not-a-real-key"))
	_, err := c.Search(context.Background(), &SearchRequest{Query: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want a 401 APIError", err)
	}
	if apiErr.Message == "" {
		t.Errorf("401 decoded without a message: %+v", apiErr)
	}
}

// Read-only list calls across every API area (no credits spent beyond auth).
func TestLiveListEndpoints(t *testing.T) {
	c := liveClient(t)
	liveCall(t, c, http.MethodGet, c.baseURL, "/agent/runs?limit=2", nil, &Page[AgentRun]{})
	liveCall(t, c, http.MethodGet, c.baseURL, "/batches?limit=2", nil, &Page[Batch]{}, withHeader("Exa-Beta", batchesBeta))
	liveCall(t, c, http.MethodGet, c.baseURL, "/monitors?limit=2", nil, &Page[SearchMonitor]{})
	liveCall(t, c, http.MethodGet, c.baseURL, "/websets/v0/websets?limit=2", nil, &Page[Webset]{})
	liveCall(t, c, http.MethodGet, c.baseURL, "/websets/v0/teams/me", nil, &TeamInfo{})
}
