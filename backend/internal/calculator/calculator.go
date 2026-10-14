package calculator

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

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

// Info describes a supported operation.
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

type operation struct {
	arity int
	apply func(operands []float64) (float64, error)
}

// Calculator evaluates operations from a registry that is read-only after New,
// so it is safe for concurrent use.
type Calculator struct {
	registry map[Op]operation
	infos    []Info // declaration order
}

func New() *Calculator {
	c := &Calculator{registry: map[Op]operation{}}
	c.register(OpAdd, 2, func(o []float64) (float64, error) { return o[0] + o[1], nil })
	c.register(OpSubtract, 2, func(o []float64) (float64, error) { return o[0] - o[1], nil })
	c.register(OpMultiply, 2, func(o []float64) (float64, error) { return o[0] * o[1], nil })
	c.register(OpDivide, 2, func(o []float64) (float64, error) { return o[0] / o[1], nil })
	c.register(OpPower, 2, func(o []float64) (float64, error) { return math.Pow(o[0], o[1]), nil })
	c.register(OpSqrt, 1, func(o []float64) (float64, error) { return math.Sqrt(o[0]), nil })
	// Literal a*b/100: an intermediate overflow is ErrOverflow (ADR-007).
	c.register(OpPercentage, 2, func(o []float64) (float64, error) { return o[0] * o[1] / 100, nil })
	return c
}

func (c *Calculator) register(name Op, arity int, apply func([]float64) (float64, error)) {
	c.registry[name] = operation{arity: arity, apply: apply}
	c.infos = append(c.infos, Info{Name: name, Arity: arity})
}

func (c *Calculator) Calculate(op Op, operands ...float64) (float64, error) {
	o, ok := c.registry[op]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownOperation, op)
	}
	if len(operands) != o.arity {
		return 0, fmt.Errorf("%w: %s takes %d, got %d", ErrArity, op, o.arity, len(operands))
	}
	return o.apply(operands)
}

// Operations lists the supported operations in declaration order.
func (c *Calculator) Operations() []Info { return slices.Clone(c.infos) }
