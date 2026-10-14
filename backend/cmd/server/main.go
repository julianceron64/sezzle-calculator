// Command server runs the calculator HTTP API and, optionally, the built SPA.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
	"github.com/julianceron64/sezzle-calculator/backend/internal/config"
	"github.com/julianceron64/sezzle-calculator/backend/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args, os.Getenv, os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run is main without process globals, so tests can drive it. It returns nil
// after a graceful shutdown triggered by ctx.
func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stdout)
	healthcheck := flags.Bool("healthcheck", false, "GET /healthz on localhost and exit 0 if it answers 200")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if *healthcheck {
		return checkHealth(ctx, cfg.Port)
	}

	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	var opts []httpapi.Option
	if cfg.StaticDir != "" {
		opts = append(opts, httpapi.WithStaticDir(cfg.StaticDir))
	}
	srv := &http.Server{
		Handler:           httpapi.NewHandler(calculator.New(), logger, opts...),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen before serving so a bind failure is returned, not logged from a goroutine.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return err
	}
	logger.Info("listening", "addr", ln.Addr().String(), "static_dir", cfg.StaticDir)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		return err // Serve only returns on failure here: Shutdown has not been called yet
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// checkHealth is the Docker HEALTHCHECK: the distroless image has no curl.
func checkHealth(ctx context.Context, port int) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/healthz", port), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}
