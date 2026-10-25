package exa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const webhookFixture = `{"id":"wh_1","object":"webhook","status":"active","events":["webset.item.created","webset.idle"],
	"url":"https://hooks.example/exa","secret":"whsec_123","metadata":{},
	"createdAt":"2026-10-01T00:00:00Z","updatedAt":"2026-10-01T00:00:00Z"}`

func TestWebsetWebhooks(t *testing.T) {
	c, rec := recordingClient(t, 200, webhookFixture)
	ctx := context.Background()
	wh, err := c.Websets.Webhooks.Create(ctx, &CreateWebhookParameters{
		Events:   []EventType{EventWebsetItemCreated, EventWebsetIdle},
		URL:      "https://hooks.example/exa",
		Metadata: map[string]string{"env": "prod"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/webhooks", `{"events":["webset.item.created","webset.idle"],"url":"https://hooks.example/exa","metadata":{"env":"prod"}}`)
	if wh.ID != "wh_1" || wh.Status != WebhookStatusActive || wh.Secret != "whsec_123" || len(wh.Events) != 2 || wh.Events[1] != EventWebsetIdle {
		t.Errorf("webhook = %+v", wh)
	}

	if _, err := c.Websets.Webhooks.Get(ctx, "wh_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/webhooks/wh_1", "")
	if _, err := c.Websets.Webhooks.Update(ctx, "wh_1", &UpdateWebhookParameters{URL: "https://hooks.example/v2"}); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPatch, "/websets/v0/webhooks/wh_1", `{"url":"https://hooks.example/v2"}`)
	if _, err := c.Websets.Webhooks.Delete(ctx, "wh_1"); err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodDelete, "/websets/v0/webhooks/wh_1", "")

	c2, rec2 := recordingClient(t, 200, `{"data":[`+webhookFixture+`],"hasMore":false,"nextCursor":null}`)
	if _, err := c2.Websets.Webhooks.List(ctx, &ListParams{Cursor: "c"}); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodGet, "/websets/v0/webhooks", "")
	if rec2.Query.Encode() != "cursor=c" {
		t.Errorf("query = %s", rec2.Query.Encode())
	}
}

func TestWebsetWebhookAttempts(t *testing.T) {
	c, rec := recordingClient(t, 200, `{"data":[{"id":"wha_1","object":"webhook_attempt","eventId":"ev_1","eventType":"webset.idle",
		"webhookId":"wh_1","url":"https://hooks.example/exa","successful":false,"responseHeaders":{"content-type":"text/plain"},
		"responseBody":null,"responseStatusCode":500,"attempt":2,"attemptedAt":"2026-10-01T00:00:05Z"}],"hasMore":false,"nextCursor":null}`)
	page, err := c.Websets.Webhooks.ListAttempts(context.Background(), "wh_1", &ListWebhookAttemptsParams{
		ListParams: ListParams{Limit: 10},
		EventType:  EventWebsetIdle,
		Successful: Ptr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodGet, "/websets/v0/webhooks/wh_1/attempts", "")
	if rec.Query.Encode() != "eventType=webset.idle&limit=10&successful=false" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
	a := page.Data[0]
	if a.Successful || a.ResponseStatusCode != 500 || a.Attempt != 2 || a.ResponseHeaders["content-type"] != "text/plain" || a.EventType != EventWebsetIdle {
		t.Errorf("attempt = %+v", a)
	}
}

func sign(secret string, ts int64, body string) string {
	m := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(m, "%d.%s", ts, body)
	return hex.EncodeToString(m.Sum(nil))
}

func TestVerifyWebhook(t *testing.T) {
	const secret = "whsec_test"
	body := []byte(`{"id":"ev_1","object":"event","type":"webset.idle","data":{},"createdAt":"2026-10-01T00:00:00Z"}`)
	now := time.Now().Unix()
	good := sign(secret, now, string(body))

	cases := []struct {
		name      string
		payload   []byte
		header    string
		tolerance time.Duration
		want      error
	}{
		{"valid", body, fmt.Sprintf("t=%d,v1=%s", now, good), 5 * time.Minute, nil},
		{"one of several v1", body, fmt.Sprintf("t=%d,v1=%s,v1=%s", now, sign("old", now, string(body)), good), 5 * time.Minute, nil},
		{"spaces around pairs", body, fmt.Sprintf("t=%d, v1=%s", now, good), 5 * time.Minute, nil},
		{"tampered body", append([]byte(nil), append(body, ' ')...), fmt.Sprintf("t=%d,v1=%s", now, good), 5 * time.Minute, ErrWebhookSignatureMismatch},
		{"wrong secret", body, fmt.Sprintf("t=%d,v1=%s", now, sign("other", now, string(body))), 5 * time.Minute, ErrWebhookSignatureMismatch},
		{"stale", body, fmt.Sprintf("t=%d,v1=%s", now-600, sign(secret, now-600, string(body))), 5 * time.Minute, ErrWebhookTimestampOutOfTolerance},
		{"future", body, fmt.Sprintf("t=%d,v1=%s", now+600, sign(secret, now+600, string(body))), 5 * time.Minute, ErrWebhookTimestampOutOfTolerance},
		{"stale but tolerance off", body, fmt.Sprintf("t=%d,v1=%s", now-600, sign(secret, now-600, string(body))), 0, nil},
		{"no timestamp", body, "v1=" + good, 0, ErrWebhookMalformedHeader},
		{"no signature", body, fmt.Sprintf("t=%d", now), 0, ErrWebhookMalformedHeader},
		{"bad timestamp", body, "t=abc,v1=" + good, 0, ErrWebhookMalformedHeader},
		{"empty", body, "", 0, ErrWebhookMalformedHeader},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyWebhook(tc.payload, tc.header, secret, tc.tolerance)
			if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
