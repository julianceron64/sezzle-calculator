package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

// Stable problem codes (ADR-005). Clients map errors by code only.
const (
	codeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	codePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	codeInvalidJSON          = "INVALID_JSON"
	codeUnknownField         = "UNKNOWN_FIELD"
	codeInvalidOperand       = "INVALID_OPERAND"
	codeMissingOperand       = "MISSING_OPERAND"
	codeUnexpectedOperand    = "UNEXPECTED_OPERAND"
	codeUnknownOperation     = "UNKNOWN_OPERATION"
	codeNotFound             = "NOT_FOUND"
	codeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
	codeDivisionByZero       = "DIVISION_BY_ZERO"
	codeNegativeSquareRoot   = "NEGATIVE_SQUARE_ROOT"
	codeUndefinedResult      = "UNDEFINED_RESULT"
	codeResultOutOfRange     = "RESULT_OUT_OF_RANGE"
	codeInternalError        = "INTERNAL_ERROR"
)

const problemContentType = "application/problem+json"

// problem is an RFC 9457 problem details object.
type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

type problemInfo struct {
	status int
	title  string
	detail string
}

var problemTable = map[string]problemInfo{
	codeUnsupportedMediaType: {http.StatusUnsupportedMediaType, "Unsupported media type", "Content-Type must be application/json"},
	codePayloadTooLarge:      {http.StatusRequestEntityTooLarge, "Payload too large", "The request body must not exceed 4 KiB"},
	codeInvalidJSON:          {http.StatusBadRequest, "Invalid JSON", "The request body must be a single JSON object"},
	codeUnknownField:         {http.StatusBadRequest, "Unknown field", "The request body contains a field other than a or b"},
	codeInvalidOperand:       {http.StatusBadRequest, "Invalid operand", "Operands must be finite JSON numbers"},
	codeMissingOperand:       {http.StatusBadRequest, "Missing operand", "A required operand is missing or null"},
	codeUnexpectedOperand:    {http.StatusBadRequest, "Unexpected operand", "This operation does not take operand b"},
	codeUnknownOperation:     {http.StatusNotFound, "Unknown operation", "The requested operation is not supported"},
	codeNotFound:             {http.StatusNotFound, "Not found", "No such route"},
	codeMethodNotAllowed:     {http.StatusMethodNotAllowed, "Method not allowed", "This route does not support the request method"},
	codeDivisionByZero:       {http.StatusUnprocessableEntity, "Division by zero", "The operation would divide by zero"},
	codeNegativeSquareRoot:   {http.StatusUnprocessableEntity, "Negative square root", "The square root of a negative number is not defined"},
	codeUndefinedResult:      {http.StatusUnprocessableEntity, "Undefined result", "The result of this operation is not a real number"},
	codeResultOutOfRange:     {http.StatusUnprocessableEntity, "Result out of range", "The result is too large to represent"},
	codeInternalError:        {http.StatusInternalServerError, "Internal error", "An unexpected error occurred"},
}

// newProblem builds the problem for a code; type is "/problems/" + kebab-case(code).
func newProblem(code string) problem {
	info := problemTable[code]
	return problem{
		Type:   "/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		Title:  info.title,
		Status: info.status,
		Detail: info.detail,
		Code:   code,
	}
}

// problemFor is the single place that maps domain errors to problems.
// Anything it does not recognise (including calculator.ErrArity, which the
// handler guarantees never happens) is a 500 with no internals in detail.
func problemFor(err error) problem {
	switch {
	case errors.Is(err, calculator.ErrDivisionByZero):
		return newProblem(codeDivisionByZero)
	case errors.Is(err, calculator.ErrNegativeSquareRoot):
		return newProblem(codeNegativeSquareRoot)
	case errors.Is(err, calculator.ErrUndefinedResult):
		return newProblem(codeUndefinedResult)
	case errors.Is(err, calculator.ErrOverflow):
		return newProblem(codeResultOutOfRange)
	case errors.Is(err, calculator.ErrInvalidOperand):
		return newProblem(codeInvalidOperand)
	case errors.Is(err, calculator.ErrUnknownOperation):
		return newProblem(codeUnknownOperation)
	default:
		return newProblem(codeInternalError)
	}
}

func writeProblem(w http.ResponseWriter, p problem) {
	body, err := json.Marshal(p)
	if err != nil { // unreachable: problem only holds strings and an int
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	_, _ = w.Write(body)
}
