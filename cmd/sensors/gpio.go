package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/gpio"
	"github.com/urfave/cli"
)

var gpioCmd = cli.Command{
	Name: "gpio",
	Subcommands: []cli.Command{
		gpioStatusCmd,
		gpioReadCmd,
		gpioConfigureCmd,
		gpioPullCmd,
	},
}

var gpioReadCmd = cli.Command{
	Name: "read",
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return console.Exit(1, "expected 1 argument, got %d", c.NArg())
		}
		bytes, err := hex.DecodeString(c.Args().Get(0))
		if err != nil {
			return console.Exit(1, "could not decode address: %v", err)
		}
		exp := gpio.NewMCP23017(adapter.NewMCP2221(), bytes[0])
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = exp.InitA(ctx, 0xFF)
		if err != nil {
			return console.Exit(1, "could not initialize gpio: %v", err)
		}
		a, err := exp.ReadA(ctx)
		if err != nil {
			return console.Exit(1, "could not read gpio A: %v", err)
		}
		fmt.Printf("\nI/O A: %#X\n", a)
		b, err := exp.ReadB(ctx)
		if err != nil {
			return console.Exit(1, "could not read gpio B: %v", err)
		}
		fmt.Printf("\nI/O B: %#X\n", b)
		return nil
	},
}

var gpioStatusCmd = cli.Command{
	Name: "status",
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return console.Exit(1, "expected 1 argument, got %d", c.NArg())
		}
		bytes, err := hex.DecodeString(c.Args().Get(0))
		if err != nil {
			return console.Exit(1, "could not decode address: %v", err)
		}
		exp := gpio.NewMCP23017(adapter.NewMCP2221(), bytes[0])
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		data, err := exp.ReadSettingsA(ctx)
		if err != nil {
			return console.Exit(1, "could not read settings: %v", err)
		}
		fmt.Printf("\nIOCON content: %#X\n", data)
		return nil
	},
}

var gpioConfigureCmd = cli.Command{
	Name: "configure",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			return console.Exit(1, "expected 2 arguments, got %d", c.NArg())
		}
		addr, err := hex.DecodeString(c.Args().Get(0))
		if err != nil {
			return console.Exit(1, "could not decode address: %v", err)
		}
		data, err := hex.DecodeString(c.Args().Get(1))
		if err != nil {
			return console.Exit(1, "could not decode data: %v", err)
		}
		exp := gpio.NewMCP23017(adapter.NewMCP2221(), addr[0])
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = exp.WriteSettingsA(ctx, data[0])
		if err != nil {
			return console.Exit(1, "could not write settings: %v", err)
		}
		fmt.Printf("\nWrote IOCON content: %#X\n", data[0])
		return nil
	},
}

var gpioPullCmd = cli.Command{
	Name: "pull",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			return console.Exit(1, "expected 2 arguments, got %d", c.NArg())
		}
		addr, err := hex.DecodeString(c.Args().Get(0))
		if err != nil {
			return console.Exit(1, "could not decode address: %v", err)
		}
		data, err := hex.DecodeString(c.Args().Get(1))
		if err != nil {
			return console.Exit(1, "could not decode data: %v", err)
		}
		exp := gpio.NewMCP23017(adapter.NewMCP2221(), addr[0])
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = exp.PullUpA(ctx, data[0])
		if err != nil {
			return console.Exit(1, "could not write pull up settings: %v", err)
		}
		fmt.Printf("\nWrote GPPU content: %#X\n", data[0])
		return nil
	},
}
