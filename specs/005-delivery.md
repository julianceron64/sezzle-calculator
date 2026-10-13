# 005 — Delivery: Docker, README, prompts

**ADRs**: 011

## Dockerfile (repo root)

1. `node:22-alpine` stage: `npm ci` + `npm run build` in `frontend/` (copy `package*.json` first for layer cache).
2. `golang:1.24-alpine` stage: `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server`.
3. `gcr.io/distroless/static-debian12:nonroot`: copy binary + `dist` to `/app/web`; `ENV STATIC_DIR=/app/web PORT=8080`;
   `EXPOSE 8080`; `USER nonroot:nonroot`; `HEALTHCHECK CMD ["/app/server", "-healthcheck"]`; `ENTRYPOINT ["/app/server"]`.
- `.dockerignore` excludes `node_modules`, `dist`, `coverage`, `.git`, `.claude`.

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| D-01 | `make docker-build && make docker-run` serves the UI at `http://localhost:8080` and the API under `/api/v1`. |
| D-02 | Container runs as non-root and reports `healthy` in `docker ps`. |
| D-03 | `scripts/smoke.sh [base_url]` (curl + `set -euo pipefail`) checks: healthz, one success per operation family, `DIVISION_BY_ZERO`, `UNKNOWN_OPERATION`, `INVALID_JSON`. Exit ≠ 0 on any mismatch. CI runs it against the built image. |
| D-04 | README complete: overview, quick start (Docker), local dev (Go ≥ 1.24, Node ≥ 22), tests & coverage, API + curl examples, error codes, operation semantics, design decisions (summary + link to ADRs), assumptions, AI usage (link to `prompts/`), "what I'd do next". |
| D-05 | `prompts/PROMPTS.md` curated: kickoff prompt + the prompt used for each spec + notable corrections; links `session-log.md`. |

## Docs

README (all sections above).
