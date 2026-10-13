# 001 — Backend: calculator domain

**Package**: `backend/internal/calculator` · **ADRs**: 002, 003, 006, 007

## Goal

A pure, dependency-free arithmetic domain with explicit, testable error semantics.

## Public API (shape is fixed; internals are yours)

```go
type Op string

const (
    OpAdd        Op = "add"
    OpSubtract   Op = "subtract"
    OpMultiply   Op = "multiply"
    OpDivide     Op = "divide"
    OpPower      Op = "power"
    OpSqrt       Op = "sqrt"
    OpPercentage Op = "percentage"
)

type Info struct {
    Name  Op
    Arity int
}

var (
    ErrUnknownOperation   = errors.New("unknown operation")
    ErrArity              = errors.New("wrong number of operands")
    ErrInvalidOperand     = errors.New("operand must be a finite number")
    ErrDivisionByZero     = errors.New("division by zero")
    ErrNegativeSquareRoot = errors.New("square root of a negative number")
    ErrUndefinedResult    = errors.New("result is undefined")
    ErrOverflow           = errors.New("result out of range")
)

type Calculator struct{ /* registry */ }

func New() *Calculator
func (c *Calculator) Calculate(op Op, operands ...float64) (float64, error)
func (c *Calculator) Operations() []Info 
```

- Errors may be wrapped with context (`fmt.Errorf("%w: ...", ErrX)`); callers use `errors.Is`.
- Allowed imports: `errors`, `fmt`, `math`, `slices`. **Not** `net/http`, `encoding/json`, `log`.

## Acceptance criteria (MUST)

| ID | Criterion |
|---|---|
| CALC-01 | `add`, `subtract`, `multiply`, `divide` return correct results for positive, negative, zero and fractional operands. |
| CALC-02 | `divide` with `b == 0` (including `-0` and `0/0`) returns `ErrDivisionByZero`. |
| CALC-03 | `power`: `2^10=1024`, `2^-1=0.5`, `5^0=1`, `0^0=1`, `4^0.5=2`; `0^-1` > `ErrDivisionByZero`; `(-8)^(1/3)` > `ErrUndefinedResult`; `10^400` > `ErrOverflow`. |
| CALC-04 | `sqrt`: `√9=3`, `√0=0`, `√2` within `1e-12` of `1.4142135623730951`; `√-1` > `ErrNegativeSquareRoot`. |
| CALC-05 | `percentage(a, b) = a*b/100`, computed literally in that order: `(20,150)=30`, `(150,20)=30`, `(0,99)=0`, `(-10,50)=-5`. An intermediate overflow returns `ErrOverflow` even when the true result is representable: `percentage(1e308, 10)` > `ErrOverflow` (`a*b` is `+Inf`). |
| CALC-06 | Any operation whose result is `±Inf` returns `ErrOverflow` (e.g. `multiply(1e308, 10)`, `add(MaxFloat64, MaxFloat64)`). |
| CALC-07 | Any operand that is `NaN` or `±Inf` returns `ErrInvalidOperand`, for every operation. |
| CALC-08 | A `-0` result is returned as `+0` (`multiply(-1, 0)` > `math.Signbit` is false). |
| CALC-09 | Unknown `Op` > `ErrUnknownOperation`; wrong operand count > `ErrArity` (checked before evaluating). Validation order inside `Calculate` is fixed: unknown operation > arity > finite operands (CALC-07) > compute > result check (CALC-06/08). Test: input violating several rules at once returns the earliest error in this order. `power` must detect `0^negative` (`ErrDivisionByZero`) before an `Inf` result is classified as `ErrOverflow`. |
| CALC-10 | `Operations()` returns all 7 operations with correct arity, in declaration order, and callers cannot mutate the registry through it. The registry stays a `map[Op]operation` (lookup); declaration order comes from a separate ordered `[]Info` built in `New`, and `Operations()` returns a copy of it (test: mutate the returned slice, call again, get the original). |
| CALC-11 | Comparison policy in tests: exact equality for exactly-representable results, `|got-want| <= 1e-12*max(1,|want|)` otherwise. One shared helper. |

## SHOULD

- `Calculate` is safe for concurrent use (registry is read-only after `New`). Prove with a `-race` test.
- Benchmarks are **not** needed.

## Out of scope

Expression parsing, operator precedence, arbitrary precision, memory functions.

## Docs

README > "Operation semantics" table (copy ADR-007), including the note that `percentage` is computed as `a*b/100`
and an intermediate overflow returns `ErrOverflow` even when the true result is representable
(e.g. `percentage(1e308, 10)`), and the validation order of `Calculate`.
