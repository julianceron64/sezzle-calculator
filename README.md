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

<!-- TODO(spec-002): endpoints, curl examples (success + 3 errors), configuration -->

## Error codes

<!-- TODO(spec-002): table code → status → meaning -->

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

<!-- TODO(spec-003): UX model, keyboard shortcuts -->

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
