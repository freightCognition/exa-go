// Package exa is a Go client for the Exa API (https://exa.ai/docs).
//
// Request and response types follow Exa's published OpenAPI specs, checked in
// under testdata/. Tests in this package fail when a spec endpoint or schema
// field has no Go counterpart.
package exa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const (
	// DefaultBaseURL is the Exa API root.
	DefaultBaseURL = "https://api.exa.ai"
	// DefaultAdminBaseURL is the root of Exa's team management (API key) API.
	DefaultAdminBaseURL = "https://admin-api.exa.ai/team-management"

	envAPIKey        = "EXA_API_KEY"
	defaultUserAgent = "exa-go"
)

// Client calls the Exa API. It is safe for concurrent use.
type Client struct {
	apiKey       string
	baseURL      string
	adminBaseURL string
	httpClient   *http.Client
	userAgent    string

	Agent   *AgentService
	Batches *BatchesService
	// Monitors is Search Monitors. Websets monitors are Websets.Monitors.
	Monitors *MonitorsService
	APIKeys  *APIKeysService
	Websets  *WebsetsService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key. Without it, New reads EXA_API_KEY.
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithBaseURL overrides the API root (for tests or proxies).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithAdminBaseURL overrides the team management API root.
func WithAdminBaseURL(u string) Option {
	return func(c *Client) { c.adminBaseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient sets the http.Client used for every request. It is used as
// given: its Timeout, Transport and redirect policy are kept.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// New returns a Client. It fails when no API key is given or set in EXA_API_KEY.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:      DefaultBaseURL,
		adminBaseURL: DefaultAdminBaseURL,
		httpClient:   http.DefaultClient,
		userAgent:    defaultUserAgent,
	}
	for _, o := range opts {
		o(c)
	}
	if c.apiKey == "" {
		c.apiKey = os.Getenv(envAPIKey)
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("exa: no API key: pass WithAPIKey or set %s", envAPIKey)
	}
	c.initServices()
	return c, nil
}

// Ptr returns a pointer to v, for optional request fields.
func Ptr[T any](v T) *T { return &v }

func pathEscape(s string) string { return url.PathEscape(s) }

// withHeader returns a request edit that sets one header.
func withHeader(key, value string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(key, value) }
}

// newRequest builds an authenticated request. body is JSON-encoded when non-nil.
func (c *Client) newRequest(ctx context.Context, method, base, path string, query url.Values, body any) (*http.Request, error) {
	// An empty ID leaves "/runs/" or "//items": refuse it rather than hit
	// the collection endpoint and decode a list as one resource.
	if strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return nil, fmt.Errorf("exa: empty ID in request path %q", path)
	}
	// A typed nil request (e.g. (*CreateAgentRunRequest)(nil)) sends no body,
	// not the JSON null.
	if v := reflect.ValueOf(body); body != nil && v.Kind() == reflect.Pointer && v.IsNil() {
		body = nil
	}
	u := base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("exa: encode request: %w", err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return nil, fmt.Errorf("exa: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// send performs req and returns the response when the status is 2xx;
// otherwise it reads the body into an *APIError.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		// Drop the query from the error: on presigned URLs it is the credential.
		var ue *url.Error
		if errors.As(err, &ue) {
			if u, perr := url.Parse(ue.URL); perr == nil && u.RawQuery != "" {
				u.RawQuery = ""
				ue.URL = u.String()
			}
		}
		return nil, fmt.Errorf("exa: %s %s: %w", req.Method, req.URL.Path, err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return nil, newAPIError(resp, b)
}

// do sends a JSON request and decodes a JSON response into out (when non-nil).
func (c *Client) do(ctx context.Context, method, base, path string, query url.Values, body, out any, edits ...func(*http.Request)) error {
	req, err := c.newRequest(ctx, method, base, path, query, body)
	if err != nil {
		return err
	}
	for _, edit := range edits {
		edit(req)
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("exa: decode %s response: %w", path, err)
	}
	return nil
}
