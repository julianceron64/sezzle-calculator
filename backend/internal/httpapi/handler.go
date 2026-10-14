// Package httpapi is the HTTP transport for the calculator service: routing,
// request decoding, problem+json errors and middleware.
package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/julianceron64/sezzle-calculator/backend/internal/calculator"
)

// Calculator is the domain behaviour the handlers need (defined by the consumer).
type Calculator interface {
	Calculate(op calculator.Op, operands ...float64) (float64, error)
	Operations() []calculator.Info
}

type api struct {
	calc   Calculator
	logger *slog.Logger
	infos  []calculator.Info
	arity  map[calculator.Op]int
}

// NewHandler returns the complete HTTP handler for the service.
func NewHandler(calc Calculator, logger *slog.Logger, opts ...Option) http.Handler {
	a := &api{calc: calc, logger: logger, infos: calc.Operations(), arity: map[calculator.Op]int{}}
	for _, info := range a.infos {
		a.arity[info.Name] = info.Arity
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/v1/{operation}", a.calculate)
	apiMux.HandleFunc("GET /api/v1/operations", a.operations)
	apiMux.HandleFunc("/api/", a.apiFallback)

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	root := http.NewServeMux()
	root.HandleFunc("/healthz", a.health)
	root.Handle("/api/", noStore(apiMux))
	if o.staticDir != "" {
		root.Handle("/", staticHandler(o.staticDir))
	} else {
		root.HandleFunc("/", http.NotFound)
	}
	return requestID(accessLog(logger, recoverer(logger, canonicalAPI(root))))
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeProblem(w, newProblem(codeMethodNotAllowed))
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	a.writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

type operationJSON struct {
	Name  calculator.Op `json:"name"`
	Arity int           `json:"arity"`
}

func (a *api) operations(w http.ResponseWriter, r *http.Request) {
	ops := make([]operationJSON, len(a.infos))
	for i, info := range a.infos {
		ops[i] = operationJSON{info.Name, info.Arity}
	}
	a.writeJSON(w, r, http.StatusOK, map[string][]operationJSON{"operations": ops})
}

// apiFallback handles every /api/ request the specific routes did not take:
// wrong methods on known routes are 405, everything else is 404.
func (a *api) apiFallback(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/")
	switch {
	case ok && rest == "operations":
		methodNotAllowed(w, http.MethodGet)
	case ok && !strings.Contains(rest, "/"):
		if _, known := a.arity[calculator.Op(rest)]; known {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		writeProblem(w, newProblem(codeNotFound))
	default:
		writeProblem(w, newProblem(codeNotFound))
	}
}

type resultJSON struct {
	Operation calculator.Op `json:"operation"`
	A         float64       `json:"a"`
	B         *float64      `json:"b,omitempty"` // pointer: b:0 must stay in the response
	Result    float64       `json:"result"`
}

func (a *api) calculate(w http.ResponseWriter, r *http.Request) {
	op := calculator.Op(r.PathValue("operation"))
	arity, ok := a.arity[op]
	if !ok {
		if op == "operations" { // POST to the read-only route
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeProblem(w, newProblem(codeUnknownOperation))
		return
	}

	operands, p := decodeRequest(w, r, arity)
	if p != nil {
		writeProblem(w, *p)
		return
	}
	result, err := a.calc.Calculate(op, operands...)
	if err != nil {
		p := problemFor(err)
		if p.Status >= http.StatusInternalServerError {
			a.logger.ErrorContext(r.Context(), "calculate failed", "operation", op, "error", err, "request_id", requestIDFrom(r.Context()))
		}
		writeProblem(w, p)
		return
	}

	res := resultJSON{Operation: op, A: operands[0], Result: result}
	if arity == 2 {
		res.B = &operands[1]
	}
	a.writeJSON(w, r, http.StatusOK, res)
}

// writeJSON marshals before writing, so an unencodable value (e.g. NaN from a
// misbehaving Calculator) becomes a clean 500 instead of a broken 200.
func (a *api) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		a.logger.ErrorContext(r.Context(), "encode response failed", "error", err, "request_id", requestIDFrom(r.Context()))
		writeProblem(w, newProblem(codeInternalError))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// canonicalAPI answers /api and any non-canonical path under /api/ (//, ., ..)
// with a NOT_FOUND problem. Without it ServeMux would redirect these before
// any handler runs, leaking a non-problem response (ADR-004).
func canonicalAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/api" || (strings.HasPrefix(p, "/api/") && path.Clean(p) != p) {
			w.Header().Set("Cache-Control", "no-store")
			writeProblem(w, newProblem(codeNotFound))
			return
		}
		next.ServeHTTP(w, r)
	})
}
