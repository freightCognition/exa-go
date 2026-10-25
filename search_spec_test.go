package exa

import (
	"context"
	"reflect"
)

// entityWire is the wire shape of one result entity variant. ResultEntity
// decodes itself (its properties land in a typed field chosen by "type"), so
// the schema test checks each variant through this mirror instead.
type entityWire[P any] struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Version    int    `json:"version"`
	Properties P      `json:"properties"`
}

// allBut returns every JSON field of v except keep, for registering one
// variant of a flat union struct.
func allBut(v any, keep ...string) []string {
	k := map[string]bool{}
	for _, n := range keep {
		k[n] = true
	}
	var out []string
	for name := range jsonFields(reflect.TypeOf(v)) {
		if !k[name] {
			out = append(out, name)
		}
	}
	return out
}

func init() {
	registerEndpoint("POST", "/search", func(ctx context.Context, c *Client) error {
		_, err := c.Search(ctx, &SearchRequest{Query: "q"})
		return err
	})
	registerEndpoint("POST", "/contents", func(ctx context.Context, c *Client) error {
		_, err := c.Contents(ctx, &ContentsRequest{URLs: []string{"https://example.com"}})
		return err
	})
	registerEndpoint("POST", "/answer", func(ctx context.Context, c *Client) error {
		_, err := c.Answer(ctx, &AnswerRequest{Query: "q"})
		return err
	})
	registerEndpoint("POST", "/findSimilar", func(ctx context.Context, c *Client) error {
		_, err := c.FindSimilar(ctx, &FindSimilarRequest{URL: "https://example.com"})
		return err
	})

	registerSchema("SearchRequest", SearchRequest{})
	registerSchema("SearchSynthesisResponse", SearchResponse{})
	registerSchema("SearchResultsResponse", SearchResponse{}, "output")
	registerSchema("SearchResultOutput", SearchResult{})
	registerSchema("SearchSynthesisOutputOutput", SearchSynthesisOutput{})
	registerSchema("SearchWarningOutput", Warning{})
	registerSchema("SearchNoticeOutput", Notice{})
	registerSchema("SearchCompanyEntityOutput", entityWire[ResultEntityCompany]{})
	registerSchema("SearchPersonEntityOutput", entityWire[ResultEntityPerson]{})
	registerSchema("SearchPublicationEntityOutput", entityWire[ResultEntityPublication]{})

	registerSchema("ContentsRequest", ContentsRequest{})
	registerSchema("ContentsResponse", ContentsResponse{})

	registerSchema("AnswerRequest", AnswerRequest{})
	registerSchema("AnswerResponse", AnswerResponse{})

	registerSchema("FindSimilarRequest", FindSimilarRequest{})
	registerSchema("FindSimilarResponse", FindSimilarResponse{})
	registerSchema("FindSimilarResult", FindSimilarResult{})

	registerSchema("X402PaymentChallenge", X402PaymentChallenge{})

	// Stream chunks: one flat Go struct per stream, each variant checked
	// against the fields it carries.
	sc := SearchStreamChunk{}
	registerSchema("SearchStreamTextDeltaChunkOutput", sc, allBut(sc, "requestId", "type", "delta", "choices")...)
	registerSchema("SearchStreamGroundingChunkOutput", sc, allBut(sc, "requestId", "type", "grounding", "citations", "choices")...)
	registerSchema("SearchStreamResultsChunkOutput", sc, allBut(sc, "requestId", "type", "results", "warnings", "notices")...)
	registerSchema("SearchStreamResetChunkOutput", sc, allBut(sc, "requestId", "type", "streamReset")...)
	registerSchema("SearchStreamDoneChunkOutput", sc, allBut(sc, "requestId", "type", "output", "searchTime", "costDollars", "choices")...)
	exemptSchema("SearchStreamErrorChunkOutput", "returned by SearchStream as a *StreamError, never yielded as a chunk")
	exemptSchema("SearchStreamChunk", "union of the SearchStream*ChunkOutput variants, each registered")

	ac := AnswerStreamChunk{}
	registerSchema("AnswerStreamTextDeltaChunk", ac, allBut(ac, "choices")...)
	registerSchema("AnswerStreamCitationsChunkOutput", ac, allBut(ac, "citations")...)
	registerSchema("AnswerStreamCostChunkOutput", ac, allBut(ac, "costDollars", "requestId")...)
	exemptSchema("AnswerStreamErrorChunkOutput", "returned by AnswerStream as a *StreamError, never yielded as a chunk")
	exemptSchema("AnswerStreamChunk", "union of the AnswerStream*Chunk variants, each registered")
	exemptSchema("SearchResponse", "union of SearchSynthesisResponse and SearchResultsResponse, both registered against SearchResponse")
}
