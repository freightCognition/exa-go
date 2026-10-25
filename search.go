package exa

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
)

var errNilRequest = errors.New("exa: nil request")

// SearchRequest is the body of POST /search. Only Query is required.
type SearchRequest struct {
	Query string `json:"query"`
	// Type picks the search mode; the API default is SearchTypeAuto.
	Type     SearchType `json:"type,omitempty"`
	Category Category   `json:"category,omitempty"`
	// NumResults is 1 to 100 (API default 10).
	NumResults     *int     `json:"numResults,omitempty"`
	IncludeDomains []string `json:"includeDomains,omitempty"`
	ExcludeDomains []string `json:"excludeDomains,omitempty"`
	// StartPublishedDate and EndPublishedDate are ISO 8601 date-times.
	StartPublishedDate string `json:"startPublishedDate,omitempty"`
	EndPublishedDate   string `json:"endPublishedDate,omitempty"`
	// Moderation filters unsafe content from results.
	Moderation *bool            `json:"moderation,omitempty"`
	Contents   *ContentsOptions `json:"contents,omitempty"`
	// AdditionalQueries are extra phrasings searched alongside Query (1 to 10).
	AdditionalQueries []string      `json:"additionalQueries,omitempty"`
	UserLocation      *UserLocation `json:"userLocation,omitempty"`
	Compliance        Compliance    `json:"compliance,omitempty"`
	// OutputSchema asks for synthesized output in SearchResponse.Output.
	// SearchStream requires it.
	OutputSchema *OutputSchema `json:"outputSchema,omitempty"`
	// Objective describes what the search is for, to steer results.
	Objective string `json:"objective,omitempty"`
	// UserIdentifier tags the request with your end user's ID
	// (1 to 64 of A-Z, a-z, 0-9, _ and -).
	UserIdentifier string `json:"userIdentifier,omitempty"`
	// SystemPrompt steers synthesized output.
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Stream is set by SearchStream. Search ignores it.
	Stream bool `json:"stream,omitempty"`

	// Deprecated: crawl dates are no longer a supported filter.
	StartCrawlDate string `json:"startCrawlDate,omitempty"`
	// Deprecated: crawl dates are no longer a supported filter.
	EndCrawlDate string `json:"endCrawlDate,omitempty"`
	// Deprecated: no longer a supported filter.
	IncludeText []string `json:"includeText,omitempty"`
	// Deprecated: no longer a supported filter.
	ExcludeText []string `json:"excludeText,omitempty"`
	// Deprecated: use Contents.Highlights or Contents.Text.
	Context *ContextOptions `json:"context,omitempty"`
}

// SearchType is the search mode. The API accepts values beyond these.
type SearchType string

const (
	SearchTypeAuto     SearchType = "auto"
	SearchTypeLite     SearchType = "lite"
	SearchTypeDeepLite SearchType = "deep-lite"
	SearchTypeDeep     SearchType = "deep"

	// Deprecated: legacy; use SearchTypeLite.
	SearchTypeInstant SearchType = "instant"
	// Deprecated: legacy; use SearchTypeLite.
	SearchTypeFast SearchType = "fast"
	// Deprecated: legacy; use SearchTypeDeep.
	SearchTypeDeepReasoning SearchType = "deep-reasoning"
)

// Category narrows results to one kind of page.
type Category string

const (
	CategoryCompany         Category = "company"
	CategoryPublication     Category = "publication"
	CategoryNews            Category = "news"
	CategoryPersonalSite    Category = "personal site"
	CategoryFinancialReport Category = "financial report"
	CategoryPeople          Category = "people"
)

// Compliance restricts processing to a compliance regime.
type Compliance string

// ComplianceHIPAA keeps the request within HIPAA-eligible processing.
const ComplianceHIPAA Compliance = "hipaa"

