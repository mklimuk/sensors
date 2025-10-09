package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func ChangelogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate or update CHANGELOG.md from git history",
		Long: `Generate CHANGELOG.md using git-chglog based on conventional commits.

This command requires git-chglog to be installed. Run the setup script first:
  ./scripts/setup-changelog.sh

The changelog is generated from git commit history following the Conventional
Commits specification. Commits should follow the format:
  <type>[optional scope]: <description>

Supported types: feat, fix, docs, refactor, test, perf, build, ci, chore

Examples:
  # Generate full changelog
  dev changelog

  # Generate for next version
  dev changelog --next v1.2.0

  # Generate for specific tag
  dev changelog --tag v1.0.0

  # Output to different file
  dev changelog --output CHANGES.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("could not get output flag: %w", err)
			}

			nextVersion, err := cmd.Flags().GetString("next")
			if err != nil {
				return fmt.Errorf("could not get next flag: %w", err)
			}

			tag, err := cmd.Flags().GetString("tag")
			if err != nil {
				return fmt.Errorf("could not get tag flag: %w", err)
			}

			// Check if git-chglog is installed
			if _, err := exec.LookPath("git-chglog"); err != nil {
				slog.Error("git-chglog not found in PATH")
				slog.Info("Please install git-chglog first:")
				slog.Info("  go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest")
				slog.Info("Or run the setup script:")
				slog.Info("  ./scripts/setup-changelog.sh")
				return fmt.Errorf("git-chglog not installed: %w", err)
			}

			// Build git-chglog command arguments
			chglogArgs := []string{}

			if nextVersion != "" {
				chglogArgs = append(chglogArgs, "--next-tag", nextVersion)
				slog.Info("Generating changelog with next version", "version", nextVersion)
			}

			if output != "" {
				chglogArgs = append(chglogArgs, "--output", output)
			} else {
				chglogArgs = append(chglogArgs, "--output", "CHANGELOG.md")
				output = "CHANGELOG.md"
			}

			if tag != "" {
				chglogArgs = append(chglogArgs, tag)
				slog.Info("Generating changelog for specific tag", "tag", tag)
			}

			// Execute git-chglog
			slog.Info("Running git-chglog", "args", chglogArgs)
			gitChglog := exec.Command("git-chglog", chglogArgs...)
			gitChglog.Stdout = os.Stdout
			gitChglog.Stderr = os.Stderr

			if err := gitChglog.Run(); err != nil {
				slog.Error("Failed to generate changelog", "error", err)
				return fmt.Errorf("failed to generate changelog: %w", err)
			}

			slog.Info("Changelog generated successfully", "output", output)
			return nil
		},
	}

	cmd.Flags().String("next", "", "Next version tag (e.g., v1.2.0)")
	cmd.Flags().String("output", "CHANGELOG.md", "Output file path")
	cmd.Flags().String("tag", "", "Generate changelog for specific tag")

	return cmd
}
