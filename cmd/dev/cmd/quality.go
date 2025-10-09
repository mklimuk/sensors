package cmd

import (
	"fmt"

	"github.com/gophertribe/devtool/test"
	"github.com/spf13/cobra"
)

func TestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := test.Test()
			if err != nil {
				return fmt.Errorf("failed to run tests: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func LintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Run linting",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := test.Lint()
			if err != nil {
				return fmt.Errorf("failed to run linting: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func IntegrationTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration-test",
		Short: "Run integration testing",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := test.Integ()
			if err != nil {
				return fmt.Errorf("failed to run integration testing: %w", err)
			}
			return nil
		},
	}
	return cmd
}
