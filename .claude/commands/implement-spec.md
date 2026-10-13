---
description: Implement one spec end-to-end (plan → approval → TDD → make check → commit)
argument-hint: <spec number, e.g. 001>
---

Implement spec `$ARGUMENTS`.

1. Read `CLAUDE.md`, `docs/DECISIONS.md`, `specs/README.md` and `specs/$ARGUMENTS-*.md`.
   Verify every spec listed under "Depends on" is `Done`; if not, stop and tell me.
2. Produce a plan. **Do not write code yet.** The plan must contain:
   - files to create / modify, grouped by layer;
   - a table mapping **every AC ID** to the test(s) that will prove it (test name + file);
   - commit sequence (Conventional Commits);
   - ambiguities or conflicts with an ADR — each with ≤2 options, trade-offs and your recommendation.
   Then stop and wait for my approval.
3. After approval, work in TDD increments: failing test → minimal code → refactor. Commit per increment.
4. Run `make check`. Fix until green. Never lower thresholds, skip tests or silence linters.
5. Update the spec's Status to `Done` in `specs/README.md` and the README sections listed under **Docs**.
6. Final report: AC → test table (✅/❌), coverage numbers, deviations (expected: none), open questions.
