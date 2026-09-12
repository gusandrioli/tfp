package main

import (
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "show <path|->",
		Short: "Review an already-generated plan (JSON file, binary plan file, or stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := loadPlanJSON(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return renderPlan(cmd, data, output)
		},
	}
	cmd.Flags().StringVar(&output, "output", "auto", "auto, text, or json")
	return cmd
}
