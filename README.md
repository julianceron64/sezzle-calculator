# Sezzle Calculator

Full-stack calculator: a **React + TypeScript** frontend consuming a **Go** REST microservice. Seven operations
(`add`, `subtract`, `multiply`, `divide`, `power`, `sqrt`, `percentage`), a pocket-calculator UI with full keyboard
support, and one small container that serves both.

```text
Browser ── React SPA ──────────────── POST /api/v1/{operation} (JSON) ──▶ Go server (net/http, stdlib only)
           pure reducer, never                                           ├─ httpapi     transport, problem+json errors
           computes a result                                             ├─ calculator  pure arithmetic domain
                                                                         └─ serves the built SPA (same origin, no CORS)
```

The backend is the single source of truth for every result; the frontend only formats input and displays what the API
returns.

```text
backend/    Go module: cmd/server (composition root), internal/{calculator,httpapi,config}
frontend/   Vite + React + TypeScript: src/api, src/features/calculator, src/lib
specs/      numbered specs with acceptance criteria       docs/DECISIONS.md   architecture decisions (ADRs)
scripts/    coverage gate, no-sleeps check, smoke test    prompts/            prompts used to build this
```

## Get the code

```bash
git clone https://github.com/julianceron64/sezzle-calculator.git
cd sezzle-calculator
```

