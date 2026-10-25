package exa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// WebhookStatus is whether a webhook receives deliveries.
type WebhookStatus string

// Webhook statuses.
const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusInactive WebhookStatus = "inactive"
)

// Webhook delivers Websets events to a URL.
type Webhook struct {
	ID     string        `json:"id"`
	Object string        `json:"object"`
	Status WebhookStatus `json:"status"`
	Events []EventType   `json:"events"`
	URL    string        `json:"url"`
	// Secret signs deliveries (see VerifyWebhook). It is returned only by
	// Create; store it then.
	Secret    string            `json:"secret"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// CreateWebhookParameters create a webhook.
type CreateWebhookParameters struct {
	Events   []EventType       `json:"events"`
	URL      string            `json:"url"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpdateWebhookParameters change a webhook. Empty fields are left alone.
type UpdateWebhookParameters struct {
	Events   []EventType       `json:"events,omitempty"`
	URL      string            `json:"url,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WebhookAttempt is one delivery attempt of an event to a webhook.
type WebhookAttempt struct {
	ID                 string            `json:"id"`
	Object             string            `json:"object"`
	EventID            string            `json:"eventId"`
	EventType          EventType         `json:"eventType"`
	WebhookID          string            `json:"webhookId"`
	URL                string            `json:"url"`
	Successful         bool              `json:"successful"`
	ResponseHeaders    map[string]string `json:"responseHeaders"`
	ResponseBody       string            `json:"responseBody"`
	ResponseStatusCode int               `json:"responseStatusCode"`
	// Attempt is the attempt number, starting at 1.
	Attempt     int       `json:"attempt"`
	AttemptedAt time.Time `json:"attemptedAt"`
}

// ListWebhookAttemptsParams filter Webhooks.ListAttempts.
type ListWebhookAttemptsParams struct {
	ListParams
	EventType  EventType
	Successful *bool
}

// Create creates a webhook. Keep the returned Secret: later reads omit it.
func (s *WebsetWebhooksService) Create(ctx context.Context, p *CreateWebhookParameters) (*Webhook, error) {
	return websetsDo[Webhook](ctx, s.c, http.MethodPost, "/webhooks", nil, p)
}

// List returns one page of webhooks.
func (s *WebsetWebhooksService) List(ctx context.Context, p *ListParams) (*Page[Webhook], error) {
	return websetsDo[Page[Webhook]](ctx, s.c, http.MethodGet, "/webhooks", p.values(), nil)
}

// Get returns a webhook.
func (s *WebsetWebhooksService) Get(ctx context.Context, id string) (*Webhook, error) {
	return websetsDo[Webhook](ctx, s.c, http.MethodGet, "/webhooks/"+pathEscape(id), nil, nil)
}

// Update changes a webhook.
func (s *WebsetWebhooksService) Update(ctx context.Context, id string, p *UpdateWebhookParameters) (*Webhook, error) {
	return websetsDo[Webhook](ctx, s.c, http.MethodPatch, "/webhooks/"+pathEscape(id), nil, p)
}

// Delete deletes a webhook.
func (s *WebsetWebhooksService) Delete(ctx context.Context, id string) (*Webhook, error) {
	return websetsDo[Webhook](ctx, s.c, http.MethodDelete, "/webhooks/"+pathEscape(id), nil, nil)
}

// ListAttempts returns one page of a webhook's delivery attempts.
func (s *WebsetWebhooksService) ListAttempts(ctx context.Context, id string, p *ListWebhookAttemptsParams) (*Page[WebhookAttempt], error) {
	var q url.Values
	if p != nil {
		q = p.values()
		if p.EventType != "" {
			q.Set("eventType", string(p.EventType))
		}
		if p.Successful != nil {
			q.Set("successful", strconv.FormatBool(*p.Successful))
		}
	}
	return websetsDo[Page[WebhookAttempt]](ctx, s.c, http.MethodGet, "/webhooks/"+pathEscape(id)+"/attempts", q, nil)
}

// ---- signature verification ----

// Errors returned by VerifyWebhook.
var (
	// ErrWebhookMalformedHeader: the Exa-Signature header lacks a numeric
	// t= or any v1= value.
	ErrWebhookMalformedHeader = errors.New("exa: malformed Exa-Signature header")
	// ErrWebhookTimestampOutOfTolerance: t= is further from now than the
	// tolerance allows (a possible replay).
	ErrWebhookTimestampOutOfTolerance = errors.New("exa: webhook timestamp outside tolerance")
	// ErrWebhookSignatureMismatch: no v1= signature matches the payload.
	ErrWebhookSignatureMismatch = errors.New("exa: webhook signature mismatch")
)

// WebhookSignatureHeader is the header carrying a delivery's signature.
const WebhookSignatureHeader = "Exa-Signature"

// VerifyWebhook checks a webhook delivery. payload is the raw request body,
// header the Exa-Signature value ("t=<unix>,v1=<hex>[,v1=<hex>...]") and
// secret the webhook's Secret. The signature is hex HMAC-SHA256 of
// "<t>.<payload>"; any matching v1 passes. tolerance bounds how far t may be
// from now in either direction; 0 skips that check. Exa suggests 5 minutes.
func VerifyWebhook(payload []byte, header, secret string, tolerance time.Duration) error {
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || len(sigs) == 0 {
		return ErrWebhookMalformedHeader
	}
	if tolerance < 0 { // only 0 turns the replay check off
		tolerance = -tolerance
	}
	if tolerance > 0 {
		if d := time.Since(time.Unix(unix, 0)); d > tolerance || d < -tolerance {
			return ErrWebhookTimestampOutOfTolerance
		}
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(ts + "."))
	m.Write(payload)
	want := m.Sum(nil)
	for _, s := range sigs {
		if got, err := hex.DecodeString(s); err == nil && hmac.Equal(got, want) {
			return nil
		}
	}
	return ErrWebhookSignatureMismatch
}
