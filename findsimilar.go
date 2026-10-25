package exa

import (
	"context"
	"net/http"
)

// FindSimilarRequest is the body of POST /findSimilar. Only URL is required.
type FindSimilarRequest struct {
	URL string `json:"url"`
	// NumResults is 1 to 100 (API default 10).
	NumResults     *int     `json:"numResults,omitempty"`
	IncludeDomains []string `json:"includeDomains,omitempty"`
	ExcludeDomains []string `json:"excludeDomains,omitempty"`
	// StartPublishedDate and EndPublishedDate are ISO 8601 date-times.
	StartPublishedDate  string           `json:"startPublishedDate,omitempty"`
	EndPublishedDate    string           `json:"endPublishedDate,omitempty"`
	Category            Category         `json:"category,omitempty"`
	ExcludeSourceDomain *bool            `json:"excludeSourceDomain,omitempty"`
	Contents            *ContentsOptions `json:"contents,omitempty"`

	// Deprecated: crawl dates are no longer a supported filter.
	StartCrawlDate string `json:"startCrawlDate,omitempty"`
	// Deprecated: crawl dates are no longer a supported filter.
	EndCrawlDate string `json:"endCrawlDate,omitempty"`
	// Deprecated: no longer a supported filter.
	IncludeText []string `json:"includeText,omitempty"`
	// Deprecated: no longer a supported filter.
	ExcludeText []string `json:"excludeText,omitempty"`
}

// FindSimilarResponse is the result of POST /findSimilar.
type FindSimilarResponse struct {
	RequestID   string              `json:"requestId,omitempty"`
	Results     []FindSimilarResult `json:"results"`
	CostDollars *CostDollars        `json:"costDollars,omitempty"`
	// SearchTime is in milliseconds.
	SearchTime float64 `json:"searchTime,omitempty"`

	// Deprecated: use Results[].Highlights or Results[].Text.
	Context string `json:"context,omitempty"`
}

// FindSimilarResult is a SearchResult with its similarity score.
type FindSimilarResult struct {
	SearchResult
	Score float64 `json:"score,omitempty"`
}

// FindSimilar finds pages similar to a URL (POST /findSimilar).
//
// Deprecated: Exa marks /findSimilar deprecated; use Search. It is still served.
func (c *Client) FindSimilar(ctx context.Context, req *FindSimilarRequest) (*FindSimilarResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}
	var out FindSimilarResponse
	if err := c.do(ctx, http.MethodPost, c.baseURL, "/findSimilar", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
