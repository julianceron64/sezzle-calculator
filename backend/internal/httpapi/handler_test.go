package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func realHandler() http.Handler {
	return NewHandler(calculator.New(), discardLogger)
}

func do(h http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	return do(h, http.MethodPost, path, jsonCT, body)
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body is not a JSON object: %v\n%s", err, rec.Body.String())
	}
	return m
}

// assertProblem checks status, code, type and content type of a problem response.
func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d\n%s", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
	m := decodeMap(t, rec)
	wantType := "/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-"))
	if m["code"] != code || m["type"] != wantType || m["status"] != float64(status) {
		t.Fatalf("problem = %v, want code %s type %s status %d", m, code, wantType, status)
	}
	if s, _ := m["title"].(string); s == "" {
		t.Fatalf("empty title: %v", m)
	}
	if s, _ := m["detail"].(string); s == "" {
		t.Fatalf("empty detail: %v", m)
	}
}

// API-01
func TestCalculate_AllOperations(t *testing.T) {
	t.Parallel()
	h := realHandler()
	tests := []struct {
		op     string
		body   string
		a      float64
		b      *float64
		result float64
	}{
		{"add", `{"a":10,"b":4}`, 10, ptr(4), 14},
		{"subtract", `{"a":10,"b":4}`, 10, ptr(4), 6},
		{"multiply", `{"a":10,"b":4}`, 10, ptr(4), 40},
		{"divide", `{"a":10,"b":4}`, 10, ptr(4), 2.5},
		{"power", `{"a":2,"b":10}`, 2, ptr(10), 1024},
		{"sqrt", `{"a":9}`, 9, nil, 3},
		{"percentage", `{"a":20,"b":150}`, 20, ptr(150), 30},
	}
	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			t.Parallel()
			rec := post(h, "/api/v1/"+tc.op, tc.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q", ct)
			}
			m := decodeMap(t, rec)
			if m["operation"] != tc.op || m["a"] != tc.a || m["result"] != tc.result {
				t.Fatalf("body = %v", m)
			}
			if tc.b == nil {
				if _, has := m["b"]; has {
					t.Fatalf("unary response has b: %v", m)
				}
			} else if m["b"] != *tc.b {
				t.Fatalf("b = %v, want %v", m["b"], *tc.b)
			}
		})
	}
}

func ptr(f float64) *float64 { return &f }

// API-02
func TestCalculate_BZeroIsKept(t *testing.T) {
	t.Parallel()
	rec := post(realHandler(), "/api/v1/add", `{"a":5,"b":0}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"b":0`) {
		t.Fatalf("b:0 missing from response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCalculate_UnaryHasNoB(t *testing.T) {
	t.Parallel()
	rec := post(realHandler(), "/api/v1/sqrt", `{"a":0}`)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), `"b"`) {
		t.Fatalf("unary response: %d %s", rec.Code, rec.Body.String())
	}
}

