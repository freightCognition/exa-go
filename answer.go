package exa

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
)

// AnswerRequest is the body of POST /answer. Only Query is required.
type AnswerRequest struct {
	Query string `json:"query"`
	// Stream is set by AnswerStream. Answer ignores it.
	Stream bool `json:"stream,omitempty"`
	// Text includes each citation's full text.
	Text bool `json:"text,omitempty"`
	// Model defaults to AnswerModelExa.
	Model        AnswerModel   `json:"model,omitempty"`
	SystemPrompt string        `json:"systemPrompt,omitempty"`
	UserLocation *UserLocation `json:"userLocation,omitempty"`
	// OutputSchema is a JSON Schema; the answer then comes back as an object
	// matching it. Not supported by AnswerStream.
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	// UserIdentifier tags the request with your end user's ID.
	UserIdentifier string `json:"userIdentifier,omitempty"`
}

// AnswerModel picks the model behind Answer.
type AnswerModel string

const (
	AnswerModelExa         AnswerModel = "exa"
	AnswerModelExaPro      AnswerModel = "exa-pro"
	AnswerModelExaResearch AnswerModel = "exa-research"
	AnswerModelExaFast     AnswerModel = "exa-fast"
)

// AnswerResponse is the result of POST /answer.
type AnswerResponse struct {
	RequestID string `json:"requestId,omitempty"`
	// Answer is a JSON string, or an object when the request had an
	// OutputSchema. Text returns the string form.
	Answer      json.RawMessage  `json:"answer"`
	Citations   []AnswerCitation `json:"citations,omitempty"`
	CostDollars *CostDollars     `json:"costDollars,omitempty"`
}

// Text returns Answer when it is a string.
func (r *AnswerResponse) Text() (string, bool) { return rawString(r.Answer) }

// AnswerCitation is a page the answer is based on.
type AnswerCitation struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	ID            string  `json:"id,omitempty"`
	PublishedDate string  `json:"publishedDate,omitempty"`
	Author        *string `json:"author,omitempty"`
	Image         string  `json:"image,omitempty"`
	Favicon       string  `json:"favicon,omitempty"`
	// Text is the page text, when AnswerRequest.Text was set.
	Text string `json:"text,omitempty"`
}

// Answer answers a question from search results (POST /answer).
func (c *Client) Answer(ctx context.Context, req *AnswerRequest) (*AnswerResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}
	r := *req
	r.Stream = false
	var out AnswerResponse
	if err := c.do(ctx, http.MethodPost, c.baseURL, "/answer", nil, &r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AnswerStream streams an answer: text deltas, then citations, then cost.
// OutputSchema is not supported when streaming. An error event from the
// server ends the stream with a *StreamError.
//
//	for chunk, err := range c.AnswerStream(ctx, req) {
//		if err != nil { return err }
//		fmt.Print(chunk.Content())
//	}
func (c *Client) AnswerStream(ctx context.Context, req *AnswerRequest) iter.Seq2[AnswerStreamChunk, error] {
	if req == nil {
		return failSeq[AnswerStreamChunk](errNilRequest)
	}
	if req.OutputSchema != nil {
		return failSeq[AnswerStreamChunk](errors.New("exa: AnswerStream does not support OutputSchema; use Answer"))
	}
	r := *req
	r.Stream = true
	return stream(ctx, c, http.MethodPost, c.baseURL, "/answer", nil, &r, decodeAnswerChunk)
}
