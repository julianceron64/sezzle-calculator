# 002 — Backend: HTTP API & server

**Packages**: `internal/httpapi`, `internal/config`, `cmd/server` · **ADRs**: 002–005, 011, 013

## Contract

### `POST /api/v1/{operation}`

Request — `Content-Type: application/json`:

```json
{ "a": 10, "b": 4 }
```

Unary operations (`sqrt`) take only `a`; sending `b` is an error.

`200 OK` — `Content-Type: application/json`, `Cache-Control: no-store`:

```json
{ "operation": "divide", "a": 10, "b": 4, "result": 2.5 }
```

For unary operations `b` is absent. (Implementation note: `b` must be `*float64` with `omitempty`,
otherwise `b: 0` would disappear from binary responses.)

### `GET /api/v1/operations`

```json
{ "operations": [ { "name": "add", "arity": 2 }, { "name": "sqrt", "arity": 1 } ] }
```

### `GET /healthz` > `200 {"status":"ok"}`

### Errors — `application/problem+json` (RFC 9457)

```json
{
  "type": "/problems/division-by-zero",
  "title": "Division by zero",
  "status": 422,
  "detail": "cannot divide 10 by 0",
  "code": "DIVISION_BY_ZERO"
}
```

`type` is `/problems/` + kebab-case of `code`.

| Situation | Status | `code` |
|---|---|---|
| Content-Type not `application/json` (params like `charset` allowed) | 415 | `UNSUPPORTED_MEDIA_TYPE` |
| Body > 4 KiB | 413 | `PAYLOAD_TOO_LARGE` |
| Empty body, malformed JSON, not an object, trailing data after the object | 400 | `INVALID_JSON` |
| Unknown field (`DisallowUnknownFields`) | 400 | `UNKNOWN_FIELD` |
| Operand not a JSON number (`"2"`, `true`, `[]`) or out of float64 range (`1e400`) | 400 | `INVALID_OPERAND` |
| Required operand missing or `null` | 400 | `MISSING_OPERAND` |
| `b` sent to a unary operation | 400 | `UNEXPECTED_OPERAND` |
| `POST /api/v1/<unknown operation>` | 404 | `UNKNOWN_OPERATION` |
| Any other unmatched route under `/api/` | 404 | `NOT_FOUND` |
| Wrong method on a known route (+ `Allow` header); includes `POST /api/v1/operations` (`Allow: GET`) | 405 | `METHOD_NOT_ALLOWED` |
| `ErrDivisionByZero` | 422 | `DIVISION_BY_ZERO` |
| `ErrNegativeSquareRoot` | 422 | `NEGATIVE_SQUARE_ROOT` |
| `ErrUndefinedResult` | 422 | `UNDEFINED_RESULT` |
| `ErrOverflow` | 422 | `RESULT_OUT_OF_RANGE` |
| Panic / unexpected error | 500 | `INTERNAL_ERROR` (no internals in `detail`) |

Defensive mapping for sentinels that the HTTP layer should never let through, but the single error-mapping
function must still handle (a fake `Calculator` can return them):

| Error | Status | `code` |
|---|---|---|
| `ErrInvalidOperand` | 400 | `INVALID_OPERAND` |
| `ErrUnknownOperation` | 404 | `UNKNOWN_OPERATION` |
| `ErrArity` | 500 | `INTERNAL_ERROR` (handler guarantees arity; no internals in `detail`) |
| any other error | 500 | `INTERNAL_ERROR` |

### Routing

- The router emits **every** error body itself as `application/problem+json`, including 404 and 405.
  No `text/plain` response from `ServeMux` defaults may reach a client under `/api/`.
- `POST /api/v1/operations` is `405` with `Allow: GET`; it is never treated as an unknown operation.
- Precedence of `/healthz` and `/api/` over the SPA fallback is fixed (see Design constraints).

### Request error precedence

When a request violates several rules, the first applicable row wins:

1. `404` (unknown operation / unmatched route) and `405`
2. `415` content type
3. `413` body too large / `400 INVALID_JSON` (empty, malformed, not an object, trailing data)
4. `400 UNKNOWN_FIELD`
5. `400 INVALID_OPERAND` (wrong type or out of range) / `400 MISSING_OPERAND` (missing or `null`)
6. `400 UNEXPECTED_OPERAND` (`b` sent to a unary operation)

`{"a":1,"b":null}` on a unary operation is `UNEXPECTED_OPERAND` (a present `b`, even `null`, is unexpected).
Because `encoding/json` returns `DisallowUnknownFields` violations as untyped errors, matching that error by its
message text is acceptable **only** inside one isolated function (e.g. `isUnknownFieldError`) with its own test
that fails if a Go upgrade changes the text.

