package calculator

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
)

// matches implements the comparison policy of CALC-11: exact equality, unless
// approx is set (for results that are not exactly representable), in which
// case a relative tolerance of 1e-12 with a floor of 1 applies.
func matches(got, want float64, approx bool) bool {
	if got == want {
		return true
	}
	return approx && math.Abs(got-want) <= 1e-12*math.Max(1, math.Abs(want))
}

func TestMatches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		got, want float64
		approx    bool
		ok        bool
	}{
		{"exact equal", 5, 5, false, true},
		{"exact rejects near miss", 5.000000000001, 5, false, false},
		{"exact rejects tiny miss", 1024.0000000001, 1024, false, false},
		{"approx accepts within tolerance", 0.30000000000000004, 0.3, true, true},
		{"approx accepts relative on large", 1e9 + 1e-4, 1e9, true, true},
		{"approx rejects outside tolerance", 1, 1.001, true, false},
		{"approx floor of 1 near zero", 0, 1e-11, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := matches(tc.got, tc.want, tc.approx); got != tc.ok {
				t.Fatalf("matches(%v, %v, %v) = %v, want %v", tc.got, tc.want, tc.approx, got, tc.ok)
			}
		})
	}
}

type calcCase struct {
	name   string
	op     Op
	in     []float64
	want   float64
	approx bool // false (default): exact equality (CALC-11)
}

func runCases(t *testing.T, cases []calcCase) {
	t.Helper()
	c := New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.Calculate(tc.op, tc.in...)
			if err != nil {
				t.Fatalf("Calculate(%s, %v) error: %v", tc.op, tc.in, err)
			}
			if !matches(got, tc.want, tc.approx) {
				t.Fatalf("Calculate(%s, %v) = %v, want %v", tc.op, tc.in, got, tc.want)
			}
		})
	}
}

// CALC-01
func TestCalculate_Basic(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"add positive", OpAdd, []float64{2, 3}, 5, false},
		{"add negative", OpAdd, []float64{-2, -3}, -5, false},
		{"add zero", OpAdd, []float64{0, 7}, 7, false},
		{"add fractional", OpAdd, []float64{0.1, 0.2}, 0.3, true},
		{"subtract positive", OpSubtract, []float64{5, 3}, 2, false},
		{"subtract negative", OpSubtract, []float64{-5, -3}, -2, false},
		{"subtract zero", OpSubtract, []float64{0, 4}, -4, false},
		{"subtract fractional", OpSubtract, []float64{1.5, 0.25}, 1.25, false},
		{"multiply positive", OpMultiply, []float64{4, 3}, 12, false},
		{"multiply negative", OpMultiply, []float64{-4, 3}, -12, false},
		{"multiply zero", OpMultiply, []float64{0, 9}, 0, false},
		{"multiply fractional", OpMultiply, []float64{0.5, 0.5}, 0.25, false},
		{"divide positive", OpDivide, []float64{12, 4}, 3, false},
		{"divide negative", OpDivide, []float64{-12, 4}, -3, false},
		{"divide zero numerator", OpDivide, []float64{0, 4}, 0, false},
		{"divide fractional", OpDivide, []float64{1, 3}, 1.0 / 3, true},
	})
}

// CALC-05
func TestCalculate_Percentage(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"20 percent of 150", OpPercentage, []float64{20, 150}, 30, false},
		{"150 percent of 20", OpPercentage, []float64{150, 20}, 30, false},
		{"zero", OpPercentage, []float64{0, 99}, 0, false},
		{"negative", OpPercentage, []float64{-10, 50}, -5, false},
	})
}

// CALC-10
func TestOperations_OrderAndArity(t *testing.T) {
	t.Parallel()
	want := []Info{
		{OpAdd, 2}, {OpSubtract, 2}, {OpMultiply, 2}, {OpDivide, 2},
		{OpPower, 2}, {OpSqrt, 1}, {OpPercentage, 2},
	}
	got := New().Operations()
	if len(got) != len(want) {
		t.Fatalf("got %d operations, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Operations()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// CALC-10
func TestOperations_ReturnsCopy(t *testing.T) {
	t.Parallel()
	c := New()
	first := c.Operations()
	first[0] = Info{Name: "hacked", Arity: 99}
	if again := c.Operations(); again[0] != (Info{OpAdd, 2}) {
		t.Fatalf("registry mutated through Operations(): %v", again[0])
	}
}

type errCase struct {
	name string
	op   Op
	in   []float64
	want error
}

func runErrCases(t *testing.T, cases []errCase) {
	t.Helper()
	c := New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.Calculate(tc.op, tc.in...)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Calculate(%s, %v) = (%v, %v), want error %v", tc.op, tc.in, got, err, tc.want)
			}
		})
	}
}

// CALC-02
func TestCalculate_DivideByZero(t *testing.T) {
	t.Parallel()
	negZero := math.Copysign(0, -1)
	runErrCases(t, []errCase{
		{"1/0", OpDivide, []float64{1, 0}, ErrDivisionByZero},
		{"1/-0", OpDivide, []float64{1, negZero}, ErrDivisionByZero},
		{"0/0", OpDivide, []float64{0, 0}, ErrDivisionByZero},
	})
}

