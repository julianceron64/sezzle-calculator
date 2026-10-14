package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func newReq(contentType, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/add", strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

const jsonCT = "application/json"

// API-03, API-14: request errors and precedence (rows 2-6 of the precedence list).
func TestDecodeRequest_Errors(t *testing.T) {
	t.Parallel()
	big := `{"a":1,"b":2,"pad":"` + strings.Repeat("x", 4096) + `"}`
	tests := []struct {
		name  string
		ct    string
		body  string
		arity int
		code  string
	}{
		// single violations
		{"wrong content type", "text/plain", `{"a":1,"b":2}`, 2, "UNSUPPORTED_MEDIA_TYPE"},
		{"missing content type", "", `{"a":1,"b":2}`, 2, "UNSUPPORTED_MEDIA_TYPE"},
		{"unparsable content type", "application/json; =", `{"a":1,"b":2}`, 2, "UNSUPPORTED_MEDIA_TYPE"},
		{"too large", jsonCT, big, 2, "PAYLOAD_TOO_LARGE"},
		{"empty body", jsonCT, ``, 2, "INVALID_JSON"},
		{"whitespace body", jsonCT, "  \n", 2, "INVALID_JSON"},
		{"malformed", jsonCT, `{"a":1,`, 2, "INVALID_JSON"},
		{"array", jsonCT, `[]`, 2, "INVALID_JSON"},
		{"null", jsonCT, `null`, 2, "INVALID_JSON"},
		{"number", jsonCT, `42`, 2, "INVALID_JSON"},
		{"trailing data", jsonCT, `{"a":1,"b":2} {"a":1}`, 2, "INVALID_JSON"},
		{"trailing garbage", jsonCT, `{"a":1,"b":2} x`, 2, "INVALID_JSON"},
		{"unknown field", jsonCT, `{"a":1,"b":2,"c":3}`, 2, "UNKNOWN_FIELD"},
		{"string operand", jsonCT, `{"a":"2","b":2}`, 2, "INVALID_OPERAND"},
		{"bool operand", jsonCT, `{"a":true,"b":2}`, 2, "INVALID_OPERAND"},
		{"array operand", jsonCT, `{"a":[],"b":2}`, 2, "INVALID_OPERAND"},
		{"object operand", jsonCT, `{"a":{},"b":2}`, 2, "INVALID_OPERAND"},
		{"out of range operand", jsonCT, `{"a":1e400,"b":2}`, 2, "INVALID_OPERAND"},
		{"negative out of range", jsonCT, `{"a":-1e400,"b":2}`, 2, "INVALID_OPERAND"},
		{"missing a", jsonCT, `{"b":2}`, 2, "MISSING_OPERAND"},
		{"missing b", jsonCT, `{"a":1}`, 2, "MISSING_OPERAND"},
		{"null a", jsonCT, `{"a":null,"b":2}`, 2, "MISSING_OPERAND"},
		{"null b", jsonCT, `{"a":1,"b":null}`, 2, "MISSING_OPERAND"},
		{"unary missing a", jsonCT, `{}`, 1, "MISSING_OPERAND"},
		{"unary with b", jsonCT, `{"a":4,"b":2}`, 1, "UNEXPECTED_OPERAND"},
		{"unary with null b", jsonCT, `{"a":1,"b":null}`, 1, "UNEXPECTED_OPERAND"},

		// API-14: several rules violated at once, the earliest wins
		{"415 beats malformed body", "text/plain", `{{{`, 2, "UNSUPPORTED_MEDIA_TYPE"},
		{"413 beats malformed body", jsonCT, `{{{` + strings.Repeat("x", 5000), 2, "PAYLOAD_TOO_LARGE"},
		{"invalid json beats unknown field", jsonCT, `{"zzz":1,`, 2, "INVALID_JSON"},
		{"trailing data beats unknown field", jsonCT, `{"zzz":1} {}`, 2, "INVALID_JSON"},
		{"unknown field beats wrong type (field first)", jsonCT, `{"zzz":1,"a":"x"}`, 2, "UNKNOWN_FIELD"},
		{"unknown field beats wrong type (field last)", jsonCT, `{"a":"x","zzz":1}`, 2, "UNKNOWN_FIELD"},
		{"unknown field beats missing operand", jsonCT, `{"zzz":1}`, 2, "UNKNOWN_FIELD"},
		{"invalid a beats missing b", jsonCT, `{"a":"x"}`, 2, "INVALID_OPERAND"},
		{"missing a beats invalid b", jsonCT, `{"b":"x"}`, 2, "MISSING_OPERAND"},
		{"invalid a beats invalid b order", jsonCT, `{"b":true,"a":"x"}`, 2, "INVALID_OPERAND"},
		{"unary: missing a beats unexpected b", jsonCT, `{"b":2}`, 1, "MISSING_OPERAND"},
		{"unary: invalid a beats unexpected b", jsonCT, `{"a":"x","b":2}`, 1, "INVALID_OPERAND"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, p := decodeRequest(httptest.NewRecorder(), newReq(tc.ct, tc.body), tc.arity)
			if p == nil {
				t.Fatalf("decodeRequest succeeded with %v, want %s", got, tc.code)
			}
			if p.Code != tc.code {
				t.Fatalf("code = %s, want %s", p.Code, tc.code)
			}
		})
	}
}

func TestDecodeRequest_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ct    string
		body  string
		arity int
		want  []float64
	}{
		{"binary", jsonCT, `{"a":10,"b":4}`, 2, []float64{10, 4}},
		{"zero operands", jsonCT, `{"a":0,"b":0}`, 2, []float64{0, 0}},
		{"exponent and negative", jsonCT, `{"a":-1.5e2,"b":2E-1}`, 2, []float64{-150, 0.2}},
		{"unary", jsonCT, `{"a":9}`, 1, []float64{9}},
		{"charset param", "application/json; charset=utf-8", `{"a":1,"b":2}`, 2, []float64{1, 2}},
		{"case-insensitive media type", "Application/JSON", `{"a":1,"b":2}`, 2, []float64{1, 2}},
		{"surrounding whitespace", jsonCT, " \n{\"a\":1,\"b\":2}\n ", 2, []float64{1, 2}},
		{"exactly 4 KiB", jsonCT, `{"a":1,"b":2}` + strings.Repeat(" ", 4096-13), 2, []float64{1, 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, p := decodeRequest(httptest.NewRecorder(), newReq(tc.ct, tc.body), tc.arity)
			if p != nil {
				t.Fatalf("unexpected problem: %+v", p)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("operands = %v, want %v", got, tc.want)
			}
		})
	}
}

// API-14: the detector matches encoding/json's untyped error by text, so this
// test fails if a Go upgrade changes that text.
func TestIsUnknownFieldError(t *testing.T) {
	t.Parallel()
	var v struct{ A int }
	dec := json.NewDecoder(strings.NewReader(`{"zzz":1}`))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err == nil || !isUnknownFieldError(err) {
		t.Fatalf("real unknown-field error not detected: %v", err)
	}
	var syntax struct{}
	if err := json.Unmarshal([]byte(`{`), &syntax); err == nil || isUnknownFieldError(err) {
		t.Fatalf("syntax error misdetected as unknown field: %v", err)
	}
	if isUnknownFieldError(nil) {
		t.Fatal("nil error detected as unknown field")
	}
}
