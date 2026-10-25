package exa

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
)

// recorded is one request seen by a recordingClient server.
type recorded struct {
	Method string
	Path   string // escaped
	Query  url.Values
	Body   string
	Header http.Header
}

// recordingClient returns a client whose server records the last request and
// answers with status and resp.
func recordingClient(t *testing.T, status int, resp string) (*Client, *recorded) {
	t.Helper()
	rec := &recorded{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*rec = recorded{r.Method, r.URL.EscapedPath(), r.URL.Query(), string(b), r.Header.Clone()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	})
	return c, rec
}

// expect checks method, path and (when wantBody != "") that the JSON body
// equals wantBody. An empty wantBody asserts no body was sent.
func (r *recorded) expect(t *testing.T, method, path, wantBody string) {
	t.Helper()
	if r.Method != method || r.Path != path {
		t.Errorf("request = %s %s, want %s %s", r.Method, r.Path, method, path)
	}
	if r.Header.Get("x-api-key") != "test-key" {
		t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
	}
	if wantBody == "" {
		if r.Body != "" {
			t.Errorf("body = %s, want none", r.Body)
		}
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
	}
	var got, want any
	if err := json.Unmarshal([]byte(r.Body), &got); err != nil {
		t.Fatalf("body %q: %v", r.Body, err)
	}
	if err := json.Unmarshal([]byte(wantBody), &want); err != nil {
		t.Fatalf("wantBody: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body =\n  %s\nwant\n  %s", r.Body, wantBody)
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

const websetSearchFixture = `{
  "id": "ws_search_1", "object": "webset_search", "websetId": "ws_1", "status": "running",
  "query": "AI startups in Europe", "entity": {"type": "company"},
  "criteria": [{"description": "Founded after 2020", "successRate": 42.5}],
  "count": 50, "maxPeoplePerCompany": null, "behavior": "override",
  "exclude": [{"source": "import", "id": "imp_1"}],
  "scope": [{"source": "webset", "id": "ws_0", "relationship": {"definition": "investors of", "limit": 3}}],
  "progress": {"found": 12, "analyzed": 80, "completion": 24, "timeLeft": 300},
  "recall": {"expected": {"total": 400, "confidence": "medium", "bounds": {"min": 300, "max": 500}}, "reasoning": "many"},
  "metadata": {"team": "growth"}, "canceledAt": null, "canceledReason": null,
  "createdAt": "2026-10-01T12:00:00Z", "updatedAt": "2026-10-01T12:05:00Z"
}`

const websetEnrichmentFixture = `{
  "id": "wenr_1", "object": "webset_enrichment", "status": "pending", "websetId": "ws_1",
  "title": "Employees", "description": "Number of employees", "format": "options",
  "options": [{"label": "1-10"}, {"label": "11-50"}], "instructions": "Use LinkedIn",
  "metadata": {}, "createdAt": "2026-10-01T12:00:00Z", "updatedAt": "2026-10-01T12:00:00Z"
}`

const websetFixture = `{
  "id": "ws_1", "object": "webset", "status": "running", "externalId": "crm-7", "title": "EU AI",
  "searches": [` + websetSearchFixture + `],
  "imports": [], "enrichments": [` + websetEnrichmentFixture + `], "monitors": [],
  "excludes": [{"source": "webset", "id": "ws_old"}],
  "metadata": {"owner": "anthony"}, "dashboardUrl": "https://websets.exa.ai/ws_1",
  "createdAt": "2026-10-01T12:00:00Z", "updatedAt": "2026-10-01T12:05:00Z"
}`

const companyItemFixture = `{
  "id": "witem_1", "object": "webset_item", "source": "search", "sourceId": "ws_search_1",
  "websetId": "ws_1",
  "properties": {
    "type": "company", "url": "https://acme.ai", "description": "Acme builds robots", "content": "Acme...",
    "company": {"name": "Acme", "location": "Berlin", "employees": 40, "industry": "Robotics",
      "about": "Robots", "logoUrl": "https://acme.ai/logo.png", "foundedYear": 2021,
      "headquarters": {"address": "1 Main", "city": "Berlin", "state": null, "postalCode": "10115", "country": "Germany", "countryCode": "DE"},
      "financials": {"revenueAnnual": null, "fundingTotal": 12500000, "fundingLatestRound": {"name": "Series A", "date": "2025-03", "amount": 10000000}},
      "webTraffic": {"visitsMonthly": 120000, "uniqueVisitors": null}}
  },
  "evaluations": [{"criterion": "Founded after 2020", "reasoning": "2021", "satisfied": "yes",
    "references": [{"title": "About", "snippet": null, "url": "https://acme.ai/about"}]}],
  "enrichments": [{"object": "enrichment_result", "status": "completed", "format": "options",
    "result": ["11-50"], "reasoning": "LinkedIn says 40",
    "references": [{"title": null, "snippet": "40 employees", "url": "https://linkedin.com/company/acme"}],
    "enrichmentId": "wenr_1"}],
  "createdAt": "2026-10-01T12:01:00Z", "updatedAt": "2026-10-01T12:02:00Z"
}`

const personItemFixture = `{
  "id": "witem_2", "object": "webset_item", "source": "import", "sourceId": "imp_1",
  "websetId": "ws_1",
  "properties": {
    "type": "person", "url": "https://linkedin.com/in/jane", "description": "Jane, CTO",
    "person": {"name": "Jane Doe", "location": "Paris", "position": "CTO",
      "company": {"name": "Acme", "location": null}, "pictureUrl": null,
      "workHistory": [{"title": "CTO", "location": "Paris", "dates": {"from": "2021-01", "to": null},
        "company": {"id": "c1", "name": "Acme", "linkedinUrl": "https://linkedin.com/company/acme"}}],
      "educationHistory": [{"degree": "MSc", "dates": null, "institution": {"id": null, "name": "ENS", "linkedinUrl": null}}]}
  },
  "evaluations": [], "enrichments": null,
  "createdAt": "2026-10-01T12:01:00Z", "updatedAt": "2026-10-01T12:02:00Z"
}`

func TestWebsetsCreate(t *testing.T) {
	c, rec := recordingClient(t, http.StatusCreated, websetFixture)
	ws, err := c.Websets.Create(context.Background(), &CreateWebsetParameters{
		Title: "EU AI",
		Search: &CreateWebsetParametersSearch{
			Query:               "AI startups in Europe",
			Count:               50,
			Entity:              &Entity{Type: EntityCompany},
			Criteria:            []CreateCriterionParameters{{Description: "Founded after 2020"}},
			MaxPeoplePerCompany: Ptr(2),
			Recall:              Ptr(false),
			Exclude:             []SourceRef{{Source: WebsetSourceImport, ID: "imp_1"}},
			Scope: []ScopeRef{{
				SourceRef:    SourceRef{Source: WebsetSourceWebset, ID: "ws_0"},
				Relationship: &ScopeRelationship{Definition: "investors of", Limit: 3},
			}},
		},
		Import:      []ImportRef{{SourceRef: SourceRef{Source: WebsetSourceImport, ID: "imp_2"}, Evaluate: Ptr(true)}},
		Enrichments: []CreateEnrichmentParameters{{Description: "Employees", Format: WebsetEnrichmentFormatOptions, Options: []WebsetEnrichmentOption{{Label: "1-10"}}}},
		Exclude:     []SourceRef{{Source: WebsetSourceWebset, ID: "ws_old"}},
		ExternalID:  "crm-7",
		Metadata:    map[string]string{"owner": "anthony"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets", `{
		"title": "EU AI",
		"search": {"query": "AI startups in Europe", "count": 50, "entity": {"type": "company"},
			"criteria": [{"description": "Founded after 2020"}], "maxPeoplePerCompany": 2, "recall": false,
			"exclude": [{"source": "import", "id": "imp_1"}],
			"scope": [{"source": "webset", "id": "ws_0", "relationship": {"definition": "investors of", "limit": 3}}]},
		"import": [{"source": "import", "id": "imp_2", "evaluate": true}],
		"enrichments": [{"description": "Employees", "format": "options", "options": [{"label": "1-10"}]}],
		"exclude": [{"source": "webset", "id": "ws_old"}],
		"externalId": "crm-7",
		"metadata": {"owner": "anthony"}
	}`)
	if ws.ID != "ws_1" || ws.Status != WebsetStatusRunning || ws.Title != "EU AI" || ws.ExternalID != "crm-7" {
		t.Errorf("webset = %+v", ws)
	}
	if !ws.CreatedAt.Equal(mustTime(t, "2026-10-01T12:00:00Z")) || ws.DashboardURL != "https://websets.exa.ai/ws_1" {
		t.Errorf("createdAt/dashboard = %v %q", ws.CreatedAt, ws.DashboardURL)
	}
	if len(ws.Searches) != 1 || len(ws.Enrichments) != 1 || len(ws.Excludes) != 1 || ws.Excludes[0].ID != "ws_old" {
		t.Fatalf("nested = %+v", ws)
	}
	s := ws.Searches[0]
	if s.Status != WebsetSearchStatusRunning || s.Entity == nil || s.Entity.Type != EntityCompany ||
		s.Criteria[0].SuccessRate != 42.5 || s.MaxPeoplePerCompany != nil || s.Progress.Found != 12 ||
		s.Progress.TimeLeft == nil || *s.Progress.TimeLeft != 300 || s.Recall == nil ||
		s.Recall.Expected.Bounds.Max != 500 || s.Recall.Expected.Confidence != "medium" ||
		s.Scope[0].Relationship.Limit != 3 || s.CanceledAt != nil {
		t.Errorf("search = %+v", s)
	}
	e := ws.Enrichments[0]
	if e.Format != WebsetEnrichmentFormatOptions || len(e.Options) != 2 || e.Instructions != "Use LinkedIn" || e.Status != WebsetEnrichmentStatusPending {
		t.Errorf("enrichment = %+v", e)
	}
}

func TestWebsetsList(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[`+websetFixture+`],"hasMore":true,"nextCursor":"cur_2"}`)
	page, err := c.Websets.List(context.Background(), &ListWebsetsParams{ListParams: ListParams{Cursor: "cur_1", Limit: 10}, Search: "ai"})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/websets", "")
	if rec.Query.Encode() != "cursor=cur_1&limit=10&search=ai" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
	if len(page.Data) != 1 || !page.HasMore || *page.NextCursor != "cur_2" {
		t.Errorf("page = %+v", page)
	}
	if _, err := c.Websets.List(context.Background(), nil); err != nil || len(rec.Query) != 0 {
		t.Errorf("nil params: err=%v query=%v", err, rec.Query)
	}
}

func TestWebsetsGetExpandsItems(t *testing.T) {
	body := websetFixture[:len(websetFixture)-1] + `, "items": [` + companyItemFixture + `]}`
	c, rec := recordingClient(t, 200, body)
	ws, err := c.Websets.Get(context.Background(), "ws/1", &GetWebsetParams{Expand: []string{"items"}})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/websets/ws%2F1", "")
	if rec.Query.Get("expand") != "items" {
		t.Errorf("query = %v", rec.Query)
	}
	if len(ws.Items) != 1 || ws.Items[0].ID != "witem_1" {
		t.Errorf("items = %+v", ws.Items)
	}
	if _, err := c.Websets.Get(context.Background(), "ws_1", nil); err != nil || len(rec.Query) != 0 {
		t.Errorf("nil params: err=%v query=%v", err, rec.Query)
	}
}

func TestWebsetsUpdateIsPost(t *testing.T) {
	c, rec := recordingClient(t, 200, websetFixture)
	if _, err := c.Websets.Update(context.Background(), "ws_1", &UpdateWebsetRequest{Title: "New", Metadata: Ptr(map[string]string{"a": "b"})}); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1", `{"title":"New","metadata":{"a":"b"}}`)

	// A nil map behind the pointer clears metadata.
	if _, err := c.Websets.Update(context.Background(), "ws_1", &UpdateWebsetRequest{Metadata: Ptr[map[string]string](nil)}); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1", `{"metadata":null}`)
}

func TestWebsetsDeleteAndCancel(t *testing.T) {
	c, rec := recordingClient(t, 200, websetFixture)
	ws, err := c.Websets.Delete(context.Background(), "ws_1")
	if err != nil || ws.ID != "ws_1" {
		t.Fatalf("Delete = %+v, %v", ws, err)
	}
	rec.expect(t, http.MethodDelete, "/websets/v0/websets/ws_1", "")
	if _, err := c.Websets.Cancel(context.Background(), "ws_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1/cancel", "")
}

func TestWebsetsPreview(t *testing.T) {
	c, rec := recordingClient(t, 200, `{
		"search": {"entity": {"type": "custom", "description": "podcasts"}, "criteria": [{"description": "About freight"}]},
		"enrichments": [{"description": "Host name", "format": "text"}, {"description": "Size", "format": "options", "options": [{"label": "big"}]}],
		"items": [{"id": "p1", "properties": {"type": "custom", "url": "https://pod.example", "description": "A podcast",
			"content": null, "custom": {"title": "Freight Talk", "author": "Bob", "publishedAt": null}}, "createdAt": "2026-10-01T12:00:00Z"}]
	}`)
	res, err := c.Websets.Preview(context.Background(), &PreviewWebsetParameters{
		Search: PreviewWebsetParametersSearch{Query: "freight podcasts", Entity: &Entity{Type: EntityCustom, Description: "podcasts"}, Count: 5},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/preview", `{"search":{"query":"freight podcasts","entity":{"type":"custom","description":"podcasts"},"count":5}}`)
	if rec.Query.Get("search") != "true" {
		t.Errorf("query = %v", rec.Query)
	}
	if res.Search.Entity.Type != EntityCustom || res.Search.Entity.Description != "podcasts" || res.Search.Criteria[0].Description != "About freight" {
		t.Errorf("search = %+v", res.Search)
	}
	if len(res.Enrichments) != 2 || res.Enrichments[1].Options[0].Label != "big" {
		t.Errorf("enrichments = %+v", res.Enrichments)
	}
	p := res.Items[0].Properties
	if p.Type != EntityCustom || p.Custom == nil || p.Custom.Title != "Freight Talk" || p.Content != "" {
		t.Errorf("item = %+v", p)
	}

	if _, err := c.Websets.Preview(context.Background(), &PreviewWebsetParameters{Search: PreviewWebsetParametersSearch{Query: "q"}}, false); err != nil {
		t.Fatal(err)
	}
	if len(rec.Query) != 0 {
		t.Errorf("search=false should send no query, got %v", rec.Query)
	}
}

func TestWebsetsTeamInfo(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"object":"team","id":"team_1","name":"Linehaul","concurrency":{"active":2,"queued":1},"limits":{"maxConcurrent":5,"maxQueued":null}}`)
	info, err := c.Websets.TeamInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/teams/me", "")
	if info.Name != "Linehaul" || info.Concurrency.Active != 2 || info.Limits.MaxConcurrent == nil || *info.Limits.MaxConcurrent != 5 || info.Limits.MaxQueued != nil {
		t.Errorf("info = %+v", info)
	}
}

func TestWebsetSearches(t *testing.T) {
	c, rec := recordingClient(t, 200, websetSearchFixture)
	ctx := context.Background()
	s, err := c.Websets.Searches.Create(ctx, "ws_1", &CreateWebsetSearchParameters{
		Count:    50,
		Query:    "AI startups in Europe",
		Entity:   &Entity{Type: EntityCompany},
		Criteria: []CreateCriterionParameters{{Description: "Founded after 2020"}},
		Behavior: WebsetSearchBehaviorAppend,
		Recall:   Ptr(true),
		Metadata: map[string]string{"team": "growth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1/searches", `{"count":50,"query":"AI startups in Europe","entity":{"type":"company"},
		"criteria":[{"description":"Founded after 2020"}],"recall":true,"behavior":"append","metadata":{"team":"growth"}}`)
	if s.ID != "ws_search_1" || s.WebsetID != "ws_1" || s.Behavior != WebsetSearchBehaviorOverride {
		t.Errorf("search = %+v", s)
	}
	if _, err := c.Websets.Searches.Get(ctx, "ws_1", "ws_search_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/websets/ws_1/searches/ws_search_1", "")
	if _, err := c.Websets.Searches.Cancel(ctx, "ws_1", "ws_search_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1/searches/ws_search_1/cancel", "")
}

func TestWebsetSearchCanceledDecodes(t *testing.T) {
	var s WebsetSearch
	err := json.Unmarshal([]byte(`{"id":"s","status":"canceled","canceledAt":"2026-10-02T00:00:00Z","canceledReason":"out_of_credits","entity":null,"recall":null}`), &s)
	if err != nil {
		t.Fatal(err)
	}
	if s.CanceledAt == nil || s.CanceledReason != WebsetSearchCanceledReasonOutOfCredits || s.Entity != nil || s.Recall != nil {
		t.Errorf("search = %+v", s)
	}
}

func TestWebsetEnrichments(t *testing.T) {
	c, rec := recordingClient(t, 200, websetEnrichmentFixture)
	ctx := context.Background()
	e, err := c.Websets.Enrichments.Create(ctx, "ws_1", &CreateEnrichmentParameters{
		Description: "Number of employees",
		Format:      WebsetEnrichmentFormatOptions,
		Options:     []WebsetEnrichmentOption{{Label: "1-10"}, {Label: "11-50"}},
		Metadata:    map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1/enrichments",
		`{"description":"Number of employees","format":"options","options":[{"label":"1-10"},{"label":"11-50"}],"metadata":{"k":"v"}}`)
	if e.ID != "wenr_1" || e.Title != "Employees" {
		t.Errorf("enrichment = %+v", e)
	}
	if _, err := c.Websets.Enrichments.Get(ctx, "ws_1", "wenr_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/websets/ws_1/enrichments/wenr_1", "")
	if _, err := c.Websets.Enrichments.Delete(ctx, "ws_1", "wenr_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodDelete, "/websets/v0/websets/ws_1/enrichments/wenr_1", "")
	if _, err := c.Websets.Enrichments.Cancel(ctx, "ws_1", "wenr_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/websets/ws_1/enrichments/wenr_1/cancel", "")
}

// PATCH enrichment answers 200 with no body.
func TestWebsetEnrichmentsUpdateEmptyBody(t *testing.T) {
	c, rec := recordingClient(t, 200, "")
	err := c.Websets.Enrichments.Update(context.Background(), "ws_1", "wenr_1", &UpdateEnrichmentParameters{
		Description: "Headcount",
		Format:      WebsetEnrichmentFormatNumber,
		Metadata:    Ptr[map[string]string](nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPatch, "/websets/v0/websets/ws_1/enrichments/wenr_1", `{"description":"Headcount","format":"number","metadata":null}`)
}

func TestWebsetItems(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[`+companyItemFixture+`,`+personItemFixture+`],"hasMore":false,"nextCursor":null}`)
	ctx := context.Background()
	page, err := c.Websets.Items.List(ctx, "ws_1", &ListItemsParams{ListParams: ListParams{Limit: 2}, SourceID: "ws_search_1"})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/websets/ws_1/items", "")
	if rec.Query.Encode() != "limit=2&sourceId=ws_search_1" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
	if len(page.Data) != 2 || page.HasMore || page.NextCursor != nil {
		t.Fatalf("page = %+v", page)
	}

	co := page.Data[0]
	if co.Source != WebsetItemSourceSearch || co.Properties.Type != EntityCompany || co.Properties.Company == nil {
		t.Fatalf("company item = %+v", co)
	}
	cp := co.Properties.Company
	if cp.Name != "Acme" || cp.Employees == nil || *cp.Employees != 40 || cp.FoundedYear == nil || *cp.FoundedYear != 2021 ||
		cp.Headquarters == nil || cp.Headquarters.CountryCode != "DE" || cp.Financials == nil ||
		cp.Financials.RevenueAnnual != nil || *cp.Financials.FundingTotal != 12500000 ||
		cp.Financials.FundingLatestRound.Name != "Series A" || *cp.WebTraffic.VisitsMonthly != 120000 || co.Properties.Content != "Acme..." {
		t.Errorf("company = %+v", cp)
	}
	ev := co.Evaluations[0]
	if ev.Satisfied != WebsetItemSatisfiedYes || ev.References[0].URL != "https://acme.ai/about" {
		t.Errorf("evaluation = %+v", ev)
	}
	en := co.Enrichments[0]
	if en.Status != WebsetEnrichmentStatusCompleted || en.Result[0] != "11-50" || en.EnrichmentID != "wenr_1" || en.References[0].Snippet != "40 employees" {
		t.Errorf("enrichment result = %+v", en)
	}

	pe := page.Data[1]
	if pe.Source != WebsetItemSourceImport || pe.Properties.Person == nil || pe.Enrichments != nil {
		t.Fatalf("person item = %+v", pe)
	}
	pp := pe.Properties.Person
	if pp.Name != "Jane Doe" || pp.Company == nil || pp.Company.Name != "Acme" || pp.WorkHistory[0].Dates.From != "2021-01" ||
		pp.WorkHistory[0].Company.LinkedinURL == "" || pp.EducationHistory[0].Dates != nil || pp.EducationHistory[0].Institution.Name != "ENS" {
		t.Errorf("person = %+v", pp)
	}

	c2, rec2 := recordingClient(t, 200, companyItemFixture)
	if _, err := c2.Websets.Items.Get(ctx, "ws_1", "witem_1"); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodGet, "/websets/v0/websets/ws_1/items/witem_1", "")
	if _, err := c2.Websets.Items.Delete(ctx, "ws_1", "witem_1"); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodDelete, "/websets/v0/websets/ws_1/items/witem_1", "")
}

func TestItemPropertiesArticleAndPaper(t *testing.T) {
	var a, r ItemProperties
	if err := json.Unmarshal([]byte(`{"type":"article","url":"u","description":"d","content":"c","article":{"title":"T","author":"A","publishedAt":"2026-01-01"}}`), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"type":"research_paper","url":"u","description":"d","content":null,"researchPaper":{"title":"P","author":null,"publishedAt":null}}`), &r); err != nil {
		t.Fatal(err)
	}
	if a.Type != EntityArticle || a.Article == nil || a.Article.Author != "A" || r.Type != EntityResearchPaper || r.ResearchPaper == nil || r.ResearchPaper.Title != "P" {
		t.Errorf("article = %+v, paper = %+v", a, r)
	}
}

func TestEntityOmitsEmptyDescription(t *testing.T) {
	b, _ := json.Marshal(Entity{Type: EntityPerson})
	if string(b) != `{"type":"person"}` {
		t.Errorf("entity = %s", b)
	}
}
