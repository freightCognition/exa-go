package exa

import (
	"context"
	"net/http"
)

// ContentsRequest is the body of POST /contents. Set IDs (from search
// results) or URLs. The content options sit at the top level of the body.
type ContentsRequest struct {
	IDs        []string   `json:"ids,omitempty"`
	URLs       []string   `json:"urls,omitempty"`
	Compliance Compliance `json:"compliance,omitempty"`
	// UserIdentifier tags the request with your end user's ID.
	UserIdentifier string `json:"userIdentifier,omitempty"`
	ContentsOptions
}

// ContentsResponse is the result of POST /contents. A page that couldn't be
// fetched is missing from Results and reported in Statuses; see Failed.
type ContentsResponse struct {
	RequestID   string          `json:"requestId,omitempty"`
	Results     []SearchResult  `json:"results"`
	Statuses    []ContentStatus `json:"statuses,omitempty"`
	CostDollars *CostDollars    `json:"costDollars,omitempty"`
	// SearchTime is in milliseconds.
	SearchTime float64 `json:"searchTime,omitempty"`

	// Deprecated: use Results[].Highlights or Results[].Text.
	Context string `json:"context,omitempty"`
}

// Failed returns the statuses of pages that could not be fetched.
func (r *ContentsResponse) Failed() []ContentStatus {
	var out []ContentStatus
	for _, s := range r.Statuses {
		if s.Status == "error" {
			out = append(out, s)
		}
	}
	return out
}

// ContentStatus is the fetch outcome for one requested ID or URL.
type ContentStatus struct {
	ID string `json:"id"`
	// Status is "success" or "error".
	Status string `json:"status"`
	// Source is "cached" or "crawled".
	Source string              `json:"source,omitempty"`
	Error  *ContentStatusError `json:"error,omitempty"`
}

// ContentStatusError says why a page could not be fetched.
type ContentStatusError struct {
	// Tag is e.g. CRAWL_NOT_FOUND or CRAWL_TIMEOUT.
	Tag string `json:"tag,omitempty"`
	// HTTPStatusCode is the status the page's server returned, when there was one.
	HTTPStatusCode *int `json:"httpStatusCode,omitempty"`
}

// Contents fetches page contents by ID or URL (POST /contents). Check
// ContentsResponse.Failed: a page that fails is not an error here.
func (c *Client) Contents(ctx context.Context, req *ContentsRequest) (*ContentsResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}
	var out ContentsResponse
	if err := c.do(ctx, http.MethodPost, c.baseURL, "/contents", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
