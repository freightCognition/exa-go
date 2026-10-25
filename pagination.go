package exa

import (
	"context"
	"iter"
	"net/url"
	"strconv"
)

// Page is one page of a cursor-paginated list. Every Exa list endpoint
// returns this shape.
type Page[T any] struct {
	// Object is "list" on the Agent and Batches APIs; empty elsewhere.
	Object     string  `json:"object,omitempty"`
	Data       []T     `json:"data"`
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor"`
}

// ListParams are the cursor and page size shared by list endpoints.
// Zero values are left to the API's defaults.
type ListParams struct {
	Cursor string
	Limit  int
}

func (p *ListParams) values() url.Values {
	v := url.Values{}
	if p == nil {
		return v
	}
	if p.Cursor != "" {
		v.Set("cursor", p.Cursor)
	}
	if p.Limit > 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	return v
}

// All yields every item across pages, calling fetch with each next cursor
// ("" for the first page). It stops at the first error.
//
//	for run, err := range exa.All(ctx, func(ctx context.Context, cursor string) (*exa.Page[exa.AgentRun], error) {
//		return c.Agent.List(ctx, &exa.ListParams{Cursor: cursor})
//	}) { ... }
func All[T any](ctx context.Context, fetch func(ctx context.Context, cursor string) (*Page[T], error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		cursor := ""
		for {
			page, err := fetch(ctx, cursor)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, v := range page.Data {
				if !yield(v, nil) {
					return
				}
			}
			if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
				return
			}
			cursor = *page.NextCursor
		}
	}
}
