package exa

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestAnswerSendsRequestAndDecodes(t *testing.T) {
	c, got := captureClient(t, 200, "application/json", `{
	  "requestId": "req_a",
	  "answer": "Acme builds rockets.",
	  "citations": [{"title": "Acme", "url": "https://acme.example", "publishedDate": "2025-01-01T00:00:00Z", "author": "Jo", "id": "https://acme.example", "image": "https://acme.example/i.png", "favicon": "https://acme.example/f.ico", "text": "full text"}],
	  "costDollars": {"total": 0.005}
	}`)
	resp, err := c.Answer(context.Background(), &AnswerRequest{
		Query:          "what does acme do?",
		Text:           true,
		Model:          AnswerModelExaPro,
		SystemPrompt:   "be brief",
		UserLocation:   &UserLocation{Country: "US", Latitude: Ptr(30.27), Longitude: Ptr(-97.74)},
		OutputSchema:   map[string]any{"type": "object", "properties": map[string]any{"what": map[string]any{"type": "string"}}},
		UserIdentifier: "user_1",
		Stream:         true, // Answer ignores it
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, got, http.MethodPost, "/answer")
	assertJSON(t, got.body, `{
	  "query": "what does acme do?", "text": true, "model": "exa-pro", "systemPrompt": "be brief",
	  "userLocation": {"country": "US", "latitude": 30.27, "longitude": -97.74},
	  "outputSchema": {"type": "object", "properties": {"what": {"type": "string"}}},
	  "userIdentifier": "user_1"
	}`)
	if s, ok := resp.Text(); !ok || s != "Acme builds rockets." {
		t.Errorf("Text() = %q %v", s, ok)
	}
	if resp.RequestID != "req_a" || resp.CostDollars.Total != 0.005 || len(resp.Citations) != 1 {
		t.Fatalf("resp = %+v", resp)
	}
	if ci := resp.Citations[0]; ci.Title != "Acme" || ci.Author == nil || *ci.Author != "Jo" || ci.Text != "full text" || ci.Favicon == "" || ci.PublishedDate == "" {
		t.Errorf("citation = %+v", ci)
	}
}

func TestAnswerObjectAnswer(t *testing.T) {
	c, _ := captureClient(t, 200, "application/json", `{"answer":{"what":"rockets"}}`)
	resp, err := c.Answer(context.Background(), &AnswerRequest{Query: "q"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resp.Text(); ok {
		t.Error("object answer reported as text")
	}
	if string(resp.Answer) != `{"what":"rockets"}` {
		t.Errorf("answer = %s", resp.Answer)
	}
}

func TestAnswerStream(t *testing.T) {
	c, got := captureClient(t, 200, "text/event-stream",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Acme \"},\"finish_reason\":null}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"builds rockets.\"},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: {\"citations\":[{\"title\":\"Acme\",\"url\":\"https://acme.example\"}]}\n\n"+
			"data: {\"costDollars\":{\"total\":0.005},\"requestId\":\"req_s\"}\n\n"+
			"data: [DONE]\n\n")
	req := &AnswerRequest{Query: "q", Model: AnswerModelExa}
	var text string
	var chunks []AnswerStreamChunk
	for ch, err := range c.AnswerStream(context.Background(), req) {
		if err != nil {
			t.Fatal(err)
		}
		text += ch.Content()
		chunks = append(chunks, ch)
	}
	assertRequest(t, got, http.MethodPost, "/answer")
	if got.header.Get("Accept") != "text/event-stream" {
		t.Errorf("Accept = %q", got.header.Get("Accept"))
	}
	assertJSON(t, got.body, `{"query":"q","model":"exa","stream":true}`)
	if req.Stream {
		t.Error("AnswerStream modified the caller's request")
	}
	if text != "Acme builds rockets." || len(chunks) != 4 {
		t.Fatalf("text = %q, chunks = %d", text, len(chunks))
	}
	if fr := chunks[1].Choices[0].FinishReason; fr == nil || *fr != "stop" || chunks[0].Choices[0].Delta.Role != "assistant" {
		t.Errorf("choices = %+v", chunks[:2])
	}
	if chunks[2].Citations[0].Title != "Acme" {
		t.Errorf("citations = %+v", chunks[2])
	}
	if chunks[3].CostDollars.Total != 0.005 || chunks[3].RequestID != "req_s" {
		t.Errorf("cost = %+v", chunks[3])
	}
}

func TestAnswerStreamEndsAtEOF(t *testing.T) {
	c, _ := captureClient(t, 200, "text/event-stream", "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n")
	var text string
	for ch, err := range c.AnswerStream(context.Background(), &AnswerRequest{Query: "q"}) {
		if err != nil {
			t.Fatal(err)
		}
		text += ch.Content()
	}
	if text != "hi" {
		t.Errorf("text = %q", text)
	}
}

func TestAnswerStreamErrorFrame(t *testing.T) {
	c, _ := captureClient(t, 200, "text/event-stream",
		"data: {\"tag\":\"ERROR\",\"payload\":{\"error\":{\"code\":500,\"message\":\"model overloaded\"},\"requestId\":\"req_e\"}}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"late\"}}]}\n\n")
	var n int
	var gotErr error
	for _, err := range c.AnswerStream(context.Background(), &AnswerRequest{Query: "q"}) {
		if err != nil {
			gotErr = err
			continue
		}
		n++
	}
	var se *StreamError
	if !errors.As(gotErr, &se) || se.Code != 500 || se.Message != "model overloaded" || se.RequestID != "req_e" {
		t.Fatalf("err = %v", gotErr)
	}
	if n != 0 {
		t.Errorf("got %d chunks after the error", n)
	}
}

func TestAnswerStreamRejectsOutputSchema(t *testing.T) {
	c, got := captureClient(t, 200, "text/event-stream", "")
	for _, err := range c.AnswerStream(context.Background(), &AnswerRequest{Query: "q", OutputSchema: map[string]any{"type": "object"}}) {
		if err == nil {
			t.Fatal("want error")
		}
	}
	if got.calls != 0 {
		t.Error("request sent with outputSchema")
	}
}

func TestPaymentChallenge(t *testing.T) {
	c, _ := captureClient(t, 402, "application/json", `{
	  "requestId": "req_p", "error": "Payment required", "tag": "PAYMENT_REQUIRED", "x402Version": 1,
	  "resource": {"url": "https://api.exa.ai/search", "description": "Exa search", "mimeType": "application/json"},
	  "accepts": [{"scheme": "exact", "network": "base", "maxAmountRequired": "5000"}],
	  "extensions": {"bazaar": {"discoverable": true}}
	}`)
	_, err := c.Search(context.Background(), &SearchRequest{Query: "q"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
	pc, ok := apiErr.PaymentChallenge()
	if !ok {
		t.Fatal("no payment challenge")
	}
	if pc.X402Version != 1 || pc.Tag != "PAYMENT_REQUIRED" || pc.RequestID != "req_p" || pc.Error != "Payment required" ||
		pc.Resource.MimeType != "application/json" || pc.Accepts[0]["network"] != "base" || pc.Extensions["bazaar"] == nil {
		t.Errorf("challenge = %+v", pc)
	}

	other := &APIError{StatusCode: 400, Body: []byte(`{"x402Version":1}`)}
	if _, ok := other.PaymentChallenge(); ok {
		t.Error("non-402 error reported a challenge")
	}
	plain := &APIError{StatusCode: 402, Body: []byte(`{"error":"out of credits"}`)}
	if _, ok := plain.PaymentChallenge(); ok {
		t.Error("402 without x402 fields reported a challenge")
	}
}