// CALC-03
func TestCalculate_Power(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"2^10", OpPower, []float64{2, 10}, 1024, false},
		{"2^-1", OpPower, []float64{2, -1}, 0.5, false},
		{"5^0", OpPower, []float64{5, 0}, 1, false},
		{"0^0", OpPower, []float64{0, 0}, 1, false},
		{"4^0.5", OpPower, []float64{4, 0.5}, 2, false},
	})
	runErrCases(t, []errCase{
		{"0^-1", OpPower, []float64{0, -1}, ErrDivisionByZero},
		{"-0^-1", OpPower, []float64{math.Copysign(0, -1), -1}, ErrDivisionByZero},
		{"(-8)^(1/3)", OpPower, []float64{-8, 1.0 / 3}, ErrUndefinedResult},
		{"10^400", OpPower, []float64{10, 400}, ErrOverflow},
	})
}

// CALC-04
func TestCalculate_Sqrt(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"sqrt 9", OpSqrt, []float64{9}, 3, false},
		{"sqrt 0", OpSqrt, []float64{0}, 0, false},
		{"sqrt 2", OpSqrt, []float64{2}, 1.4142135623730951, true},
	})
	runErrCases(t, []errCase{
		{"sqrt -1", OpSqrt, []float64{-1}, ErrNegativeSquareRoot},
	})
}

// CALC-05, CALC-06
func TestCalculate_Overflow(t *testing.T) {
	t.Parallel()
	runErrCases(t, []errCase{
		{"multiply", OpMultiply, []float64{1e308, 10}, ErrOverflow},
		{"add", OpAdd, []float64{math.MaxFloat64, math.MaxFloat64}, ErrOverflow},
		{"subtract", OpSubtract, []float64{-math.MaxFloat64, math.MaxFloat64}, ErrOverflow},
		{"divide", OpDivide, []float64{1e308, 1e-10}, ErrOverflow},
		{"percentage intermediate", OpPercentage, []float64{1e308, 10}, ErrOverflow},
	})
}

// CALC-07
func TestCalculate_InvalidOperand(t *testing.T) {
	t.Parallel()
	c := New()
	bad := map[string]float64{"NaN": math.NaN(), "+Inf": math.Inf(1), "-Inf": math.Inf(-1)}
	for _, info := range c.Operations() {
		for name, v := range bad {
			for pos := 0; pos < info.Arity; pos++ {
				in := make([]float64, info.Arity)
				for i := range in {
					in[i] = 1
				}
				in[pos] = v
				t.Run(fmt.Sprintf("%s/%s/arg%d", info.Name, name, pos), func(t *testing.T) {
					t.Parallel()
					if _, err := c.Calculate(info.Name, in...); !errors.Is(err, ErrInvalidOperand) {
						t.Fatalf("got %v, want ErrInvalidOperand", err)
					}
				})
			}
		}
	}
}

// CALC-08
func TestCalculate_NegativeZeroNormalized(t *testing.T) {
	t.Parallel()
	c := New()
	negZero := math.Copysign(0, -1)
	for name, tc := range map[string]calcCase{
		"multiply": {op: OpMultiply, in: []float64{-1, 0}},
		"sqrt":     {op: OpSqrt, in: []float64{negZero}},
		"divide":   {op: OpDivide, in: []float64{negZero, 5}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := c.Calculate(tc.op, tc.in...)
			if err != nil || got != 0 || math.Signbit(got) {
				t.Fatalf("got (%v, %v), want +0", got, err)
			}
		})
	}
}

// CALC-09
func TestCalculate_UnknownOperation(t *testing.T) {
	t.Parallel()
	runErrCases(t, []errCase{{"modulo", Op("modulo"), []float64{1, 2}, ErrUnknownOperation}})
}

// CALC-09
func TestCalculate_Arity(t *testing.T) {
	t.Parallel()
	runErrCases(t, []errCase{
		{"binary with one", OpAdd, []float64{1}, ErrArity},
		{"binary with three", OpAdd, []float64{1, 2, 3}, ErrArity},
		{"unary with two", OpSqrt, []float64{4, 4}, ErrArity},
		{"none", OpSqrt, nil, ErrArity},
	})
}

// CALC-09: the earliest rule in the fixed order wins.
func TestCalculate_ValidationOrder(t *testing.T) {
	t.Parallel()
	nan, inf := math.NaN(), math.Inf(1)
	runErrCases(t, []errCase{
		{"unknown beats arity and NaN", Op("nope"), []float64{nan}, ErrUnknownOperation},
		{"arity beats NaN", OpAdd, []float64{nan}, ErrArity},
		{"invalid operand beats division by zero", OpDivide, []float64{nan, 0}, ErrInvalidOperand},
		{"invalid operand beats negative sqrt", OpSqrt, []float64{math.Inf(-1)}, ErrInvalidOperand},
		{"0^negative is division by zero, not overflow", OpPower, []float64{0, -1}, ErrDivisionByZero},
		{"infinite beats 0^negative", OpPower, []float64{0, -inf}, ErrInvalidOperand},
	})
}

// SHOULD: Calculate is safe for concurrent use (meaningful under -race).
func TestCalculate_Concurrent(t *testing.T) {
	t.Parallel()
	c := New()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := c.Calculate(OpAdd, 1, 2); err != nil || got != 3 {
				t.Errorf("got (%v, %v)", got, err)
			}
			_ = c.Operations()
		}()
	}
	wg.Wait()
}

// ADR-006: underflow is not an error; 1e-328 is below the smallest subnormal, so the result is 0.
func TestCalculate_Underflow(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{{"divide", OpDivide, []float64{1e-308, 1e20}, 0, false}})
}
