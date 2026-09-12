package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <path|->",
		Short: "Review an already-generated plan (JSON file, binary plan file, or stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("tfp show: not yet implemented")
		},
	}
	return cmd
}
