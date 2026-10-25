package exa

import (
	"encoding/json"
	"reflect"
)

// ContentsOptions choose what page content comes back with results. Used by
// Search, FindSimilar, Contents (flattened) and Search Monitors.
//
// Text, Highlights, Summary and Context accept "true" or an options object:
// a pointer to an empty options struct sends true; a nil pointer omits it.
type ContentsOptions struct {
	Text       *TextOptions       `json:"text,omitempty"`
	Highlights *HighlightsOptions `json:"highlights,omitempty"`
	Summary    *SummaryOptions    `json:"summary,omitempty"`
	Extras     *ExtrasOptions     `json:"extras,omitempty"`
	// LivecrawlTimeout is in milliseconds (max 90000).
	LivecrawlTimeout *int `json:"livecrawlTimeout,omitempty"`
	// MaxAgeHours caps cached content age: -1 cache only, 0 always fetch
	// fresh, up to 720.
	MaxAgeHours *int `json:"maxAgeHours,omitempty"`
	// SnapshotAsOf returns the newest stored version of each page as of this
	// ISO 8601 date or date-time.
	SnapshotAsOf string `json:"snapshotAsOf,omitempty"`

	// Deprecated: use Highlights or Text.
	Context *ContextOptions `json:"context,omitempty"`
	// Deprecated: use MaxAgeHours.
	Livecrawl Livecrawl `json:"livecrawl,omitempty"`
	// Deprecated: alias of SnapshotAsOf.
	CrawledBeforeDate string `json:"crawledBeforeDate,omitempty"`
	// Deprecated: number of subpages to crawl per result.
	Subpages *int `json:"subpages,omitempty"`
	// Deprecated: terms to find specific subpages.
	SubpageTarget []string `json:"subpageTarget,omitempty"`
}

// TextOptions controls full-text extraction. An empty struct sends true.
type TextOptions struct {
	MaxCharacters   *int          `json:"maxCharacters,omitempty"`
	IncludeHTMLTags *bool         `json:"includeHtmlTags,omitempty"`
	Verbosity       TextVerbosity `json:"verbosity,omitempty"`
	IncludeSections []Section     `json:"includeSections,omitempty"`
	ExcludeSections []Section     `json:"excludeSections,omitempty"`
}

// HighlightsOptions steers highlight extraction. An empty struct sends true,
// which Exa recommends for the best default.
type HighlightsOptions struct {
	Query     string              `json:"query,omitempty"`
	Verbosity HighlightsVerbosity `json:"verbosity,omitempty"`
	// Dynamic requires the dynamic-highlights beta.
	Dynamic       *bool `json:"dynamic,omitempty"`
	MaxCharacters *int  `json:"maxCharacters,omitempty"`

	// Deprecated: use MaxCharacters.
	NumSentences *int `json:"numSentences,omitempty"`
	// Deprecated: use MaxCharacters.
	HighlightsPerURL *int `json:"highlightsPerUrl,omitempty"`
}

// SummaryOptions requests an LLM summary of each page. An empty struct sends true.
type SummaryOptions struct {
	Query string `json:"query,omitempty"`
	// Schema is a JSON Schema for a structured summary.
	Schema map[string]any `json:"schema,omitempty"`
}

// ContextOptions is the deprecated combined-context option. An empty struct sends true.
type ContextOptions struct {
	MaxCharacters *int `json:"maxCharacters,omitempty"`
}

// ExtrasOptions asks for links found on each page.
type ExtrasOptions struct {
	Links          *int `json:"links,omitempty"`
	ImageLinks     *int `json:"imageLinks,omitempty"`
	RichImageLinks *int `json:"richImageLinks,omitempty"`
	// Deprecated.
	RichLinks *int `json:"richLinks,omitempty"`
	// Deprecated.
	CodeBlocks *int `json:"codeBlocks,omitempty"`
}

func (o TextOptions) MarshalJSON() ([]byte, error) {
	type plain TextOptions
	return trueWhenEmpty(plain(o))
}

func (o HighlightsOptions) MarshalJSON() ([]byte, error) {
	type plain HighlightsOptions
	return trueWhenEmpty(plain(o))
}

func (o SummaryOptions) MarshalJSON() ([]byte, error) {
	type plain SummaryOptions
	return trueWhenEmpty(plain(o))
}

func (o ContextOptions) MarshalJSON() ([]byte, error) {
	type plain ContextOptions
	return trueWhenEmpty(plain(o))
}

// trueWhenEmpty encodes a zero struct as true (the API's "use defaults") and
// anything else as an object.
func trueWhenEmpty(v any) ([]byte, error) {
	if reflect.ValueOf(v).IsZero() {
		return []byte("true"), nil
	}
	return json.Marshal(v)
}

// TextVerbosity sets how much of a page's text is kept.
type TextVerbosity string

const (
	TextVerbosityCompact  TextVerbosity = "compact"
	TextVerbosityStandard TextVerbosity = "standard"
	TextVerbosityFull     TextVerbosity = "full"
)

// HighlightsVerbosity sets highlight length.
type HighlightsVerbosity string

const (
	HighlightsVerbosityLow    HighlightsVerbosity = "low"
	HighlightsVerbosityMedium HighlightsVerbosity = "medium"
	HighlightsVerbosityHigh   HighlightsVerbosity = "high"
)

// Section is a part of a page for include/exclude filters.
type Section string

const (
	SectionHeader     Section = "header"
	SectionNavigation Section = "navigation"
	SectionBanner     Section = "banner"
	SectionBody       Section = "body"
	SectionSidebar    Section = "sidebar"
	SectionFooter     Section = "footer"
	SectionMetadata   Section = "metadata"
)

// Livecrawl is the deprecated freshness setting; use ContentsOptions.MaxAgeHours.
type Livecrawl string

const (
	LivecrawlNever     Livecrawl = "never"
	LivecrawlAlways    Livecrawl = "always"
	LivecrawlFallback  Livecrawl = "fallback"
	LivecrawlPreferred Livecrawl = "preferred"
)

// OutputSchema asks for synthesized output: plain text (Type "text") or a
// JSON object matching a schema (Type "object").
type OutputSchema struct {
	Type        OutputSchemaType `json:"type"`
	Description string           `json:"description,omitempty"`
	// Properties, Required and AdditionalProperties apply to Type "object".
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	AdditionalProperties *bool          `json:"additionalProperties,omitempty"`
}

// OutputSchemaType is "text" or "object".
type OutputSchemaType string

const (
	OutputSchemaTypeText   OutputSchemaType = "text"
	OutputSchemaTypeObject OutputSchemaType = "object"
)

// CostDollars is what a request cost. Which parts are set varies by endpoint.
type CostDollars struct {
	Total    float64       `json:"total,omitempty"`
	Search   *SearchCost   `json:"search,omitempty"`
	Summary  float64       `json:"summary,omitempty"`
	Contents *ContentsCost `json:"contents,omitempty"`
}

// SearchCost breaks down search cost.
type SearchCost struct {
	Neural  float64 `json:"neural,omitempty"`
	Keyword float64 `json:"keyword,omitempty"`
}

// ContentsCost breaks down content retrieval cost.
type ContentsCost struct {
	Text       float64 `json:"text,omitempty"`
	Highlights float64 `json:"highlights,omitempty"`
	Summary    float64 `json:"summary,omitempty"`
}
