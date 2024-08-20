package main

import (
	"context"
	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/environment"
	"github.com/urfave/cli/v2"
)

var lightCmd = cli.Command{
	Name: "light",
	Subcommands: []*cli.Command{
		&lightReadCmd,
	},
}

var lightReadCmd = cli.Command{
	Name:    "read",
	Aliases: []string{"rd"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		&cli.StringFlag{
			Name:  "sensor,s",
			Value: "bh1750",
		},
		&cli.StringFlag{
			Name:  "addr",
			Value: "l",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.String("sensor") {
		case "bh1750":
			switch c.String("adapter") {
			case "mcp2221":
				a := adapter.NewMCP2221()
				err := a.Init()
				if err != nil {
					return console.Exit(1, "adapter initialization error: %s", console.Red(err))
				}
				var addr byte
				switch c.String("addr") {
				case "h":
					addr = environment.BH1750AddrHigh
				default:
					addr = environment.BH1750AddrLow
				}
				s := environment.NewBH1750(a, addr)
				lux, err := s.GetLux(ctx)
				if err != nil {
					console.Errorf("error getting light sensor read: %s", console.Red(err))
				}
				console.Printf("%s lux\n", console.White(lux))
			}
		}
		return nil
	},
}