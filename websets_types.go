package exa

import "time"

// Types shared across the Websets API: entities, references to other
// websets and imports, and the item shapes returned by items, previews and
// events.

// EntityType is the kind of thing a webset, search or import collects.
type EntityType string

// Entity types.
const (
	EntityCompany       EntityType = "company"
	EntityPerson        EntityType = "person"
	EntityArticle       EntityType = "article"
	EntityResearchPaper EntityType = "research_paper"
	EntityCustom        EntityType = "custom"
)

// Entity is the entity a search or import targets. Description is set only
// for EntityCustom (2 to 200 characters).
type Entity struct {
	Type        EntityType `json:"type"`
	Description string     `json:"description,omitempty"`
}

// WebsetSource is where a referenced collection lives.
type WebsetSource string

// Webset sources.
const (
	WebsetSourceImport WebsetSource = "import"
	WebsetSourceWebset WebsetSource = "webset"
)

// SourceRef points at an import or another webset, e.g. to exclude its items.
type SourceRef struct {
	Source WebsetSource `json:"source"`
	ID     string       `json:"id"`
}

// ScopeRef limits a search to an import or webset. With Relationship set, the
// search looks for entities related to each scoped item (a hop search).
type ScopeRef struct {
	SourceRef
	Relationship *ScopeRelationship `json:"relationship,omitempty"`
}

// ScopeRelationship describes how searched entities relate to scoped items.
type ScopeRelationship struct {
	// Definition is the relationship, e.g. "investors of".
	Definition string `json:"definition"`
	// Limit is how many related entities to find per scoped item (1 to 10).
	Limit int `json:"limit"`
}

// ImportRef attaches an import or webset's items to a new webset.
type ImportRef struct {
	SourceRef
	// Evaluate runs the webset's criteria against the imported items.
	Evaluate *bool `json:"evaluate,omitempty"`
}

// CreateCriterionParameters is one criterion items must satisfy.
type CreateCriterionParameters struct {
	// Description is 1 to 1000 characters.
	Description string `json:"description"`
}

// Criterion is a search criterion with how often found items satisfied it.
type Criterion struct {
	Description string `json:"description"`
	// SuccessRate is the percentage (0 to 100) of analyzed items that matched.
	SuccessRate float64 `json:"successRate"`
}

// WebsetItemSource is how an item entered a webset.
type WebsetItemSource string

// Webset item sources.
const (
	WebsetItemSourceSearch WebsetItemSource = "search"
	WebsetItemSourceImport WebsetItemSource = "import"
)

