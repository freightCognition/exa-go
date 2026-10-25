package exa

import (
	"context"
	"net/http"
)

// The team management spec defines its schemas inline (no components), so
// there is nothing to register with registerSchema; apikeys_test.go checks
// the field names against the spec's examples.
func init() {
	registerAdminEndpoint(http.MethodPost, "/api-keys", func(ctx context.Context, c *Client) error {
		_, err := c.APIKeys.Create(ctx, &APIKeyRequest{})
		return err
	})
	registerAdminEndpoint(http.MethodGet, "/api-keys", func(ctx context.Context, c *Client) error {
		_, err := c.APIKeys.List(ctx)
		return err
	})
	registerAdminEndpoint(http.MethodGet, "/api-keys/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.APIKeys.Get(ctx, "k")
		return err
	})
	registerAdminEndpoint(http.MethodPut, "/api-keys/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.APIKeys.Update(ctx, "k", &APIKeyRequest{})
		return err
	})
	registerAdminEndpoint(http.MethodDelete, "/api-keys/{id}", func(ctx context.Context, c *Client) error {
		return c.APIKeys.Delete(ctx, "k")
	})
	registerAdminEndpoint(http.MethodGet, "/api-keys/{id}/usage", func(ctx context.Context, c *Client) error {
		_, err := c.APIKeys.Usage(ctx, "k", nil)
		return err
	})
}
