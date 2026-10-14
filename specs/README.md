# Specs

Spec-driven workflow: the human writes specs and ADRs; Claude Code implements them one at a time.

## Workflow

1. `/implement-spec 00X` → Claude reads spec + ADRs and returns a plan with an **AC → test** table.
2. Human approves or corrects the plan.
3. Claude implements with TDD, small Conventional Commits, `make check` green.
4. `/review 00X` → read-only `spec-reviewer` subagent audits the diff. Human chooses what to fix.
5. Status set to `Done` here.

## Index

| # | Spec | Depends on | Status |
|---|---|---|---|
| 001 | [Backend — calculator domain](001-backend-domain.md) | — | Done |
| 002 | [Backend — HTTP API & server](002-backend-http-api.md) | 001 | Todo |
| 003 | [Frontend — calculator app](003-frontend-calculator.md) | 002 (contract only) | Todo |
| 004 | [Quality — tests, coverage, lint, CI](004-quality.md) | 001–003 | Todo |
| 005 | [Delivery — Docker, README, prompts](005-delivery.md) | 001–004 | Todo |

## Conventions

- AC IDs are stable (`CALC-03`, `API-07`, `FE-12`…). Tests reference them in their name or a comment.
- **MUST** = required for Done. **SHOULD** = do it if it fits the timebox; report if skipped.
- Out-of-scope items are listed explicitly in each spec. Don't build them.