// WebsetItem is one entity in a webset, with its criteria evaluations and
// enrichment results.
type WebsetItem struct {
	ID     string           `json:"id"`
	Object string           `json:"object"`
	Source WebsetItemSource `json:"source"`
	// SourceID is the search or import that produced the item.
	SourceID       string `json:"sourceId"`
	SourceEntityID string `json:"sourceEntityId,omitempty"`
	// ScopeID is the scoped item a hop search found this item through.
	ScopeID     string                 `json:"scopeId,omitempty"`
	WebsetID    string                 `json:"websetId"`
	Properties  ItemProperties         `json:"properties"`
	Evaluations []WebsetItemEvaluation `json:"evaluations"`
	// Enrichments is nil until enrichments have run on the item.
	Enrichments []EnrichmentResult `json:"enrichments"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// WebsetItemPreview is an example item returned by Preview.
type WebsetItemPreview struct {
	ID         string         `json:"id"`
	Properties ItemProperties `json:"properties"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// ItemProperties describes an item. Type says which one of Person, Company,
// Article, ResearchPaper or Custom is set. Content is the page text and is
// empty for people.
type ItemProperties struct {
	Type        EntityType `json:"type"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Content     string     `json:"content,omitempty"`

	Person        *WebsetItemPerson      `json:"person,omitempty"`
	Company       *WebsetItemCompany     `json:"company,omitempty"`
	Article       *WebsetItemPublication `json:"article,omitempty"`
	ResearchPaper *WebsetItemPublication `json:"researchPaper,omitempty"`
	Custom        *WebsetItemPublication `json:"custom,omitempty"`
}

// WebsetItemPerson is a person item.
type WebsetItemPerson struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Position string `json:"position"`
	// Company is the person's current employer, if known.
	Company          *WebsetItemPersonCompany `json:"company"`
	PictureURL       string                   `json:"pictureUrl"`
	WorkHistory      []WebsetItemPersonRole   `json:"workHistory"`
	EducationHistory []WebsetItemEducation    `json:"educationHistory"`
}

// WebsetItemPersonCompany is a person's current employer.
type WebsetItemPersonCompany struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

// WebsetItemPersonRole is one job in a person's work history.
type WebsetItemPersonRole struct {
	Title    string                  `json:"title"`
	Location string                  `json:"location"`
	Dates    *WebsetItemDateRange    `json:"dates"`
	Company  *WebsetItemOrganization `json:"company"`
}

// WebsetItemEducation is one entry in a person's education history.
type WebsetItemEducation struct {
	Degree      string                  `json:"degree"`
	Dates       *WebsetItemDateRange    `json:"dates"`
	Institution *WebsetItemOrganization `json:"institution"`
}

// WebsetItemDateRange is a free-form from/to period; either end may be empty.
type WebsetItemDateRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// WebsetItemOrganization is a company or school in a person's history.
type WebsetItemOrganization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	LinkedinURL string `json:"linkedinUrl"`
}

// WebsetItemCompany is a company item. Pointer fields are nil when unknown.
type WebsetItemCompany struct {
	Name         string                       `json:"name"`
	Location     string                       `json:"location"`
	Employees    *int                         `json:"employees"`
	Industry     string                       `json:"industry"`
	About        string                       `json:"about"`
	LogoURL      string                       `json:"logoUrl"`
	FoundedYear  *int                         `json:"foundedYear"`
	Headquarters *WebsetItemCompanyHQ         `json:"headquarters"`
	Financials   *WebsetItemCompanyFinancials `json:"financials"`
	WebTraffic   *WebsetItemCompanyWebTraffic `json:"webTraffic"`
}

// WebsetItemCompanyHQ is a company's headquarters address.
type WebsetItemCompanyHQ struct {
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

// WebsetItemCompanyFinancials are a company's revenue and funding, in USD.
type WebsetItemCompanyFinancials struct {
	RevenueAnnual      *float64                       `json:"revenueAnnual"`
	FundingTotal       *float64                       `json:"fundingTotal"`
	FundingLatestRound *WebsetItemCompanyFundingRound `json:"fundingLatestRound"`
}

// WebsetItemCompanyFundingRound is a company's latest funding round.
type WebsetItemCompanyFundingRound struct {
	Name   string   `json:"name"`
	Date   string   `json:"date"`
	Amount *float64 `json:"amount"`
}

// WebsetItemCompanyWebTraffic is a company website's monthly traffic.
type WebsetItemCompanyWebTraffic struct {
	VisitsMonthly  *float64 `json:"visitsMonthly"`
	UniqueVisitors *float64 `json:"uniqueVisitors"`
}

// WebsetItemPublication is an article, research paper or custom item.
type WebsetItemPublication struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	// PublishedAt is the date as the source gives it (not normalised).
	PublishedAt string `json:"publishedAt"`
}

// WebsetItemSatisfied is whether an item met a criterion.
type WebsetItemSatisfied string

// Criterion verdicts.
const (
	WebsetItemSatisfiedYes     WebsetItemSatisfied = "yes"
	WebsetItemSatisfiedNo      WebsetItemSatisfied = "no"
	WebsetItemSatisfiedUnclear WebsetItemSatisfied = "unclear"
)

// WebsetItemEvaluation is how an item fared against one criterion.
type WebsetItemEvaluation struct {
	Criterion  string              `json:"criterion"`
	Reasoning  string              `json:"reasoning"`
	Satisfied  WebsetItemSatisfied `json:"satisfied"`
	References []WebsetReference   `json:"references"`
}

// WebsetReference is a source backing an evaluation or enrichment.
type WebsetReference struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

// EnrichmentResult is one enrichment's value for one item.
type EnrichmentResult struct {
	Object string                 `json:"object"`
	Status WebsetEnrichmentStatus `json:"status"`
	Format WebsetEnrichmentFormat `json:"format"`
	// Result holds the value(s); nil until the enrichment completes.
	Result       []string          `json:"result"`
	Reasoning    string            `json:"reasoning"`
	References   []WebsetReference `json:"references"`
	EnrichmentID string            `json:"enrichmentId"`
}
