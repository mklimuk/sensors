package main

import (
	"context"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/environment"
	"github.com/urfave/cli"
)

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
				s := environment.NewHIH6021(adapter.NewMCP2221())
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
