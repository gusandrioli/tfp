// Package main is tfp's CLI entry point.
package main

import (
	"context"
	"errors"
)

// usageError marks an error as a CLI usage mistake (bad flags/args),
// distinct from a runtime failure, so Execute can map it to exit code 2.
type usageError struct{ error }

func newUsageError(err error) error { return usageError{err} }

func exitCodeForError(err error) int {
	switch {
	case errors.Is(err, context.Canceled):
		return 130
	case errors.As(err, &usageError{}):
		return 2
	default:
		return 1
	}
}
