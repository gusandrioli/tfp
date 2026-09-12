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
	if errors.Is(err, context.Canceled) {
		return 130
	}
	var usage usageError
	if errors.As(err, &usage) {
		return 2
	}
	return 1
}
