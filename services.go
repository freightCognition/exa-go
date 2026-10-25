package exa

// Search, Contents, Answer and FindSimilar (and their stream variants) are
// methods on Client. Everything else hangs off these services.

// AgentService is the Agent API (/agent/runs): async multi-step research.
type AgentService struct{ c *Client }

// BatchesService is the Batches API (/batches).
type BatchesService struct{ c *Client }

// MonitorsService is the Search Monitors API (/monitors). Websets monitors
// are a different resource: see WebsetsService.Monitors.
type MonitorsService struct{ c *Client }

// APIKeysService is the Team Management API (admin base URL).
type APIKeysService struct{ c *Client }

// WebsetsService is the Websets API (/websets/v0).
type WebsetsService struct {
	c *Client

	Searches    *WebsetSearchesService
	Enrichments *WebsetEnrichmentsService
	Items       *WebsetItemsService
	Monitors    *WebsetMonitorsService
	Imports     *WebsetImportsService
	Webhooks    *WebsetWebhooksService
	Events      *WebsetEventsService
}

type (
	// WebsetSearchesService is /websets/v0/websets/{webset}/searches.
	WebsetSearchesService struct{ c *Client }
	// WebsetEnrichmentsService is /websets/v0/websets/{webset}/enrichments.
	WebsetEnrichmentsService struct{ c *Client }
	// WebsetItemsService is /websets/v0/websets/{webset}/items.
	WebsetItemsService struct{ c *Client }
	// WebsetMonitorsService is /websets/v0/monitors.
	WebsetMonitorsService struct{ c *Client }
	// WebsetImportsService is /websets/v0/imports.
	WebsetImportsService struct{ c *Client }
	// WebsetWebhooksService is /websets/v0/webhooks.
	WebsetWebhooksService struct{ c *Client }
	// WebsetEventsService is /websets/v0/events.
	WebsetEventsService struct{ c *Client }
)

func (c *Client) initServices() {
	c.Agent = &AgentService{c}
	c.Batches = &BatchesService{c}
	c.Monitors = &MonitorsService{c}
	c.APIKeys = &APIKeysService{c}
	c.Websets = &WebsetsService{
		c:           c,
		Searches:    &WebsetSearchesService{c},
		Enrichments: &WebsetEnrichmentsService{c},
		Items:       &WebsetItemsService{c},
		Monitors:    &WebsetMonitorsService{c},
		Imports:     &WebsetImportsService{c},
		Webhooks:    &WebsetWebhooksService{c},
		Events:      &WebsetEventsService{c},
	}
}
