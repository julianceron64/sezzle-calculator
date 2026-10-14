package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const waitTimeout = 10 * time.Second

func getenvFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// freePort obtains a port by listening on :0, reading it and closing the listener (API-11).
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// startServer runs the server on a free port and returns its base URL once /healthz answers.
// The returned stop function cancels the context and returns run's result.
func startServer(t *testing.T, extraEnv map[string]string) (baseURL, port string, stop func() error) {
	t.Helper()
	port = freePort(t)
	env := map[string]string{"PORT": port}
	for k, v := range extraEnv {
		env[k] = v
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"server"}, getenvFrom(env), io.Discard) }()

	baseURL = "http://127.0.0.1:" + port
	// Event-driven readiness: poll /healthz until it answers, run fails, or the deadline passes.
	deadline := time.After(waitTimeout)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for ready := false; !ready; {
		select {
		case err := <-done:
			cancel()
			t.Fatalf("run returned before becoming ready: %v", err)
		case <-deadline:
			cancel()
			t.Fatal("server did not become ready in time")
		case <-tick.C:
			if resp, err := http.Get(baseURL + "/healthz"); err == nil {
				resp.Body.Close()
				ready = resp.StatusCode == http.StatusOK
			}
		}
	}

	var stopped bool
	stop = func() error {
		if stopped {
			return nil
		}
		stopped = true
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(waitTimeout):
			t.Fatal("run did not return after context cancel")
			return nil
		}
	}
	t.Cleanup(func() { _ = stop() })
	return baseURL, port, stop
}

// API-11
func TestRun_ServesHealthzAndShutsDown(t *testing.T) {
	t.Parallel()
	base, _, stop := startServer(t, nil)

	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("healthz: %d %s", resp.StatusCode, body)
	}
	if err := stop(); err != nil {
		t.Fatalf("run returned %v after context cancel, want nil", err)
	}
}

// The real wiring: calculator behind the API, SPA from STATIC_DIR.
func TestRun_WiresAPIAndStatic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<title>INDEX</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, _, _ := startServer(t, map[string]string{"STATIC_DIR": dir})

	resp, err := http.Post(base+"/api/v1/divide", "application/json", strings.NewReader(`{"a":10,"b":4}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"result":2.5`) {
		t.Fatalf("divide: %d %s", resp.StatusCode, body)
	}

	resp, err = http.Get(base + "/some/client/route")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), "INDEX") {
		t.Fatalf("fallback: %d %s", resp.StatusCode, body)
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	t.Parallel()
	for name, env := range map[string]map[string]string{
		"port zero":     {"PORT": "0"},
		"port too big":  {"PORT": "70000"},
		"bad log level": {"LOG_LEVEL": "loud"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := run(t.Context(), []string{"server"}, getenvFrom(env), io.Discard); err == nil {
				t.Fatal("run succeeded with invalid config")
			}
		})
	}
}

func TestRun_UnknownFlag(t *testing.T) {
	t.Parallel()
	if err := run(t.Context(), []string{"server", "-nope"}, getenvFrom(nil), io.Discard); err == nil {
		t.Fatal("run succeeded with an unknown flag")
	}
}

func TestRun_PortInUse(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	if err := run(t.Context(), []string{"server"}, getenvFrom(map[string]string{"PORT": port}), io.Discard); err == nil {
		t.Fatal("run succeeded on a port that is already in use")
	}
}

func TestRun_Healthcheck(t *testing.T) {
	t.Parallel()
	t.Run("healthy server", func(t *testing.T) {
		t.Parallel()
		_, port, _ := startServer(t, nil)
		if err := run(t.Context(), []string{"server", "-healthcheck"}, getenvFrom(map[string]string{"PORT": port}), io.Discard); err != nil {
			t.Fatalf("healthcheck failed against a healthy server: %v", err)
		}
	})
	t.Run("nothing listening", func(t *testing.T) {
		t.Parallel()
		env := map[string]string{"PORT": freePort(t)}
		if err := run(t.Context(), []string{"server", "-healthcheck"}, getenvFrom(env), io.Discard); err == nil {
			t.Fatal("healthcheck succeeded with nothing listening")
		}
	})
	t.Run("unhealthy status", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()
		u, _ := url.Parse(srv.URL)
		env := map[string]string{"PORT": u.Port()}
		if err := run(t.Context(), []string{"server", "-healthcheck"}, getenvFrom(env), io.Discard); err == nil {
			t.Fatal("healthcheck succeeded against a 503")
		}
	})
	t.Run("invalid config", func(t *testing.T) {
		t.Parallel()
		if err := run(t.Context(), []string{"server", "-healthcheck"}, getenvFrom(map[string]string{"PORT": "x"}), io.Discard); err == nil {
			t.Fatal("healthcheck succeeded with invalid PORT")
		}
	})
}
