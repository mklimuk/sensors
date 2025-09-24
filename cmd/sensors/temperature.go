package main

import (
	"context"

	"github.com/mklimuk/sensors"
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

		var a sensors.I2CBus
		switch c.String("adapter") {
		case "mcp2221":
			mcp2221 := adapter.NewMCP2221()
			err := mcp2221.Init()
			if err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			a = mcp2221
		}
		switch c.String("sensor") {
		case "tc74":
			s := environment.NewTC74(a)
			temp, err := s.GetTemperature(ctx)
			if err != nil {
				return console.Exit(1, "error getting temperature read: %s", console.Red(err))
			}
			console.Printf("%s %s\n", console.PictoThermometer, console.White(temp))
		case "hih6021":
			s := environment.NewHIH6021(a)
			temp, hum, err := s.GetTempAndHum(ctx)
			if err != nil {
				console.Errorf("error getting temperature read: %s", console.Red(err))
			}
			console.Printf("%s  %s\n%s %s\n", console.PictoThermometer, console.White(temp), console.PictoHumidity, console.White(hum))
		case "shtc3":
			s := environment.NewSHTC3(a)
			temp, hum, err := s.GetTempAndHum(ctx)
			if err != nil {
				return console.Exit(1, "error getting temperature read: %s", console.Red(err))
			}
			console.Printf("%s  %s\n%s %s\n", console.PictoThermometer, console.White(temp), console.PictoHumidity, console.White(hum))

		}
		return nil
	},
}
