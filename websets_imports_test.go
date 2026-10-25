package exa

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const importFixture = `{"id":"imp_1","object":"import","status":"pending","format":"csv",
	"entity":{"type":"company"},"title":"Q4 leads","count":120,"metadata":{"src":"crm"},
	"failedReason":null,"failedAt":null,"failedMessage":null,
	"createdAt":"2026-10-01T00:00:00Z","updatedAt":"2026-10-01T00:00:00Z"`

func TestWebsetImports(t *testing.T) {
	c, rec := recordingClient(t, http.StatusCreated, importFixture+`,"uploadUrl":"https://s3.example/put?sig=1","uploadValidUntil":"2026-10-01T01:00:00Z"}`)
	ctx := context.Background()
	res, err := c.Websets.Imports.Create(ctx, &CreateImportParameters{
		Size:     2048,
		Count:    120,
		Title:    "Q4 leads",
		Format:   ImportFormatCSV,
		Entity:   Entity{Type: EntityCompany},
		CSV:      &CreateImportCSV{Identifier: Ptr(0)},
		Metadata: map[string]string{"src": "crm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.expect(t, http.MethodPost, "/websets/v0/imports",
		`{"size":2048,"count":120,"title":"Q4 leads","format":"csv","metadata":{"src":"crm"},"entity":{"type":"company"},"csv":{"identifier":0}}`)
	if res.ID != "imp_1" || res.Status != ImportStatusPending || res.Format != ImportFormatCSV || res.Count != 120 ||
		res.Entity == nil || res.UploadURL != "https://s3.example/put?sig=1" || res.UploadValidUntil != "2026-10-01T01:00:00Z" {
		t.Errorf("import = %+v", res)
	}

	c2, rec2 := recordingClient(t, 200, importFixture+`}`)
	if _, err := c2.Websets.Imports.Get(ctx, "imp_1"); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodGet, "/websets/v0/imports/imp_1", "")
	if _, err := c2.Websets.Imports.Update(ctx, "imp_1", &UpdateImport{Title: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodPatch, "/websets/v0/imports/imp_1", `{"title":"Renamed"}`)
	if _, err := c2.Websets.Imports.Delete(ctx, "imp_1"); err != nil {
		t.Fatal(err)
	}
	rec2.expect(t, http.MethodDelete, "/websets/v0/imports/imp_1", "")

	c3, rec3 := recordingClient(t, 200, `{"data":[`+importFixture+`}],"hasMore":false,"nextCursor":null}`)
	page, err := c3.Websets.Imports.List(ctx, &ListParams{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	rec3.expect(t, http.MethodGet, "/websets/v0/imports", "")
	if rec3.Query.Encode() != "limit=50" || len(page.Data) != 1 {
		t.Errorf("query = %s page = %+v", rec3.Query.Encode(), page)
	}
}

func TestImportFailedDecodes(t *testing.T) {
	c, _ := recordingClient(t, 200, `{"id":"imp_2","status":"failed","failedReason":"invalid_format","failedAt":"2026-10-01T00:00:01Z","failedMessage":"bad header"}`)
	imp, err := c.Websets.Imports.Get(context.Background(), "imp_2")
	if err != nil {
		t.Fatal(err)
	}
	if imp.Status != ImportStatusFailed || imp.FailedReason != ImportFailedReasonInvalidFormat || imp.FailedAt == nil || imp.FailedMessage != "bad header" {
		t.Errorf("import = %+v", imp)
	}
}

// Upload goes to a presigned URL: a CSV PUT that must not carry the API key.
func TestWebsetImportsUpload(t *testing.T) {
	var got struct {
		method, ct, key, body string
		length                int64
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.method, got.ct, got.key, got.body, got.length = r.Method, r.Header.Get("Content-Type"), r.Header.Get("x-api-key"), string(b), r.ContentLength
	}))
	defer srv.Close()
	c, err := New(WithAPIKey("secret-key"))
	if err != nil {
		t.Fatal(err)
	}
	csv := "name,url\nAcme,https://acme.ai\n"
	if err := c.Websets.Imports.Upload(context.Background(), srv.URL+"/put?sig=1", strings.NewReader(csv)); err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPut || got.ct != "text/csv" || got.key != "" || got.body != csv || got.length != int64(len(csv)) {
		t.Errorf("upload = %+v", got)
	}

	// A file has no Len; its size is found by seeking so the PUT isn't chunked.
	path := filepath.Join(t.TempDir(), "leads.csv")
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := c.Websets.Imports.Upload(context.Background(), srv.URL+"/put", f); err != nil {
		t.Fatal(err)
	}
	if got.length != int64(len(csv)) || got.body != csv {
		t.Errorf("file upload = %+v", got)
	}

	// Plain readers still upload.
	if err := c.Websets.Imports.Upload(context.Background(), srv.URL+"/put", io.MultiReader(bytes.NewReader([]byte(csv)))); err != nil || got.body != csv {
		t.Errorf("reader upload: err=%v body=%q", err, got.body)
	}
}

func TestWebsetImportsUploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>SignatureDoesNotMatch</Code></Error>`))
	}))
	defer srv.Close()
	c, _ := New(WithAPIKey("k"))
	err := c.Websets.Imports.Upload(context.Background(), srv.URL, strings.NewReader("a"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 403 || !strings.Contains(string(apiErr.Body), "SignatureDoesNotMatch") {
		t.Fatalf("err = %v", err)
	}
}
