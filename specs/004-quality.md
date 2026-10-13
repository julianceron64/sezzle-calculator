# 004 — Quality: tests, coverage, lint, CI

**ADRs**: 012 · Mostly verification and wiring; most tests were written in 001–003.

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| Q-01 | `make test` runs `go test -race -count=1 ./...` and `vitest run`. |
| Q-02 | `make coverage` produces `coverage/backend.out`, `coverage/backend.html`, `frontend/coverage/` (html + lcov + text). |
| Q-03 | Gates enforced (build fails below): backend `./internal/...` ≥ 90 % (via `scripts/check-go-coverage.sh`); frontend ≥ 85 % lines/statements/functions, ≥ 80 % branches (Vitest `thresholds`). |
| Q-04 | `make lint` = `gofmt -l` empty + `go vet` + `eslint` + `tsc --noEmit`. Zero warnings. |
| Q-05 | `make check` = lint + coverage. It is green on a clean clone after `npm ci`. |
| Q-06 | `.github/workflows/ci.yml` runs the same `make` targets (no duplicated logic) and uploads coverage artifacts. |
| Q-07 | `docs/COVERAGE.md` committed: date, backend total + per-package, frontend totals, and how to regenerate. |
| Q-08 | No fixed sleeps (`time.Sleep`, `setTimeout`-based waits) and no dependence on test execution order. Event-driven polling with a timeout (server readiness, RTL `waitFor` / `findBy*`) is allowed. |

## Evidence types

Every AC in every spec is verified by exactly one primary evidence type, and the final report's AC table shows it:

- **test**: an automated test that fails if the behavior breaks (the default).
- **command**: a command whose result is checked (`make lint`, `make check`, `npm run build`, `docker ps`, `scripts/smoke.sh`).
- **review**: checked by `/review` (the `spec-reviewer` subagent) or by a human, because no meaningful test exists.

Assignments for the ACs that are not plain tests:

| AC | Evidence |
|---|---|
| FE-02 | review (spec-reviewer confirms no arithmetic on operands in `reducer.ts`) |
| FE-12, Q-01..Q-05 | command |
| Q-06 | review (workflow uses `make` targets) + command (CI run) |
| Q-07, D-04, D-05 | review |
| Q-08 | command: `scripts/check-no-sleeps.sh` (grep for `time.Sleep` in `backend/**/*_test.go` and `setTimeout` in `frontend/src/**/*.test.*`), wired into `make lint`; it lives in `scripts/` next to `check-go-coverage.sh`. Order independence: review (the plan may add `-shuffle=on` to `make test-backend` only if Q-01's command text is amended by the human). |
| D-01, D-02 | command (`make docker-build && make docker-run`, `docker ps`) |
| D-03 | command (`scripts/smoke.sh`) |

## Coverage scope

The backend gate measures `./internal/...` only (ADR-012). `cmd/server` is exercised by API-10/API-11 tests but is
intentionally **not** counted in the gate; `docs/COVERAGE.md` reports `cmd/server` coverage separately for information.

## SHOULD

- Run `/review` for 001–003 and resolve all Blockers before marking this Done.

## Docs

README > "Testing & coverage".
