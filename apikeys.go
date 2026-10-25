package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// APIKeyRequest creates or updates an API key. Unset fields are left as they are.
type APIKeyRequest struct {
	Name string `json:"name,omitempty"`
	// RateLimit is in requests per second.
	RateLimit *int `json:"rateLimit,omitempty"`
	// BudgetCents caps spend per billing period, in cents; it resets each period.
	BudgetCents *int `json:"budgetCents,omitempty"`
	// RemoveBudget sends budgetCents: null, removing the key's budget. It
	// overrides BudgetCents.
	RemoveBudget bool `json:"-"`
}

// MarshalJSON sends budgetCents: null when RemoveBudget is set.
func (r APIKeyRequest) MarshalJSON() ([]byte, error) {
	type plain APIKeyRequest
	if !r.RemoveBudget {
		return json.Marshal(plain(r))
	}
	r.BudgetCents = nil
	return json.Marshal(struct {
		plain
		BudgetCents *int `json:"budgetCents"`
	}{plain: plain(r)})
}

// APIKey is a team API key. Which fields are set varies by endpoint: List
// omits TeamID, UserID and the timestamps.
type APIKey struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// RateLimit is in requests per second; nil when unlimited.
	RateLimit *int `json:"rateLimit"`
	// BudgetCents is the per-billing-period budget; nil when there is none.
	BudgetCents  *int       `json:"budgetCents"`
	IsOverBudget bool       `json:"isOverBudget"`
	TeamID       string     `json:"teamId,omitempty"`
	UserID       string     `json:"userId,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
}

// APIKeyUsageParams choose the usage period. Zero dates are left to the API
// (the last 30 days); the start must be within the last 180 days.
type APIKeyUsageParams struct {
	StartDate time.Time
	EndDate   time.Time
	// GroupBy is accepted by the API but, per its docs, has no effect.
	GroupBy APIKeyUsageGroupBy
}

// APIKeyUsageGroupBy is APIKeyUsageParams.GroupBy.
type APIKeyUsageGroupBy string

// APIKeyUsageGroupBy values.
const (
	APIKeyUsageGroupByHour  APIKeyUsageGroupBy = "hour"
	APIKeyUsageGroupByDay   APIKeyUsageGroupBy = "day"
	APIKeyUsageGroupByMonth APIKeyUsageGroupBy = "month"
)

// APIKeyUsage is an API key's spend over a period.
type APIKeyUsage struct {
	ID            string              `json:"id"`
	APIKeyID      string              `json:"api_key_id"`
	APIKeyName    string              `json:"api_key_name"`
	TeamID        string              `json:"team_id"`
	Period        APIKeyUsagePeriod   `json:"period"`
	TotalCostUSD  float64             `json:"total_cost_usd"`
	CostBreakdown []APIKeyUsageCost   `json:"cost_breakdown"`
	Metadata      APIKeyUsageMetadata `json:"metadata"`
}

// APIKeyUsagePeriod is the period an APIKeyUsage covers.
type APIKeyUsagePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// APIKeyUsageCost is the spend on one price, e.g. "Neural Search".
type APIKeyUsageCost struct {
	PriceID   string  `json:"price_id"`
	PriceName string  `json:"price_name"`
	Quantity  float64 `json:"quantity"`
	AmountUSD float64 `json:"amount_usd"`
}

// APIKeyUsageMetadata describes the usage report.
type APIKeyUsageMetadata struct {
	GeneratedAt time.Time `json:"generated_at"`
}

// apiKeyEnvelope is the {"apiKey": ...} wrapper of single-key responses.
type apiKeyEnvelope struct {
	APIKey APIKey `json:"apiKey"`
}

// Create creates an API key.
func (s *APIKeysService) Create(ctx context.Context, req *APIKeyRequest) (*APIKey, error) {
	return s.key(ctx, http.MethodPost, "/api-keys", req)
}

// List returns the team's API keys.
func (s *APIKeysService) List(ctx context.Context) ([]APIKey, error) {
	var out struct {
		APIKeys []APIKey `json:"apiKeys"`
	}
	if err := s.c.do(ctx, http.MethodGet, s.c.adminBaseURL, "/api-keys", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.APIKeys, nil
}

// Get returns an API key.
func (s *APIKeysService) Get(ctx context.Context, id string) (*APIKey, error) {
	return s.key(ctx, http.MethodGet, "/api-keys/"+pathEscape(id), nil)
}

// Update changes an API key (a PUT, but unset fields are kept).
func (s *APIKeysService) Update(ctx context.Context, id string, req *APIKeyRequest) (*APIKey, error) {
	return s.key(ctx, http.MethodPut, "/api-keys/"+pathEscape(id), req)
}

func (s *APIKeysService) key(ctx context.Context, method, path string, body any) (*APIKey, error) {
	var out apiKeyEnvelope
	if err := s.c.do(ctx, method, s.c.adminBaseURL, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out.APIKey, nil
}

// Delete deletes an API key.
func (s *APIKeysService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, s.c.adminBaseURL, "/api-keys/"+pathEscape(id), nil, nil, nil)
}

// Usage returns an API key's spend over a period.
func (s *APIKeysService) Usage(ctx context.Context, id string, p *APIKeyUsageParams) (*APIKeyUsage, error) {
	q := url.Values{}
	if p != nil {
		if !p.StartDate.IsZero() {
			q.Set("start_date", p.StartDate.Format(time.RFC3339))
		}
		if !p.EndDate.IsZero() {
			q.Set("end_date", p.EndDate.Format(time.RFC3339))
		}
		if p.GroupBy != "" {
			q.Set("group_by", string(p.GroupBy))
		}
	}
	var out APIKeyUsage
	if err := s.c.do(ctx, http.MethodGet, s.c.adminBaseURL, "/api-keys/"+pathEscape(id)+"/usage", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
