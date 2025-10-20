package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	chlog "github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/air"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/i2c"
)

var airCmd = cli.Command{
	Name: "air",
	Subcommands: []*cli.Command{
		&airReadCmd,
		&airCalibrateCmd,
	},
}

var airCalibrateCmd = cli.Command{
	Name: "calibrate",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		&cli.StringFlag{
			Name:  "device,d",
			Value: "/dev/i2c-1",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		verbose := c.Bool("verbose")
		ctx := console.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s := air.NewAGS02MA(ad)
			err := s.Calibrate(ctx)
			if err != nil {
				return console.Exit(1, "error calibrating: %s", console.Red(err))
			}
			console.Printf("calibrated\n")
		case "generic":
			fallthrough
		case "nanopi":
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
			bus.SetSpeed(20_000_000_000) // 20 kHz
			s := air.NewAGS02MA(bus)
			err = s.Calibrate(ctx)
			if err != nil {
				return console.Exit(1, "error calibrating: %s", console.Red(err))
			}
			console.Printf("calibrated\n")
		}
		return nil
	},
}

var airReadCmd = cli.Command{
	Name:    "read",
	Aliases: []string{"rd"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		&cli.StringFlag{
			Name:  "device,d",
			Value: "/dev/i2c-1",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		verbose := c.Bool("verbose")
		ctx := console.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s := air.NewAGS02MA(ad)
			ppb, err := s.GetTVOC(ctx)
			if err != nil {
				return console.Exit(1, "error getting TVOC read: %s", console.Red(err))
			}
			console.Printf("%d ppb\n", ppb)
		case "generic":
			fallthrough
		case "nanopi":
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
			bus.SetSpeed(20_000_000_000) // 20 kHz
			s := air.NewAGS02MA(bus)
			ver, err := s.ReadVersion(ctx)
			if err != nil {
				return console.Exit(1, "error reading version: %s", console.Red(err))
			}
			resistance, err := s.ReadResistance(ctx)
			if err != nil {
				return console.Exit(1, "error reading resistance: %s", console.Red(err))
			}
			console.Printf("resistance: %d\n", resistance)
			console.Printf("version: %d\n", ver)
			ppb, err := s.GetTVOCWithRegisterRead(ctx)
			if err != nil {
				return console.Exit(1, "error getting TVOC read: %s", console.Red(err))
			}
			console.Printf("%d ppb\n", ppb)
		}
		return nil
	},
}
