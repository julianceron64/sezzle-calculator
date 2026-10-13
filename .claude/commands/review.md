---
description: Audit the current changes against a spec using the read-only spec-reviewer subagent
argument-hint: <spec number, e.g. 002>
---

Use the `spec-reviewer` subagent to review the work for spec `$ARGUMENTS`
(diff against the commit before the spec started; if unsure, use `git log` to find it).

Show me its findings exactly as returned, grouped by severity (Blocker / Should fix / Nit).
**Do not fix anything** until I tell you which findings to address.
