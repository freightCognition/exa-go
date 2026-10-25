package exa

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Imports bring your own CSV of entities into Websets: Create returns a
// presigned upload URL, Upload sends the file to it.

// ImportStatus is an import's lifecycle state.
type ImportStatus string

// Import statuses.
const (
	ImportStatusPending    ImportStatus = "pending"
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
	ImportStatusCanceled   ImportStatus = "canceled"
)

// ImportFormat is the kind of data an import holds.
type ImportFormat string

// Import formats. Only CSV can be created.
const (
	ImportFormatCSV    ImportFormat = "csv"
	ImportFormatWebset ImportFormat = "webset"
)

// ImportFailedReason is why an import failed.
type ImportFailedReason string

// Import failure reasons.
const (
	ImportFailedReasonInvalidFormat      ImportFailedReason = "invalid_format"
	ImportFailedReasonInvalidFileContent ImportFailedReason = "invalid_file_content"
	ImportFailedReasonMissingIdentifier  ImportFailedReason = "missing_identifier"
)

// Import is a set of entities uploaded for use by websets.
type Import struct {
	ID     string       `json:"id"`
	Object string       `json:"object"`
	Status ImportStatus `json:"status"`
	Format ImportFormat `json:"format"`
	Entity *Entity      `json:"entity"`
	Title  string       `json:"title"`
	// Count is the number of records.
	Count         int                `json:"count"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
	FailedReason  ImportFailedReason `json:"failedReason"`
	FailedAt      *time.Time         `json:"failedAt"`
	FailedMessage string             `json:"failedMessage"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

// CreateImportResponse is a new import plus where to upload its file.
type CreateImportResponse struct {
	Import
	// UploadURL takes the file; pass it to Upload.
	UploadURL string `json:"uploadUrl"`
	// UploadValidUntil is when UploadURL expires (an hour after creation).
	UploadValidUntil time.Time `json:"uploadValidUntil"`
}

// CreateImportParameters create a CSV import.
type CreateImportParameters struct {
	// Size is the file size in bytes (up to 50 MB).
	Size int64 `json:"size"`
	// Count is the number of records in the file.
	Count int    `json:"count"`
	Title string `json:"title,omitempty"`
	// Format is required; set it to ImportFormatCSV.
	Format   ImportFormat      `json:"format"`
	Metadata map[string]string `json:"metadata,omitempty"`
	// Entity is the kind of entity each row describes.
	Entity Entity           `json:"entity"`
	CSV    *CreateImportCSV `json:"csv,omitempty"`
}

// CreateImportCSV are CSV-specific import options.
type CreateImportCSV struct {
	// Identifier is the zero-based column holding each row's key (URL, name,
	// ...). Inferred when nil.
	Identifier *int `json:"identifier,omitempty"`
}

// UpdateImport changes an import. Empty fields are left alone.
type UpdateImport struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Title    string            `json:"title,omitempty"`
}

// Create creates an import. Upload the file to the returned UploadURL.
func (s *WebsetImportsService) Create(ctx context.Context, p *CreateImportParameters) (*CreateImportResponse, error) {
	return websetsDo[CreateImportResponse](ctx, s.c, http.MethodPost, "/imports", nil, p)
}

// List returns one page of imports.
func (s *WebsetImportsService) List(ctx context.Context, p *ListParams) (*Page[Import], error) {
	return websetsDo[Page[Import]](ctx, s.c, http.MethodGet, "/imports", p.values(), nil)
}

// Get returns an import.
func (s *WebsetImportsService) Get(ctx context.Context, id string) (*Import, error) {
	return websetsDo[Import](ctx, s.c, http.MethodGet, "/imports/"+pathEscape(id), nil, nil)
}

// Update changes an import's title or metadata.
func (s *WebsetImportsService) Update(ctx context.Context, id string, p *UpdateImport) (*Import, error) {
	return websetsDo[Import](ctx, s.c, http.MethodPatch, "/imports/"+pathEscape(id), nil, p)
}

// Delete deletes an import.
func (s *WebsetImportsService) Delete(ctx context.Context, id string) (*Import, error) {
	return websetsDo[Import](ctx, s.c, http.MethodDelete, "/imports/"+pathEscape(id), nil, nil)
}

// Upload PUTs CSV data to an import's presigned UploadURL. The API key is
// not sent: the URL carries its own authorisation. Presigned URLs need a
// known length, so pass a *bytes.Reader, *strings.Reader, *bytes.Buffer or
// any io.Seeker such as an *os.File; other readers are sent chunked, which
// the storage service may refuse. A non-2xx answer is an *APIError.
func (s *WebsetImportsService) Upload(ctx context.Context, uploadURL string, r io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, r)
	if err != nil {
		return fmt.Errorf("exa: build upload request: %w", err)
	}
	if req.ContentLength == 0 {
		if sk, ok := r.(io.Seeker); ok {
			cur, err1 := sk.Seek(0, io.SeekCurrent)
			end, err2 := sk.Seek(0, io.SeekEnd)
			if _, err3 := sk.Seek(cur, io.SeekStart); err1 == nil && err2 == nil && err3 == nil {
				req.ContentLength = end - cur
			}
		}
	}
	req.Header.Set("Content-Type", "text/csv")
	req.Header.Set("User-Agent", s.c.userAgent)
	resp, err := s.c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
