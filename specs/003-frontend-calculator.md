# 003 — Frontend: calculator app

**Folder**: `frontend/` · **ADRs**: 005, 006, 008, 009, 010

## Setup (step 1 of this spec)

`frontend/` intentionally contains only `.gitkeep` before this step. `create-vite` prompts on a non-empty folder, so
first run `git rm frontend/.gitkeep`. If any command still needs interactive input, **stop and ask the human to run it**.

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend && npm install
npm install -D vitest @vitest/coverage-v8 jsdom @testing-library/react @testing-library/user-event \
  @testing-library/jest-dom msw prettier eslint-plugin-jsx-a11y
```

- `tsconfig.app.json`: keep Vite's `strict`, add `"noUncheckedIndexedAccess": true`.
- `vite.config.ts`: dev proxy `/api` > `http://localhost:8080`; Vitest `environment: "jsdom"`,
  `setupFiles: ["src/test/setup.ts"]`, coverage provider `v8`, reporters `text`, `html`, `lcov`,
  thresholds per spec 004, exclude `src/main.tsx`, `src/test/**`, `**/*.d.ts`.
- ESLint: Vite's flat config + `jsx-a11y` recommended. Prettier defaults + `singleQuote: true`.
- `package.json` scripts: `dev`, `build`, `preview`, `test` (`vitest`), `coverage` (`vitest run --coverage`),
  `lint` (`eslint . --max-warnings=0`), `typecheck` (`tsc -b --noEmit`; if the generated tsconfig does not support it, report and propose the closest equivalent), `format` (`prettier --write .`).
- Delete Vite demo assets/styles.
- **Verify at the start of this spec and report**: that `AbortSignal.any` / `AbortSignal.timeout` and MSW's fetch
  interception work under jsdom. If jsdom's `AbortController`/`AbortSignal` break MSW (fetch rejects the signal),
  `src/test/setup.ts` restores Node's `AbortController` and `AbortSignal` on `globalThis`. No new dependency.

## Structure (fixed)

```
src/
  api/types.ts            OperationName, CalculateResponse, ProblemDetails, ApiError (discriminated by code)
  api/client.ts           calculate(op, operands, { signal, timeoutMs }) > Promise<Result<CalculateResponse, ApiError>>
  features/calculator/
    model/types.ts        State, Action, BinaryOp, UnaryOp
    model/reducer.ts      pure state machine (no I/O, no Date/Math.random)
    model/keymap.ts       keyboard key > Action
    useCalculator.ts      useReducer + effect that executes requests, abort, keyboard listener
    errorMessages.ts      ApiError code > user copy (single place)
    components/           Calculator, Display, Keypad, Key (+ *.module.css)
  lib/formatNumber.ts     display formatting
  test/setup.ts, test/msw.ts
```

## API client rules

- Base URL `import.meta.env.VITE_API_BASE_URL ?? ""` (same-origin by default).
- Timeout via `AbortSignal.any([callerSignal, AbortSignal.timeout(timeoutMs)])`; both `signal` and `timeoutMs`
  are optional, `timeoutMs` defaults to `5000`. Tests inject a short `timeoutMs` (no real 5 s waiting).
- Returns a `Result` — **never throws** for HTTP/network/timeout errors:
  - 2xx > validate body shape with a type guard (no `as`); invalid shape > `INVALID_RESPONSE`.
  - non-2xx with `application/problem+json` > `ApiError` with backend `code` + `status`.
  - non-2xx otherwise > `UNKNOWN_ERROR`. Network failure > `NETWORK_ERROR`. Timeout > `TIMEOUT`.
  - Caller abort > `ABORTED` (the hook ignores it silently).

## State machine (behavioral contract)

State is a discriminated union on `status`: `editing` | `result` | `evaluating` | `error`.
`evaluating` carries the request to execute (`{ id, op, operands }`) — **effects as data**; the hook runs it.
Actions: `DIGIT`, `DECIMAL`, `TOGGLE_SIGN`, `BACKSPACE`, `CLEAR`, `BINARY_OP`, `UNARY_OP`, `EQUALS`,
`EVAL_SUCCESS {id, result}`, `EVAL_FAILURE {id, error}`.

Rules (immediate execution, no precedence):

