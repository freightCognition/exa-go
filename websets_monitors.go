package exa

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Websets monitors re-run a search on a webset on a schedule. They are not
// Search Monitors (Client.Monitors).

// MonitorStatus is whether a monitor runs.
type MonitorStatus string

// Monitor statuses.
const (
	MonitorStatusEnabled  MonitorStatus = "enabled"
	MonitorStatusDisabled MonitorStatus = "disabled"
)

// MonitorRunStatus is a monitor run's lifecycle state.
type MonitorRunStatus string

// Monitor run statuses.
const (
	MonitorRunStatusCreated   MonitorRunStatus = "created"
	MonitorRunStatusRunning   MonitorRunStatus = "running"
	MonitorRunStatusCompleted MonitorRunStatus = "completed"
	MonitorRunStatusCanceled  MonitorRunStatus = "canceled"
	MonitorRunStatusFailed    MonitorRunStatus = "failed"
)

// MonitorRunType is what a monitor run did.
type MonitorRunType string

// Monitor run types.
const (
	MonitorRunTypeSearch  MonitorRunType = "search"
	MonitorRunTypeRefresh MonitorRunType = "refresh"
)

// MonitorBehaviorSearch is the only MonitorBehavior type: run a search.
const MonitorBehaviorSearch = "search"

// Monitor runs a search on a webset on a cron schedule.
type Monitor struct {
	ID       string          `json:"id"`
	Object   string          `json:"object"`
	Status   MonitorStatus   `json:"status"`
	WebsetID string          `json:"websetId"`
	Cadence  MonitorCadence  `json:"cadence"`
	Behavior MonitorBehavior `json:"behavior"`
	// LastRun is nil before the first run.
	LastRun   *MonitorRun       `json:"lastRun"`
	NextRunAt *time.Time        `json:"nextRunAt"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// MonitorCadence is when a monitor runs.
type MonitorCadence struct {
	// Cron is a 5-field cron expression; runs at most once a day.
	Cron string `json:"cron"`
	// Timezone is an IANA zone; the API defaults to Etc/UTC.
	Timezone string `json:"timezone,omitempty"`
}

// MonitorBehavior is what a monitor does on each run.
type MonitorBehavior struct {
	// Type is required; set it to MonitorBehaviorSearch.
	Type   string                `json:"type"`
	Config MonitorBehaviorConfig `json:"config"`
}

// MonitorBehaviorConfig is the search a monitor runs. Query, Criteria and
// Entity default to the webset's last search.
type MonitorBehaviorConfig struct {
	Query    string                      `json:"query,omitempty"`
	Criteria []CreateCriterionParameters `json:"criteria,omitempty"`
	Entity   *Entity                     `json:"entity,omitempty"`
	// Count is how many new items each run looks for.
	Count int `json:"count"`
	// Behavior defaults to append.
	Behavior WebsetSearchBehavior `json:"behavior,omitempty"`
}

// MonitorRun is one execution of a monitor.
type MonitorRun struct {
	ID           string           `json:"id"`
	Object       string           `json:"object"`
	MonitorID    string           `json:"monitorId"`
	Status       MonitorRunStatus `json:"status"`
	Type         MonitorRunType   `json:"type"`
	CompletedAt  *time.Time       `json:"completedAt"`
	FailedAt     *time.Time       `json:"failedAt"`
	FailedReason string           `json:"failedReason"`
	CanceledAt   *time.Time       `json:"canceledAt"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// CreateMonitorParameters create a monitor.
type CreateMonitorParameters struct {
	WebsetID string            `json:"websetId"`
	Cadence  MonitorCadence    `json:"cadence"`
	Behavior MonitorBehavior   `json:"behavior"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpdateMonitor changes a monitor. Empty fields are left alone.
type UpdateMonitor struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Status   MonitorStatus     `json:"status,omitempty"`
	Cadence  *MonitorCadence   `json:"cadence,omitempty"`
	Behavior *MonitorBehavior  `json:"behavior,omitempty"`
}

// ListWebsetMonitorsParams filter Monitors.List.
type ListWebsetMonitorsParams struct {
	ListParams
	// WebsetID keeps one webset's monitors.
	WebsetID string
}

// Create creates a monitor.
func (s *WebsetMonitorsService) Create(ctx context.Context, p *CreateMonitorParameters) (*Monitor, error) {
	return websetsDo[Monitor](ctx, s.c, http.MethodPost, "/monitors", nil, p)
}

// List returns one page of monitors.
func (s *WebsetMonitorsService) List(ctx context.Context, p *ListWebsetMonitorsParams) (*Page[Monitor], error) {
	var q url.Values
	if p != nil {
		q = p.values()
		if p.WebsetID != "" {
			q.Set("websetId", p.WebsetID)
		}
	}
	return websetsDo[Page[Monitor]](ctx, s.c, http.MethodGet, "/monitors", q, nil)
}

// Get returns a monitor.
func (s *WebsetMonitorsService) Get(ctx context.Context, id string) (*Monitor, error) {
	return websetsDo[Monitor](ctx, s.c, http.MethodGet, "/monitors/"+pathEscape(id), nil, nil)
}

// Update changes a monitor.
func (s *WebsetMonitorsService) Update(ctx context.Context, id string, p *UpdateMonitor) (*Monitor, error) {
	return websetsDo[Monitor](ctx, s.c, http.MethodPatch, "/monitors/"+pathEscape(id), nil, p)
}

// Delete deletes a monitor.
func (s *WebsetMonitorsService) Delete(ctx context.Context, id string) (*Monitor, error) {
	return websetsDo[Monitor](ctx, s.c, http.MethodDelete, "/monitors/"+pathEscape(id), nil, nil)
}

// ListRuns returns one page of a monitor's runs. The spec declares no
// cursor or limit here, but the response pages, so p is passed through.
func (s *WebsetMonitorsService) ListRuns(ctx context.Context, monitorID string, p *ListParams) (*Page[MonitorRun], error) {
	return websetsDo[Page[MonitorRun]](ctx, s.c, http.MethodGet, "/monitors/"+pathEscape(monitorID)+"/runs", p.values(), nil)
}

// GetRun returns one monitor run.
func (s *WebsetMonitorsService) GetRun(ctx context.Context, monitorID, id string) (*MonitorRun, error) {
	return websetsDo[MonitorRun](ctx, s.c, http.MethodGet, "/monitors/"+pathEscape(monitorID)+"/runs/"+pathEscape(id), nil, nil)
}
