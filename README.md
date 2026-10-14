# Sezzle Calculator

Full-stack calculator: **React + TypeScript** frontend consuming a **Go** REST microservice.

> Sections marked `TODO(spec-00X)` are completed as each spec is implemented.

## Quick start (Docker)

```bash
make docker-build && make docker-run   # UI + API on http://localhost:8080
make smoke                             # end-to-end smoke test with curl
```

## Local development

Prerequisites: Go ≥ 1.24, Node ≥ 22, make.

```bash
make dev-backend    # API on :8080
make dev-frontend   # UI on :5173 (proxies /api → :8080)
make check          # lint + typecheck + tests + coverage gates
```

## API

| Method & path | Description |
|---|---|
| `POST /api/v1/{operation}` | Run `add`, `subtract`, `multiply`, `divide`, `power`, `sqrt` or `percentage` |
| `GET /api/v1/operations` | List operations and their arity |
| `GET /healthz` | Liveness: `{"status":"ok"}` |

Requests are `Content-Type: application/json` with named operands: `{"a": 10, "b": 4}`. Unary operations (`sqrt`)
take only `a`; sending `b` (even `null`) is an error. Bodies are limited to 4 KiB and unknown fields are rejected.
Every response carries `X-Request-ID` (an incoming valid id, `[A-Za-z0-9-]{1,64}`, is echoed; otherwise one is
generated); responses under `/api/` also carry `Cache-Control: no-store`.

```bash
# success
curl -s -X POST localhost:8080/api/v1/divide -H 'Content-Type: application/json' -d '{"a":10,"b":4}'
# {"operation":"divide","a":10,"b":4,"result":2.5}

# 422: well-formed input that cannot be computed
curl -s -X POST localhost:8080/api/v1/divide -H 'Content-Type: application/json' -d '{"a":10,"b":0}'
# {"type":"/problems/division-by-zero","title":"Division by zero","status":422,"detail":"The operation would divide by zero","code":"DIVISION_BY_ZERO"}

# 400: b sent to a unary operation
curl -s -X POST localhost:8080/api/v1/sqrt -H 'Content-Type: application/json' -d '{"a":4,"b":1}'
# {"type":"/problems/unexpected-operand","title":"Unexpected operand","status":400,"detail":"This operation does not take operand b","code":"UNEXPECTED_OPERAND"}

# 405: wrong method (response has Allow: POST)
curl -si localhost:8080/api/v1/add
```

Binary responses always include `b` (also when it is `0`); unary responses have no `b` key.

### Configuration

Environment variables only:

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | Listen port, integer in 1–65535 (empty = default) |
| `STATIC_DIR` | unset | If set, serves the built SPA from this directory; any other non-`/api/` GET/HEAD returns `index.html` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |

Invalid values make the server exit with status 1. `server -healthcheck` does `GET http://127.0.0.1:$PORT/healthz`
and exits 0 on `200` (used by the Docker `HEALTHCHECK`). The server shuts down gracefully (10 s) on SIGINT/SIGTERM.
Logs are JSON (`slog`): one access line per request (method, path, status, duration, request_id); bodies are never logged.

## Error codes

Every error under `/api/` is `application/problem+json` (RFC 9457) with `type`, `title`, `status`, `detail` and a stable
`code`; `type` is `/problems/` + the kebab-case of `code`. Clients must branch on `code`, never on `detail` text.