// API-03, API-04: every row of the error table through the real handler.
func TestProblems_ErrorTable(t *testing.T) {
	t.Parallel()
	h := realHandler()
	tests := []struct {
		name   string
		method string
		path   string
		ct     string
		body   string
		status int
		code   string
		allow  string
	}{
		{"415", "POST", "/api/v1/add", "text/plain", `{"a":1,"b":2}`, 415, "UNSUPPORTED_MEDIA_TYPE", ""},
		{"413", "POST", "/api/v1/add", jsonCT, `{"a":1,"b":2,"p":"` + strings.Repeat("x", 4096) + `"}`, 413, "PAYLOAD_TOO_LARGE", ""},
		{"400 empty", "POST", "/api/v1/add", jsonCT, ``, 400, "INVALID_JSON", ""},
		{"400 malformed", "POST", "/api/v1/add", jsonCT, `{`, 400, "INVALID_JSON", ""},
		{"400 not an object", "POST", "/api/v1/add", jsonCT, `[1,2]`, 400, "INVALID_JSON", ""},
		{"400 trailing", "POST", "/api/v1/add", jsonCT, `{"a":1,"b":2}{}`, 400, "INVALID_JSON", ""},
		{"400 unknown field", "POST", "/api/v1/add", jsonCT, `{"a":1,"b":2,"c":3}`, 400, "UNKNOWN_FIELD", ""},
		{"400 string operand", "POST", "/api/v1/add", jsonCT, `{"a":"2","b":2}`, 400, "INVALID_OPERAND", ""},
		{"400 out of range", "POST", "/api/v1/add", jsonCT, `{"a":1e400,"b":2}`, 400, "INVALID_OPERAND", ""},
		{"400 missing", "POST", "/api/v1/add", jsonCT, `{"a":1}`, 400, "MISSING_OPERAND", ""},
		{"400 null", "POST", "/api/v1/add", jsonCT, `{"a":1,"b":null}`, 400, "MISSING_OPERAND", ""},
		{"400 unexpected b", "POST", "/api/v1/sqrt", jsonCT, `{"a":4,"b":1}`, 400, "UNEXPECTED_OPERAND", ""},
		{"404 unknown op", "POST", "/api/v1/modulo", jsonCT, `{"a":1,"b":2}`, 404, "UNKNOWN_OPERATION", ""},
		{"404 unknown op, bad body", "POST", "/api/v1/modulo", "text/plain", `{{`, 404, "UNKNOWN_OPERATION", ""},
		{"404 GET unknown op", "GET", "/api/v1/modulo", "", ``, 404, "NOT_FOUND", ""},
		{"404 unknown api route", "GET", "/api/nope", "", ``, 404, "NOT_FOUND", ""},
		{"404 api root", "GET", "/api/", "", ``, 404, "NOT_FOUND", ""},
		{"404 v1 only", "GET", "/api/v1", "", ``, 404, "NOT_FOUND", ""},
		{"404 v1 slash", "POST", "/api/v1/", jsonCT, `{}`, 404, "NOT_FOUND", ""},
		{"404 trailing slash", "POST", "/api/v1/add/", jsonCT, `{"a":1,"b":2}`, 404, "NOT_FOUND", ""},
		{"404 extra segment", "POST", "/api/v1/add/x", jsonCT, `{"a":1,"b":2}`, 404, "NOT_FOUND", ""},
		{"404 other version", "POST", "/api/v2/add", jsonCT, `{"a":1,"b":2}`, 404, "NOT_FOUND", ""},
		{"405 GET operation", "GET", "/api/v1/add", "", ``, 405, "METHOD_NOT_ALLOWED", "POST"},
		{"405 PUT operation", "PUT", "/api/v1/sqrt", jsonCT, `{"a":1}`, 405, "METHOD_NOT_ALLOWED", "POST"},
		{"405 POST operations", "POST", "/api/v1/operations", jsonCT, `{"a":1,"b":2}`, 405, "METHOD_NOT_ALLOWED", "GET"},
		{"405 DELETE operations", "DELETE", "/api/v1/operations", "", ``, 405, "METHOD_NOT_ALLOWED", "GET"},
		{"422 division by zero", "POST", "/api/v1/divide", jsonCT, `{"a":10,"b":0}`, 422, "DIVISION_BY_ZERO", ""},
		{"422 negative sqrt", "POST", "/api/v1/sqrt", jsonCT, `{"a":-1}`, 422, "NEGATIVE_SQUARE_ROOT", ""},
		{"422 undefined", "POST", "/api/v1/power", jsonCT, `{"a":-8,"b":0.3333333333333333}`, 422, "UNDEFINED_RESULT", ""},
		{"422 overflow", "POST", "/api/v1/multiply", jsonCT, `{"a":1e308,"b":10}`, 422, "RESULT_OUT_OF_RANGE", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := do(h, tc.method, tc.path, tc.ct, tc.body)
			assertProblem(t, rec, tc.status, tc.code)
			if got := rec.Header().Get("Allow"); got != tc.allow {
				t.Fatalf("Allow = %q, want %q", got, tc.allow)
			}
		})
	}
}

