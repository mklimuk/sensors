package main

import (
	"errors"
	"fmt"
	"log"
	"os"

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
