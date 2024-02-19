package main

import (
	"context"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/environment"
	"github.com/urfave/cli/v2"
)

var tempReadCmd = cli.Command{
	Name:    "temperature",
	Aliases: []string{"temp"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		&cli.StringFlag{
			Name:  "sensor,s",
			Value: "hih6021",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.String("sensor") {
		case "hih6021":
			switch c.String("adapter") {
			case "mcp2221":
				a := adapter.NewMCP2221()
				err := a.Init()
				if err != nil {
					return console.Exit(1, "adapter initialization error: %s", console.Red(err))
				}
				s := environment.NewHIH6021(a)
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
