# exa-go

A Go client for the [Exa API](https://exa.ai/docs). Standard library only, Go 1.23+.

```bash
go get github.com/freightCognition/exa-go
```

```go
c, err := exa.New() // reads EXA_API_KEY; or exa.WithAPIKey("...")
if err != nil {
	log.Fatal(err)
}
resp, err := c.Search(ctx, &exa.SearchRequest{
	Query:      "freight brokers in the Midwest",
	NumResults: exa.Ptr(10),
	Contents:   &exa.ContentsOptions{Highlights: &exa.HighlightsOptions{}}, // empty options = true
})
```

## Coverage

Every operation in Exa's published OpenAPI specs (`testdata/`) has a method:

| Area | Methods |
|------|---------|
| Search | `Search`, `SearchStream`, `Contents`, `Answer`, `AnswerStream`, `FindSimilar` (deprecated by Exa) |
| Agent | `Agent.Create`, `CreateStream`, `List`, `Get`, `Delete`, `Cancel`, `Stop`, `Events`, `StreamEvents` |
| Batches | `Batches.Create`, `List`, `Get`, `Delete`, `Cancel` |
| Search Monitors | `Monitors.Create`, `List`, `Get`, `Update`, `Delete`, `Batch`, `Trigger`, `ListRuns`, `GetRun` |
| Websets | `Websets.Create`, `List`, `Get`, `Update`, `Delete`, `Cancel`, `Preview`, `TeamInfo`, plus `Searches`, `Enrichments`, `Items`, `Monitors`, `Imports` (with CSV `Upload`), `Webhooks`, `Events` |
| Team management | `APIKeys.Create`, `List`, `Get`, `Update`, `Delete`, `Usage` |

`VerifyWebhook` checks the `Exa-Signature` header on Websets webhooks.

## Conventions

- **Optional fields** are pointers or omitted when empty. `exa.Ptr(v)` builds a pointer.
- **"true or object" options** (`Text`, `Highlights`, `Summary`, `Context`): a pointer to an empty struct sends `true`; a nil pointer omits it.
- **Errors**: any non-2xx is an `*exa.APIError` with `StatusCode`, `Tag`, `Message`, `RequestID` and the raw `Body`. On a 402, `PaymentChallenge()` decodes the x402 challenge.
- **Streams** return `iter.Seq2[Chunk, error]`; the response closes when you stop ranging.
- **Pagination**: list methods return `*exa.Page[T]`. `exa.All` walks every page:

  ```go
  for run, err := range exa.All(ctx, func(ctx context.Context, cursor string) (*exa.Page[exa.AgentRun], error) {
  	return c.Agent.List(ctx, &exa.ListParams{Cursor: cursor})
  }) { ... }
  ```

- **Your `http.Client` is used as given** (`exa.WithHTTPClient`), timeout included.

## Staying in sync with Exa

`testdata/` holds Exa's OpenAPI specs. The spec coverage tests fail when an operation has no method, or a schema field has no Go field (or the other way round). To pick up API changes:

```bash
curl -sSL https://exa.ai/docs/exa-spec.yaml -o testdata/exa-spec.yaml
npx -y js-yaml testdata/exa-spec.yaml > testdata/exa-spec.json
go test ./...
```

Live tests call the real API and decode its responses strictly. Without a key they check only the error replies (free):

```bash
EXA_LIVE=1 go test -run Live -v
EXA_LIVE=1 EXA_API_KEY=... go test -run Live -v   # spends a few cents
```
