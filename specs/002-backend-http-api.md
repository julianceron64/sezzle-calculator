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
| Unknown operation, or any unmatched `/api/` route | 404 | `UNKNOWN_OPERATION` / `NOT_FOUND` |
| Wrong method on a known route (+ `Allow` header) | 405 | `METHOD_NOT_ALLOWED` |
| `ErrDivisionByZero` | 422 | `DIVISION_BY_ZERO` |
| `ErrNegativeSquareRoot` | 422 | `NEGATIVE_SQUARE_ROOT` |
| `ErrUndefinedResult` | 422 | `UNDEFINED_RESULT` |
| `ErrOverflow` | 422 | `RESULT_OUT_OF_RANGE` |
| Panic / unexpected error | 500 | `INTERNAL_ERROR` (no internals in `detail`) |

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
- `-healthcheck` flag: GET `http://127.0.0.1:$PORT/healthz`, exit 0 on 200 else 1 (used by Docker).

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| API-01 | Each of the 7 operations returns `200` with the documented body for a valid request (table test). |
| API-02 | Binary response with `b: 0` includes `"b": 0`; unary response has no `b` key. |
| API-03 | Every row of the error table produces the exact status, `code`, `type`, and `Content-Type: application/problem+json`. |
| API-04 | `405` responses include `Allow: POST` (or `GET` for read routes). |
| API-05 | `GET /api/v1/operations` returns all operations with arity, matching `calculator.Operations()`. |
| API-06 | `GET /healthz` > `200 {"status":"ok"}`. |
| API-07 | API responses carry `Cache-Control: no-store` and `X-Request-ID` (echoed when valid, generated otherwise). |
| API-08 | A handler panic returns `500 INTERNAL_ERROR` problem, is logged with request_id, and the server keeps serving. |
| API-09 | Handler tests use a fake `Calculator` to prove error mapping independent of the real domain. |
| API-10 | Config: defaults, valid overrides and each invalid value are covered by table tests. |
| API-11 | `run` starts on an ephemeral port in a test, serves `/healthz`, and returns `nil` after context cancel. |
| API-12 | With `STATIC_DIR` set: existing file served; unknown non-API path > `index.html`; unknown `/api/...` > `404` problem (not `index.html`). |
| API-13 | Fuzz test `FuzzCalculateHandler` (seeded with every error-table body): never 5xx, body always valid JSON. |

## Out of scope

Auth, rate limiting, metrics endpoint, OpenAPI generation, CORS (same-origin by ADR-011).

## Docs

README > "API" (endpoints, curl examples for success + 3 error cases), "Error codes" table, "Configuration".
