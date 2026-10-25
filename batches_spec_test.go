package exa

import (
	"context"
	"net/http"
)

func init() {
	registerEndpoint(http.MethodPost, "/batches", func(ctx context.Context, c *Client) error {
		_, err := c.Batches.Create(ctx, &CreateBatchRequest{})
		return err
	})
	registerEndpoint(http.MethodGet, "/batches", func(ctx context.Context, c *Client) error {
		_, err := c.Batches.List(ctx, nil)
		return err
	})
	registerEndpoint(http.MethodGet, "/batches/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Batches.Get(ctx, "b")
		return err
	})
	registerEndpoint(http.MethodDelete, "/batches/{id}", func(ctx context.Context, c *Client) error {
		_, err := c.Batches.Delete(ctx, "b")
		return err
	})
	registerEndpoint(http.MethodPost, "/batches/{id}/cancel", func(ctx context.Context, c *Client) error {
		_, err := c.Batches.Cancel(ctx, "b")
		return err
	})

	registerSchema("CreateBatchRequest", CreateBatchRequest{})
	registerSchema("BatchRequestItem", BatchRequestItem{})
	registerSchema("Batch", Batch{})
	registerSchema("BatchRequestCounts", BatchRequestCounts{})
	registerSchema("BatchList", Page[Batch]{})
	registerSchema("DeletedBatch", DeletedBatch{})
}
