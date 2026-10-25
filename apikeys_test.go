package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const apiKeyFields = `"id":"550e8400-e29b-41d4-a716-446655440000","name":"Production API Key","rateLimit":1000,"budgetCents":5000,"isOverBudget":false,"teamId":"660e8400-e29b-41d4-a716-446655440000","userId":"770e8400-e29b-41d4-a716-446655440000","createdAt":"2025-01-01T00:00:00Z","updatedAt":"2025-01-02T00:00:00Z"`

func checkAPIKey(t *testing.T, k *APIKey) {
	t.Helper()
	if k.ID != "550e8400-e29b-41d4-a716-446655440000" || k.Name != "Production API Key" || *k.RateLimit != 1000 || *k.BudgetCents != 5000 || k.IsOverBudget {
		t.Errorf("key = %+v", k)
	}
	if k.TeamID == "" || k.UserID == "" || k.CreatedAt == nil || k.UpdatedAt == nil || k.UpdatedAt.Day() != 2 {
		t.Errorf("key = %+v", k)
	}
}

func TestAPIKeysCreate(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"apiKey":{`+apiKeyFields+`}}`)
	k, err := c.APIKeys.Create(context.Background(), &APIKeyRequest{Name: "Production API Key", RateLimit: Ptr(1000), BudgetCents: Ptr(5000)})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/team-management/api-keys", `{"name":"Production API Key","rateLimit":1000,"budgetCents":5000}`)
	if got.Header.Get("x-api-key") != "test-key" {
		t.Error("missing x-api-key")
	}
	checkAPIKey(t, k)
}

func TestAPIKeysList(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"apiKeys":[{`+apiKeyFields+`},{"id":"k2","name":"","rateLimit":null,"budgetCents":null,"isOverBudget":true}]}`)
	keys, err := c.APIKeys.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/team-management/api-keys", "")
	if len(keys) != 2 {
		t.Fatalf("keys = %+v", keys)
	}
	checkAPIKey(t, &keys[0])
	if keys[1].RateLimit != nil || keys[1].BudgetCents != nil || !keys[1].IsOverBudget {
		t.Errorf("key 2 = %+v", keys[1])
	}
}

func TestAPIKeysGet(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"apiKey":{`+apiKeyFields+`}}`)
	k, err := c.APIKeys.Get(context.Background(), "550e8400")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/team-management/api-keys/550e8400", "")
	checkAPIKey(t, k)
}

func TestAPIKeysUpdate(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"apiKey":{`+apiKeyFields+`}}`)
	k, err := c.APIKeys.Update(context.Background(), "550e8400", &APIKeyRequest{Name: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPut, "/team-management/api-keys/550e8400", `{"name":"Updated"}`)
	checkAPIKey(t, k)
}

func TestAPIKeysUpdateRemovesBudget(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"apiKey":{`+apiKeyFields+`}}`)
	if _, err := c.APIKeys.Update(context.Background(), "k", &APIKeyRequest{RemoveBudget: true, BudgetCents: Ptr(1)}); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPut, "/team-management/api-keys/k", `{"budgetCents":null}`)
}

func TestAPIKeyRequestZeroBudgetIsSent(t *testing.T) {
	b, err := json.Marshal(APIKeyRequest{BudgetCents: Ptr(0)})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"budgetCents":0}` {
		t.Errorf("got %s", b)
	}
}

func TestAPIKeysDelete(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{"success":true}`)
	if err := c.APIKeys.Delete(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/team-management/api-keys/k", "")
}

func TestAPIKeysUsage(t *testing.T) {
	c, got := newAgentCaptureClient(t, `{
	  "id": "key_abc123def456",
	  "api_key_id": "550e8400-e29b-41d4-a716-446655440000",
	  "api_key_name": "Production API Key",
	  "team_id": "660e8400-e29b-41d4-a716-446655440000",
	  "period": {"start": "2025-01-01T00:00:00Z", "end": "2025-01-31T23:59:59Z"},
	  "total_cost_usd": 45.67,
	  "cost_breakdown": [
	    {"price_id": "price_neural_search", "price_name": "Neural Search", "quantity": 1000, "amount_usd": 30},
	    {"price_id": "price_content_retrieval", "price_name": "Content Retrieval", "quantity": 500, "amount_usd": 15.67}
	  ],
	  "metadata": {"generated_at": "2025-02-01T10:30:00Z"}
	}`)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 31, 23, 59, 59, 0, time.FixedZone("x", -5*3600))
	u, err := c.APIKeys.Usage(context.Background(), "550e8400", &APIKeyUsageParams{StartDate: start, EndDate: end, GroupBy: APIKeyUsageGroupByDay})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/team-management/api-keys/550e8400/usage", "")
	q := got.Query
	if q.Get("start_date") != "2025-01-01T00:00:00Z" || q.Get("end_date") != "2025-01-31T23:59:59-05:00" || q.Get("group_by") != "day" {
		t.Errorf("query = %v", q)
	}
	if u.ID != "key_abc123def456" || u.APIKeyID == "" || u.APIKeyName != "Production API Key" || u.TeamID == "" || u.TotalCostUSD != 45.67 {
		t.Errorf("usage = %+v", u)
	}
	if !u.Period.Start.Equal(start) || u.Period.End.Day() != 31 || u.Metadata.GeneratedAt.Hour() != 10 {
		t.Errorf("period/metadata = %+v %+v", u.Period, u.Metadata)
	}
	if len(u.CostBreakdown) != 2 || u.CostBreakdown[1] != (APIKeyUsageCost{PriceID: "price_content_retrieval", PriceName: "Content Retrieval", Quantity: 500, AmountUSD: 15.67}) {
		t.Errorf("breakdown = %+v", u.CostBreakdown)
	}

	if _, err := c.APIKeys.Usage(context.Background(), "k", nil); err != nil {
		t.Fatal(err)
	}
	if len(got.Query) != 0 {
		t.Errorf("nil params sent query %v", got.Query)
	}
}
