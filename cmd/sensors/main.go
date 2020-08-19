package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
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
		tempReadCmd,
		usbCmd,
		mcp2221Cmd,
		gpioCmd,
		motionCmd,
	}
	err := app.Run(os.Args)
	if err != nil {
		if exerr, ok := err.(cli.ExitCoder); ok {
			log.Printf("unexpected error: %v", err)
			return exerr.ExitCode()
		}
		return 1
	}
	return 0
}
