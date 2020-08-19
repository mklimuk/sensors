package main

import (
	"context"
	"os"

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
