package exa

import (
	"encoding/json"
	"fmt"
	"iter"
	"strings"
)

// SearchStreamChunk is one event of SearchStream. Type says which fields
// are set:
//
//   - text-delta: Delta (the next piece of output text), Choices
//   - grounding: Grounding, Citations, Choices
//   - results: Results, Warnings, Notices
//   - stream-reset: StreamReset; discard the text received so far
//   - done: Output, SearchTime, CostDollars, Choices
type SearchStreamChunk struct {
	RequestID string                `json:"requestId,omitempty"`
	Type      SearchStreamChunkType `json:"type"`

	Delta       string                 `json:"delta,omitempty"`
	Choices     []StreamChoice         `json:"choices,omitempty"`
	Grounding   []Grounding            `json:"grounding,omitempty"`
	Citations   []StreamCitation       `json:"citations,omitempty"`
	Results     []SearchResult         `json:"results,omitempty"`
	Warnings    []Warning              `json:"warnings,omitempty"`
	Notices     []Notice               `json:"notices,omitempty"`
	StreamReset bool                   `json:"streamReset,omitempty"`
	Output      *SearchSynthesisOutput `json:"output,omitempty"`
	// SearchTime is in milliseconds.
	SearchTime  float64      `json:"searchTime,omitempty"`
	CostDollars *CostDollars `json:"costDollars,omitempty"`
}

// SearchStreamChunkType is the kind of a SearchStreamChunk.
type SearchStreamChunkType string

const (
	SearchStreamChunkTextDelta SearchStreamChunkType = "text-delta"
	SearchStreamChunkGrounding SearchStreamChunkType = "grounding"
	SearchStreamChunkResults   SearchStreamChunkType = "results"
	SearchStreamChunkReset     SearchStreamChunkType = "stream-reset"
	SearchStreamChunkDone      SearchStreamChunkType = "done"
)

// StreamChoice is the OpenAI-style choice carried by search stream chunks.
type StreamChoice struct {
	Index int         `json:"index"`
	Delta StreamDelta `json:"delta"`
	// FinishReason is "stop" on the last choice, nil before.
	FinishReason *string `json:"finish_reason"`
}

// StreamDelta is the new content of a StreamChoice.
type StreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	Citations []StreamCitation `json:"citations,omitempty"`
}

// StreamCitation is a page cited while streaming.
type StreamCitation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	ID    string `json:"id"`
}

// AnswerStreamChunk is one event of AnswerStream: text (Choices, see
// Content), then Citations, then CostDollars and RequestID.
type AnswerStreamChunk struct {
	Choices     []AnswerStreamChoice `json:"choices,omitempty"`
	Citations   []AnswerCitation     `json:"citations,omitempty"`
	CostDollars *CostDollars         `json:"costDollars,omitempty"`
	RequestID   string               `json:"requestId,omitempty"`
}

// Content returns the text this chunk adds to the answer.
func (c AnswerStreamChunk) Content() string {
	var b strings.Builder
	for _, ch := range c.Choices {
		b.WriteString(ch.Delta.Content)
	}
	return b.String()
}

// AnswerStreamChoice is the OpenAI-style choice carried by answer stream chunks.
type AnswerStreamChoice struct {
	Index        int               `json:"index"`
	Delta        AnswerStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason,omitempty"`
}

// AnswerStreamDelta is the new content of an AnswerStreamChoice.
type AnswerStreamDelta struct {
	Role    string  `json:"role,omitempty"`
	Content string  `json:"content,omitempty"`
	Refusal *string `json:"refusal,omitempty"`
}

// StreamError is an error event sent inside a stream, after the HTTP
// response had already started with 200.
type StreamError struct {
	// Code is the error code the answer stream sends; 0 for search.
	Code      int
	Message   string
	RequestID string
}

func (e *StreamError) Error() string {
	s := "exa: stream error"
	if e.Code != 0 {
		s += fmt.Sprintf(" %d", e.Code)
	}
	s += ": " + e.Message
	if e.RequestID != "" {
		s += " (request " + e.RequestID + ")"
	}
	return s
}

// decodeSearchChunk decodes one /search stream event; an "error" event
// becomes a *StreamError.
func decodeSearchChunk(ev sseEvent) (SearchStreamChunk, bool, error) {
	var f struct {
		SearchStreamChunk
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(ev.Data), &f); err != nil {
		return SearchStreamChunk{}, false, fmt.Errorf("exa: decode search stream event: %w", err)
	}
	if f.Type == "error" {
		se := &StreamError{RequestID: f.RequestID}
		if f.Error != nil {
			se.Message = f.Error.Message
		}
		return SearchStreamChunk{}, false, se
	}
	return f.SearchStreamChunk, true, nil
}

// decodeAnswerChunk decodes one /answer stream event. These carry no type
// field; an error is {"tag":"ERROR","payload":{...}}.
func decodeAnswerChunk(ev sseEvent) (AnswerStreamChunk, bool, error) {
	var f struct {
		AnswerStreamChunk
		Tag     string `json:"tag"`
		Payload struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			RequestID string `json:"requestId"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(ev.Data), &f); err != nil {
		return AnswerStreamChunk{}, false, fmt.Errorf("exa: decode answer stream event: %w", err)
	}
	if f.Tag == "ERROR" {
		return AnswerStreamChunk{}, false, &StreamError{Code: f.Payload.Error.Code, Message: f.Payload.Error.Message, RequestID: f.Payload.RequestID}
	}
	return f.AnswerStreamChunk, true, nil
}

// failSeq is a stream that yields only err.
func failSeq[T any](err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		yield(zero, err)
	}
}
