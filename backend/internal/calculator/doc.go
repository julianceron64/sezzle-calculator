// Package calculator implements the pure arithmetic domain of the service.
//
// It has no knowledge of HTTP or JSON (see docs/DECISIONS.md, ADR-003). Every
// operation is registered with its arity, and failures are reported through the
// sentinel errors declared in this package so callers can use errors.Is.
package calculator
