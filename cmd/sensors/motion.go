package main

import (
	"context"

	"github.com/mklimuk/sensors/accel"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/urfave/cli"
)

var motionCmd = cli.Command{
	Name: "motion",
	Subcommands: cli.Commands{
		motionInitCmd,
		motionCheckCmd,
		motionResetCmd,
	},
}

var motionInitCmd = cli.Command{
	Name: "init",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		cli.StringFlag{
			Name:  "sensor,s",
			Value: "bma220",
		},
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.String("sensor") {
		case "bma220":
			switch c.String("adapter") {
			case "mcp2221":
				s := accel.NewBMA220(adapter.NewMCP2221())
				err := s.InitMotionDetection(ctx)
				if err != nil {
					console.Errorf("error initializing BMA220: %s", console.Red(err))
				}
			}
		}
		return nil
	},
}

var motionCheckCmd = cli.Command{
	Name: "check",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		cli.StringFlag{
			Name:  "sensor,s",
			Value: "bma220",
		},
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.String("sensor") {
		case "bma220":
			switch c.String("adapter") {
			case "mcp2221":
				s := accel.NewBMA220(adapter.NewMCP2221())
				motion, err := s.CheckMotionInterrupt(ctx)
				if err != nil {
					console.Errorf("error checking motion detection on BMA220: %s", console.Red(err))
				}
				if motion {
					console.Printf("motion interrupt: %s\n", console.Yellow(motion))
				} else {
					console.Printf("motion interrupt: %s\n", console.Green(motion))
				}
			}
		}
		return nil
	},
}

var motionResetCmd = cli.Command{
	Name: "reset",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		cli.StringFlag{
			Name:  "sensor,s",
			Value: "bma220",
		},
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		switch c.String("sensor") {
		case "bma220":
			switch c.String("adapter") {
			case "mcp2221":
				s := accel.NewBMA220(adapter.NewMCP2221())
				err := s.ResetMotionInterrupt(ctx)
				if err != nil {
					console.Errorf("error resetting motion detection on BMA220: %s", console.Red(err))
				}
			}
		}
		return nil
	},
}
