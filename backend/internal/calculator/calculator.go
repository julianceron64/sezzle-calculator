package calculator

import "errors"

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

// Calculator evaluates operations from a read-only registry.
type Calculator struct{}

func New() *Calculator { return &Calculator{} }

func (c *Calculator) Calculate(op Op, operands ...float64) (float64, error) {
	return 0, ErrUnknownOperation
}

func (c *Calculator) Operations() []Info { return nil }