// API-05
func TestOperations_MatchesCalculator(t *testing.T) {
	t.Parallel()
	rec := do(realHandler(), "GET", "/api/v1/operations", "", "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status/type: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var got struct {
		Operations []struct {
			Name  string `json:"name"`
			Arity int    `json:"arity"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := calculator.New().Operations()
	if len(got.Operations) != len(want) {
		t.Fatalf("got %d operations, want %d", len(got.Operations), len(want))
	}
	for i, w := range want {
		if got.Operations[i].Name != string(w.Name) || got.Operations[i].Arity != w.Arity {
			t.Errorf("operations[%d] = %+v, want %+v", i, got.Operations[i], w)
		}
	}
}

// API-06
func TestHealthz(t *testing.T) {
	t.Parallel()
	h := realHandler()
	rec := do(h, "GET", "/healthz", "", "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("healthz: %d %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	rec = do(h, "POST", "/healthz", jsonCT, `{}`)
	assertProblem(t, rec, 405, "METHOD_NOT_ALLOWED")
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q", got)
	}
}

// fakeCalc lets tests drive the handler with arbitrary calculator behaviour.
type fakeCalc struct {
	calc func(op calculator.Op, operands ...float64) (float64, error)
}

func (f fakeCalc) Calculate(op calculator.Op, operands ...float64) (float64, error) {
	return f.calc(op, operands...)
}

func (fakeCalc) Operations() []calculator.Info { return calculator.New().Operations() }

func failingWith(err error) Calculator {
	return fakeCalc{func(calculator.Op, ...float64) (float64, error) { return 0, err }}
}

// API-09: error mapping is proven with a fake, independent of the real domain.
func TestErrorMapping_FakeCalculator(t *testing.T) {
	t.Parallel()
	secret := errors.New("secret internal failure: db password")
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
		{"invalid operand", calculator.ErrInvalidOperand, 400, "INVALID_OPERAND"},
		{"unknown operation", calculator.ErrUnknownOperation, 404, "UNKNOWN_OPERATION"},
		{"arity", calculator.ErrArity, 500, "INTERNAL_ERROR"},
		{"unknown error", secret, 500, "INTERNAL_ERROR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := post(NewHandler(failingWith(tc.err), discardLogger), "/api/v1/add", `{"a":1,"b":2}`)
			assertProblem(t, rec, tc.status, tc.code)
			if tc.status == 500 {
				body := rec.Body.String()
				for _, leak := range []string{"secret", "password", "wrong number", "arity"} {
					if strings.Contains(body, leak) {
						t.Fatalf("500 body leaks %q: %s", leak, body)
					}
				}
			}
		})
	}
}

// API-09: a calculator that returns a non-finite number must not produce a broken 200.
func TestCalculate_NonFiniteResultIs500(t *testing.T) {
	t.Parallel()
	for name, v := range map[string]float64{"NaN": math.NaN(), "Inf": math.Inf(1)} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := NewHandler(fakeCalc{func(calculator.Op, ...float64) (float64, error) { return v, nil }}, discardLogger)
			assertProblem(t, post(h, "/api/v1/add", `{"a":1,"b":2}`), 500, "INTERNAL_ERROR")
		})
	}
}

// API-04: non-canonical /api paths get a problem, never a ServeMux redirect or text/html body.
func TestRouting_NonCanonicalAPIPaths(t *testing.T) {
	t.Parallel()
	h := realHandler()
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"bare /api", "GET", "/api"},
		{"bare /api POST", "POST", "/api"},
		{"double slash", "POST", "/api//v1/add"},
		{"dot-dot segment", "POST", "/api/v1/../v1/add"},
		{"dot segment", "GET", "/api/./x"},
		{"trailing slash", "POST", "/api/v1/add/"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := do(h, tc.method, tc.path, jsonCT, `{"a":1,"b":2}`)
			assertProblem(t, rec, 404, "NOT_FOUND")
			if loc := rec.Header().Get("Location"); loc != "" {
				t.Errorf("redirected to %q", loc)
			}
			if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Request-ID") == "" {
				t.Errorf("missing headers: %v", rec.Header())
			}
		})
	}
}