1. Digits append; a lone `0` is replaced; max **15 digits** (sign and `.` not counted); a second `.` is ignored; `.` on empty input yields `0.`.
2. `BINARY_OP` with no pending op > current value becomes the left operand; next digit starts a new input.
3. `BINARY_OP` right after another op (no new digits) > replaces the pending op.
4. `BINARY_OP` with pending op and a new operand > evaluates the pending op; on success the result is the new left operand and the new op is pending (`2 + 3 ×` shows `5`).
5. `EQUALS` with pending op and new operand > evaluate; show result; pending cleared. Otherwise no-op (no "repeat last op").
6. `UNARY_OP` (√) > evaluates on the current value immediately; result replaces the current value; pending op is kept (`2 + √9 =` > `5`).
7. After a result, a digit starts a fresh input; `BACKSPACE` on a result is a no-op; `TOGGLE_SIGN` on a result: see rule 12d.
8. While `evaluating`, every action except `CLEAR` and `EVAL_*` is ignored. `CLEAR` aborts the request.
9. `EVAL_*` with an `id` that isn't the current request is ignored (stale response).
10. On failure > `error` status with the `ApiError`; the next `DIGIT` or `CLEAR` starts clean; other keys ignored.
11. `CLEAR` always resets to the initial state (`editing`, input `"0"`, no pending op) **except `nextId`, which never resets**
    (see 12f).
12. Cases that would otherwise be undefined. "Committed left operand" = the state right after a binary operator,
    where the display shows the left operand and no digits have been typed yet.
    - **12a** `TOGGLE_SIGN` with a committed left operand: no-op.
    - **12b** `BACKSPACE` with a committed left operand: no-op.
    - **12c** `UNARY_OP` with a committed left operand (`2 + √`): evaluates on the displayed left operand with a new
      request id. On success the left operand becomes the result, the pending operator is kept, and the next digit
      starts a fresh input. `EQUALS` right after is a no-op (rule 5).
    - **12d** `TOGGLE_SIGN` on a result: if `formatNumber(value)` is plain (no `e`), the state becomes `editing` with
      that string's sign flipped (lossy: the input holds the 12-significant-digit display string, not the full
      float64; document this in the README). If it is exponential, `TOGGLE_SIGN` is a no-op, because an exponential
      string is not valid input under ADR-009.
    - **12e** `TOGGLE_SIGN` when the value is zero (input `"0"`, `"0."`, `"0.0"`, or a result of 0): no-op. There is never
      a `-0` input.
    - **12f** Request ids: `nextId: number` lives at the root of the state in every status. Starting an evaluation
      uses `id = nextId` and increments it. `CLEAR` preserves it, so a late response of an aborted request can never
      match a later request.

Tests for rule 12 (reducer unit tests unless noted; all count toward FE-01):

| Case | Test |
|---|---|
| 12a | `2 +` then `TOGGLE_SIGN`: state unchanged; then `3 =` evaluates `{a:2,b:3}`. |
| 12b | `2 +` then `BACKSPACE`: state unchanged. |
| 12c | `2 + √`: state is `evaluating` with request `{op:"sqrt", operands:[2]}` and pending `+` kept; after `EVAL_SUCCESS`, left operand is the result, a digit starts fresh input, `EQUALS` before any digit is a no-op, and `3 =` then evaluates `{a:<result>,b:3}` with `add`. |
| 12d | Result `0.3` then `TOGGLE_SIGN`: `editing`, input `"-0.3"`. Result `0.30000000000000004` then `TOGGLE_SIGN`: input `"-0.3"` (loss documented). Result `1e15` then `TOGGLE_SIGN`: state unchanged. |
| 12e | `TOGGLE_SIGN` on initial state, on `"0."`, and on a result of 0: state unchanged; `0 ± + 5 =` evaluates `{a:0,b:5}`. |
| 12f | Two evaluations get ids `n` and `n+1`. Late-response test: start request (id 1), `CLEAR`, start another (id 2), deliver `EVAL_SUCCESS` for id 1: display and state unchanged. |

## Display

- Main line: input as typed while editing; otherwise `formatNumber(value)`.
- Secondary line (expression): e.g. `2 +`, `20 % of`.
- `formatNumber(n)` (never use `toPrecision`, it switches to exponential on its own at ≥ 1e12 and < 1e-6):
  1. `n === 0` (including `-0`) > `"0"`. Otherwise `sign = n < 0 ? "-" : ""`, `abs = |n|`.
  2. `[digits, exp]` from `abs.toExponential(11)` (12 significant digits, correctly rounded, already normalized:
     `digits` is the 12 mantissa digits, `exp` the integer exponent).
  3. **The threshold is decided on the rounded exponent** (step 2), not on the raw value: if `exp >= 12` or `exp < -9`
     > exponential: `abs.toExponential(9)` (10 significant digits, rounded from the raw value), trailing zeros of the
     mantissa trimmed (and the dot if nothing is left), exponent as JS prints it (`e+12`, `e-10`), prefixed with `sign`.
  4. Otherwise plain: if `exp >= 0`, put the dot after `exp + 1` digits of `digits` (none if `exp + 1 == 12`);
     if `exp < 0`, `"0." + "0".repeat(-exp - 1) + digits`. Trim trailing zeros and a trailing dot. Prefix `sign`.
  Consequences: a plain display never shows a value that rounds to ≥ 1e12; `0.30000000000000004` > `0.3`;
  `123456789012345` > `1.23456789e+14`.
