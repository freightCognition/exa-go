package exa

import (
	"context"
	"net/http"
)

func init() {
	registerEndpoint(http.MethodPost, "/monitors", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Create(ctx, &CreateSearchMonitorParameters{})
		return err
	})
	registerEndpoint(http.MethodGet, "/monitors", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.List(ctx, nil)
		return err
	})
	registerEndpoint(http.MethodPost, "/monitors/batch", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Batch(ctx, &BatchMonitorsRequest{})
		return err
	})
	registerEndpoint(http.MethodGet, "/monitors/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Get(ctx, "m")
		return err
	})
	registerEndpoint(http.MethodPatch, "/monitors/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Update(ctx, "m", &UpdateSearchMonitorParameters{})
		return err
	})
	registerEndpoint(http.MethodDelete, "/monitors/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Delete(ctx, "m")
		return err
	})
	registerEndpoint(http.MethodPost, "/monitors/{id}/trigger", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.Trigger(ctx, "m")
		return err
	})
	registerEndpoint(http.MethodGet, "/monitors/{id}/runs", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.ListRuns(ctx, "m", nil)
		return err
	})
	registerEndpoint(http.MethodGet, "/monitors/{id}/runs/{runId}", func(ctx context.Context, c *Client) error {
		_, err := c.Monitors.GetRun(ctx, "m", "r")
		return err
	})

	registerSchema("CreateSearchMonitorParameters", CreateSearchMonitorParameters{})
	registerSchema("UpdateSearchMonitorParameters", UpdateSearchMonitorParameters{})
	registerSchema("CreateSearchMonitorResponse", CreateSearchMonitorResponse{})
	registerSchema("SearchMonitor", SearchMonitor{})
	registerSchema("SearchMonitorSearch", SearchMonitorSearch{})
	registerSchema("SearchMonitorSearchOutput", SearchMonitorSearch{})
	// SearchMonitorContents decodes itself, which hides its fields from the
	// nested comparison; the methodless twin is compared field by field here.
	registerSchema("SearchMonitorContents", searchMonitorContentsFields{})
	registerSchema("SearchMonitorContentsOutput", searchMonitorContentsFields{})
	registerSchema("SummaryWithMaxTokensOptions", SummaryWithMaxTokensOptions{})
	registerSchema("SummaryWithMaxTokensOptionsOutput", SummaryWithMaxTokensOptions{})
	registerSchema("SearchMonitorTrigger", SearchMonitorTrigger{})
	registerSchema("SearchMonitorTriggerOutput", SearchMonitorTrigger{})
	registerSchema("SearchMonitorWebhook", SearchMonitorWebhook{})
	registerSchema("SearchMonitorWebhookOutput", SearchMonitorWebhook{})
	// A text|object union of the shared OutputSchema; no single object shape.
	registerSchema("SearchMonitorOutputSchema", OutputSchema{})
	registerSchema("SearchMonitorOutputSchemaOutput", OutputSchema{})
	registerSchema("BatchMonitorsRequest", BatchMonitorsRequest{})
	registerSchema("BatchMonitorsResponse", BatchMonitorsResponse{})
	registerSchema("TriggerSearchMonitorResponse", TriggerSearchMonitorResponse{})
	// The spec's run citations list only "url" yet require "url" and "title";
	// runs reuse AgentGrounding, so its nested citations aren't compared.
	registerSchema("SearchMonitorRun", SearchMonitorRun{}, "output")
	registerSchema("SearchMonitorRunOutput", SearchMonitorRunOutput{}, "grounding")
	registerSchema("ListSearchMonitorsResponse", Page[SearchMonitor]{}, "object")
	registerSchema("ListSearchMonitorRunsResponse", Page[SearchMonitorRun]{}, "object")
}
