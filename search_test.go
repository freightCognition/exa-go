package exa

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// captured is what a test server saw.
type captured struct {
	method, path string
	header       http.Header
	body         []byte
	calls        int
}

// captureClient returns a client whose server records the request and
// answers with status, content type and body.
func captureClient(t *testing.T, status int, contentType, resp string) (*Client, *captured) {
	t.Helper()
	got := &captured{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got.calls++
		got.method, got.path, got.header = r.Method, r.URL.Path, r.Header.Clone()
		got.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	})
	return c, got
}

// assertJSON compares two JSON documents by value, ignoring key order.
func assertJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("sent body is not JSON: %v\n%s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want JSON: %v", err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("body\n got  %s\n want %s", got, want)
	}
}

func assertRequest(t *testing.T, got *captured, method, path string) {
	t.Helper()
	if got.method != method || got.path != path {
		t.Errorf("request = %s %s, want %s %s", got.method, got.path, method, path)
	}
	if got.header.Get("x-api-key") != "test-key" || got.header.Get("Content-Type") != "application/json" {
		t.Errorf("headers = %v", got.header)
	}
}

const searchResponseFixture = `{
  "requestId": "req_123",
  "searchTime": 412.5,
  "resolvedSearchType": "neural",
  "results": [{
    "title": "Acme Corp",
    "url": "https://acme.example/about",
    "publishedDate": "2025-03-01T00:00:00.000Z",
    "author": null,
    "id": "https://acme.example/about",
    "image": "https://acme.example/og.png",
    "favicon": "https://acme.example/favicon.ico",
    "text": "Acme builds rockets.",
    "highlights": ["Acme builds rockets."],
    "highlightScores": [0.82],
    "summary": "A rocket company.",
    "subpages": [{"title": "Careers", "url": "https://acme.example/careers", "id": "https://acme.example/careers", "author": "HR"}],
    "entities": [{
      "id": "ent_1", "type": "company", "version": 2,
      "properties": {
        "name": "Acme", "foundedYear": 1999, "description": "Rockets",
        "workforce": {"total": 1200},
        "headquarters": {"address": "1 Main St", "city": "Austin", "postalCode": "78701", "country": "US"},
        "financials": {"revenueAnnual": null, "fundingTotal": 5000000, "fundingLatestRound": {"name": "Series A", "date": "2020-01-01", "amount": 5000000}},
        "webTraffic": {"visitsMonthly": 10000, "countryRank": 900, "avgDurationSeconds": 61.5, "history": [{"value": 9000, "dateFrom": "2025-01-01", "dateTo": "2025-01-31"}]},
        "research": null
      }
    }],
    "extras": {
      "links": ["https://acme.example/a"],
      "imageLinks": ["https://acme.example/i.png"],
      "richImageLinks": [{"url": "https://acme.example/i.png", "alt": "logo"}],
      "richLinks": [{"url": "https://acme.example/a", "anchor": "About"}],
      "codeBlocks": [{"text": "print(1)", "source": "python"}]
    }
  }],
  "output": {
    "content": "Acme is a rocket company.",
    "grounding": [{"field": "content", "citations": [{"url": "https://acme.example/about", "title": "Acme Corp"}], "confidence": "high"}]
  },
  "warnings": [{"code": "DEPRECATED_FIELD", "path": "type", "message": "fast is legacy"}],
  "notices": [{"code": "QUERY_REWRITTEN", "path": "query", "message": "rewrote query", "details": {"from": "acme"}}],
  "costDollars": {"total": 0.015, "search": {"neural": 0.005}, "contents": {"text": 0.001}}
}`

