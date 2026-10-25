package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CreateSearchMonitorParameters create a Search Monitor: a search that runs
// on a schedule and posts each run to a webhook.
type CreateSearchMonitorParameters struct {
	Name   string              `json:"name,omitempty"`
	Search SearchMonitorSearch `json:"search"`
	// Trigger schedules runs; without one the monitor runs only on Trigger.
	Trigger *SearchMonitorTrigger `json:"trigger,omitempty"`
	// OutputSchema asks for synthesized output on each run.
	OutputSchema *OutputSchema        `json:"outputSchema,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
	Webhook      SearchMonitorWebhook `json:"webhook"`
}

// UpdateSearchMonitorParameters change a monitor. Nil and empty fields are
// left as they are; Search and Webhook are merged field by field.
type UpdateSearchMonitorParameters struct {
	Name string `json:"name,omitempty"`
	// Status is SearchMonitorStatusActive or SearchMonitorStatusPaused.
	Status       SearchMonitorStatus   `json:"status,omitempty"`
	Search       *SearchMonitorSearch  `json:"search,omitempty"`
	Trigger      *SearchMonitorTrigger `json:"trigger,omitempty"`
	OutputSchema *OutputSchema         `json:"outputSchema,omitempty"`
	Metadata     map[string]string     `json:"metadata,omitempty"`
	Webhook      *SearchMonitorWebhook `json:"webhook,omitempty"`

	// The Clear fields send null, removing the value; each overrides its
	// field above. ClearTrigger leaves the monitor running only on Trigger.
	ClearName         bool `json:"-"`
	ClearTrigger      bool `json:"-"`
	ClearOutputSchema bool `json:"-"`
	ClearMetadata     bool `json:"-"`
}

// MarshalJSON sends null for each set Clear field.
func (p UpdateSearchMonitorParameters) MarshalJSON() ([]byte, error) {
	type plain UpdateSearchMonitorParameters
	b, err := json.Marshal(plain(p))
	if err != nil || !(p.ClearName || p.ClearTrigger || p.ClearOutputSchema || p.ClearMetadata) {
		return b, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for key, clear := range map[string]bool{"name": p.ClearName, "trigger": p.ClearTrigger, "outputSchema": p.ClearOutputSchema, "metadata": p.ClearMetadata} {
		if clear {
			m[key] = json.RawMessage("null")
		}
	}
	return json.Marshal(m)
}

// SearchMonitorSearch is the search a monitor runs.
type SearchMonitorSearch struct {
	// Query is required when creating a monitor.
	Query string `json:"query,omitempty"`
	// NumResults is 1-100; the API default is 10.
	NumResults     *int                   `json:"numResults,omitempty"`
	IncludeDomains []string               `json:"includeDomains,omitempty"`
	ExcludeDomains []string               `json:"excludeDomains,omitempty"`
	Contents       *SearchMonitorContents `json:"contents,omitempty"`
}

// SearchMonitorContents chooses the page content each run returns. It is
// ContentsOptions without SnapshotAsOf/CrawledBeforeDate, with a different
// Summary and with FilterEmptyResults.
type SearchMonitorContents struct {
	Text       *TextOptions                 `json:"text,omitempty"`
	Highlights *HighlightsOptions           `json:"highlights,omitempty"`
	Summary    *SummaryWithMaxTokensOptions `json:"summary,omitempty"`
	Extras     *ExtrasOptions               `json:"extras,omitempty"`
	// LivecrawlTimeout is in milliseconds.
	LivecrawlTimeout *int `json:"livecrawlTimeout,omitempty"`
	// MaxAgeHours caps cached content age: -1 cache only, 0 always fetch fresh.
	MaxAgeHours *int `json:"maxAgeHours,omitempty"`
	// FilterEmptyResults drops results whose requested content came back empty.
	FilterEmptyResults *bool `json:"filterEmptyResults,omitempty"`

	// Deprecated: use Highlights or Text.
	Context *ContextOptions `json:"context,omitempty"`
	// Deprecated: use MaxAgeHours.
	Livecrawl Livecrawl `json:"livecrawl,omitempty"`
	// Deprecated: number of subpages to crawl per result.
	Subpages *int `json:"subpages,omitempty"`
	// Deprecated: terms to find specific subpages.
	SubpageTarget Strings `json:"subpageTarget,omitempty"`
}

// SummaryWithMaxTokensOptions requests an LLM summary of each result. An
// empty struct sends true.
type SummaryWithMaxTokensOptions struct {
	toggle
	Query string `json:"query,omitempty"`
	// Deprecated: no longer used.
	MaxTokens *int `json:"maxTokens,omitempty"`
}

// MarshalJSON sends true for an empty struct.
func (o SummaryWithMaxTokensOptions) MarshalJSON() ([]byte, error) {
	type plain SummaryWithMaxTokensOptions
	return marshalToggle(o.off, plain(o))
}

// UnmarshalJSON accepts true, false or an object.
func (o *SummaryWithMaxTokensOptions) UnmarshalJSON(b []byte) error {
	type plain SummaryWithMaxTokensOptions
	return unmarshalToggle(b, (*plain)(o), &o.toggle)
}

// SearchMonitorTrigger schedules a monitor.
type SearchMonitorTrigger struct {
	// Type is SearchMonitorTriggerInterval, the only kind.
	Type string `json:"type"`
	// Period is a single-unit duration such as "1h", "6h", "1d" or "7d"
	// (at least 1h), anchored to the monitor's creation time.
	Period string `json:"period"`
}

// SearchMonitorTriggerInterval is SearchMonitorTrigger.Type.
const SearchMonitorTriggerInterval = "interval"

// SearchMonitorWebhook is where a monitor posts its events.
type SearchMonitorWebhook struct {
	// URL is required when creating a monitor.
	URL string `json:"url,omitempty"`
	// Events limits which events are sent; empty means all.
	Events []SearchMonitorEvent `json:"events,omitempty"`
}

// SearchMonitorEvent is a webhook event type.
type SearchMonitorEvent string

// SearchMonitorEvent values.
const (
	SearchMonitorEventCreated      SearchMonitorEvent = "monitor.created"
	SearchMonitorEventUpdated      SearchMonitorEvent = "monitor.updated"
	SearchMonitorEventDeleted      SearchMonitorEvent = "monitor.deleted"
	SearchMonitorEventRunCreated   SearchMonitorEvent = "monitor.run.created"
	SearchMonitorEventRunCompleted SearchMonitorEvent = "monitor.run.completed"
)

// SearchMonitor is a Search Monitor.
type SearchMonitor struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Status       SearchMonitorStatus   `json:"status"`
	Search       SearchMonitorSearch   `json:"search"`
	Trigger      *SearchMonitorTrigger `json:"trigger"`
	OutputSchema *OutputSchema         `json:"outputSchema"`
	Metadata     map[string]string     `json:"metadata"`
	Webhook      SearchMonitorWebhook  `json:"webhook"`
	// NextRunAt is nil when the monitor has no trigger.
	NextRunAt *time.Time `json:"nextRunAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// SearchMonitorStatus is a monitor's state.
type SearchMonitorStatus string

// SearchMonitorStatus values.
const (
	SearchMonitorStatusActive   SearchMonitorStatus = "active"
	SearchMonitorStatusPaused   SearchMonitorStatus = "paused"
	SearchMonitorStatusDisabled SearchMonitorStatus = "disabled"
)

// CreateSearchMonitorResponse is the new monitor and its webhook secret.
type CreateSearchMonitorResponse struct {
	SearchMonitor
	// WebhookSecret signs webhook deliveries. It is returned only here; store it.
	WebhookSecret string `json:"webhookSecret"`
}

// ListSearchMonitorsParams filter Monitors.List.
type ListSearchMonitorsParams struct {
	ListParams
	Status SearchMonitorStatus
	// Name filters by monitor name.
	Name string
	// Metadata matches monitors having every key/value given.
	Metadata map[string]string
}

// BatchMonitorsRequest applies one action to every monitor matching Filter.
type BatchMonitorsRequest struct {
	Action BatchMonitorsAction `json:"action"`
	Filter BatchMonitorsFilter `json:"filter"`
	// DryRun lists the monitors that would be affected without acting.
	// The API defaults it to true: set Ptr(false) to apply the action.
	DryRun *bool `json:"dry_run,omitempty"`
	// Limit caps monitors processed per call (API default 50, max 500).
	Limit int `json:"limit,omitempty"`
}

// BatchMonitorsFilter selects monitors for Monitors.Batch.
type BatchMonitorsFilter struct {
	Name     string              `json:"name,omitempty"`
	Status   SearchMonitorStatus `json:"status,omitempty"`
	Metadata map[string]string   `json:"metadata,omitempty"`
}

// BatchMonitorsAction is what Monitors.Batch does.
type BatchMonitorsAction string

// BatchMonitorsAction values.
const (
	BatchMonitorsActionDelete  BatchMonitorsAction = "delete"
	BatchMonitorsActionPause   BatchMonitorsAction = "pause"
	BatchMonitorsActionUnpause BatchMonitorsAction = "unpause"
)

// BatchMonitorsResponse reports what Monitors.Batch did (or would do).
type BatchMonitorsResponse struct {
	Action   BatchMonitorsAction `json:"action"`
	Affected int                 `json:"affected"`
	IDs      []string            `json:"ids"`
	DryRun   bool                `json:"dry_run"`
	// HasMore is true when more monitors match than Limit; call again.
	HasMore bool `json:"has_more"`
}

// TriggerSearchMonitorResponse reports whether a run was started.
type TriggerSearchMonitorResponse struct {
	Triggered bool `json:"triggered"`
}

// SearchMonitorRun is one run of a monitor.
type SearchMonitorRun struct {
	ID        string                  `json:"id"`
	MonitorID string                  `json:"monitorId"`
	Status    SearchMonitorRunStatus  `json:"status"`
	Output    *SearchMonitorRunOutput `json:"output"`
	// FailReason is set for failed runs, e.g. "insufficient_credits".
	FailReason  string     `json:"failReason"`
	StartedAt   *time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	FailedAt    *time.Time `json:"failedAt"`
	CancelledAt *time.Time `json:"cancelledAt"`
	DurationMs  *int       `json:"durationMs"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// SearchMonitorRunStatus is a run's state.
type SearchMonitorRunStatus string

// SearchMonitorRunStatus values.
const (
	SearchMonitorRunStatusPending   SearchMonitorRunStatus = "pending"
	SearchMonitorRunStatusRunning   SearchMonitorRunStatus = "running"
	SearchMonitorRunStatusCompleted SearchMonitorRunStatus = "completed"
	SearchMonitorRunStatusFailed    SearchMonitorRunStatus = "failed"
	SearchMonitorRunStatusCancelled SearchMonitorRunStatus = "cancelled"
)

// SearchMonitorRunOutput is what a run found.
type SearchMonitorRunOutput struct {
	// Results are the search results, as returned by Search.
	Results []map[string]any `json:"results"`
	// Content is the output synthesized per OutputSchema: a JSON string for
	// type text, an object for type object; null without an OutputSchema.
	Content   json.RawMessage  `json:"content"`
	Grounding []AgentGrounding `json:"grounding"`
}

// Create creates a monitor. Keep the returned WebhookSecret: it isn't shown again.
func (s *MonitorsService) Create(ctx context.Context, req *CreateSearchMonitorParameters) (*CreateSearchMonitorResponse, error) {
	var out CreateSearchMonitorResponse
	if err := s.c.do(ctx, http.MethodPost, s.c.baseURL, "/monitors", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns monitors.
func (s *MonitorsService) List(ctx context.Context, p *ListSearchMonitorsParams) (*Page[SearchMonitor], error) {
	var lp *ListParams
	if p != nil {
		lp = &p.ListParams
	}
	q := lp.values()
	if p != nil {
		if p.Status != "" {
			q.Set("status", string(p.Status))
		}
		if p.Name != "" {
			q.Set("name", p.Name)
		}
		for k, v := range p.Metadata {
			q.Set("metadata["+k+"]", v)
		}
	}
	var out Page[SearchMonitor]
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/monitors", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns a monitor.
func (s *MonitorsService) Get(ctx context.Context, id string) (*SearchMonitor, error) {
	return s.monitor(ctx, http.MethodGet, id, nil)
}

// Update changes a monitor.
func (s *MonitorsService) Update(ctx context.Context, id string, req *UpdateSearchMonitorParameters) (*SearchMonitor, error) {
	return s.monitor(ctx, http.MethodPatch, id, req)
}

// Delete deletes a monitor and returns it.
func (s *MonitorsService) Delete(ctx context.Context, id string) (*SearchMonitor, error) {
	return s.monitor(ctx, http.MethodDelete, id, nil)
}

func (s *MonitorsService) monitor(ctx context.Context, method, id string, body any) (*SearchMonitor, error) {
	var out SearchMonitor
	if err := s.c.do(ctx, method, s.c.baseURL, "/monitors/"+pathEscape(id), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Batch deletes, pauses or unpauses every monitor matching a filter. It is a
// dry run unless req.DryRun is Ptr(false).
func (s *MonitorsService) Batch(ctx context.Context, req *BatchMonitorsRequest) (*BatchMonitorsResponse, error) {
	var out BatchMonitorsResponse
	if err := s.c.do(ctx, http.MethodPost, s.c.baseURL, "/monitors/batch", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Trigger starts a run now.
func (s *MonitorsService) Trigger(ctx context.Context, id string) (*TriggerSearchMonitorResponse, error) {
	var out TriggerSearchMonitorResponse
	if err := s.c.do(ctx, http.MethodPost, s.c.baseURL, "/monitors/"+pathEscape(id)+"/trigger", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRuns returns a monitor's runs.
func (s *MonitorsService) ListRuns(ctx context.Context, id string, p *ListParams) (*Page[SearchMonitorRun], error) {
	var out Page[SearchMonitorRun]
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/monitors/"+pathEscape(id)+"/runs", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRun returns one run of a monitor.
func (s *MonitorsService) GetRun(ctx context.Context, id, runID string) (*SearchMonitorRun, error) {
	var out SearchMonitorRun
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/monitors/"+pathEscape(id)+"/runs/"+pathEscape(runID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
