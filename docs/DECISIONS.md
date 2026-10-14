# Architecture Decision Records

Format: **Context → Decision → Alternatives considered → Consequences**. Status: Accepted unless noted.

---

## ADR-001 — Monorepo, two deployables, one image

- **Decision**: single repo with `backend/` (Go module) and `frontend/` (Vite app). Built into one container.
- **Alternatives**: two repos (overkill for a take-home; harder to review atomically).
- **Consequences**: one PR shows the whole change; CI runs both pipelines; a root `Makefile` is the single entry point.

## ADR-002 — Go standard library only

- **Context**: the API is tiny; reviewers value idiomatic Go over framework knowledge.
- **Decision**: `net/http` with Go 1.22+ method/path patterns (`POST /api/v1/{operation}`), `encoding/json`,
  `log/slog`, `testing` + `httptest`. **Zero third-party modules**, tests included.
- **Alternatives**: chi/gin/echo (unneeded since 1.22 routing), testify (table tests + `cmp`-free helpers suffice).
- **Consequences**: no `go.sum`, no supply-chain surface, nothing to upgrade.

## ADR-003 — Layering and an operation registry

- **Decision**: `cmd/server` (composition root) → `internal/httpapi` (transport) → `internal/calculator` (pure domain).
  - The domain exposes `Calculate(op, operands) (float64, error)` and `Operations() []Info`, backed by a
    registry `map[Op]operation{arity, apply}`. Adding an operation = one registry entry + tests.
  - `httpapi` declares the `Calculator` interface it consumes (consumer-defined interface) → handlers are
    testable with a fake, the domain is testable without HTTP.
- **Alternatives**: one handler per operation with duplicated decoding (repetitive); a switch statement (OCP violation).
- **Consequences**: the HTTP layer knows each operation's arity (to validate `a`/`b`) via `Operations()`.
- **Amendment 2026-10-13 — ordered operations**: the registry stays a `map[Op]operation` for lookup; declaration order
  comes from a separate ordered `[]Info` built in `New`. `Operations()` returns a copy of that slice, so callers cannot
  mutate the registry (spec 001, CALC-10).

## ADR-004 — API shape: one endpoint per operation

- **Decision**:
  - `POST /api/v1/{operation}` with named operands `{"a": number, "b": number}` (`b` omitted for unary ops).
  - `GET /api/v1/operations` lists supported operations and their arity.
  - `GET /healthz` for liveness.
  - Versioned under `/api/v1`.
- **Why POST**: inputs are typed JSON numbers validated uniformly; avoids float encoding in query strings.
- **Alternatives**:
  - single `POST /calculate {operation, operands[]}` — rejected: unknown operation becomes a body-validation
    error instead of a 404, and the spec literally asks to "expose endpoints for calculator operations".
  - `GET /add?a=1&b=2` — rejected: string→float parsing per param, weaker validation story.
- **Consequences**: unknown operation → `404 UNKNOWN_OPERATION`; routing is trivial; easy curl examples.
- **Amendment 2026-10-13 — routing rules**: `POST /api/v1/<unknown>` → `404 UNKNOWN_OPERATION`; any other unmatched
  `/api/` route → `404 NOT_FOUND`; `POST /api/v1/operations` → `405` with `Allow: GET`. The router emits every error body
  itself as `application/problem+json` (no `ServeMux` `text/plain` defaults) (spec 002, API-04).

## ADR-005 — Error model: RFC 9457 Problem Details + stable codes

- **Decision**: every error under `/api/` is `application/problem+json`:
  `{"type","title","status","detail","code"}`. `code` is a stable SCREAMING_SNAKE identifier.
  Status mapping: `400` malformed/invalid input · `404` unknown operation/route · `405` wrong method (with `Allow`)
  · `413` body too large · `415` wrong content type · `422` well-formed input that cannot be computed
  (domain errors) · `500` unexpected.