// UserLocation localizes results. Set Country (ISO 3166-1 alpha-2, e.g. "US"),
// or Latitude and Longitude, or all three.
type UserLocation struct {
	Country   string   `json:"country,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// SearchResponse is the result of POST /search. Output is set only when the
// request had an OutputSchema.
type SearchResponse struct {
	RequestID   string                 `json:"requestId,omitempty"`
	Results     []SearchResult         `json:"results"`
	Output      *SearchSynthesisOutput `json:"output,omitempty"`
	CostDollars *CostDollars           `json:"costDollars,omitempty"`
	// SearchTime is in milliseconds.
	SearchTime float64   `json:"searchTime,omitempty"`
	Warnings   []Warning `json:"warnings,omitempty"`
	Notices    []Notice  `json:"notices,omitempty"`

	// Deprecated: no longer meaningful.
	ResolvedSearchType string `json:"resolvedSearchType,omitempty"`
	// Deprecated: use Results[].Highlights or Results[].Text.
	Context string `json:"context,omitempty"`
}

// SearchResult is one page. Content fields are set when requested through
// ContentsOptions.
type SearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	// ID identifies the page for Contents.
	ID string `json:"id,omitempty"`
	// PublishedDate is ISO 8601, when known.
	PublishedDate   string          `json:"publishedDate,omitempty"`
	Author          *string         `json:"author,omitempty"`
	Image           string          `json:"image,omitempty"`
	Favicon         string          `json:"favicon,omitempty"`
	Text            string          `json:"text,omitempty"`
	Highlights      []string        `json:"highlights,omitempty"`
	HighlightScores []float64       `json:"highlightScores,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	Subpages        []SearchSubpage `json:"subpages,omitempty"`
	Entities        []ResultEntity  `json:"entities,omitempty"`
	Extras          *ResultExtras   `json:"extras,omitempty"`
}

// SearchSubpage is a subpage of a result (see the deprecated
// ContentsOptions.Subpages). The spec lists only these fields.
type SearchSubpage struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	ID            string  `json:"id,omitempty"`
	PublishedDate string  `json:"publishedDate,omitempty"`
	Author        *string `json:"author,omitempty"`
	Image         string  `json:"image,omitempty"`
	Favicon       string  `json:"favicon,omitempty"`
}

// ResultExtras are the links asked for with ContentsOptions.Extras.
type ResultExtras struct {
	Links          []string          `json:"links,omitempty"`
	ImageLinks     []string          `json:"imageLinks,omitempty"`
	RichImageLinks []ResultImageLink `json:"richImageLinks,omitempty"`
	// Deprecated.
	RichLinks []ResultLink `json:"richLinks,omitempty"`
	// Deprecated.
	CodeBlocks []ResultCodeBlock `json:"codeBlocks,omitempty"`
}

// ResultImageLink is an image found on a page.
type ResultImageLink struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

// ResultLink is a link found on a page.
type ResultLink struct {
	URL    string `json:"url"`
	Anchor string `json:"anchor,omitempty"`
}

// ResultCodeBlock is a code block found on a page.
type ResultCodeBlock struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

// SearchSynthesisOutput is the synthesized answer for a request with an
// OutputSchema.
type SearchSynthesisOutput struct {
	// Content is a JSON string for a text schema, or an object matching an
	// object schema. Text returns the string form.
	Content   json.RawMessage `json:"content"`
	Grounding []Grounding     `json:"grounding"`
}

// Text returns Content when it is a string.
func (o *SearchSynthesisOutput) Text() (string, bool) { return rawString(o.Content) }

// Grounding ties one output field to the pages that support it.
type Grounding struct {
	// Field is the output path, e.g. "content" or a property name.
	Field     string              `json:"field"`
	Citations []GroundingCitation `json:"citations"`
	// Confidence is "low", "medium" or "high".
	Confidence string `json:"confidence"`
}

// GroundingCitation is a page cited by a Grounding.
type GroundingCitation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// Warning flags something about the request that the API changed or ignored.
type Warning struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Notice is informational, e.g. a rewritten query.
type Notice struct {
	Code    string         `json:"code"`
	Path    string         `json:"path"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// rawString decodes raw as a JSON string.
func rawString(raw json.RawMessage) (string, bool) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}

// Search runs a web search (POST /search).
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}
	r := *req
	r.Stream = false
	var out SearchResponse
	if err := c.do(ctx, http.MethodPost, c.baseURL, "/search", nil, &r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchStream runs a search and streams its synthesized output. The API
// streams only when an output schema is set, so req.OutputSchema is required.
// An error event from the server ends the stream with a *StreamError.
//
//	for chunk, err := range c.SearchStream(ctx, req) {
//		if err != nil { return err }
//		fmt.Print(chunk.Delta)
//	}
func (c *Client) SearchStream(ctx context.Context, req *SearchRequest) iter.Seq2[SearchStreamChunk, error] {
	if req == nil {
		return failSeq[SearchStreamChunk](errNilRequest)
	}
	if req.OutputSchema == nil {
		return failSeq[SearchStreamChunk](errors.New("exa: SearchStream needs an OutputSchema; without one /search does not stream"))
	}
	r := *req
	r.Stream = true
	return stream(ctx, c, http.MethodPost, c.baseURL, "/search", nil, &r, decodeSearchChunk)
}
