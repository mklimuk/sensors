package main

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	chlog "github.com/charmbracelet/log"
	"github.com/muesli/termenv"
	"github.com/urfave/cli/v2"
)

var version string
var commit string
var date string

func main() {
	os.Exit(run())
}

func run() int {
	app := cli.NewApp()
	app.Name = "sensors"
	app.EnableBashCompletion = true
	app.Version = fmt.Sprintf("%s-%s-%s", version, date, commit)
	app.Usage = "sensors cli"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "enable verbose logging",
		},
	}
	app.Before = func(ctx *cli.Context) error {
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		charm.SetColorProfile(termenv.TrueColor)
		charm.SetLevel(chlog.InfoLevel)
		if ctx.Bool("verbose") {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))
		return nil
	}
	app.Commands = cli.Commands{
		&tempReadCmd,
		&usbCmd,
		&mcp2221Cmd,
		&gpioCmd,
		&motionCmd,
		&lightCmd,
		&airCmd,
	}
	err := app.Run(os.Args)
	if err != nil {
		var exerr cli.ExitCoder
		if errors.As(err, &exerr) {
			log.Printf("unexpected error: %v", err)
			return exerr.ExitCode()
		}
		return 1
	}
	return 0
}
