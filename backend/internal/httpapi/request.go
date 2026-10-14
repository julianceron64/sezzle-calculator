package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const maxBodyBytes = 4 << 10

type operandState int

const (
	operandAbsent operandState = iota // key not in the body
	operandNull
	operandNumber
	operandInvalid // not a number, or outside float64 range
)

// operand records what the body said about one operand. It distinguishes a
// missing key from null (a unary op rejects a present b, even null) and never
// fails in UnmarshalJSON, so a bad operand cannot hide an unknown-field error
// that appears later in the body.
type operand struct {
	state operandState
	value float64
}

func (o *operand) UnmarshalJSON(b []byte) error {
	s := string(bytes.TrimSpace(b))
	if s == "null" {
		o.state = operandNull
		return nil
	}
	o.state = operandInvalid
	if s == "" || (s[0] != '-' && (s[0] < '0' || s[0] > '9')) {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(v, 0) {
		return nil
	}
	o.state, o.value = operandNumber, v
	return nil
}

type calcRequest struct {
	A operand `json:"a"`
	B operand `json:"b"`
}

// isUnknownFieldError reports whether err comes from DisallowUnknownFields.
// encoding/json returns it untyped, so this is the one place that matches its
// message text; request_test.go fails if a Go upgrade changes it.
func isUnknownFieldError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "json: unknown field ")
}

// decodeRequest validates the request in the precedence order of spec 002:
// 415, 413 / INVALID_JSON, UNKNOWN_FIELD, INVALID/MISSING_OPERAND (a, then b),
// UNEXPECTED_OPERAND. It returns the operands in order (one for a unary op).
func decodeRequest(w http.ResponseWriter, r *http.Request, arity int) ([]float64, *problem) {
	fail := func(code string) ([]float64, *problem) {
		p := newProblem(code)
		return nil, &p
	}

	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return fail(codeUnsupportedMediaType)
	}

	// Read the whole body first so the size limit wins over syntax errors.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fail(codePayloadTooLarge)
		}
		return fail(codeInvalidJSON)
	}
	if trimmed := bytes.TrimSpace(body); len(trimmed) == 0 || trimmed[0] != '{' {
		return fail(codeInvalidJSON)
	}

	var req calcRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	decodeErr := dec.Decode(&req)
	if decodeErr != nil && !isUnknownFieldError(decodeErr) {
		return fail(codeInvalidJSON)
	}
	if _, err := dec.Token(); err != io.EOF { // trailing data after the object
		return fail(codeInvalidJSON)
	}
	if decodeErr != nil {
		return fail(codeUnknownField)
	}

	operands := make([]float64, 0, 2)
	for _, o := range []operand{req.A, req.B}[:arity] {
		switch o.state {
		case operandNumber:
			operands = append(operands, o.value)
		case operandInvalid:
			return fail(codeInvalidOperand)
		default:
			return fail(codeMissingOperand)
		}
	}
	if arity == 1 && req.B.state != operandAbsent {
		return fail(codeUnexpectedOperand)
	}
	return operands, nil
}
