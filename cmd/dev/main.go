package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/mklimuk/sensors/cmd/dev/cmd"
)

var (
	debug   bool
	version string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dev",
		Short: "build/test/run/deploy tool for sensors project",
		Long:  "A custom build tool easing common build/test/run/deploy tasks",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			charm := log.NewWithOptions(os.Stdout, log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				TimeFormat:      time.DateTime,
				Prefix:          "sns",
			})
			charm.SetColorProfile(termenv.TrueColor)

			if debug {
				charm.SetLevel(log.DebugLevel)
			} else {
				charm.SetLevel(log.InfoLevel)
			}
			slogger := slog.New(charm)
			slog.SetDefault(slogger)
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&version, "version", "latest", "Version for build")

	rootCmd.AddCommand(cmd.BuildCmd())
	rootCmd.AddCommand(cmd.ChangelogCmd())
	rootCmd.AddCommand(cmd.TestCmd())
	rootCmd.AddCommand(cmd.LintCmd())
	rootCmd.AddCommand(cmd.IntegrationTestCmd())

	err := rootCmd.Execute()
	if err != nil {
		slog.Error("unexpected error", "error", err)
		os.Exit(1)
	}
}
