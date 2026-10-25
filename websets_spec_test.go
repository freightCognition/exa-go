package exa

import "context"

// Spec coverage for the Websets API (/v0 paths, served under /websets).

func init() {
	w := func(c *Client) *WebsetsService { return c.Websets }
	ep := func(method, path string, call func(context.Context, *Client) error) {
		registerEndpoint(method, path, call)
	}
	ignore := func(_ any, err error) error { return err }

	ep("GET", "/v0/teams/me", func(ctx context.Context, c *Client) error { return ignore(w(c).TeamInfo(ctx)) })
	ep("POST", "/v0/websets", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Create(ctx, &CreateWebsetParameters{}))
	})
	ep("GET", "/v0/websets", func(ctx context.Context, c *Client) error { return ignore(w(c).List(ctx, nil)) })
	ep("GET", "/v0/websets/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Get(ctx, "ws", nil)) })
	ep("POST", "/v0/websets/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Update(ctx, "ws", &UpdateWebsetRequest{}))
	})
	ep("DELETE", "/v0/websets/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Delete(ctx, "ws")) })
	ep("POST", "/v0/websets/{id}/cancel", func(ctx context.Context, c *Client) error { return ignore(w(c).Cancel(ctx, "ws")) })
	ep("POST", "/v0/websets/preview", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Preview(ctx, &PreviewWebsetParameters{}, false))
	})

	ep("POST", "/v0/websets/{webset}/searches", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Searches.Create(ctx, "ws", &CreateWebsetSearchParameters{}))
	})
	ep("GET", "/v0/websets/{webset}/searches/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Searches.Get(ctx, "ws", "s"))
	})
	ep("POST", "/v0/websets/{webset}/searches/{id}/cancel", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Searches.Cancel(ctx, "ws", "s"))
	})

	ep("POST", "/v0/websets/{webset}/enrichments", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Enrichments.Create(ctx, "ws", &CreateEnrichmentParameters{}))
	})
	ep("GET", "/v0/websets/{webset}/enrichments/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Enrichments.Get(ctx, "ws", "e"))
	})
	ep("PATCH", "/v0/websets/{webset}/enrichments/{id}", func(ctx context.Context, c *Client) error {
		return w(c).Enrichments.Update(ctx, "ws", "e", &UpdateEnrichmentParameters{})
	})
	ep("DELETE", "/v0/websets/{webset}/enrichments/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Enrichments.Delete(ctx, "ws", "e"))
	})
	ep("POST", "/v0/websets/{webset}/enrichments/{id}/cancel", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Enrichments.Cancel(ctx, "ws", "e"))
	})

	ep("GET", "/v0/websets/{webset}/items", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Items.List(ctx, "ws", nil))
	})
	ep("GET", "/v0/websets/{webset}/items/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Items.Get(ctx, "ws", "i"))
	})
	ep("DELETE", "/v0/websets/{webset}/items/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Items.Delete(ctx, "ws", "i"))
	})

	ep("POST", "/v0/monitors", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Monitors.Create(ctx, &CreateMonitorParameters{}))
	})
	ep("GET", "/v0/monitors", func(ctx context.Context, c *Client) error { return ignore(w(c).Monitors.List(ctx, nil)) })
	ep("GET", "/v0/monitors/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Monitors.Get(ctx, "m")) })
	ep("PATCH", "/v0/monitors/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Monitors.Update(ctx, "m", &UpdateMonitor{}))
	})
	ep("DELETE", "/v0/monitors/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Monitors.Delete(ctx, "m")) })
	ep("GET", "/v0/monitors/{monitor}/runs", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Monitors.ListRuns(ctx, "m", nil))
	})
	ep("GET", "/v0/monitors/{monitor}/runs/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Monitors.GetRun(ctx, "m", "r"))
	})

	ep("POST", "/v0/imports", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Imports.Create(ctx, &CreateImportParameters{}))
	})
	ep("GET", "/v0/imports", func(ctx context.Context, c *Client) error { return ignore(w(c).Imports.List(ctx, nil)) })
	ep("GET", "/v0/imports/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Imports.Get(ctx, "i")) })
	ep("PATCH", "/v0/imports/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Imports.Update(ctx, "i", &UpdateImport{}))
	})
	ep("DELETE", "/v0/imports/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Imports.Delete(ctx, "i")) })

	ep("POST", "/v0/webhooks", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Webhooks.Create(ctx, &CreateWebhookParameters{}))
	})
	ep("GET", "/v0/webhooks", func(ctx context.Context, c *Client) error { return ignore(w(c).Webhooks.List(ctx, nil)) })
	ep("GET", "/v0/webhooks/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Webhooks.Get(ctx, "h")) })
	ep("PATCH", "/v0/webhooks/{id}", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Webhooks.Update(ctx, "h", &UpdateWebhookParameters{}))
	})
	ep("DELETE", "/v0/webhooks/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Webhooks.Delete(ctx, "h")) })
	ep("GET", "/v0/webhooks/{id}/attempts", func(ctx context.Context, c *Client) error {
		return ignore(w(c).Webhooks.ListAttempts(ctx, "h", nil))
	})

	ep("GET", "/v0/events", func(ctx context.Context, c *Client) error { return ignore(w(c).Events.List(ctx, nil)) })
	ep("GET", "/v0/events/{id}", func(ctx context.Context, c *Client) error { return ignore(w(c).Events.Get(ctx, "e")) })

	// ---- schemas ----

	registerSchema("WebsetsTeamInfo", TeamInfo{})
	registerSchema("Webset", Webset{})
	registerSchema("GetWebsetResponse", GetWebsetResponse{})
	registerSchema("CreateWebsetParameters", CreateWebsetParameters{})
	registerSchema("UpdateWebsetRequest", UpdateWebsetRequest{})
	registerSchema("ListWebsetsResponse", Page[Webset]{}, "object")
	registerSchema("PreviewWebsetParameters", PreviewWebsetParameters{})
	registerSchema("PreviewWebsetResponse", PreviewWebsetResponse{})

	// Entity is one struct; only custom entities carry a description.
	registerSchema("Entity", Entity{})
	registerSchema("CompanyEntity", Entity{}, "description")
	registerSchema("PersonEntity", Entity{}, "description")
	registerSchema("ArticleEntity", Entity{}, "description")
	registerSchema("ResearchPaperEntity", Entity{}, "description")
	registerSchema("CustomEntity", Entity{})
	registerSchema("CreateCriterionParameters", CreateCriterionParameters{})

	registerSchema("WebsetSearch", WebsetSearch{})
	registerSchema("CreateWebsetSearchParameters", CreateWebsetSearchParameters{})

	registerSchema("WebsetEnrichment", WebsetEnrichment{})
	registerSchema("CreateEnrichmentParameters", CreateEnrichmentParameters{})
	registerSchema("UpdateEnrichmentParameters", UpdateEnrichmentParameters{})
	registerSchema("EnrichmentResult", EnrichmentResult{})

	registerSchema("WebsetItem", WebsetItem{})
	registerSchema("WebsetItemPreview", WebsetItemPreview{})
	registerSchema("WebsetItemEvaluation", WebsetItemEvaluation{})
	registerSchema("ListWebsetItemResponse", Page[WebsetItem]{}, "object")
	// ItemProperties is one struct; each variant sets only its own payload key.
	payloads := []string{"person", "company", "article", "researchPaper", "custom"}
	without := func(keep string, extra ...string) []string {
		out := extra
		for _, p := range payloads {
			if p != keep {
				out = append(out, p)
			}
		}
		return out
	}
	registerSchema("WebsetItemPersonProperties", ItemProperties{}, without("person", "content")...)
	registerSchema("WebsetItemCompanyProperties", ItemProperties{}, without("company")...)
	registerSchema("WebsetItemArticleProperties", ItemProperties{}, without("article")...)
	registerSchema("WebsetItemResearchPaperProperties", ItemProperties{}, without("researchPaper")...)
	registerSchema("WebsetItemCustomProperties", ItemProperties{}, without("custom")...)

	registerSchema("Monitor", Monitor{})
	registerSchema("MonitorRun", MonitorRun{})
	registerSchema("MonitorCadence", MonitorCadence{})
	registerSchema("MonitorBehavior", MonitorBehavior{})
	registerSchema("CreateMonitorParameters", CreateMonitorParameters{})
	registerSchema("UpdateMonitor", UpdateMonitor{})
	registerSchema("ListMonitorsResponse", Page[Monitor]{}, "object")
	registerSchema("ListMonitorRunsResponse", Page[MonitorRun]{}, "object")

	registerSchema("Import", Import{})
	registerSchema("CreateImportParameters", CreateImportParameters{})
	registerSchema("CreateImportResponse", CreateImportResponse{})
	registerSchema("UpdateImport", UpdateImport{})
	registerSchema("ListImportsResponse", Page[Import]{}, "object")

	registerSchema("Webhook", Webhook{})
	registerSchema("WebhookAttempt", WebhookAttempt{})
	registerSchema("CreateWebhookParameters", CreateWebhookParameters{})
	registerSchema("UpdateWebhookParameters", UpdateWebhookParameters{})
	registerSchema("ListWebhooksResponse", Page[Webhook]{}, "object")
	registerSchema("ListWebhookAttemptsResponse", Page[WebhookAttempt]{}, "object")

	// Event is a union over Type; Data stays raw with typed accessors.
	registerSchema("Event", Event{})
	registerSchema("ListEventsResponse", Page[Event]{}, "object")
}
