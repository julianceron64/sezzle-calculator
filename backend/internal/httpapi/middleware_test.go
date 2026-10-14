package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

// syncBuffer is a bytes.Buffer that is safe to write from server goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func jsonLogger(buf *syncBuffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

// logLines parses the JSON log lines written so far.
func logLines(t *testing.T, buf *syncBuffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", raw)
		}
		lines = append(lines, m)
	}
	return lines
}

var generatedID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// API-07: X-Request-ID on every response, echoed when valid, generated otherwise.
func TestRequestID(t *testing.T) {
	t.Parallel()
	h := realHandler()
	tests := []struct {
		name     string
		header   string
		wantEcho bool
	}{
		{"valid id is echoed", "abc-123-XYZ", true},
		{"64 chars is echoed", strings.Repeat("a", 64), true},
		{"65 chars is replaced", strings.Repeat("a", 65), false},
		{"bad characters are replaced", "a b", false},
		{"underscore is replaced", "a_b", false},
		{"newline-ish is replaced", "a\tb", false},
		{"absent is generated", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", "/healthz", nil)
			if tc.header != "" {
				r.Header.Set("X-Request-ID", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)
			got := rec.Header().Get("X-Request-ID")
			if tc.wantEcho && got != tc.header {
				t.Fatalf("X-Request-ID = %q, want echo of %q", got, tc.header)
			}
			if !tc.wantEcho && !generatedID.MatchString(got) {
				t.Fatalf("X-Request-ID = %q, want 32 hex chars", got)
			}
		})
	}
}

func TestRequestID_GeneratedIDsDiffer(t *testing.T) {
	t.Parallel()
	h := realHandler()
	a := do(h, "GET", "/healthz", "", "").Header().Get("X-Request-ID")
	b := do(h, "GET", "/healthz", "", "").Header().Get("X-Request-ID")
	if a == b {
		t.Fatalf("two requests got the same generated id %q", a)
	}
}

// API-07: X-Request-ID is on every response; no-store only under /api/.
func TestHeaders_NoStoreAndRequestID(t *testing.T) {
	t.Parallel()
	h := realHandler()
	tests := []struct {
		name    string
		method  string
		path    string
		noStore bool
	}{
		{"calculate 200", "POST", "/api/v1/add", true},
		{"operations", "GET", "/api/v1/operations", true},
		{"api 404", "GET", "/api/nope", true},
		{"api 405", "GET", "/api/v1/add", true},
		{"healthz", "GET", "/healthz", false},
		{"non-api 404", "GET", "/somewhere", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := do(h, tc.method, tc.path, jsonCT, `{"a":1,"b":2}`)
			if rec.Header().Get("X-Request-ID") == "" {
				t.Error("missing X-Request-ID")
			}
			if got := rec.Header().Get("Cache-Control") == "no-store"; got != tc.noStore {
				t.Errorf("Cache-Control no-store = %v, want %v", got, tc.noStore)
			}
		})
	}
}

// API-07: the access log has method, path, status, duration and request_id, and never the body.
func TestAccessLog(t *testing.T) {
	t.Parallel()
	buf := &syncBuffer{}
	h := NewHandler(calculator.New(), jsonLogger(buf))
	r := httptest.NewRequest("POST", "/api/v1/divide", strings.NewReader(`{"a":1,"b":0,"note":"topsecret"}`))
	r.Header.Set("Content-Type", jsonCT)
	r.Header.Set("X-Request-ID", "req-42")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if strings.Contains(buf.String(), "topsecret") {
		t.Fatalf("log contains the request body: %s", buf.String())
	}
	var found bool
	for _, l := range logLines(t, buf) {
		if l["msg"] != "request" {
			continue
		}
		found = true
		if l["method"] != "POST" || l["path"] != "/api/v1/divide" || l["status"] != float64(400) || l["request_id"] != "req-42" {
			t.Errorf("access log = %v", l)
		}
		if _, ok := l["duration"]; !ok {
			t.Errorf("access log has no duration: %v", l)
		}
	}
	if !found {
		t.Fatalf("no access log line: %s", buf.String())
	}
}

// API-08: a panic is a 500 problem, logged with the request id, and the server keeps serving.
func TestRecover_Panic500AndKeepsServing(t *testing.T) {
	t.Parallel()
	buf := &syncBuffer{}
	calc := fakeCalc{func(op calculator.Op, operands ...float64) (float64, error) {
		if op == calculator.OpAdd {
			panic("boom: secret internals")
		}
		return 7, nil
	}}
	srv := httptest.NewServer(NewHandler(calc, jsonLogger(buf)))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/add", strings.NewReader(`{"a":1,"b":2}`))
	req.Header.Set("Content-Type", jsonCT)
	req.Header.Set("X-Request-ID", "panic-req-1")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var p map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&p)
	if resp.StatusCode != 500 || p["code"] != "INTERNAL_ERROR" || resp.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("panic response: %d %v", resp.StatusCode, p)
	}
	if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Request-ID") != "panic-req-1" {
		t.Errorf("headers lost on panic: %v", resp.Header)
	}
	if strings.Contains(p["detail"].(string), "boom") {
		t.Errorf("detail leaks the panic value: %v", p["detail"])
	}

	var logged bool
	for _, l := range logLines(t, buf) {
		if l["level"] == "ERROR" && l["request_id"] == "panic-req-1" && strings.Contains(l["msg"].(string)+toString(l["panic"]), "boom") {
			logged = true
		}
	}
	if !logged {
		t.Fatalf("panic not logged with request_id: %s", buf.String())
	}

	// The server keeps serving.
	resp2, err := srv.Client().Post(srv.URL+"/api/v1/sqrt", jsonCT, strings.NewReader(`{"a":4}`))
	if err != nil {
		t.Fatalf("server stopped serving after a panic: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("follow-up request status = %d", resp2.StatusCode)
	}
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