func TestSearchSendsRequestAndDecodes(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", searchResponseFixture)
	resp, err := c.Search(context.Background(), &SearchRequest{
		Query:              "rocket companies",
		Type:               SearchTypeDeep,
		Category:           CategoryCompany,
		NumResults:         Ptr(5),
		IncludeDomains:     []string{"acme.example"},
		ExcludeDomains:     []string{"spam.example"},
		StartPublishedDate: "2025-01-01T00:00:00Z",
		EndPublishedDate:   "2025-12-31T00:00:00Z",
		Moderation:         Ptr(false),
		Contents:           &ContentsOptions{Text: &TextOptions{}, Highlights: &HighlightsOptions{MaxCharacters: Ptr(300)}},
		AdditionalQueries:  []string{"launch providers"},
		UserLocation:       &UserLocation{Country: "US"},
		Compliance:         ComplianceHIPAA,
		OutputSchema:       &OutputSchema{Type: OutputSchemaTypeText, Description: "one paragraph"},
		Objective:          "find suppliers",
		UserIdentifier:     "user_1",
		SystemPrompt:       "be brief",
		Stream:             true, // Search ignores it
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, got, http.MethodPost, "/search")
	assertJSON(t, got.body, `{
		"query": "rocket companies", "type": "deep", "category": "company", "numResults": 5,
		"includeDomains": ["acme.example"], "excludeDomains": ["spam.example"],
		"startPublishedDate": "2025-01-01T00:00:00Z", "endPublishedDate": "2025-12-31T00:00:00Z",
		"moderation": false,
		"contents": {"text": true, "highlights": {"maxCharacters": 300}},
		"additionalQueries": ["launch providers"],
		"userLocation": {"country": "US"},
		"compliance": "hipaa",
		"outputSchema": {"type": "text", "description": "one paragraph"},
		"objective": "find suppliers", "userIdentifier": "user_1", "systemPrompt": "be brief"
	}`)

	if resp.RequestID != "req_123" || resp.SearchTime != 412.5 || resp.ResolvedSearchType != "neural" {
		t.Errorf("resp = %+v", resp)
	}
	if resp.CostDollars == nil || resp.CostDollars.Total != 0.015 {
		t.Errorf("cost = %+v", resp.CostDollars)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results = %d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Title != "Acme Corp" || r.URL != "https://acme.example/about" || r.Author != nil || r.Text != "Acme builds rockets." ||
		r.Summary != "A rocket company." || len(r.HighlightScores) != 1 || r.HighlightScores[0] != 0.82 || r.Favicon == "" || r.Image == "" {
		t.Errorf("result = %+v", r)
	}
	if len(r.Subpages) != 1 || r.Subpages[0].Title != "Careers" || r.Subpages[0].Author == nil || *r.Subpages[0].Author != "HR" {
		t.Errorf("subpages = %+v", r.Subpages)
	}
	if r.Extras == nil || len(r.Extras.RichImageLinks) != 1 || r.Extras.RichImageLinks[0].Alt != "logo" ||
		r.Extras.RichLinks[0].Anchor != "About" || r.Extras.CodeBlocks[0].Source != "python" || r.Extras.Links[0] == "" {
		t.Errorf("extras = %+v", r.Extras)
	}
	if len(r.Entities) != 1 {
		t.Fatalf("entities = %d", len(r.Entities))
	}
	e := r.Entities[0]
	if e.ID != "ent_1" || e.Type != ResultEntityTypeCompany || e.Version != 2 || e.Company == nil || e.Person != nil {
		t.Fatalf("entity = %+v", e)
	}
	co := e.Company
	if co.Name != "Acme" || co.FoundedYear == nil || *co.FoundedYear != 1999 || co.Workforce.Total == nil || *co.Workforce.Total != 1200 ||
		co.Headquarters.City != "Austin" || co.Financials.RevenueAnnual != nil || *co.Financials.FundingLatestRound.Amount != 5e6 ||
		co.WebTraffic.History[0].DateTo != "2025-01-31" || co.Research != nil {
		t.Errorf("company = %+v", co)
	}

	if resp.Output == nil {
		t.Fatal("output missing")
	}
	if s, ok := resp.Output.Text(); !ok || s != "Acme is a rocket company." {
		t.Errorf("output text = %q %v", s, ok)
	}
	if g := resp.Output.Grounding; len(g) != 1 || g[0].Confidence != "high" || g[0].Citations[0].Title != "Acme Corp" {
		t.Errorf("grounding = %+v", g)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0].Code != "DEPRECATED_FIELD" {
		t.Errorf("warnings = %+v", resp.Warnings)
	}
	if len(resp.Notices) != 1 || resp.Notices[0].Details["from"] != "acme" {
		t.Errorf("notices = %+v", resp.Notices)
	}
}

func TestSearchMinimalBodyAndPlainResults(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{"requestId":"r","results":[{"title":"t","url":"https://x.example"}]}`)
	resp, err := c.Search(context.Background(), &SearchRequest{Query: "q"})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got.body, `{"query":"q"}`)
	if resp.Output != nil || len(resp.Results) != 1 {
		t.Errorf("resp = %+v", resp)
	}
}

func TestSearchDeprecatedFields(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{"results":[],"context":"ctx"}`)
	resp, err := c.Search(context.Background(), &SearchRequest{
		Query:          "q",
		StartCrawlDate: "2024-01-01T00:00:00Z",
		EndCrawlDate:   "2024-02-01T00:00:00Z",
		IncludeText:    []string{"rocket"},
		ExcludeText:    []string{"toy"},
		Context:        &ContextOptions{},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got.body, `{"query":"q","startCrawlDate":"2024-01-01T00:00:00Z","endCrawlDate":"2024-02-01T00:00:00Z","includeText":["rocket"],"excludeText":["toy"],"context":true}`)
	if resp.Context != "ctx" {
		t.Errorf("context = %q", resp.Context)
	}
}

func TestUserLocationCoordinates(t *testing.T) {
	got := marshal(t, UserLocation{Latitude: Ptr(30.27), Longitude: Ptr(-97.74)})
	if got != `{"latitude":30.27,"longitude":-97.74}` {
		t.Fatalf("got %s", got)
	}
}

func TestSearchNilRequest(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{}`)
	if _, err := c.Search(context.Background(), nil); err == nil {
		t.Fatal("want error for nil request")
	}
	if got.calls != 0 {
		t.Error("nil request was sent")
	}
}

func TestSearchAPIError(t *testing.T) {
	c, _ := captureClient(t, 400, "application/json", `{"requestId":"r1","error":"query is required","tag":"INVALID_REQUEST_BODY"}`)
	_, err := c.Search(context.Background(), &SearchRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Tag != "INVALID_REQUEST_BODY" {
		t.Fatalf("err = %v", err)
	}
}

func TestResultEntityVariants(t *testing.T) {
	var es []ResultEntity
	err := json.Unmarshal([]byte(`[
	  {"id":"p1","type":"person","version":1,"properties":{"name":"Ada","firstName":"Ada","lastName":"Lovelace","location":"London",
	    "workHistory":[{"title":"Analyst","location":null,"dates":{"from":"1840","to":null},"company":{"id":null,"name":"Babbage & Co"}}],
	    "educationHistory":[{"degree":"none","dates":null,"institution":{"id":"i1","name":"Home"}}],
	    "research":{"worksCount":3,"citationCount":100,"hIndex":2,"firstPublicationYear":1843,"latestPublicationYear":1843,"areas":["math"],
	      "notableWorks":[{"title":"Notes","year":1843,"venue":null,"citationCount":100,"doi":null,"id":"w1","type":"article"}]}}},
	  {"id":"pub1","type":"publication","version":1,"properties":{"title":"Notes","year":1843,"date":null,"type":"article","language":"en",
	    "citationCount":100,"authors":[{"name":"Ada","id":"p1"}],"referenceCount":0,"abstract":"On the engine","doi":"10.1/x"}},
	  {"id":"x1","type":"vehicle","version":1,"properties":{"wheels":4}}
	]`), &es)
	if err != nil {
		t.Fatal(err)
	}
	p := es[0].Person
	if p == nil || p.LastName != "Lovelace" || p.WorkHistory[0].Company.Name != "Babbage & Co" || p.WorkHistory[0].Dates.From != "1840" ||
		p.EducationHistory[0].Dates != nil || *p.Research.HIndex != 2 || p.Research.NotableWorks[0].Type != "article" {
		t.Errorf("person = %+v", p)
	}
	pub := es[1].Publication
	if pub == nil || pub.Type != "article" || pub.Authors[0].ID != "p1" || *pub.ReferenceCount != 0 || pub.Abstract != "On the engine" {
		t.Errorf("publication = %+v", pub)
	}
	// An unknown type keeps its raw properties and sets no typed field.
	if x := es[2]; x.Company != nil || x.Person != nil || x.Publication != nil || string(x.Properties) != `{"wheels":4}` {
		t.Errorf("unknown = %+v", x)
	}
	// Re-encoding keeps the properties.
	b, _ := json.Marshal(es[1])
	if !strings.Contains(string(b), `"abstract":"On the engine"`) {
		t.Errorf("re-encoded = %s", b)
	}
}

func TestSearchSynthesisOutputObject(t *testing.T) {
	var o SearchSynthesisOutput
	if err := json.Unmarshal([]byte(`{"content":{"name":"Acme"},"grounding":[]}`), &o); err != nil {
		t.Fatal(err)
	}
	if _, ok := o.Text(); ok {
		t.Error("object content reported as text")
	}
	var m map[string]string
	if err := json.Unmarshal(o.Content, &m); err != nil || m["name"] != "Acme" {
		t.Errorf("content = %s", o.Content)
	}
}

const searchStreamFixture = `data: {"requestId":"r1","type":"text-delta","delta":"Acme ","choices":[{"index":0,"delta":{"role":"assistant","content":"Acme "},"finish_reason":null}]}

data: {"requestId":"r1","type":"grounding","grounding":[{"field":"content","citations":[{"url":"https://acme.example","title":"Acme"}],"confidence":"medium"}],"citations":[{"url":"https://acme.example","title":"Acme","id":"c1"}]}

data: {"requestId":"r1","type":"results","results":[{"title":"Acme","url":"https://acme.example"}],"warnings":[{"code":"W","path":"p","message":"m"}],"notices":[]}

data: {"requestId":"r1","type":"stream-reset","streamReset":true}

data: {"requestId":"r1","type":"done","output":{"content":"Acme builds rockets.","grounding":[]},"searchTime":900,"costDollars":{"total":0.02},"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

func TestSearchStream(t *testing.T) {
	c, got := captureClient(t, 200, "text/event-stream", searchStreamFixture)
	req := &SearchRequest{Query: "q", OutputSchema: &OutputSchema{Type: OutputSchemaTypeText}}
	var chunks []SearchStreamChunk
	for ch, err := range c.SearchStream(context.Background(), req) {
		if err != nil {
			t.Fatal(err)
		}
		chunks = append(chunks, ch)
	}
	assertRequest(t, got, http.MethodPost, "/search")
	if got.header.Get("Accept") != "text/event-stream" {
		t.Errorf("Accept = %q", got.header.Get("Accept"))
	}
	assertJSON(t, got.body, `{"query":"q","outputSchema":{"type":"text"},"stream":true}`)
	if req.Stream {
		t.Error("SearchStream modified the caller's request")
	}
	if len(chunks) != 5 {
		t.Fatalf("chunks = %d", len(chunks))
	}
	if ch := chunks[0]; ch.Type != SearchStreamChunkTextDelta || ch.Delta != "Acme " || ch.RequestID != "r1" ||
		ch.Choices[0].Delta.Content != "Acme " || ch.Choices[0].Delta.Role != "assistant" || ch.Choices[0].FinishReason != nil {
		t.Errorf("text-delta = %+v", ch)
	}
	if ch := chunks[1]; ch.Type != SearchStreamChunkGrounding || ch.Grounding[0].Confidence != "medium" || ch.Citations[0].ID != "c1" {
		t.Errorf("grounding = %+v", ch)
	}
	if ch := chunks[2]; ch.Type != SearchStreamChunkResults || ch.Results[0].Title != "Acme" || ch.Warnings[0].Code != "W" {
		t.Errorf("results = %+v", ch)
	}
	if ch := chunks[3]; ch.Type != SearchStreamChunkReset || !ch.StreamReset {
		t.Errorf("reset = %+v", ch)
	}
	ch := chunks[4]
	if ch.Type != SearchStreamChunkDone || ch.SearchTime != 900 || ch.CostDollars.Total != 0.02 || ch.Choices[0].FinishReason == nil || *ch.Choices[0].FinishReason != "stop" {
		t.Errorf("done = %+v", ch)
	}
	if s, _ := ch.Output.Text(); s != "Acme builds rockets." {
		t.Errorf("done output = %q", s)
	}
}

func TestSearchStreamErrorFrame(t *testing.T) {
	c, _ := captureClient(t, 200, "text/event-stream",
		"data: {\"requestId\":\"r1\",\"type\":\"text-delta\",\"delta\":\"A\"}\n\n"+
			"data: {\"requestId\":\"r1\",\"type\":\"error\",\"error\":{\"message\":\"synthesis failed\"}}\n\n"+
			"data: {\"requestId\":\"r1\",\"type\":\"text-delta\",\"delta\":\"B\"}\n\n")
	var n int
	var gotErr error
	for _, err := range c.SearchStream(context.Background(), &SearchRequest{Query: "q", OutputSchema: &OutputSchema{Type: OutputSchemaTypeText}}) {
		if err != nil {
			gotErr = err
			continue
		}
		n++
	}
	var se *StreamError
	if !errors.As(gotErr, &se) || se.Message != "synthesis failed" || se.RequestID != "r1" {
		t.Fatalf("err = %v", gotErr)
	}
	if n != 1 {
		t.Errorf("chunks after error: got %d chunks, want 1", n)
	}
}

func TestSearchStreamRequiresOutputSchema(t *testing.T) {
	c, got := captureClient(t, 200, "text/event-stream", "")
	for _, err := range c.SearchStream(context.Background(), &SearchRequest{Query: "q"}) {
		if err == nil {
			t.Fatal("want error")
		}
	}
	if got.calls != 0 {
		t.Error("request sent without outputSchema")
	}
}
