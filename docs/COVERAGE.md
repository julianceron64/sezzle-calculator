# Coverage

Snapshot taken **2026-10-13** at the end of spec 004, from `make coverage` (153 frontend tests, all backend tests
with `-race`). The numbers below go stale as the code changes; regenerate them with the commands at the end.

## Gates

| Scope | Gate | Result |
|---|---|---|
| Backend `./internal/...` (statements) | ≥ 90 % | **97.1 %** |
| Frontend lines / statements / functions | ≥ 85 % | **94.76 % / 95.39 % / 95.91 %** |
| Frontend branches | ≥ 80 % | **95.23 %** |

The gates fail the build below the threshold: the backend one in `scripts/check-go-coverage.sh`, the frontend ones
as Vitest `thresholds` in `frontend/vite.config.ts`.

## Backend, per package (inside the gate)

| Package | Statements |
|---|---|
| `internal/calculator` | 100.0 % |
| `internal/config` | 100.0 % |
| `internal/httpapi` | 96.3 % |
| **Total (`./internal/...`)** | **97.1 %** |

### `cmd/server` (reported separately, outside the gate)

`cmd/server` is tested (`run` on a free port, graceful shutdown, `-healthcheck`, invalid config, port in use) but is
intentionally not counted in the gate (ADR-012).

| Package | Statements |
|---|---|
| `cmd/server` | 80.0 % |

Per function: `main` is 0 % (signal wiring and `os.Exit`, kept thin on purpose so `run` carries the logic), `run` is
90.6 % and `checkHealth` is 91.7 %; the uncovered statements in those two are defensive error returns.

## Frontend

| Metric | Covered | Total | % |
|---|---|---|---|
| Statements | 228 | 239 | 95.39 |
| Branches | 200 | 210 | 95.23 |
| Functions | 47 | 49 | 95.91 |
| Lines | 199 | 210 | 94.76 |

Files below 100 %:

| File | Lines | Why |
|---|---|---|
| `src/App.tsx` | 0 % | Three-line wrapper around `<Calculator />`; the component is covered through its own tests |
| `src/lib/assertNever.ts` | 0 % | The exhaustiveness helper; it throws only if a `never` check is violated, so tests do not reach it |
| `src/features/calculator/model/reducer.ts` | 90.7 % | The three `assertNever` defaults of its exhaustive switches, and one branch of the error check in `getExpression` |
| `src/api/client.ts` | 100 % lines, 97.3 % branches | The `?? ''` fallback when `VITE_API_BASE_URL` is unset: tests either set it (`http://api.test`) or stub it to an empty string, so the truly unset case is not exercised |

`src/main.tsx`, `src/test/**` and `*.d.ts` are excluded from coverage by configuration.

## How to regenerate

```bash
make coverage                                  # backend + frontend reports and gates
cd backend && go test -race -count=1 -cover ./cmd/...   # cmd/server, for information
```

Outputs (git-ignored, uploaded as artifacts by CI):

- `coverage/backend.out` and `coverage/backend.html`
- `frontend/coverage/` (`index.html`, `lcov.info`; the text table is printed by the command)