- Error copy (in `errorMessages.ts`): division by zero, negative square root, out of range, undefined,
  network, timeout, generic fallback. Display shows `Error`; message below with `role="alert"`.

## Keypad & accessibility

- Keys: `C`, `⌫`, `±`, `÷`, `7 8 9 ×`, `4 5 6 −`, `1 2 3 +`, `0 . =`, plus `xʸ`, `√`, `%` (percent of).
- Every key is a `<button type="button">` with an accessible name (`aria-label` for symbols):
  `÷` divide · `×` multiply · `−` subtract · `+` add · `xʸ` power · `√` square root · `%` percent of ·
  `±` toggle sign · `⌫` backspace · `C` clear · `.` decimal point · `=` equals · digits use their own number (`0`–`9`).
- Display is `role="status"` + `aria-live="polite"`. Visible `:focus-visible` ring.
- While `evaluating`, keys are `aria-disabled="true"` with a disabled look and the reducer ignores their actions;
  **no native `disabled` attribute** (focus is kept). `C` is always enabled.
- Keyboard: `0–9`, `.`/`,`, `+ - * /`, `^`, `%`, `r` (√), `Enter`/`=`, `Backspace`, `Escape`/`Delete` (clear).
  Listener on `window`, removed on unmount; ignored when modifier keys are held. The handler calls
  `preventDefault()` for every key the keymap handles (so `Enter` on a focused button does not also click it).

## Responsive

- Single centered card, `max-width: 360px`; usable at 320 px width without horizontal scroll.
- CSS grid 4 columns; touch targets ≥ 48 px; `prefers-color-scheme` light/dark via tokens on `:root`.

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| FE-01 | Reducer: each rule 1–12 (including 12a–12f, with the tests listed under rule 12) has at least one unit test (pure, no React). |
| FE-02 | Reducer never contains arithmetic on operands (review-checked) — results only come from `EVAL_SUCCESS`. |
| FE-03 | Client: success, problem+json, non-problem error, invalid 2xx shape, network error, timeout, caller abort — each tested with MSW. |
| FE-04 | `formatNumber`: table test with exactly these expectations (algorithm in Display). Basic: `0`>`0`, `-0`>`0`, `0.1+0.2`>`0.3`, `2/3`>`0.666666666667`, `-1.5`>`-1.5`, `1e11`>`100000000000`, `1e12`>`1e+12`, `123456789012345`>`1.23456789e+14`, `-123456789012345`>`-1.23456789e+14`, `1e15`>`1e+15`, `-1e15`>`-1e+15`, `12345678901234567890`>`1.23456789e+19`, `1e-6`>`0.000001`, `1e-7`>`0.0000001`, `1e-9`>`0.000000001`, `1e-10`>`1e-10`, `-1e-10`>`-1e-10`. Rounding-crossing (threshold decided on the **rounded** value): `999999999999.4`>`999999999999` (control, stays plain); `999999999999.9`>`1e+12` (raw < 1e12, rounds to 1e12); `-999999999999.9`>`-1e+12`; `999999999999999.9`>`1e+15`; `9.9999999999999e-10` (raw < 1e-9, rounds to 1e-9)>`0.000000001`; `9.99999e-10` (control)>`9.99999e-10`. |
| FE-05 | Integration: clicking `1 2 + 3 =` shows `15` (MSW returns 15) and sends `{a:12,b:3}` to `/api/v1/add`. |
| FE-06 | Integration: chaining `2 + 3 × 4 =` issues two requests in order and shows the final result. |
| FE-07 | Integration: `÷ 0 =` with MSW 422 `DIVISION_BY_ZERO` shows the mapped message in an `alert`; next digit recovers. |
| FE-08 | Integration: the same flows work with the keyboard only. |
| FE-09 | Integration: while a request is pending, keys have `aria-disabled="true"` (no native `disabled`) and their presses/keystrokes are ignored; `C` stays enabled, aborts the request, and a late response doesn't change the display. |
| FE-10 | Integration: network error and timeout show their messages (timeout test injects a short `timeoutMs`; no real 5 s wait). |
| FE-11 | All keys reachable by the accessible names listed above (`getByRole('button', { name: /divide/i })`, etc.). |
| FE-12 | `npm run lint` (`--max-warnings=0`), `npm run typecheck`, `npm run build` pass with zero warnings. |
| FE-13 | Keyboard: pressing a handled key calls `preventDefault()` (e.g. `Enter` with a digit key focused triggers only `EQUALS`, not also that key's click). |

## Out of scope

History panel, memory keys, theming toggle, i18n, routing, E2E browser tests.

## Docs

README > "Frontend" (run, keyboard shortcuts, UX model and its assumptions).
