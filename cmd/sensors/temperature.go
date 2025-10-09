package main

import (
	"context"
	"strconv"

	"github.com/mklimuk/sensors"
	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/environment"
	"github.com/mklimuk/sensors/i2c"
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
		&cli.StringFlag{
			Name:  "device,d",
			Value: "/dev/i2c-1",
		},
		&cli.StringFlag{
			Name:  "addr",
			Value: "4d",
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
		case "generic":
			fallthrough
		case "nanopi":
			var err error
			bus, err := i2c.NewGenericBus(c.String("device"))
			if err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			defer func() {
				err := bus.Close()
				if err != nil {
					console.Errorf("error closing bus: %s", console.Red(err))
				}
			}()
			a = bus
		}
		switch c.String("sensor") {
		case "tc74":
			addr := c.String("addr")
			opts := []environment.TC74ConfigOption{}
			if addr != "" {
				addrInt, err := strconv.ParseInt(addr, 16, 8)
				if err != nil {
					return console.Exit(1, "invalid address: %s", console.Red(err))
				}
				opts = append(opts, environment.WithAddress(byte(addrInt)))
			}
			s := environment.NewTC74(a, opts...)
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
