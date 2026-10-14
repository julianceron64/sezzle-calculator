package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

// API-03, API-09: one function maps domain errors to problems.
func TestProblemFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"division by zero", calculator.ErrDivisionByZero, 422, "DIVISION_BY_ZERO"},
		{"negative sqrt", calculator.ErrNegativeSquareRoot, 422, "NEGATIVE_SQUARE_ROOT"},
		{"undefined", calculator.ErrUndefinedResult, 422, "UNDEFINED_RESULT"},
		{"overflow", calculator.ErrOverflow, 422, "RESULT_OUT_OF_RANGE"},
		{"wrapped overflow", fmt.Errorf("multiply: %w", calculator.ErrOverflow), 422, "RESULT_OUT_OF_RANGE"},
		{"invalid operand (defensive)", calculator.ErrInvalidOperand, 400, "INVALID_OPERAND"},
		{"unknown operation (defensive)", calculator.ErrUnknownOperation, 404, "UNKNOWN_OPERATION"},
		{"arity (defensive)", calculator.ErrArity, 500, "INTERNAL_ERROR"},
		{"unknown error", errors.New("db password is hunter2"), 500, "INTERNAL_ERROR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := problemFor(tc.err)
			if p.Status != tc.status || p.Code != tc.code {
				t.Fatalf("problemFor = %d %s, want %d %s", p.Status, p.Code, tc.status, tc.code)
			}
			if tc.status == 500 && (strings.Contains(p.Detail, "hunter2") || strings.Contains(p.Detail, "operands")) {
				t.Fatalf("500 detail leaks internals: %q", p.Detail)
			}
		})
	}
}

func TestNewProblem_TypeIsKebabCaseOfCode(t *testing.T) {
	t.Parallel()
	p := newProblem(codeDivisionByZero)
	if p.Type != "/problems/division-by-zero" || p.Status != 422 || p.Title == "" || p.Detail == "" {
		t.Fatalf("unexpected problem: %+v", p)
	}
}

func TestWriteProblem(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeProblem(rec, newProblem(codeMethodNotAllowed))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	for _, k := range []string{"type", "title", "status", "detail", "code"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing key %q in %v", k, got)
		}
	}
}