- **Alternatives**: ad-hoc `{"error": "..."}` (no standard, clients parse strings).
- **Consequences**: the frontend maps errors by `code` only — never by `detail` text. User-facing copy lives in the frontend.
- **Amendment 2026-10-13 — precedence and defensive mapping**: request errors resolve in this order: 404/405 → 415 →
  413 / 400 `INVALID_JSON` → unknown fields → operand type or missing → `UNEXPECTED_OPERAND` (`{"a":1,"b":null}` on a unary
  operation is `UNEXPECTED_OPERAND`). Matching `DisallowUnknownFields` errors by message text is allowed only inside one
  isolated, tested function. Sentinels outside the table map defensively: `ErrInvalidOperand` → 400 `INVALID_OPERAND`,
  `ErrUnknownOperation` → 404 `UNKNOWN_OPERATION`, `ErrArity` → 500 `INTERNAL_ERROR` with no internals in `detail`.

## ADR-006 — Numbers are IEEE-754 float64 end to end

- **Decision**: operands and results are `float64`. Inputs must be finite; results that are `±Inf` or `NaN`
  are errors (JSON cannot encode them anyway). `-0` is normalized to `0`.
  **Display formatting is a frontend concern** (12 significant digits, trimmed, exponential at extremes).
- **Alternatives**: `math/big` / decimal library — rejected: third-party dep or awkward `sqrt`/`pow`,
  irrationals can't be exact anyway, out of scope for a 2–4 h exercise.
- **Consequences**: `POST /add {0.1, 0.2}` returns `0.30000000000000004`; the UI shows `0.3`. Documented in README.
- **Amendment 2026-10-13 — display formatting**: 12 significant digits in plain notation; exponential (10 significant
  digits, trimmed) when the 12-digit-rounded exponent is `>= 12` (i.e. `>= 1e12`) or `< -9` (i.e. `< 1e-9`). The threshold is
  decided on the **rounded** value, so a plain display never shows a value that rounds to `>= 1e12` (`999999999999.9` →
  `1e+12`). Formatting is string-based from `toExponential`, never `toPrecision` (spec 003, FE-04).

## ADR-007 — Operation semantics (assumptions)

| Operation | Arity | Semantics | Domain errors |
|---|---|---|---|
| `add` | 2 | a + b | overflow |
| `subtract` | 2 | a − b | overflow |
| `multiply` | 2 | a × b | overflow |
| `divide` | 2 | a ÷ b | b = 0 (incl. 0/0, −0) |
| `power` | 2 | aᵇ (`0⁰ = 1`, as `math.Pow`) | `0^negative` → division by zero; NaN (e.g. `(−8)^(1/3)`) → undefined; overflow |
| `sqrt` | 1 | √a | a < 0 (no complex numbers) |
| `percentage` | 2 | "a percent of b" = a × b ÷ 100 | overflow |

- **Amendment 2026-10-13 — percentage evaluation**: computed literally as `a*b/100`. An intermediate overflow returns
  `ErrOverflow` even when the true result is representable (e.g. `percentage(1e308, 10)`). Documented in the README.
- **Amendment 2026-10-13 — validation order in `Calculate`**: unknown operation → arity → finite operands → compute →
  result check. `0^negative` is detected as `ErrDivisionByZero` before an `Inf` result can become `ErrOverflow`.

## ADR-008 — Frontend stack

- **Decision**: Vite + React + TypeScript (`strict`, `noUncheckedIndexedAccess`). No router, no state library,
  no UI kit, no data-fetching library. CSS Modules + CSS custom properties. Feature-folder structure.
- **Alternatives**: Redux/Zustand (one screen, one reducer — unjustified), React Query (one mutation,
  needs abort + stale-response control we want to show explicitly), Tailwind/MUI (dependency weight).
- **Consequences**: runtime deps are only `react` + `react-dom`.
- **Amendment 2026-10-13 — client timeout and test environment**: `calculate(op, operands, { signal, timeoutMs })`, with
  `timeoutMs` optional (default 5000) so tests inject a short timeout instead of waiting. If jsdom's `AbortController` /
  `AbortSignal` break MSW, `src/test/setup.ts` restores Node's. Lint runs `eslint . --max-warnings=0`.

## ADR-009 — Validation split: format on the client, rules on the server

