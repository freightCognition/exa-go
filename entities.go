package exa

import "encoding/json"

// ResultEntity is structured data about a result's subject. Exactly one of
// Company, Person and Publication is set, matching Type; for any other Type
// only Properties (the raw JSON) is.
//
// Nullable strings decode to "", nullable numbers to nil.
type ResultEntity struct {
	ID      string           `json:"id"`
	Type    ResultEntityType `json:"type"`
	Version int              `json:"version"`
	// Properties is the raw "properties" object.
	Properties json.RawMessage `json:"properties"`

	Company     *ResultEntityCompany     `json:"-"`
	Person      *ResultEntityPerson      `json:"-"`
	Publication *ResultEntityPublication `json:"-"`
}

// ResultEntityType is the kind of a ResultEntity.
type ResultEntityType string

const (
	ResultEntityTypeCompany     ResultEntityType = "company"
	ResultEntityTypePerson      ResultEntityType = "person"
	ResultEntityTypePublication ResultEntityType = "publication"
)

// UnmarshalJSON decodes Properties into the field for Type.
func (e *ResultEntity) UnmarshalJSON(b []byte) error {
	type plain ResultEntity
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*e = ResultEntity(p)
	var dst any
	switch e.Type {
	case ResultEntityTypeCompany:
		e.Company = new(ResultEntityCompany)
		dst = e.Company
	case ResultEntityTypePerson:
		e.Person = new(ResultEntityPerson)
		dst = e.Person
	case ResultEntityTypePublication:
		e.Publication = new(ResultEntityPublication)
		dst = e.Publication
	default:
		return nil
	}
	return json.Unmarshal(e.Properties, dst)
}

// ResultEntityCompany describes a company.
type ResultEntityCompany struct {
	Name         string                       `json:"name"`
	FoundedYear  *int                         `json:"foundedYear"`
	Description  string                       `json:"description"`
	Workforce    *ResultEntityWorkforce       `json:"workforce"`
	Headquarters *ResultEntityHeadquarters    `json:"headquarters"`
	Financials   *ResultEntityFinancials      `json:"financials"`
	WebTraffic   *ResultEntityWebTraffic      `json:"webTraffic"`
	Research     *ResultEntityCompanyResearch `json:"research"`
}

// ResultEntityWorkforce is a company's headcount.
type ResultEntityWorkforce struct {
	Total *float64 `json:"total"`
}

// ResultEntityHeadquarters is a company's main address.
type ResultEntityHeadquarters struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// ResultEntityFinancials are a company's revenue and funding.
type ResultEntityFinancials struct {
	RevenueAnnual      *float64                  `json:"revenueAnnual"`
	FundingTotal       *float64                  `json:"fundingTotal"`
	FundingLatestRound *ResultEntityFundingRound `json:"fundingLatestRound"`
}

// ResultEntityFundingRound is one funding round.
type ResultEntityFundingRound struct {
	Name   string   `json:"name"`
	Date   string   `json:"date"`
	Amount *float64 `json:"amount"`
}

// ResultEntityWebTraffic is a company site's traffic.
type ResultEntityWebTraffic struct {
	VisitsMonthly      *float64                   `json:"visitsMonthly"`
	CountryRank        *int                       `json:"countryRank"`
	AvgDurationSeconds *float64                   `json:"avgDurationSeconds"`
	History            []ResultEntityTrafficPoint `json:"history"`
}

// ResultEntityTrafficPoint is visits over one period.
type ResultEntityTrafficPoint struct {
	Value    float64 `json:"value"`
	DateFrom string  `json:"dateFrom"`
	DateTo   string  `json:"dateTo"`
}

// ResultEntityCompanyResearch is a company's research output.
type ResultEntityCompanyResearch struct {
	WorksCount     *int                     `json:"worksCount"`
	CitationCount  *int                     `json:"citationCount"`
	Areas          []string                 `json:"areas"`
	NotableWorks   []ResultEntityWork       `json:"notableWorks"`
	TopResearchers []ResultEntityResearcher `json:"topResearchers"`
}

// ResultEntityResearcher is a person and their research totals.
type ResultEntityResearcher struct {
	Person        *ResultEntityRef `json:"person"`
	WorksCount    *int             `json:"worksCount"`
	CitationCount *int             `json:"citationCount"`
}

// ResultEntityWork is one published work.
type ResultEntityWork struct {
	Title         string `json:"title"`
	Year          *int   `json:"year"`
	Venue         string `json:"venue"`
	CitationCount *int   `json:"citationCount"`
	DOI           string `json:"doi"`
	ID            string `json:"id"`
	// Type is article, book, book-chapter, dataset, dissertation, preprint,
	// report or review.
	Type string `json:"type"`
}

// ResultEntityRef names another entity (a person, company or institution).
type ResultEntityRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResultEntityPerson describes a person.
type ResultEntityPerson struct {
	Name             string                      `json:"name"`
	FirstName        string                      `json:"firstName"`
	LastName         string                      `json:"lastName"`
	Location         string                      `json:"location"`
	WorkHistory      []ResultEntityJob           `json:"workHistory"`
	EducationHistory []ResultEntityEducation     `json:"educationHistory"`
	Research         *ResultEntityPersonResearch `json:"research"`
}

// ResultEntityDates is a from/to span; either end may be "".
type ResultEntityDates struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ResultEntityJob is one role in a person's work history.
type ResultEntityJob struct {
	Title    string             `json:"title"`
	Location string             `json:"location"`
	Dates    *ResultEntityDates `json:"dates"`
	Company  *ResultEntityRef   `json:"company"`
}

// ResultEntityEducation is one entry in a person's education history.
type ResultEntityEducation struct {
	Degree      string             `json:"degree"`
	Dates       *ResultEntityDates `json:"dates"`
	Institution *ResultEntityRef   `json:"institution"`
}

// ResultEntityPersonResearch is a person's research record.
type ResultEntityPersonResearch struct {
	WorksCount            *int               `json:"worksCount"`
	CitationCount         *int               `json:"citationCount"`
	HIndex                *int               `json:"hIndex"`
	FirstPublicationYear  *int               `json:"firstPublicationYear"`
	LatestPublicationYear *int               `json:"latestPublicationYear"`
	Areas                 []string           `json:"areas"`
	NotableWorks          []ResultEntityWork `json:"notableWorks"`
}

// ResultEntityPublication describes a published work.
type ResultEntityPublication struct {
	Title string `json:"title"`
	Year  *int   `json:"year"`
	Date  string `json:"date"`
	// Type is article, book, book-chapter, dataset, dissertation, preprint,
	// report or review.
	Type           string            `json:"type"`
	Language       string            `json:"language"`
	CitationCount  *int              `json:"citationCount"`
	Authors        []ResultEntityRef `json:"authors"`
	ReferenceCount *int              `json:"referenceCount"`
	Abstract       string            `json:"abstract"`
	DOI            string            `json:"doi"`
}
