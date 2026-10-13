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
| Q-08 | No test uses `time.Sleep`/real timers for synchronization; no test depends on execution order. |

## SHOULD

- Run `/review` for 001–003 and resolve all Blockers before marking this Done.

## Docs

README > "Testing & coverage".