- **Decision**: the frontend validates **input format** (digits, one decimal point, max length, sign) and
  prevents impossible key sequences. It never computes results nor pre-checks domain rules (division by
  zero, negative sqrt). The backend is the single source of truth.
- **Consequences**: one round trip per operation (acceptable); no duplicated business rules to drift apart.

## ADR-010 — UX model: pocket calculator with immediate execution

- **Decision**: keypad UI. Binary operations execute left-to-right without precedence (`2 + 3 × 4 = 20`),
  each step being one API call. `√` applies immediately to the current value. Full keyboard support.
- **Alternatives**: expression parser with precedence — rejected: needs a parser/AST and a different API
  contract; scope creep. Two-input form — rejected: less intuitive, doesn't feel like a calculator.
- **Consequences**: state machine in a pure reducer; effects described as data (`status: "evaluating"` +
  request), executed by the hook; stale responses ignored via request id; `C` aborts in-flight requests.
- **Amendment 2026-10-13 — undefined key sequences (rule 12)**: `±` and `⌫` right after a binary operator are no-ops;
  `√` there evaluates on the displayed left operand; `±` on a plain result becomes editable input with the sign flipped
  (lossy to 12 significant digits), on an exponential result or on zero it is a no-op (never a `-0` input). The request id
  is a counter `nextId` at the state root in every status, and `CLEAR` resets everything except `nextId`, so a late
  response can never match a later request.
- **Amendment 2026-10-13 — keyboard and busy state**: the window keydown handler calls `preventDefault()` for every key the
  keymap handles. While `evaluating`, keys use `aria-disabled` and the reducer ignores their actions (no native `disabled`,
  focus is kept); `C` is always enabled. Accessible names are fixed in spec 003.

## ADR-011 — Deployment: one binary serves API + SPA

- **Decision**: the Go server serves the built SPA from `STATIC_DIR` (with `index.html` fallback for
  non-`/api` routes). Same origin in production → **no CORS**. In dev, Vite proxies `/api` → `:8080`.
  Image: multi-stage (node build → go build `CGO_ENABLED=0` → `distroless/static:nonroot`).
- **Alternatives**: nginx + API via docker-compose (two containers, CORS or reverse-proxy config to explain).
- **Consequences**: one port, one process, tiny image. Healthcheck via `server -healthcheck` (distroless has no curl).
- **Amendment 2026-10-13 — static serving details**: `/healthz` and `/api/` take precedence over files and the SPA fallback;
  non-GET/HEAD to a non-API path → 405 problem+json with `Allow: GET, HEAD`. `X-Request-ID` is on every response;
  `Cache-Control: no-store` only on `/api/` responses.

## ADR-012 — Testing strategy and gates

- **Backend**: table-driven domain tests; `httptest` handler tests for every status/code; one native fuzz test
  on the request decoder (never 5xx, always valid JSON). `go test -race`.
- **Frontend**: reducer unit tests (highest value: pure), API client with MSW, formatter tests, and
  component integration tests driven by `user-event` (click + keyboard).
- **No E2E framework** (scope). Replaced by `scripts/smoke.sh` (curl) against the container.
- **Gates**: backend `internal/...` ≥ 90 % statements; frontend ≥ 85 % lines/statements/functions, ≥ 80 % branches.
- **Amendment 2026-10-13 — sleeps and evidence types**: tests may not use fixed sleeps or depend on execution order;
  event-driven polling with a timeout (server readiness, `waitFor` / `findBy*`) is allowed. Every AC carries an evidence
  type (test, command or review) shown in the final report. `cmd/server` is tested but not counted in the backend gate;
  `docs/COVERAGE.md` reports it separately.

## ADR-013 — Minimal production hygiene

- `slog` JSON logs; middleware chain `RequestID → AccessLog → Recover → mux`.
- `http.Server` timeouts (`ReadHeader 5s`, `Read 10s`, `Write 10s`, `Idle 60s`), body limit 4 KiB,
  graceful shutdown on SIGINT/SIGTERM (10 s), `Cache-Control: no-store` on API responses.
- Config via env only (`PORT`, `STATIC_DIR`, `LOG_LEVEL`). `main` delegates to `run(ctx, args, getenv, stdout) error` for testability.
