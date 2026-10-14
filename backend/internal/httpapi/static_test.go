package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

const (
	indexHTML = "<!doctype html><title>INDEX</title>"
	appJS     = "console.log('app')"
)

// staticSite writes a small SPA build and returns a handler serving it. A
// secret file sits next to (not inside) the served directory.
func staticSite(t *testing.T) http.Handler {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "www")
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(dir, "index.html"):         indexHTML,
		filepath.Join(dir, "assets", "app.js"):   appJS,
		filepath.Join(parent, "secret.txt"):      "TOP-SECRET",
		filepath.Join(dir, "assets", "other.js"): "other",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return NewHandler(calculator.New(), discardLogger, WithStaticDir(dir))
}

// API-12
func TestStatic_ServesFilesAndFallsBack(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	tests := []struct {
		name string
		path string
		want string
	}{
		{"existing file", "/assets/app.js", appJS},
		{"root", "/", indexHTML},
		{"index.html directly", "/index.html", indexHTML},
		{"unknown path falls back", "/history/42", indexHTML},
		{"unknown path with extension falls back", "/missing.js", indexHTML},
		{"directory is not listed", "/assets/", indexHTML},
		{"directory without slash is not listed", "/assets", indexHTML},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := do(h, "GET", tc.path, "", "")
			if rec.Code != 200 || rec.Body.String() != tc.want {
				t.Fatalf("GET %s = %d %q, want 200 %q", tc.path, rec.Code, rec.Body.String(), tc.want)
			}
			if rec.Header().Get("X-Request-ID") == "" {
				t.Error("static response has no X-Request-ID")
			}
			if rec.Header().Get("Cache-Control") == "no-store" {
				t.Error("static response must not be forced uncacheable")
			}
		})
	}
}

func TestStatic_ContentTypesAndHead(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	if ct := do(h, "GET", "/assets/app.js", "", "").Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("js Content-Type = %q", ct)
	}
	if ct := do(h, "GET", "/", "", "").Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("html Content-Type = %q", ct)
	}
	if rec := do(h, "HEAD", "/", "", ""); rec.Code != 200 || rec.Body.Len() != 0 {
		t.Errorf("HEAD / = %d with %d body bytes", rec.Code, rec.Body.Len())
	}
}

// API-12: /api/ and /healthz are never shadowed by files or the fallback.
func TestStatic_ApiAndHealthzTakePrecedence(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	assertProblem(t, do(h, "GET", "/api/unknown", "", ""), 404, "NOT_FOUND")
	assertProblem(t, do(h, "GET", "/api/v1/modulo", "", ""), 404, "NOT_FOUND")
	assertProblem(t, do(h, "GET", "/api/v1/add", "", ""), 405, "METHOD_NOT_ALLOWED")
	if rec := do(h, "GET", "/healthz", "", ""); rec.Body.String() == indexHTML || rec.Code != 200 || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("/healthz shadowed: %d %q", rec.Code, rec.Body.String())
	}
	if rec := post(h, "/api/v1/add", `{"a":1,"b":2}`); rec.Code != 200 {
		t.Fatalf("API broken by static handler: %d", rec.Code)
	}
}

// API-12
func TestStatic_NonGetIs405Problem(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			rec := do(h, method, "/anything", jsonCT, `{}`)
			assertProblem(t, rec, 405, "METHOD_NOT_ALLOWED")
			if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
				t.Fatalf("Allow = %q", got)
			}
		})
	}
}

func TestStatic_NoPathTraversal(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	for _, path := range []string{"/../secret.txt", "/assets/../../secret.txt", "/%2e%2e/secret.txt", "/..%2fsecret.txt"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			if rec := do(h, "GET", path, "", ""); strings.Contains(rec.Body.String(), "TOP-SECRET") {
				t.Fatalf("GET %s leaked a file outside STATIC_DIR", path)
			}
		})
	}
}

// Without STATIC_DIR every non-API path other than /healthz is a plain 404.
func TestNoStatic_Returns404(t *testing.T) {
	t.Parallel()
	h := realHandler()
	for _, method := range []string{"GET", "POST"} {
		if rec := do(h, method, "/", "", ""); rec.Code != 404 {
			t.Errorf("%s / = %d, want 404", method, rec.Code)
		}
	}
}

// If STATIC_DIR has no index.html there is nothing to fall back to.
func TestStatic_MissingIndexIs404(t *testing.T) {
	t.Parallel()
	h := NewHandler(calculator.New(), discardLogger, WithStaticDir(t.TempDir()))
	if rec := do(h, "GET", "/anything", "", ""); rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// The /api guard must not swallow SPA routes that merely start with "api".
func TestStatic_ApiLookalikePathsStillFallBack(t *testing.T) {
	t.Parallel()
	h := staticSite(t)
	for _, path := range []string{"/apple", "/apis/docs"} {
		if rec := do(h, "GET", path, "", ""); rec.Code != 200 || rec.Body.String() != indexHTML {
			t.Errorf("GET %s = %d %q, want the SPA index", path, rec.Code, rec.Body.String())
		}
	}
}