| Code | Status | Meaning |
|---|---|---|
| `UNSUPPORTED_MEDIA_TYPE` | 415 | `Content-Type` is not `application/json` (parameters such as `charset` are fine) |
| `PAYLOAD_TOO_LARGE` | 413 | Body larger than 4 KiB |
| `INVALID_JSON` | 400 | Empty, malformed, not an object, or trailing data after the object |
| `UNKNOWN_FIELD` | 400 | A field other than `a` or `b` |
| `INVALID_OPERAND` | 400 | Operand is not a JSON number, or is outside the float64 range (`1e400`) |
| `MISSING_OPERAND` | 400 | A required operand is missing or `null` |
| `UNEXPECTED_OPERAND` | 400 | `b` sent to a unary operation |
| `UNKNOWN_OPERATION` | 404 | `POST /api/v1/<unknown>` |
| `NOT_FOUND` | 404 | Any other unmatched route under `/api/` |
| `METHOD_NOT_ALLOWED` | 405 | Wrong method on a known route; the `Allow` header lists the right one |
| `DIVISION_BY_ZERO` | 422 | `divide` by 0 (incl. 0/0), or `0^negative` |
| `NEGATIVE_SQUARE_ROOT` | 422 | `sqrt` of a negative number |
| `UNDEFINED_RESULT` | 422 | Result is not a real number (e.g. `(-8)^(1/3)`) |
| `RESULT_OUT_OF_RANGE` | 422 | Result (or an intermediate value) overflows float64 |
| `INTERNAL_ERROR` | 500 | Unexpected failure or panic; `detail` never contains internals |

When a request breaks several rules, the first applies: 404/405 → 415 → 413 / `INVALID_JSON` → `UNKNOWN_FIELD` →
`INVALID_OPERAND` / `MISSING_OPERAND` (operand `a` is checked before `b`) → `UNEXPECTED_OPERAND`.

Non-canonical paths under `/api` (`/api`, `/api//v1/add`, `/api/v1/../v1/add`) are `404 NOT_FOUND` problems, never redirects.

Known limit: a request whose target is not valid HTTP, such as `GET /api/v1/%zz` (bad percent-encoding), is rejected
by Go's `net/http` itself with a `400` `text/plain` response before any handler runs, so it cannot be a problem+json.

## Operation semantics

| Operation | Arity | Semantics | Domain errors |
|---|---|---|---|
| `add` | 2 | a + b | overflow |
| `subtract` | 2 | a − b | overflow |
| `multiply` | 2 | a × b | overflow |
| `divide` | 2 | a ÷ b | b = 0 (incl. 0/0, −0) |
| `power` | 2 | aᵇ (`0⁰ = 1`, as `math.Pow`) | `0^negative` → division by zero; NaN (e.g. `(−8)^(1/3)`) → undefined; overflow |
| `sqrt` | 1 | √a | a < 0 (no complex numbers) |
| `percentage` | 2 | "a percent of b" = a × b ÷ 100 | overflow |

- `percentage` is computed literally as `a*b/100`. An intermediate overflow returns `ErrOverflow` even when the true
  result is representable (e.g. `percentage(1e308, 10)`).
- Operands and results are `float64`; operands must be finite, results that are `±Inf` are overflow, `NaN` is
  undefined, and `-0` is returned as `0`.
- Validation order in `Calculate`: unknown operation → arity → finite operands → compute → result check.
  `0^negative` is detected as division by zero before an `Inf` result can become overflow.

## Frontend

React 19 + TypeScript (strict) + Vite, CSS Modules, no UI kit. Runtime dependencies are `react` and `react-dom` only.

```bash
make dev-backend    # API on :8080
make dev-frontend   # UI on :5173, proxies /api to :8080
cd frontend && npm test          # Vitest watch mode
cd frontend && npm run coverage  # tests + coverage gates
```

`VITE_API_BASE_URL` overrides the API origin; unset means same-origin (the production setup, no CORS).

### UX model

A pocket calculator with **immediate execution**: binary operators run left to right with no precedence, so
`2 + 3 × 4 =` is `20`, and each step is one API call. The frontend never computes a result: it only parses typed
digits into request operands, and every number on screen after an operation comes from the backend.