## Design constraints

- `httpapi` declares the interface it needs:
  ```go
  type Calculator interface {
      Calculate(op calculator.Op, operands ...float64) (float64, error)
      Operations() []calculator.Info
  }
  func NewHandler(calc Calculator, logger *slog.Logger) http.Handler
  ```
- One generic calculate handler; arity comes from `Operations()`. One function maps domain errors > problem.
- Request DTO uses `*float64` fields to distinguish "missing" from `0`.
- Middleware order: `RequestID > AccessLog > Recover > mux`. `X-Request-ID` is echoed if present
  (≤ 64 chars, `[A-Za-z0-9-]`), generated otherwise (`crypto/rand`, 16 hex bytes). Access log: method,
  path, status, duration, request_id. **Never log bodies.**
- `cmd/server`: `main` only calls `run(ctx, os.Args, os.Getenv, os.Stdout)`; `run` returns `error`.
  `signal.NotifyContext` for SIGINT/SIGTERM, graceful shutdown 10 s, server timeouts per ADR-013.
- `config`: `PORT` (default `8080`, must be 1–65535), `STATIC_DIR` (optional), `LOG_LEVEL` (`debug|info|warn|error`, default `info`). Invalid config > `run` returns an error, exit code 1.
- If `STATIC_DIR` is set: serve files from it; any non-`/api/` GET that doesn't match a file > `index.html`.
  Routes `/healthz` and everything under `/api/` take precedence over files and the fallback.
  Non-GET/HEAD requests to a non-`/api/` path > `405` problem+json with `Allow: GET, HEAD`.
  If `STATIC_DIR` is unset, non-`/api/` paths other than `/healthz` > `404` (plain default body is acceptable;
  the problem+json guarantee covers `/api/` only, per ADR-005).
- Header scope: `X-Request-ID` is set on **every** response (the middleware wraps the whole mux);
  `Cache-Control: no-store` only on `/api/` responses (static assets are not forced uncacheable).
- `-healthcheck` flag: GET `http://127.0.0.1:$PORT/healthz`, exit 0 on 200 else 1 (used by Docker).

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| API-01 | Each of the 7 operations returns `200` with the documented body for a valid request (table test). |
| API-02 | Binary response with `b: 0` includes `"b": 0`; unary response has no `b` key. |
| API-03 | Every row of the error table produces the exact status, `code`, `type`, and `Content-Type: application/problem+json`. |
| API-04 | `405` responses include `Allow: POST` (or `GET` for read routes). `POST /api/v1/operations` > `405` + `Allow: GET`; `GET /api/v1/add` > `405` + `Allow: POST`; neither is `404`. All 404/405 bodies are `application/problem+json` (never `text/plain`). |
| API-05 | `GET /api/v1/operations` returns all operations with arity, matching `calculator.Operations()`. |
| API-06 | `GET /healthz` > `200 {"status":"ok"}`. |
| API-07 | API responses carry `Cache-Control: no-store` and `X-Request-ID` (echoed when valid, generated otherwise). |
| API-08 | A handler panic returns `500 INTERNAL_ERROR` problem, is logged with request_id, and the server keeps serving. |
| API-09 | Handler tests use a fake `Calculator` to prove error mapping independent of the real domain, including the defensive mapping table (`ErrInvalidOperand`, `ErrUnknownOperation`, `ErrArity`, unknown error) and that `detail` of a 500 never contains the underlying error text. |
| API-10 | Config: defaults, valid overrides and each invalid value are covered by table tests. |
| API-11 | `run` starts in a test on a free port, serves `/healthz`, and returns `nil` after context cancel. The test obtains the port by listening on `:0`, reading the port and closing the listener, then passes it as `PORT`; `PORT=0` stays invalid in config (API-10). Readiness uses event-driven polling with a timeout (no fixed sleep). |
| API-12 | With `STATIC_DIR` set: existing file served; unknown non-API path > `index.html`; unknown `/api/...` > `404` problem (not `index.html`); `/healthz` is not shadowed by the fallback; `POST` to a non-API path > `405` problem with `Allow: GET, HEAD`. |
| API-13 | Fuzz test `FuzzCalculateHandler` (seeded with every error-table body): never 5xx, body always valid JSON. |
| API-14 | Precedence: a table test sends requests violating several rules at once (e.g. wrong content type + malformed body; unknown field + wrong operand type; `b: null` on `sqrt`) and asserts the winner per the "Request error precedence" list. The unknown-field detector has its own test. |

## Out of scope

Auth, rate limiting, metrics endpoint, OpenAPI generation, CORS (same-origin by ADR-011).

## Docs

README > "API" (endpoints, curl examples for success + 3 error cases), "Error codes" table, "Configuration".
