package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var mcp2221Cmd = cli.Command{
	Name: "mcp2221",
	Subcommands: cli.Commands{
		mcp2221StatusCmd,
		mcp2221ReleaseCmd,
		mcp2221GPIOCmd,
		mcp2221ResetCmd,
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
		settings, err := a.GetGPIOParameters(ctx)
		if err != nil {
			return console.Exit(1, "could not read gpio parameters: %s", console.Red(err))
		}
		fmt.Println("GPIO settings:")
		err = enc.Encode(settings)
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

var mcp2221ResetCmd = cli.Command{
	Name: "reset",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a := adapter.NewMCP2221()
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		err := a.Reset(ctx)
		if err != nil {
			return console.Exit(1, "adapter communication error: %s", console.Red(err))
		}
		return nil
	},
}

var mcp2221GPIOCmd = cli.Command{
	Name: "gpio",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose,v"},
	},
	Subcommands: cli.Commands{
		mcp2221GPIOReadCmd,
	},
}

var mcp2221GPIOReadCmd = cli.Command{
	Name: "read",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a := adapter.NewMCP2221()
		ctx := console.SetVerbose(context.Background(), c.Bool("verbose"))
		err := a.SetGPIOParameters(ctx, adapter.MCP2221GPIOParameters{
			GPIO0Mode: adapter.GPIOModeIn,
			GPIO1Mode: adapter.GPIOModeIn,
			GPIO2Mode: adapter.GPIOModeIn,
			GPIO3Mode: adapter.GPIOModeIn,
		})
		if err != nil {
			return console.Exit(1, "could not set parameters: %s", console.Red(err))
		}
		time.Sleep(100 * time.Millisecond)
		params, err := a.GetGPIOParameters(ctx)
		if err != nil {
			return console.Exit(1, "could not get parameters: %s", console.Red(err))
		}
		enc := yaml.NewEncoder(os.Stdout)
		err = enc.Encode(params)
		if err != nil {
			return console.Exit(1, "params encoding error: %s", console.Red(err))
		}
		vals, err := a.ReadGPIO(ctx)
		if err != nil {
			return console.Exit(1, "could not read values: %s", console.Red(err))
		}
		err = enc.Encode(vals)
		if err != nil {
			return console.Exit(1, "values encoding error: %s", console.Red(err))
		}
		return nil
	},
}
