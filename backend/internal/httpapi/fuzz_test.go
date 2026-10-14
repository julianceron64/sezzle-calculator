package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// API-13: whatever the body, the handler never answers 5xx and always returns valid JSON.
func FuzzCalculateHandler(f *testing.F) {
	ops := []string{"add", "subtract", "multiply", "divide", "power", "sqrt", "percentage"}

	// Seeds: every error-table body, plus valid ones.
	seeds := []string{
		`{"a":1,"b":2}`, `{"a":9}`, `{"a":10,"b":0}`, `{"a":0,"b":-1}`, `{"a":-1}`,
		`{"a":-8,"b":0.3333333333333333}`, `{"a":1e308,"b":10}`, `{"a":1e400,"b":2}`,
		``, `{`, `[]`, `null`, `42`, `{"a":1,"b":2}{}`, `{"a":1,"b":2} x`,
		`{"a":1,"b":2,"c":3}`, `{"a":"2","b":2}`, `{"a":true,"b":2}`, `{"a":[],"b":2}`,
		`{"a":1}`, `{"a":null,"b":2}`, `{"a":1,"b":null}`, `{"a":4,"b":1}`,
		`{"a":1,"b":2,"p":"` + strings.Repeat("x", 4096) + `"}`,
	}
	for i, s := range seeds {
		f.Add(uint8(i), []byte(s))
	}

	h := realHandler()
	f.Fuzz(func(t *testing.T, opIdx uint8, body []byte) {
		rec := post(h, "/api/v1/"+ops[int(opIdx)%len(ops)], string(body))
		if rec.Code >= 500 {
			t.Fatalf("status %d for body %q: %s", rec.Code, body, rec.Body.String())
		}
		if !json.Valid(rec.Body.Bytes()) {
			t.Fatalf("response is not valid JSON for body %q: %s", body, rec.Body.String())
		}
	})
}
