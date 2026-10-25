# AGENTS.md

Guidance for coding agents working in this repo. README.md is the user-facing doc; this file is how to change the code without breaking it.

## What this is

A Go client for the Exa API (package `exa`, module `github.com/freightCognition/exa-go`). One package at the repo root, standard library only, Go 1.23 (it uses `iter.Seq2` and range-over-func). Do not add dependencies.

The contract is Exa's OpenAPI specs, checked in under `testdata/`:

- `exa-spec.yaml` / `exa-spec.json`: the main API (`https://api.exa.ai`)
- `team-management-spec.yaml` / `.json`: API key management (`https://admin-api.exa.ai/team-management`)

The tests read the `.json` copies. When you update a spec, regenerate the JSON from the YAML (README has the commands). Never hand-edit either.

## Commands

```bash
gofmt -l .                                                  # must print nothing
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...      # an older installed staticcheck can't read Go 1.27 export data
go test -race ./...
EXA_LIVE=1 go test -run Live -v                             # free: error replies only
EXA_LIVE=1 EXA_API_KEY=... go test -run Live -v             # spends a few cents
```

CI (`.github/workflows/ci.yml`) runs the first four.

## Layout

| File | Owns |
|------|------|
| `client.go` | `Client`, options, `newRequest`, `send`, `do`, `Ptr`, `pathEscape`, `withHeader` |
| `errors.go` | `APIError` (both error shapes: `{error, tag, requestId}` and the Agent API's `{error:{type, code, message}}`) |
| `sse.go` | the SSE reader and the generic `stream[T]` |
| `pagination.go` | `Page[T]`, `ListParams`, `All` |
| `contents_options.go` | shared `ContentsOptions`, the true/false/object option types, `OutputSchema`, `CostDollars`, `Strings` |
| `services.go` | every service struct and their wiring in `initServices` |
| `search.go`, `contents.go`, `answer.go`, `findsimilar.go`, `stream_chunks.go`, `entities.go`, `x402.go` | Search area (methods on `*Client`) |
| `agent.go`, `batches.go`, `monitors.go`, `apikeys.go` | Agent, Batches, Search Monitors, team management |
| `websets*.go`, `webhooks.go`, `events.go` | Websets (`c.Websets.*`), webhook verification, events |
| `spec_coverage_test.go` | the spec coverage harness (see below) |
| `live_test.go` | opt-in tests against the real API |

## How to add or change an endpoint

1. **Find it in the spec.** Use `jq` on `testdata/exa-spec.json`. Check for a per-path `servers` override.
2. **Write the failing test first.** Use `newTestClient` (or the area's own helper: `recordingClient` in websets, `newAgentCaptureClient` in agent). Assert the method, path, query, headers and the exact JSON body. Decode a realistic response fixture.
3. **Implement it with the core helpers.** Use `c.do(ctx, method, base, path, query, body, &out, edits...)`, or `stream[T]` for SSE. Escape every path ID with `pathEscape`. Websets calls go through `websetsDo`.
4. **Register it for coverage** in the area's `*_spec_test.go` `init()`:
   - `registerEndpoint(method, specPath, call)`. Use `registerAdminEndpoint` for team management.
   - `registerSchema("SpecSchemaName", GoType{})` for every component schema it uses.
   - `exemptSchema(name, reason)`, only when a schema truly has no Go type of its own.
5. **Run the verification commands above.**

## The spec coverage tests

These two tests are the reason this SDK exists instead of the community one.

- **`TestSpecEndpointsCovered`** calls every registered endpoint against a recording server. It checks the method and path actually sent, then fails for any spec operation with no registration.
- **`TestSpecSchemasCovered`** compares JSON field names in both directions: spec properties missing from Go, and Go fields the spec doesn't define. It recurses into nested objects, and it fails for any object schema that isn't registered or exempted.

Rules that keep them honest:

- **Don't weaken the harness to get green.** Fix the type instead.
- **Use `registerSchema`'s skip list rarely.** It applies only at the top level of that schema, not to nested comparisons.
- **A type with its own `UnmarshalJSON` is skipped by the comparison** unless it embeds `toggle`, which carries the `acceptsBool()` marker. If you add a custom decoder, cover the fields another way (e.g. a methodless mirror type, as `entities.go` does with `entityWire`).

## Conventions

- **Optional fields.**
  - Use `omitempty`.
  - Use a pointer when the zero value means something (`numResults: 0`, `dryRun: false`).
  - Callers build pointers with `exa.Ptr`.
- **True-or-object options.** `TextOptions`, `HighlightsOptions`, `SummaryOptions`, `ContextOptions` and `SummaryWithMaxTokensOptions` embed `toggle`.
  - A pointer to an empty struct sends `true`; nil omits the field.
  - Decoding accepts `true`, `false` or an object, and re-encodes the value unchanged.
- **Never give `ContentsOptions` a `MarshalJSON` or `UnmarshalJSON`.** `ContentsRequest` embeds it to flatten the fields, and a method on it would be promoted and break that.
- **Sending `null` to clear a field** takes an explicit opt-in. Use `RemoveBudget` on `APIKeyRequest`, the `Clear*` fields on `UpdateSearchMonitorParameters`, or `*map[string]string` metadata on Websets updates.
- **Deprecated spec fields stay, marked `// Deprecated:`.** Exa still serves them, and the schema test expects them.
- **Open enums are string types with constants.** Unknown values must still decode.
- **Never mutate the caller's request.** Copy it before setting `Stream` or similar.
- **Godoc every exported identifier.** Match the surrounding comment density.

## Exa API facts that aren't obvious from the spec

- **Websets are served under `/websets/v0/...`.**
  - The spec sets this per path.
  - `/v0/teams/me` lacks the override, but is also served at `/websets/v0/teams/me` (`/v0/teams/me` is a 404). This was checked live.
- **A request with no API key gets `402 X402_PAYMENT_REQUIRED`** with an x402 challenge (see `APIError.PaymentChallenge`). A bad key gets `401 INVALID_API_KEY`.
- **Every Batches call needs `Exa-Beta: batches-2026-06-06`.**
- **Agent SSE `data:` is the payload only, not the `AgentRunEvent` envelope.** The envelope is built from the frame's `id` and `event`. A failed run's error is only in the event (`AgentRunEvent.Failure`).
- **`/search` streams only when `stream: true` and `outputSchema` are both set.** `/answer` stream frames have no `type` field.
- **Search Monitor webhooks reuse the `monitor.*` event names** with different payloads. The Websets `Event` accessors check `data.object` for that reason.
- **`/findSimilar` is deprecated in the spec but still served.**

## Working rules

- **Commit on a branch, never directly on `main`.** End commit messages with the co-author line your tooling requires.
- **When parallel agents build different areas, give each its own git worktree.** They share one Go package, so a half-written file in a shared checkout breaks everyone's build. Only one owner edits each core file listed under Layout.
- **For a bug fix, add a regression test that fails without the fix.** `review_regressions_test.go` is the example.
