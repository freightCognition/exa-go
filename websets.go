package exa

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// websetsPrefix roots every Websets path. The spec serves its /v0 paths from
// https://api.exa.ai/websets; /v0/teams/me omits that server but lives there too.
const websetsPrefix = "/websets/v0"

// websetsDo sends a Websets request and decodes the response into a new T.
func websetsDo[T any](ctx context.Context, c *Client, method, path string, query url.Values, body any) (*T, error) {
	var out T
	if err := c.do(ctx, method, c.baseURL, websetsPrefix+path, query, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- websets ----

// WebsetStatus is a webset's lifecycle state.
type WebsetStatus string

// Webset statuses.
const (
	WebsetStatusIdle    WebsetStatus = "idle"
	WebsetStatusPending WebsetStatus = "pending"
	WebsetStatusRunning WebsetStatus = "running"
	WebsetStatusPaused  WebsetStatus = "paused"
)

// Webset is a collection of entities found by searches and imports, with
// enrichments and monitors attached.
type Webset struct {
	ID          string             `json:"id"`
	Object      string             `json:"object"`
	Status      WebsetStatus       `json:"status"`
	ExternalID  string             `json:"externalId"`
	Title       string             `json:"title"`
	Searches    []WebsetSearch     `json:"searches"`
	Imports     []Import           `json:"imports"`
	Enrichments []WebsetEnrichment `json:"enrichments"`
	Monitors    []Monitor          `json:"monitors"`
	// Excludes are imports and websets whose items this webset skips.
	Excludes     []SourceRef       `json:"excludes,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	DashboardURL string            `json:"dashboardUrl"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// GetWebsetResponse is a webset as returned by Get.
type GetWebsetResponse struct {
	Webset
	// Items is set only when Get is called with Expand "items".
	Items []WebsetItem `json:"items,omitempty"`
}

// CreateWebsetParameters create a webset. Set Search, Import, or both.
type CreateWebsetParameters struct {
	Title       string                        `json:"title,omitempty"`
	Search      *CreateWebsetParametersSearch `json:"search,omitempty"`
	Import      []ImportRef                   `json:"import,omitempty"`
	Enrichments []CreateEnrichmentParameters  `json:"enrichments,omitempty"`
	Exclude     []SourceRef                   `json:"exclude,omitempty"`
	// ExternalID is your own id for the webset (up to 300 characters); a
	// second create with the same one fails with 409.
	ExternalID string            `json:"externalId,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// CreateWebsetParametersSearch is the initial search of a new webset.
type CreateWebsetParametersSearch struct {
	Query string `json:"query"`
	// Count is how many items to find; the API defaults to 10.
	Count  int     `json:"count,omitempty"`
	Entity *Entity `json:"entity,omitempty"`
	// Criteria (1 to 5) are generated from Query when omitted.
	Criteria            []CreateCriterionParameters `json:"criteria,omitempty"`
	MaxPeoplePerCompany *int                        `json:"maxPeoplePerCompany,omitempty"`
	// Recall asks for an estimate of how many matching items exist.
	Recall  *bool       `json:"recall,omitempty"`
	Exclude []SourceRef `json:"exclude,omitempty"`
	Scope   []ScopeRef  `json:"scope,omitempty"`
}

// UpdateWebsetRequest changes a webset's title or metadata.
type UpdateWebsetRequest struct {
	Title string `json:"title,omitempty"`
	// Metadata replaces the webset's metadata. Nil leaves it unchanged; a
	// pointer to a nil map sends null and clears it.
	Metadata *map[string]string `json:"metadata,omitempty"`
}

// ListWebsetsParams filter List.
type ListWebsetsParams struct {
	ListParams
	// Search matches webset titles (2 to 50 characters).
	Search string
}

// GetWebsetParams tune Get.
type GetWebsetParams struct {
	// Expand adds related resources; the only value is "items".
	Expand []string
}

// Create creates a webset and starts its search.
func (s *WebsetsService) Create(ctx context.Context, p *CreateWebsetParameters) (*Webset, error) {
	return websetsDo[Webset](ctx, s.c, http.MethodPost, "/websets", nil, p)
}

// List returns one page of websets.
func (s *WebsetsService) List(ctx context.Context, p *ListWebsetsParams) (*Page[Webset], error) {
	var q url.Values
	if p != nil {
		q = p.values()
		if p.Search != "" {
			q.Set("search", p.Search)
		}
	}
	return websetsDo[Page[Webset]](ctx, s.c, http.MethodGet, "/websets", q, nil)
}

// Get returns a webset. p may be nil.
func (s *WebsetsService) Get(ctx context.Context, id string, p *GetWebsetParams) (*GetWebsetResponse, error) {
	var q url.Values
	if p != nil && len(p.Expand) > 0 {
		q = url.Values{"expand": p.Expand}
	}
	return websetsDo[GetWebsetResponse](ctx, s.c, http.MethodGet, "/websets/"+pathEscape(id), q, nil)
}

// Update changes a webset's title or metadata. The API uses POST for this.
func (s *WebsetsService) Update(ctx context.Context, id string, p *UpdateWebsetRequest) (*Webset, error) {
	return websetsDo[Webset](ctx, s.c, http.MethodPost, "/websets/"+pathEscape(id), nil, p)
}

// Delete deletes a webset and its items, returning the deleted webset.
func (s *WebsetsService) Delete(ctx context.Context, id string) (*Webset, error) {
	return websetsDo[Webset](ctx, s.c, http.MethodDelete, "/websets/"+pathEscape(id), nil, nil)
}

// Cancel stops every running search and enrichment on a webset.
func (s *WebsetsService) Cancel(ctx context.Context, id string) (*Webset, error) {
	return websetsDo[Webset](ctx, s.c, http.MethodPost, "/websets/"+pathEscape(id)+"/cancel", nil, nil)
}

// ---- preview ----

// PreviewWebsetParameters describe a search to preview.
type PreviewWebsetParameters struct {
	Search PreviewWebsetParametersSearch `json:"search"`
}

// PreviewWebsetParametersSearch is the search to preview.
type PreviewWebsetParametersSearch struct {
	Query  string  `json:"query"`
	Entity *Entity `json:"entity,omitempty"`
	// Count is how many example items to return (1 to 10, default 10).
	Count int `json:"count,omitempty"`
}

// PreviewWebsetResponse is how Exa would interpret a search.
type PreviewWebsetResponse struct {
	Search      PreviewWebsetResponseSearch       `json:"search"`
	Enrichments []PreviewWebsetResponseEnrichment `json:"enrichments"`
	// Items are example results; empty unless Preview was called with search.
	Items []WebsetItemPreview `json:"items"`
}

// PreviewWebsetResponseSearch is the detected entity and generated criteria.
type PreviewWebsetResponseSearch struct {
	Entity   Entity                      `json:"entity"`
	Criteria []CreateCriterionParameters `json:"criteria"`
}

// PreviewWebsetResponseEnrichment is a suggested enrichment.
type PreviewWebsetResponseEnrichment struct {
	Description string                   `json:"description"`
	Format      WebsetEnrichmentFormat   `json:"format"`
	Options     []WebsetEnrichmentOption `json:"options,omitempty"`
}

// Preview shows how a query would be interpreted before creating a webset.
// With search true it also returns example items.
func (s *WebsetsService) Preview(ctx context.Context, p *PreviewWebsetParameters, search bool) (*PreviewWebsetResponse, error) {
	var q url.Values
	if search {
		q = url.Values{"search": {"true"}}
	}
	return websetsDo[PreviewWebsetResponse](ctx, s.c, http.MethodPost, "/websets/preview", q, p)
}

// ---- team info ----

// TeamInfo is the calling team's Websets concurrency and limits.
type TeamInfo struct {
	Object      string              `json:"object"`
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Concurrency TeamInfoConcurrency `json:"concurrency"`
	Limits      TeamInfoLimits      `json:"limits"`
}

// TeamInfoConcurrency counts the team's active and queued searches.
type TeamInfoConcurrency struct {
	Active int `json:"active"`
	Queued int `json:"queued"`
}

// TeamInfoLimits are the team's caps; nil means unlimited.
type TeamInfoLimits struct {
	MaxConcurrent *int `json:"maxConcurrent"`
	MaxQueued     *int `json:"maxQueued"`
}

// TeamInfo returns the calling team's concurrency and limits.
func (s *WebsetsService) TeamInfo(ctx context.Context) (*TeamInfo, error) {
	return websetsDo[TeamInfo](ctx, s.c, http.MethodGet, "/teams/me", nil, nil)
}

// ---- searches ----

// WebsetSearchStatus is a search's lifecycle state.
type WebsetSearchStatus string

// Webset search statuses.
const (
	WebsetSearchStatusCreated   WebsetSearchStatus = "created"
	WebsetSearchStatusPending   WebsetSearchStatus = "pending"
	WebsetSearchStatusRunning   WebsetSearchStatus = "running"
	WebsetSearchStatusCompleted WebsetSearchStatus = "completed"
	WebsetSearchStatusCanceled  WebsetSearchStatus = "canceled"
)

// WebsetSearchBehavior is how a new search treats the webset's existing items.
type WebsetSearchBehavior string

// Webset search behaviors.
const (
	// WebsetSearchBehaviorOverride replaces items that don't match the new search.
	WebsetSearchBehaviorOverride WebsetSearchBehavior = "override"
	// WebsetSearchBehaviorAppend adds to the existing items.
	WebsetSearchBehaviorAppend WebsetSearchBehavior = "append"
)

// WebsetSearchCanceledReason is why a search was canceled.
type WebsetSearchCanceledReason string

// Webset search cancel reasons.
const (
	WebsetSearchCanceledReasonWebsetDeleted  WebsetSearchCanceledReason = "webset_deleted"
	WebsetSearchCanceledReasonWebsetCanceled WebsetSearchCanceledReason = "webset_canceled"
	WebsetSearchCanceledReasonOutOfCredits   WebsetSearchCanceledReason = "out_of_credits"
)

// WebsetSearch is a search that finds items for a webset.
type WebsetSearch struct {
	ID       string             `json:"id"`
	Object   string             `json:"object"`
	WebsetID string             `json:"websetId"`
	Status   WebsetSearchStatus `json:"status"`
	Query    string             `json:"query"`
	// Entity is nil until detected.
	Entity              *Entity              `json:"entity"`
	Criteria            []Criterion          `json:"criteria"`
	Count               int                  `json:"count"`
	MaxPeoplePerCompany *int                 `json:"maxPeoplePerCompany"`
	Behavior            WebsetSearchBehavior `json:"behavior,omitempty"`
	Exclude             []SourceRef          `json:"exclude"`
	Scope               []ScopeRef           `json:"scope"`
	Progress            WebsetSearchProgress `json:"progress"`
	// Recall is set when the search asked for it.
	Recall         *WebsetSearchRecall        `json:"recall"`
	Metadata       map[string]string          `json:"metadata,omitempty"`
	CanceledAt     *time.Time                 `json:"canceledAt"`
	CanceledReason WebsetSearchCanceledReason `json:"canceledReason"`
	CreatedAt      time.Time                  `json:"createdAt"`
	UpdatedAt      time.Time                  `json:"updatedAt"`
}

// WebsetSearchProgress is how far a search has got.
type WebsetSearchProgress struct {
	Found    int `json:"found"`
	Analyzed int `json:"analyzed"`
	// Completion is a percentage (0 to 100).
	Completion float64 `json:"completion"`
	// TimeLeft is the estimated seconds remaining, when known.
	TimeLeft *float64 `json:"timeLeft"`
}

// WebsetSearchRecall estimates how many matching items exist.
type WebsetSearchRecall struct {
	Expected  WebsetSearchRecallExpected `json:"expected"`
	Reasoning string                     `json:"reasoning"`
}

// WebsetSearchRecallExpected is the recall estimate.
type WebsetSearchRecallExpected struct {
	Total float64 `json:"total"`
	// Confidence is "high", "medium" or "low".
	Confidence string                   `json:"confidence"`
	Bounds     WebsetSearchRecallBounds `json:"bounds"`
}

// WebsetSearchRecallBounds is the range of the recall estimate.
type WebsetSearchRecallBounds struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// CreateWebsetSearchParameters start a new search on an existing webset.
type CreateWebsetSearchParameters struct {
	// Count is how many items to find (at least 1).
	Count  int     `json:"count"`
	Query  string  `json:"query"`
	Entity *Entity `json:"entity,omitempty"`
	// Criteria (1 to 5) are generated from Query when omitted.
	Criteria            []CreateCriterionParameters `json:"criteria,omitempty"`
	MaxPeoplePerCompany *int                        `json:"maxPeoplePerCompany,omitempty"`
	Exclude             []SourceRef                 `json:"exclude,omitempty"`
	Scope               []ScopeRef                  `json:"scope,omitempty"`
	Recall              *bool                       `json:"recall,omitempty"`
	// Behavior defaults to override.
	Behavior WebsetSearchBehavior `json:"behavior,omitempty"`
	Metadata map[string]string    `json:"metadata,omitempty"`
}

// Create starts a search on a webset.
func (s *WebsetSearchesService) Create(ctx context.Context, websetID string, p *CreateWebsetSearchParameters) (*WebsetSearch, error) {
	return websetsDo[WebsetSearch](ctx, s.c, http.MethodPost, "/websets/"+pathEscape(websetID)+"/searches", nil, p)
}

// Get returns a search.
func (s *WebsetSearchesService) Get(ctx context.Context, websetID, id string) (*WebsetSearch, error) {
	return websetsDo[WebsetSearch](ctx, s.c, http.MethodGet, "/websets/"+pathEscape(websetID)+"/searches/"+pathEscape(id), nil, nil)
}

// Cancel stops a running search; items found so far are kept.
func (s *WebsetSearchesService) Cancel(ctx context.Context, websetID, id string) (*WebsetSearch, error) {
	return websetsDo[WebsetSearch](ctx, s.c, http.MethodPost, "/websets/"+pathEscape(websetID)+"/searches/"+pathEscape(id)+"/cancel", nil, nil)
}

// ---- enrichments ----

// WebsetEnrichmentStatus is an enrichment's (or enrichment result's) state.
type WebsetEnrichmentStatus string

// Webset enrichment statuses.
const (
	WebsetEnrichmentStatusPending   WebsetEnrichmentStatus = "pending"
	WebsetEnrichmentStatusCanceled  WebsetEnrichmentStatus = "canceled"
	WebsetEnrichmentStatusCompleted WebsetEnrichmentStatus = "completed"
)

// WebsetEnrichmentFormat is the type of value an enrichment extracts.
type WebsetEnrichmentFormat string

// Webset enrichment formats.
const (
	WebsetEnrichmentFormatText    WebsetEnrichmentFormat = "text"
	WebsetEnrichmentFormatDate    WebsetEnrichmentFormat = "date"
	WebsetEnrichmentFormatNumber  WebsetEnrichmentFormat = "number"
	WebsetEnrichmentFormatOptions WebsetEnrichmentFormat = "options"
	WebsetEnrichmentFormatEmail   WebsetEnrichmentFormat = "email"
	WebsetEnrichmentFormatPhone   WebsetEnrichmentFormat = "phone"
	WebsetEnrichmentFormatURL     WebsetEnrichmentFormat = "url"
)

// WebsetEnrichmentOption is one allowed value of an options enrichment.
type WebsetEnrichmentOption struct {
	Label string `json:"label"`
}

// WebsetEnrichment extracts one value from every item of a webset.
type WebsetEnrichment struct {
	ID          string                   `json:"id"`
	Object      string                   `json:"object"`
	Status      WebsetEnrichmentStatus   `json:"status"`
	WebsetID    string                   `json:"websetId"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	Format      WebsetEnrichmentFormat   `json:"format"`
	Options     []WebsetEnrichmentOption `json:"options"`
	// Instructions are the prompt Exa generated from Description.
	Instructions string            `json:"instructions"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// CreateEnrichmentParameters create an enrichment.
type CreateEnrichmentParameters struct {
	// Description says what to extract (1 to 5000 characters).
	Description string `json:"description"`
	// Format is inferred from Description when empty.
	Format WebsetEnrichmentFormat `json:"format,omitempty"`
	// Options (1 to 150) are required when Format is options.
	Options  []WebsetEnrichmentOption `json:"options,omitempty"`
	Metadata map[string]string        `json:"metadata,omitempty"`
}

// UpdateEnrichmentParameters change an enrichment. Empty fields are left alone.
type UpdateEnrichmentParameters struct {
	Description string                   `json:"description,omitempty"`
	Format      WebsetEnrichmentFormat   `json:"format,omitempty"`
	Options     []WebsetEnrichmentOption `json:"options,omitempty"`
	// Metadata replaces the enrichment's metadata. Nil leaves it unchanged; a
	// pointer to a nil map sends null and clears it.
	Metadata *map[string]string `json:"metadata,omitempty"`
}

func enrichmentPath(websetID, id string) string {
	return "/websets/" + pathEscape(websetID) + "/enrichments/" + pathEscape(id)
}

// Create adds an enrichment to a webset and runs it on every item.
func (s *WebsetEnrichmentsService) Create(ctx context.Context, websetID string, p *CreateEnrichmentParameters) (*WebsetEnrichment, error) {
	return websetsDo[WebsetEnrichment](ctx, s.c, http.MethodPost, "/websets/"+pathEscape(websetID)+"/enrichments", nil, p)
}

// Get returns an enrichment.
func (s *WebsetEnrichmentsService) Get(ctx context.Context, websetID, id string) (*WebsetEnrichment, error) {
	return websetsDo[WebsetEnrichment](ctx, s.c, http.MethodGet, enrichmentPath(websetID, id), nil, nil)
}

// Update changes an enrichment. The API answers with an empty body, so
// there is nothing to return; Get the enrichment to see the result.
func (s *WebsetEnrichmentsService) Update(ctx context.Context, websetID, id string, p *UpdateEnrichmentParameters) error {
	return s.c.do(ctx, http.MethodPatch, s.c.baseURL, websetsPrefix+enrichmentPath(websetID, id), nil, p, nil)
}

// Delete removes an enrichment and its results.
func (s *WebsetEnrichmentsService) Delete(ctx context.Context, websetID, id string) (*WebsetEnrichment, error) {
	return websetsDo[WebsetEnrichment](ctx, s.c, http.MethodDelete, enrichmentPath(websetID, id), nil, nil)
}

// Cancel stops a running enrichment; results so far are kept.
func (s *WebsetEnrichmentsService) Cancel(ctx context.Context, websetID, id string) (*WebsetEnrichment, error) {
	return websetsDo[WebsetEnrichment](ctx, s.c, http.MethodPost, enrichmentPath(websetID, id)+"/cancel", nil, nil)
}

// ---- items ----

// ListItemsParams filter Items.List.
type ListItemsParams struct {
	ListParams
	// SourceID keeps items from one search or import.
	SourceID string
}

// List returns one page of a webset's items.
func (s *WebsetItemsService) List(ctx context.Context, websetID string, p *ListItemsParams) (*Page[WebsetItem], error) {
	var q url.Values
	if p != nil {
		q = p.values()
		if p.SourceID != "" {
			q.Set("sourceId", p.SourceID)
		}
	}
	return websetsDo[Page[WebsetItem]](ctx, s.c, http.MethodGet, "/websets/"+pathEscape(websetID)+"/items", q, nil)
}

// Get returns one item.
func (s *WebsetItemsService) Get(ctx context.Context, websetID, id string) (*WebsetItem, error) {
	return websetsDo[WebsetItem](ctx, s.c, http.MethodGet, "/websets/"+pathEscape(websetID)+"/items/"+pathEscape(id), nil, nil)
}

// Delete removes an item from a webset.
func (s *WebsetItemsService) Delete(ctx context.Context, websetID, id string) (*WebsetItem, error) {
	return websetsDo[WebsetItem](ctx, s.c, http.MethodDelete, "/websets/"+pathEscape(websetID)+"/items/"+pathEscape(id), nil, nil)
}
