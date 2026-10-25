package exa

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"time"
)

// CreateAgentRunRequest starts an Agent run.
type CreateAgentRunRequest struct {
	// Query is the natural-language question or instruction. Required.
	Query        string         `json:"query"`
	SystemPrompt string         `json:"systemPrompt,omitempty"`
	Effort       AgentEffort    `json:"effort,omitempty"`
	Input        *AgentRunInput `json:"input,omitempty"`
	// OutputSchema is a JSON Schema the structured output must match.
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	// PreviousRunID continues from a completed run.
	PreviousRunID string            `json:"previousRunId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	// DataSources enables Exa Connect providers (at most 5).
	DataSources []AgentDataSource `json:"dataSources,omitempty"`
	Budget      *AgentBudget      `json:"budget,omitempty"`
	// UserIdentifier names the end user the run is for (1-64 of A-Z a-z 0-9 _ -).
	UserIdentifier string `json:"userIdentifier,omitempty"`

	// Beta is sent as the Exa-Beta header: comma-separated beta feature tokens.
	Beta string `json:"-"`
}

// AgentRunInput gives the agent rows to work from and rows to leave out.
type AgentRunInput struct {
	Data      []map[string]any `json:"data,omitempty"`
	Exclusion []map[string]any `json:"exclusion,omitempty"`
}

// AgentDataSource enables one Exa Connect data provider, e.g. "fiber".
type AgentDataSource struct {
	Provider string `json:"provider"`
}

// AgentBudget caps a run. The run stops with AgentStopReasonBudgetReached or
// AgentStopReasonTimeLimitReached when a cap is hit.
type AgentBudget struct {
	MaxCostDollars     *float64 `json:"maxCostDollars,omitempty"`
	MaxDurationSeconds *int     `json:"maxDurationSeconds,omitempty"`
}

// AgentEffort trades cost for reasoning depth. The API default is auto.
type AgentEffort string

// AgentEffort values.
const (
	AgentEffortMinimal AgentEffort = "minimal"
	AgentEffortLow     AgentEffort = "low"
	AgentEffortMedium  AgentEffort = "medium"
	AgentEffortHigh    AgentEffort = "high"
	AgentEffortXHigh   AgentEffort = "xhigh"
	AgentEffortAuto    AgentEffort = "auto"
	AgentEffortUltra   AgentEffort = "ultra"
)

// AgentRun is an Agent run and, once finished, its output.
type AgentRun struct {
	ID     string         `json:"id"`
	Object string         `json:"object"`
	Status AgentRunStatus `json:"status"`
	// StopReason is set once the run ends.
	StopReason  AgentStopReason `json:"stopReason"`
	CreatedAt   time.Time       `json:"createdAt"`
	CompletedAt *time.Time      `json:"completedAt"`
	// Request echoes the run's parameters; nil when not stored.
	Request     *AgentRunRequest `json:"request"`
	Output      AgentRunOutput   `json:"output"`
	Usage       AgentUsage       `json:"usage"`
	CostDollars AgentCostDollars `json:"costDollars"`
}

// AgentRunStatus is a run's lifecycle state.
type AgentRunStatus string

// AgentRunStatus values.
const (
	AgentRunStatusQueued    AgentRunStatus = "queued"
	AgentRunStatusRunning   AgentRunStatus = "running"
	AgentRunStatusCompleted AgentRunStatus = "completed"
	AgentRunStatusFailed    AgentRunStatus = "failed"
	AgentRunStatusCancelled AgentRunStatus = "cancelled"
)

// AgentStopReason is why a run ended. The set is open: expect other values.
type AgentStopReason string

// Known AgentStopReason values.
const (
	AgentStopReasonSchemaSatisfied       AgentStopReason = "schema_satisfied"
	AgentStopReasonBudgetReached         AgentStopReason = "budget_reached"
	AgentStopReasonTimeLimitReached      AgentStopReason = "time_limit_reached"
	AgentStopReasonIterationLimitReached AgentStopReason = "iteration_limit_reached"
	AgentStopReasonTokenLimitReached     AgentStopReason = "token_limit_reached"
	AgentStopReasonStopped               AgentStopReason = "stopped"
	AgentStopReasonError                 AgentStopReason = "error"
	AgentStopReasonCancelled             AgentStopReason = "cancelled"
)

// AgentRunRequest is the request a run was created with, as the API stores it.
type AgentRunRequest struct {
	Query         string            `json:"query"`
	SystemPrompt  string            `json:"systemPrompt,omitempty"`
	Effort        AgentEffort       `json:"effort,omitempty"`
	Input         *AgentRunInput    `json:"input,omitempty"`
	OutputSchema  map[string]any    `json:"outputSchema,omitempty"`
	PreviousRunID string            `json:"previousRunId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	DataSources   []AgentDataSource `json:"dataSources,omitempty"`
	Budget        *AgentBudget      `json:"budget,omitempty"`
}

// AgentRunOutput is a run's answer.
type AgentRunOutput struct {
	Text string `json:"text"`
	// Structured is the JSON matching the request's OutputSchema, or null.
	Structured json.RawMessage  `json:"structured"`
	Grounding  []AgentGrounding `json:"grounding"`
}

// AgentGrounding lists the sources behind one output field.
type AgentGrounding struct {
	Field     string          `json:"field"`
	Citations []AgentCitation `json:"citations"`
	// Confidence is low, medium or high; empty when not given.
	Confidence string `json:"confidence,omitempty"`
}

// AgentCitation is one source.
type AgentCitation struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// AgentUsage counts what a run consumed.
type AgentUsage struct {
	AgentComputeUnits float64 `json:"agentComputeUnits"`
	Searches          float64 `json:"searches"`
	Emails            float64 `json:"emails"`
	PhoneNumbers      float64 `json:"phoneNumbers"`
	// DataSources is calls per Exa Connect provider.
	DataSources map[string]int `json:"dataSources,omitempty"`
}

