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
	"github.com/mklimuk/sensors/snsctx"
)

var airCmd = cli.Command{
	Name: "air",
	Subcommands: []*cli.Command{
		&airVersionCmd,
		&airReadTvocCmd,
		&airCalibrateCmd,
		&airReadResistanceCmd,
	},
}

var airVersionCmd = cli.Command{
	Name: "version",
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
		ctx := snsctx.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		var s *air.AGS02MA
		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s = air.NewAGS02MA(ad)
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
			s = air.NewAGS02MA(bus)
		}
		ver, err := s.ReadVersion(ctx)
		if err != nil {
			return console.Exit(1, "error reading version: %s", console.Red(err))
		}
		console.Printf("firmwareversion: %d\n", ver)
		return nil
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
		ctx := snsctx.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		var s *air.AGS02MA
		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s = air.NewAGS02MA(ad)
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
			s = air.NewAGS02MA(bus)
		}
		err := s.Configure(ctx)
		if err != nil {
			return console.Exit(1, "error configuring: %s", console.Red(err))
		}
		console.Printf("ags02ma configured\n")
		err = s.Calibrate(ctx)
		if err != nil {
			return console.Exit(1, "error calibrating: %s", console.Red(err))
		}
		console.Printf("sensor calibrated\n")
		return nil
	},
}

var airReadTvocCmd = cli.Command{
	Name: "tvoc",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "adapter,a",
			Value: "mcp2221",
		},
		&cli.StringFlag{
			Name:  "device,d",
			Value: "/dev/i2c-1",
		},
		&cli.StringFlag{
			Name:  "mode,m",
			Value: "register-write",
			Usage: "mode to use for TVOC read",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		modeStr := c.String("mode")
		if modeStr != "direct-read" && modeStr != "register-write" {
			return console.Exit(1, "invalid mode: %s", console.Red(modeStr))
		}
		mode := air.TVOCModeRegisterWrite
		if modeStr == "direct-read" {
			mode = air.TVOCModeDirectRead
		}

		verbose := c.Bool("verbose")
		ctx := snsctx.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		var s *air.AGS02MA
		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s = air.NewAGS02MA(ad, air.WithTVOCMode(mode))
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
			s = air.NewAGS02MA(bus, air.WithTVOCMode(mode))
		}

		err := s.Configure(ctx)
		if err != nil {
			return console.Exit(1, "error configuring: %s", console.Red(err))
		}
		console.Printf("sensor configured\n")

		ppb, err := s.GetTVOC(ctx)
		if err != nil {
			return console.Exit(1, "error getting TVOC read: %s", console.Red(err))
		}
		console.Printf("%d ppb\n", ppb)
		return nil
	},
}

var airReadResistanceCmd = cli.Command{
	Name: "resistance",
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
		ctx := snsctx.SetVerbose(context.Background(), verbose)
		charm := chlog.NewWithOptions(os.Stdout, chlog.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		if verbose {
			charm.SetLevel(chlog.DebugLevel)
		}
		slog.SetDefault(slog.New(charm))

		var s *air.AGS02MA
		switch c.String("adapter") {
		case "mcp2221":
			ad := adapter.NewMCP2221()
			if err := ad.Init(); err != nil {
				return console.Exit(1, "adapter initialization error: %s", console.Red(err))
			}
			s = air.NewAGS02MA(ad)
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
			s = air.NewAGS02MA(bus)
		}

		err := s.Configure(ctx)
		if err != nil {
			return console.Exit(1, "error configuring: %s", console.Red(err))
		}
		console.Printf("sensor configured\n")

		resistance, err := s.ReadResistance(ctx)
		if err != nil {
			return console.Exit(1, "error getting resistance read: %s", console.Red(err))
		}
		console.Printf("%d Ohms resistance\n", resistance)
		return nil
	},
}
