package exa

import (
	"context"
	"net/http"
)

func init() {
	registerEndpoint(http.MethodPost, "/agent/runs", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Create(ctx, &CreateAgentRunRequest{Query: "q"})
		return err
	})
	registerEndpoint(http.MethodGet, "/agent/runs", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.List(ctx, nil)
		return err
	})
	registerEndpoint(http.MethodGet, "/agent/runs/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Get(ctx, "r")
		return err
	})
	registerEndpoint(http.MethodDelete, "/agent/runs/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Delete(ctx, "r")
		return err
	})
	registerEndpoint(http.MethodPost, "/agent/runs/{id}/cancel", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Cancel(ctx, "r")
		return err
	})
	registerEndpoint(http.MethodPost, "/agent/runs/{id}/stop", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Stop(ctx, "r")
		return err
	})
	registerEndpoint(http.MethodGet, "/agent/runs/{id}/events", func(ctx context.Context, c *Client) error {
		_, err := c.Agent.Events(ctx, "r", nil)
		return err
	})

	registerSchema("CreateAgentRunRequest", CreateAgentRunRequest{})
	registerSchema("AgentRunRequest", AgentRunRequest{})
	registerSchema("AgentDataSource", AgentDataSource{})
	registerSchema("AgentDataSourceOutput", AgentDataSource{})
	registerSchema("AgentBudget", AgentBudget{})
	registerSchema("AgentBudgetOutput", AgentBudget{})
	registerSchema("AgentRun", AgentRun{})
	registerSchema("AgentRunOutput", AgentRunOutput{})
	registerSchema("AgentGrounding", AgentGrounding{})
	registerSchema("AgentCitation", AgentCitation{})
	registerSchema("AgentUsage", AgentUsage{})
	registerSchema("AgentCostDollars", AgentCostDollars{})
	registerSchema("AgentDataSourceUsage", map[string]int{})
	registerSchema("AgentDataSourceCost", map[string]float64{})
	registerSchema("DeleteAgentRunResponse", DeleteAgentRunResponse{})
	registerSchema("AgentRunEvent", AgentRunEvent{})
	registerSchema("AgentRunList", Page[AgentRun]{})
	registerSchema("AgentRunEventList", Page[AgentRunEvent]{})
}