// AgentCostDollars is what a run cost, in USD.
type AgentCostDollars struct {
	Total        float64 `json:"total"`
	AgentCompute float64 `json:"agentCompute"`
	Search       float64 `json:"search"`
	Emails       float64 `json:"emails"`
	PhoneNumbers float64 `json:"phoneNumbers"`
	// DataSources is cost per Exa Connect provider.
	DataSources map[string]float64 `json:"dataSources,omitempty"`
}

// DeleteAgentRunResponse confirms a deleted run.
type DeleteAgentRunResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// AgentRunEvent is one event of a run. Streams also carry progress events
// (e.g. "agent_run.source.added") beyond the constants below; ignore the ones
// you don't handle.
type AgentRunEvent struct {
	// ID orders events within the run; pass the last one seen to StreamEvents
	// to resume.
	ID    string `json:"id"`
	Event string `json:"event"`
	// Data is the event payload: a partial run for created/started/failed/
	// cancelled, the full run for completed.
	Data json.RawMessage `json:"data"`
	// CreatedAt is set on events from Events; SSE frames don't carry it.
	CreatedAt time.Time `json:"createdAt"`
}

// Lifecycle event names of AgentRunEvent.Event.
const (
	AgentRunEventCreated   = "agent_run.created"
	AgentRunEventStarted   = "agent_run.started"
	AgentRunEventCompleted = "agent_run.completed"
	AgentRunEventFailed    = "agent_run.failed"
	AgentRunEventCancelled = "agent_run.cancelled"
)

// Run decodes Data as a run: the finished run for AgentRunEventCompleted, the
// fields sent so far for other lifecycle events.
func (e AgentRunEvent) Run() (*AgentRun, error) {
	var r AgentRun
	if err := json.Unmarshal(e.Data, &r); err != nil {
		return nil, fmt.Errorf("exa: decode %s event: %w", e.Event, err)
	}
	return &r, nil
}

func (r *CreateAgentRunRequest) edits() []func(*http.Request) {
	if r == nil || r.Beta == "" {
		return nil
	}
	return []func(*http.Request){withHeader("Exa-Beta", r.Beta)}
}

// Create starts a run and returns it queued. Poll Get, or use CreateStream.
func (s *AgentService) Create(ctx context.Context, req *CreateAgentRunRequest) (*AgentRun, error) {
	var out AgentRun
	if err := s.c.do(ctx, http.MethodPost, s.c.baseURL, "/agent/runs", nil, req, &out, req.edits()...); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateStream starts a run and yields its events until the run ends. The
// last event is agent_run.completed (see AgentRunEvent.Run), .failed or
// .cancelled.
func (s *AgentService) CreateStream(ctx context.Context, req *CreateAgentRunRequest) iter.Seq2[AgentRunEvent, error] {
	return stream(ctx, s.c, http.MethodPost, s.c.baseURL, "/agent/runs", nil, req, decodeAgentRunEvent, req.edits()...)
}

// decodeAgentRunEvent builds an event from an SSE frame, whose data is the
// payload only.
func decodeAgentRunEvent(ev sseEvent) (AgentRunEvent, bool, error) {
	return AgentRunEvent{ID: ev.ID, Event: ev.Event, Data: json.RawMessage(ev.Data)}, true, nil
}

// List returns runs, newest first.
func (s *AgentService) List(ctx context.Context, p *ListParams) (*Page[AgentRun], error) {
	var out Page[AgentRun]
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/agent/runs", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns a run.
func (s *AgentService) Get(ctx context.Context, id string) (*AgentRun, error) {
	return s.run(ctx, http.MethodGet, "/agent/runs/"+pathEscape(id))
}

// Delete deletes a run.
func (s *AgentService) Delete(ctx context.Context, id string) (*DeleteAgentRunResponse, error) {
	var out DeleteAgentRunResponse
	if err := s.c.do(ctx, http.MethodDelete, s.c.baseURL, "/agent/runs/"+pathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel cancels a run, discarding its output.
func (s *AgentService) Cancel(ctx context.Context, id string) (*AgentRun, error) {
	return s.run(ctx, http.MethodPost, "/agent/runs/"+pathEscape(id)+"/cancel")
}

// Stop ends a run early and keeps what it has found (stop reason "stopped").
func (s *AgentService) Stop(ctx context.Context, id string) (*AgentRun, error) {
	return s.run(ctx, http.MethodPost, "/agent/runs/"+pathEscape(id)+"/stop")
}

func (s *AgentService) run(ctx context.Context, method, path string) (*AgentRun, error) {
	var out AgentRun
	if err := s.c.do(ctx, method, s.c.baseURL, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Events returns a page of a run's stored events. Zero-data-retention runs
// keep none.
func (s *AgentService) Events(ctx context.Context, id string, p *ListParams) (*Page[AgentRunEvent], error) {
	var out Page[AgentRunEvent]
	if err := s.c.do(ctx, http.MethodGet, s.c.baseURL, "/agent/runs/"+pathEscape(id)+"/events", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StreamEvents replays a run's stored events as SSE, then ends. A non-empty
// lastEventID skips events up to and including it.
func (s *AgentService) StreamEvents(ctx context.Context, id, lastEventID string) iter.Seq2[AgentRunEvent, error] {
	var edits []func(*http.Request)
	if lastEventID != "" {
		edits = append(edits, withHeader("Last-Event-ID", lastEventID))
	}
	return stream(ctx, s.c, http.MethodGet, s.c.baseURL, "/agent/runs/"+pathEscape(id)+"/events", nil, nil, decodeAgentRunEvent, edits...)
}
