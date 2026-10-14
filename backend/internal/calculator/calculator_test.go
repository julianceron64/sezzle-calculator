package calculator

import (
	"errors"
	"math"
	"testing"
)

// approxEqual implements the comparison policy of CALC-11: exact equality when
// want is exactly representable (callers pass exact values), otherwise a
// relative tolerance of 1e-12 with a floor of 1.
func approxEqual(got, want float64) bool {
	if got == want {
		return true
	}
	return math.Abs(got-want) <= 1e-12*math.Max(1, math.Abs(want))
}

func TestApproxEqual(t *testing.T) {
	t.Parallel()
	if !approxEqual(0.5, 0.5) || !approxEqual(1e9+1e-4, 1e9) || approxEqual(1, 1.001) || approxEqual(0, 1e-11) {
		t.Fatal("approxEqual does not follow CALC-11")
	}
}

type calcCase struct {
	name string
	op   Op
	in   []float64
	want float64
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
			if !approxEqual(got, tc.want) {
				t.Fatalf("Calculate(%s, %v) = %v, want %v", tc.op, tc.in, got, tc.want)
			}
		})
	}
}

// CALC-01
func TestCalculate_Basic(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"add positive", OpAdd, []float64{2, 3}, 5},
		{"add negative", OpAdd, []float64{-2, -3}, -5},
		{"add zero", OpAdd, []float64{0, 7}, 7},
		{"add fractional", OpAdd, []float64{0.1, 0.2}, 0.3},
		{"subtract positive", OpSubtract, []float64{5, 3}, 2},
		{"subtract negative", OpSubtract, []float64{-5, -3}, -2},
		{"subtract zero", OpSubtract, []float64{0, 4}, -4},
		{"subtract fractional", OpSubtract, []float64{1.5, 0.25}, 1.25},
		{"multiply positive", OpMultiply, []float64{4, 3}, 12},
		{"multiply negative", OpMultiply, []float64{-4, 3}, -12},
		{"multiply zero", OpMultiply, []float64{0, 9}, 0},
		{"multiply fractional", OpMultiply, []float64{0.5, 0.5}, 0.25},
		{"divide positive", OpDivide, []float64{12, 4}, 3},
		{"divide negative", OpDivide, []float64{-12, 4}, -3},
		{"divide zero numerator", OpDivide, []float64{0, 4}, 0},
		{"divide fractional", OpDivide, []float64{1, 3}, 1.0 / 3},
	})
}

// CALC-05
func TestCalculate_Percentage(t *testing.T) {
	t.Parallel()
	runCases(t, []calcCase{
		{"20 percent of 150", OpPercentage, []float64{20, 150}, 30},
		{"150 percent of 20", OpPercentage, []float64{150, 20}, 30},
		{"zero", OpPercentage, []float64{0, 99}, 0},
		{"negative", OpPercentage, []float64{-10, 50}, -5},
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
		{"2^10", OpPower, []float64{2, 10}, 1024},
		{"2^-1", OpPower, []float64{2, -1}, 0.5},
		{"5^0", OpPower, []float64{5, 0}, 1},
		{"0^0", OpPower, []float64{0, 0}, 1},
		{"4^0.5", OpPower, []float64{4, 0.5}, 2},
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
		{"sqrt 9", OpSqrt, []float64{9}, 3},
		{"sqrt 0", OpSqrt, []float64{0}, 0},
		{"sqrt 2", OpSqrt, []float64{2}, 1.4142135623730951},
	})
	runErrCases(t, []errCase{
		{"sqrt -1", OpSqrt, []float64{-1}, ErrNegativeSquareRoot},
	})
}