- `√` applies to the current value immediately (`2 + √9 =` shows `5`); `%` is "a percent of b" (`20 % 150 =` is `30`).
- An operator right after an operator replaces it. `=` with no new operand does nothing (there is no "repeat last
  operation"), and `C` clears and cancels an in-flight request.
- While a request is pending the keys are `aria-disabled` (not `disabled`, so focus stays) and ignore presses; only `C`
  stays active. Errors show `Error` on the display and a message below it, chosen by the backend's error `code`.
- Results are displayed with 12 significant digits (`0.1 + 0.2` shows `0.3`) and switch to exponential notation at
  `1e12` and above or below `1e-9`. The full-precision value is what is sent in the next request.
- Input is limited to 15 digits, and every digit counts, including the leading `0` of `0.5`. So a number typed as
  `0.xxx` holds at most 14 fractional digits, while `1.xxx` holds 14 as well and `123.xxx` holds 12. The sign and the
  decimal point are not counted. The frontend validates input format only; rules such as division by zero come from
  the backend.

**Assumption:** `±` on a result turns it into editable input holding the *12-digit display string*, so
`0.30000000000000004 ±` becomes `-0.3`. This is lossy by design (exponential displays are left unchanged, because an
exponential string is not valid input). `±` on a zero or on a number right after an operator does nothing.

### Keyboard

| Key | Action |
|---|---|
| `0`–`9` | digit |
| `.` or `,` | decimal point |
| `+` `-` `*` `/` | add, subtract, multiply, divide |
| `^` | power |
| `%` | percent of |
| `r` | square root |
| `Enter` or `=` | equals |
| `Backspace` | delete last digit |
| `Escape` or `Delete` | clear |

Keys are ignored while Ctrl, Alt or Meta is held (Shift is allowed, since `+`, `*`, `^` and `%` need it on many
layouts). Handled keys call `preventDefault()`, so `Enter` on a focused button does not also click it.

### Test environment notes

- **jsdom + MSW + abort signals (verified):** `AbortSignal.any`, `AbortSignal.timeout` and MSW's `fetch` interception all
  work under jsdom with the pinned versions (Vitest 5, jsdom 29, MSW 3, Node 22). `src/test/setup.ts` therefore does
  not restore Node's `AbortController`/`AbortSignal`, which ADR-008 allowed only if they had broken MSW.
- Node's `fetch` cannot resolve relative URLs, so tests set `VITE_API_BASE_URL=http://api.test` (`test.env` in
  `vite.config.ts`) and MSW intercepts that origin. The same-origin default is covered by its own test.
- MSW 3 renamed the unhandled-request option to `onUnhandledFrame`; the setup uses `'error'`, so any request without a
  handler fails the test.
- The real client rejects as soon as the hook aborts, so a late response can never reach the hook through it. The
  reducer's request-id guard is therefore proven in `Calculator.stale.test.tsx`, which swaps in a fake `calculate`
  that ignores the abort signal.

## Testing & coverage

<!-- TODO(spec-004): how to run, gates, link to docs/COVERAGE.md -->

## Design decisions

Full rationale in [`docs/DECISIONS.md`](docs/DECISIONS.md). Highlights:

- **Stdlib-only Go** with 1.22+ routing; pure domain package behind a consumer-defined interface.
- **One endpoint per operation** (`POST /api/v1/{operation}`), errors as **RFC 9457 problem+json** with stable codes.
- **float64 end to end**, non-finite results rejected, display formatting owned by the frontend.
- **Pocket-calculator UX** driven by a pure reducer; effects as data, stale responses ignored, requests abortable.
- **Single container**: Go serves API + SPA, same origin, no CORS.

## Assumptions

<!-- TODO(spec-005) -->

## How this was built (AI usage)

Spec-driven development with Claude Code: I wrote the specs (`specs/`) and ADRs (`docs/DECISIONS.md`);
Claude Code implemented them one spec at a time under the rules in `CLAUDE.md`.
Every prompt is in [`prompts/`](prompts/).

## What I'd do next

<!-- TODO(spec-005) -->
