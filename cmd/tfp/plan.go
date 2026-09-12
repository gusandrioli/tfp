package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [-- terraform-flags...]",
		Short: "Run terraform plan and review the result",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("tfp plan: not yet implemented")
		},
	}
	return cmd
}
