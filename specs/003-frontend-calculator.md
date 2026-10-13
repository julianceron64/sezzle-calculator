# 003 — Frontend: calculator app

**Folder**: `frontend/` · **ADRs**: 005, 006, 008, 009, 010

## Setup (step 1 of this spec)

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
  `lint` (`eslint .`), `typecheck` (`tsc -b --noEmit`), `format` (`prettier --write .`).
- Delete Vite demo assets/styles.

## Structure (fixed)

```
src/
  api/types.ts            OperationName, CalculateResponse, ProblemDetails, ApiError (discriminated by code)
  api/client.ts           calculate(op, operands, { signal }) > Promise<Result<CalculateResponse, ApiError>>
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
- Timeout 5 s via `AbortSignal.any([callerSignal, AbortSignal.timeout(5000)])`.
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
7. After a result, a digit starts a fresh input; `TOGGLE_SIGN`/`BACKSPACE` on a result: sign toggles into editable input / backspace is a no-op.
8. While `evaluating`, every action except `CLEAR` and `EVAL_*` is ignored. `CLEAR` aborts the request.
9. `EVAL_*` with an `id` that isn't the current request is ignored (stale response).
10. On failure > `error` status with the `ApiError`; the next `DIGIT` or `CLEAR` starts clean; other keys ignored.
11. `CLEAR` always resets to the initial state (`editing`, input `"0"`, no pending op).

## Display

- Main line: input as typed while editing; otherwise `formatNumber(value)`.
- Secondary line (expression): e.g. `2 +`, `20 % of`.
- `formatNumber`: `-0` > `0`; `|n| >= 1e15` or `0 < |n| < 1e-9` > exponential with ≤ 10 significant digits;
  else `toPrecision(12)` with trailing zeros trimmed. `0.30000000000000004` > `0.3`.
- Error copy (in `errorMessages.ts`): division by zero, negative square root, out of range, undefined,
  network, timeout, generic fallback. Display shows `Error`; message below with `role="alert"`.

## Keypad & accessibility

- Keys: `C`, `⌫`, `±`, `÷`, `7 8 9 ×`, `4 5 6 −`, `1 2 3 +`, `0 . =`, plus `xʸ`, `√`, `%` (percent of).
- Every key is a `<button type="button">` with an `aria-label` for symbols ("divide", "square root", "percent of", "power").
- Display is `role="status"` + `aria-live="polite"`. Visible `:focus-visible` ring. Disabled look while evaluating.
- Keyboard: `0–9`, `.`/`,`, `+ - * /`, `^`, `%`, `r` (√), `Enter`/`=`, `Backspace`, `Escape`/`Delete` (clear).
  Listener on `window`, removed on unmount; ignored when modifier keys are held.

## Responsive

- Single centered card, `max-width: 360px`; usable at 320 px width without horizontal scroll.
- CSS grid 4 columns; touch targets ≥ 48 px; `prefers-color-scheme` light/dark via tokens on `:root`.

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| FE-01 | Reducer: each rule 1–11 has at least one unit test (pure, no React). |
| FE-02 | Reducer never contains arithmetic on operands (review-checked) — results only come from `EVAL_SUCCESS`. |
| FE-03 | Client: success, problem+json, non-problem error, invalid 2xx shape, network error, timeout, caller abort — each tested with MSW. |
| FE-04 | `formatNumber`: table test covering `-0`, `0.1+0.2`, large, tiny, negative, integers. |
| FE-05 | Integration: clicking `1 2 + 3 =` shows `15` (MSW returns 15) and sends `{a:12,b:3}` to `/api/v1/add`. |
| FE-06 | Integration: chaining `2 + 3 × 4 =` issues two requests in order and shows the final result. |
| FE-07 | Integration: `÷ 0 =` with MSW 422 `DIVISION_BY_ZERO` shows the mapped message in an `alert`; next digit recovers. |
| FE-08 | Integration: the same flows work with the keyboard only. |
| FE-09 | Integration: keys are disabled/ignored while a request is pending; `C` during a pending request aborts it and a late response doesn't change the display. |
| FE-10 | Integration: network error and timeout show their messages (timeout test uses fake timers or a short injected timeout). |
| FE-11 | All keys reachable by accessible name (`getByRole('button', { name: /divide/i })`). |
| FE-12 | `npm run lint`, `npm run typecheck`, `npm run build` pass with zero warnings. |

## Out of scope

History panel, memory keys, theming toggle, i18n, routing, E2E browser tests.

## Docs

README > "Frontend" (run, keyboard shortcuts, UX model and its assumptions).
