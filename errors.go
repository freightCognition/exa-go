package exa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned for any non-2xx response.
type APIError struct {
	StatusCode int
	// Message is the API's error message, or the raw body when it isn't JSON.
	Message string
	// Tag is Exa's machine-readable error code, e.g. INVALID_REQUEST_BODY or,
	// on the Agent API, RUN_NOT_FOUND.
	Tag string
	// Type is the Agent API's error class, e.g. NOT_FOUND. Empty elsewhere.
	Type      string
	RequestID string
	// Body is the raw response body (up to 1 MiB). For 402 it holds the x402
	// payment challenge.
	Body []byte
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "exa: %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	if e.Tag != "" {
		fmt.Fprintf(&b, " [%s]", e.Tag)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " (request %s)", e.RequestID)
	}
	return b.String()
}

func newAPIError(resp *http.Response, body []byte) *APIError {
	e := &APIError{StatusCode: resp.StatusCode, Body: body}
	var raw struct {
		Error     json.RawMessage `json:"error"`
		Tag       string          `json:"tag"`
		RequestID string          `json:"requestId"`
	}
	if json.Unmarshal(body, &raw) == nil {
		e.Tag, e.RequestID = raw.Tag, raw.RequestID
		var msg string
		var agent AgentError
		switch {
		case json.Unmarshal(raw.Error, &msg) == nil:
			e.Message = msg
		case json.Unmarshal(raw.Error, &agent) == nil && (agent.Message != "" || agent.Code != ""):
			e.Message, e.Tag, e.Type = agent.Message, agent.Code, agent.Type
		}
	}
	if e.Message == "" && e.Tag == "" {
		e.Message = strings.TrimSpace(string(body))
	}
	if e.RequestID == "" {
		e.RequestID = resp.Header.Get("x-request-id")
	}
	return e
}

// ErrorResponse is the JSON error body most Exa endpoints return.
type ErrorResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Error     string `json:"error,omitempty"`
	Tag       string `json:"tag,omitempty"`
}

// AgentErrorResponse is the error body of the Agent API.
type AgentErrorResponse struct {
	Error AgentError `json:"error"`
}

// AgentError is the nested error object of the Agent API.
type AgentError struct {
	// Type is one of INVALID_REQUEST, AUTHENTICATION_ERROR, RATE_LIMIT_ERROR,
	// NOT_FOUND, SERVER_ERROR.
	Type string `json:"type"`
	// Code is e.g. RUN_NOT_FOUND, CONCURRENCY_LIMIT_REACHED, INVALID_OUTPUT_SCHEMA.
	Code    string `json:"code"`
	Message string `json:"message"`
}