Next: if you only want to **run** it, follow [Quick start (Docker)](#quick-start-docker); if you want to **work on**
it, follow [Local development](#local-development).

## Quick start (Docker)

Prerequisites:

- **Docker, with the daemon running** (Docker Desktop on Windows and macOS). Tested with Docker 29.8.2.
- **make**, for the shortcuts below (GNU Make 4.4.1). No make? See [Without make](#without-make).
- **bash and curl**, for the smoke test, `scripts/smoke.sh` (GNU bash 5.2.37, curl 8.15.0). Git Bash on Windows
  provides both.

These versions are the ones this was tested with, on Windows 11 with Git Bash; Linux and macOS were not tried.

`make docker-run` runs in the foreground, so use two terminals:

```bash
# terminal 1: build the image and run it (UI + API on http://localhost:8080)
make docker-build && make docker-run

# terminal 2: end-to-end smoke test with curl (11 checks)
make smoke
```

Detached equivalent, in one terminal:

```bash
make docker-build
docker run -d --rm -p 8080:8080 --name sezzle-calculator sezzle-calculator:local
make smoke
docker stop sezzle-calculator
```

Open <http://localhost:8080> for the calculator. The image is about 16 MB on disk (3.7 MB compressed), runs as a
non-root user, and `docker ps` shows it as `healthy` after the first health check (the interval is 10 s). The
captured output is in [`docs/VERIFICATION.md`](docs/VERIFICATION.md). To use another host port:
`make docker-run PORT=9090`, then `make smoke BASE_URL=http://localhost:9090`.

### Without make

These are the exact commands the Makefile runs (`make docker-build`, `make docker-run`, `make smoke`):

```bash
# build the image (make docker-build)
docker build -t sezzle-calculator:local .

# run it in the foreground (make docker-run): UI + API on http://localhost:8080
docker run --rm -p 8080:8080 --name sezzle-calculator sezzle-calculator:local

# in a second terminal: the smoke test (make smoke)
bash scripts/smoke.sh                       # defaults to http://localhost:8080
bash scripts/smoke.sh http://localhost:9090 # another base URL

# stop the container; --rm removes it
docker stop sezzle-calculator
```

For another host port, change the left side of `-p`: `docker run --rm -p 9090:8080 ...`. The container always listens
on 8080 inside. To run in the background, add `-d` (as in the detached example above).

## Local development

Prerequisites: Go ≥ 1.24, Node ≥ 22, make (tested with Go 1.27.2 and Node 22.18.0), after [getting the code](#get-the-code).

1. Install the frontend dependencies once:

   ```bash
   cd frontend && npm ci && cd ..
   ```

2. In **terminal 1**, start the API (http://localhost:8080):

   ```bash
   make dev-backend
   ```

3. In **terminal 2**, start the frontend dev server, and open **http://localhost:5173**. It proxies `/api` to the API
   on port 8080, so terminal 1 must be running:

   ```bash
   make dev-frontend
   ```

4. Before committing, run the local quality gate (see "Testing & coverage" for what it leaves out):

   ```bash
   make check   # lint + typecheck + coverage gates
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

```bash
make test        # go test -race -count=1 ./...  +  vitest run
make coverage    # backend + frontend coverage reports, with the gates below
make lint        # gofmt, go vet, eslint (--max-warnings=0), tsc --noEmit, no-fixed-sleeps check
make check       # lint + coverage: the local quality gate (not the same set of steps as CI, see below)
```

`make check` needs `cd frontend && npm ci` once. Reports: `coverage/backend.html` and `frontend/coverage/index.html`
(plus `lcov.info`); the latest numbers are in [`docs/COVERAGE.md`](docs/COVERAGE.md).

`make check` does **not** run the `cmd/server` tests (the backend coverage run covers `./internal/...` only), the
production build, or Docker. Run `make test` for every test, `make build-frontend` for the build, and see the CI
section below for what runs on every push.

**Gates (the build fails below them):**

| Scope | Gate |
|---|---|
| Backend `./internal/...` | ≥ 90 % statements (`scripts/check-go-coverage.sh`) |
| Frontend | ≥ 85 % lines, statements and functions; ≥ 80 % branches (Vitest `thresholds`) |

`cmd/server` is tested but not counted in the backend gate; `docs/COVERAGE.md` reports it separately.

**What is tested, and how** (ADR-012):

- **Backend:** table-driven domain tests, `httptest` handler tests for every status and error code, a fake `Calculator`
  to prove error mapping, a fuzz test on the request decoder (`FuzzCalculateHandler`; the seed corpus runs in every
  `go test`, run it for real with `go test -fuzz=FuzzCalculateHandler ./internal/httpapi`), and `run` tested on a free
  port.
- **Frontend:** pure reducer unit tests, the API client against MSW, `formatNumber` tables, and component integration
  tests driven by `user-event` (clicks and keyboard). There is no browser E2E suite; `scripts/smoke.sh` covers the
  running container instead.
- **No fixed sleeps:** `scripts/check-no-sleeps.sh` (part of `make lint`) fails on `time.Sleep` in Go tests and
  `setTimeout` in frontend tests. Tests wait on events instead: a ticker with a deadline for server readiness,
  `waitFor` / `findBy*` in the UI.
- **No order dependence:** tests do not rely on execution order. `go test -race -shuffle=on -count=3 ./...` and Vitest
  with file and test shuffling both pass. `make test` itself does not shuffle.

**CI** (`.github/workflows/ci.yml`) does not call `make check`. It runs the individual `make` targets as three jobs,
and uploads `coverage/` and `frontend/coverage/` as artifacts:

| Job | Steps |
|---|---|
| `backend` | `make lint-backend`, `make test-backend` (all Go tests, including `cmd/server`), `make coverage-backend` |
| `frontend` | `npm ci`, `make lint-frontend`, `make coverage-frontend`, `make build-frontend` |
| `docker` (after both) | `make docker-build`, run the container, wait for `/healthz`, `make smoke`; prints the container's logs if a step fails |

So CI runs everything `make check` runs (lint and coverage), plus the `cmd/server` tests, the production build and the
Docker smoke test.

## Design decisions

Full rationale in [`docs/DECISIONS.md`](docs/DECISIONS.md). Highlights:

- **Stdlib-only Go** with 1.22+ routing; pure domain package behind a consumer-defined interface.
- **One endpoint per operation** (`POST /api/v1/{operation}`), errors as **RFC 9457 problem+json** with stable codes.
- **float64 end to end**, non-finite results rejected, display formatting owned by the frontend.
- **Pocket-calculator UX** driven by a pure reducer; effects as data, stale responses ignored, requests abortable.
- **Single container**: Go serves API + SPA, same origin, no CORS.

## Assumptions

The assignment leaves these open; each is a decision recorded in [`docs/DECISIONS.md`](docs/DECISIONS.md) or in the
spec it comes from.

**Arithmetic and API**

- Numbers are IEEE-754 `float64` end to end, so `POST /add {0.1, 0.2}` returns `0.30000000000000004`; the UI displays
  `0.3`. Inputs must be finite, `±Inf` and `NaN` results are errors, and `-0` is returned as `0`.
- `percentage(a, b)` means "`a` percent of `b`" and is computed literally as `a*b/100`. An intermediate overflow is an
  error even when the true result fits (`percentage(1e308, 10)`).
- `0⁰ = 1`, `0` to a negative power is a division by zero, and `(−8)^(1/3)` is undefined (there are no complex numbers).
- Operands are JSON numbers only (`"2"` is rejected). Bodies are limited to 4 KiB, unknown fields are rejected, and
  sending `b` to a unary operation is an error (even `null`).
- `POST` to an unknown operation is `UNKNOWN_OPERATION`; any other unmatched `/api/` route, including `GET` on an
  unknown operation, is `NOT_FOUND`.
- No auth, rate limiting or CORS: the UI and API share one origin.

**Calculator behaviour**

- Immediate execution, left to right, no precedence (`2 + 3 × 4 = 20`). `=` with no new operand does nothing; there is
  no "repeat last operation".
- Input is limited to 15 digits and every digit counts, including the leading `0` of `0.5`.
- Results are shown with 12 significant digits, in exponential notation at `1e12` and above or below `1e-9`. The
  full-precision value is kept for the next operation.
- `±` on a result is lossy: it edits the displayed 12-digit string (`0.30000000000000004` becomes `-0.3`).
- Keyboard shortcuts work with Shift held (`+`, `*`, `^` and `%` need it on many layouts); Ctrl, Alt and Meta
  combinations are left to the browser.

**Operations**

- `STATIC_DIR` is not validated at startup: a wrong path serves 404s instead of failing fast.
- The Docker `HEALTHCHECK` runs every 10 s (the Dockerfile sets this explicitly).

**Known limits**

- A request target that is not valid HTTP, such as `GET /api/v1/%zz`, is rejected by Go's `net/http` with a `400`
  `text/plain` before any handler runs, so it cannot be a problem+json.
- The layout and dark mode were built from the CSS and the component tests; I have not reviewed them on a real
  phone or in several browsers.

## How this was built (AI usage)

Spec-driven development with Claude Code. I wrote the specs ([`specs/`](specs/)) and the ADRs
([`docs/DECISIONS.md`](docs/DECISIONS.md)); Claude Code implemented them one spec at a time under the rules in
[`CLAUDE.md`](CLAUDE.md):

1. `/implement-spec 00X` produced a plan with an AC-to-test table. **No code was written until I approved the plan.**
2. Code followed in small TDD increments, one Conventional Commit each, and `make check` had to pass.
3. After a spec, a read-only `spec-reviewer` subagent audits it against its ACs and the ADRs (`/review 00X`), and I
   choose which findings to fix.
4. Evidence for each AC is a test, a command or a review, and is listed in the final report of each spec.

Where Claude's first attempt was wrong or incomplete, the correction is recorded rather than hidden: see
[`prompts/PROMPTS.md`](prompts/PROMPTS.md) for the curated prompts and corrections, and
[`prompts/session-log.md`](prompts/session-log.md) for every prompt verbatim.

## What I'd do next

- **Run CI for real.** The workflow (including the Docker and smoke job) has not run on GitHub yet; the first push is
  the real test of it.
- **Look at the UI on real devices** and in other browsers, and add a browser end-to-end suite (for example
  Playwright) on top of the smoke test.
- **Expression input** with precedence and parentheses. It needs a parser and a new API contract, so it would start
  with an ADR.
- **Exact decimal arithmetic** (`math/big` or a decimal type) so `0.1 + 0.2` is exactly `0.3`; ADR-006 records why
  float64 was chosen for now.
- **History and memory keys**, a theme toggle, and localisation (decimal comma).
- **Operability:** a metrics endpoint, rate limiting, OpenAPI generation, fail-fast validation of `STATIC_DIR`, and a
  time-boxed fuzz run in CI.
