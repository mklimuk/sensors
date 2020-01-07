package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/mklimuk/sensors"

	"github.com/karalabe/hid"
	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
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

var tempReadCmd = cli.Command{
	Name:      "temperature",
	ShortName: "temp",
	Aliases:   []string{"t"},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return console.Exit(1, "expected 1 argument, got %d", c.NArg())
		}
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.Args().Get(0) {
		case "hih6021":
			switch c.String("adapter") {
			case "mcp2221":
				s := sensors.NewHIH6021(adapter.NewMCP2221())
				temp, hum, err := s.GetTempAndHum(ctx)
				if err != nil {
					console.Errorf("error getting temperature read: %s", console.Red(err))
				}
				console.Printf("%s  %s\n%s %s\n", console.PictoThermometer, console.White(temp), console.PictoHumidity, console.White(hum))
			}
		}
		return nil
	},
}

var usbCmd = cli.Command{
	Name: "usb",
	Subcommands: cli.Commands{
		usbLsCmd,
	},
}

var usbLsCmd = cli.Command{
	Name: "ls",
	Action: func(c *cli.Context) error {
		devices := hid.Enumerate(0, 0)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, '.', tabwriter.AlignRight|tabwriter.Debug)
		for _, d := range devices {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n", d.Path, d.Serial, d.VendorID, d.ProductID, d.Manufacturer, d.Product)
		}
		_ = w.Flush()
		return nil
	},
}

var mcp2221Cmd = cli.Command{
	Name: "mcp2221",
	Subcommands: cli.Commands{
		mcp2221StatusCmd,
		mcp2221ReleaseCmd,
	},
}

var mcp2221StatusCmd = cli.Command{
	Name: "status",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a := adapter.NewMCP2221()
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		status, err := a.Status(ctx)
		if err != nil {
			return console.Exit(1, "adapter communication error: %s", console.Red(err))
		}
		enc := yaml.NewEncoder(os.Stdout)
		err = enc.Encode(status)
		if err != nil {
			return console.Exit(1, "encoding error: %s", console.Red(err))
		}
		return nil
	},
}

var mcp2221ReleaseCmd = cli.Command{
	Name: "release",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a := adapter.NewMCP2221()
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		status, err := a.ReleaseBus(ctx)
		if err != nil {
			return console.Exit(1, "adapter communication error: %s", console.Red(err))
		}
		enc := yaml.NewEncoder(os.Stdout)
		err = enc.Encode(status)
		if err != nil {
			return console.Exit(1, "encoding error: %s", console.Red(err))
		}
		return nil
	},
}
