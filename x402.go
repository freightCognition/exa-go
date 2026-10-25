package exa

import (
	"encoding/json"
	"net/http"
)

// X402PaymentChallenge is the body of a 402 Payment Required response when
// the request can be paid for per call with the x402 protocol.
type X402PaymentChallenge struct {
	RequestID   string       `json:"requestId"`
	Error       string       `json:"error"`
	Tag         string       `json:"tag"`
	X402Version int          `json:"x402Version"`
	Resource    X402Resource `json:"resource"`
	// Accepts lists the payment requirements the server will take, as defined
	// by the x402 protocol.
	Accepts    []map[string]any `json:"accepts"`
	Extensions map[string]any   `json:"extensions,omitempty"`
}

// X402Resource is what an x402 payment buys.
type X402Resource struct {
	URL         string `json:"url"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// PaymentChallenge decodes the x402 payment challenge of a 402 response. It
// reports false for any other error.
func (e *APIError) PaymentChallenge() (*X402PaymentChallenge, bool) {
	if e == nil || e.StatusCode != http.StatusPaymentRequired {
		return nil, false
	}
	var pc X402PaymentChallenge
	if json.Unmarshal(e.Body, &pc) != nil || pc.X402Version == 0 {
		return nil, false
	}
	return &pc, true
}
