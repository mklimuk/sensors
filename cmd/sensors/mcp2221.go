package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mklimuk/sensors/adapter"
	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/snsctx"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func toUint16(value string) (uint16, error) {
	clean := strings.TrimSpace(strings.ToLower(value))
	clean = strings.TrimPrefix(clean, "0x")
	productID, err := strconv.ParseUint(clean, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid product value %q, expected hex like 00dd or 0x00dd", value)
	}
	return uint16(productID), nil
}

func mcp2221FromContext(c *cli.Context) (*adapter.MCP2221, error) {
	productID, err := toUint16(c.String("product"))
	if err != nil {
		return nil, cli.Exit(err.Error(), 1)
	}
	return adapter.NewMCP2221(adapter.WithProductID(productID)), nil
}

var mcp2221Cmd = cli.Command{
	Name: "mcp2221",
	Subcommands: cli.Commands{
		&mcp2221StatusCmd,
		&mcp2221ReleaseCmd,
		&mcp2221GPIOCmd,
		&mcp2221ResetCmd,
		&mcp2221ChipCmd,
		&mcp2221ButtonCmd,
	},
}

var mcp2221StatusCmd = cli.Command{
	Name: "status",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		err = a.Init()
		if err != nil {
			return console.Exit(1, "adapter initialization error: %s", console.Red(err))
		}
		ctx := snsctx.SetVerbose(context.Background(), c.Bool("verbose"))
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
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		err = a.Init()
		if err != nil {
			return console.Exit(1, "adapter initialization error: %s", console.Red(err))
		}
		ctx := snsctx.SetVerbose(context.Background(), c.Bool("verbose"))
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
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		err = a.Init()
		if err != nil {
			return console.Exit(1, "adapter initialization error: %s", console.Red(err))
		}
		ctx := snsctx.SetVerbose(context.Background(), c.Bool("verbose"))
		err = a.Reset(ctx)
		if err != nil {
			return console.Exit(1, "adapter communication error: %s", console.Red(err))
		}
		return nil
	},
}

var mcp2221GPIOCmd = cli.Command{
	Name: "gpio",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Subcommands: cli.Commands{
		&mcp2221GPIOReadCmd,
	},
}

var mcp2221GPIOReadCmd = cli.Command{
	Name: "read",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		err = a.Init()
		if err != nil {
			return console.Exit(1, "adapter initialization error: %s", console.Red(err))
		}
		ctx := snsctx.SetVerbose(context.Background(), c.Bool("verbose"))
		err = a.SetGPIOParameters(ctx, adapter.MCP2221GPIOParameters{
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

var mcp2221ChipCmd = cli.Command{
	Name:        "chip",
	Description: "read mcp2221 chip settings",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
	},
	Subcommands: []*cli.Command{
		&mcp2221UpdateVendorCmd,
	},
	Action: func(c *cli.Context) error {
		mcp, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		err = mcp.ReadChipSettings(c.Context)
		if err != nil {
			console.Errorf("could not read chip settings: %v", err)
		}
		return nil
	},
}

var mcp2221UpdateVendorCmd = cli.Command{
	Name:        "update-vendor",
	Description: "update mcp2221 vendor and product id",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "vendor", Value: "04d8"},
		&cli.StringFlag{Name: "product", Value: "00dd"},
		&cli.StringFlag{Name: "device-product,P", Value: "00dd"},
		&cli.BoolFlag{Name: "dryrun"},
	},
	Action: func(c *cli.Context) error {
		vendor, err := hex.DecodeString(strings.TrimPrefix(c.String("vendor"), "0x"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("could not decode vendor from string: %v", err), 1)
		}
		product, err := hex.DecodeString(strings.TrimPrefix(c.String("product"), "0x"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("could not decode product from string: %v", err), 1)
		}
		deviceProductID, err := toUint16(c.String("device-product"))
		if err != nil {
			return cli.Exit(err.Error(), 1)
		}
		mcp := adapter.NewMCP2221(adapter.WithProductID(deviceProductID))
		err = mcp.UpdateVendorAndProductID(c.Context, vendor, product, c.Bool("dryrun"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("could not read chip settings: %v", err), 1)
		}
		return nil
	},
}

var mcp2221ButtonCmd = cli.Command{
	Name:        "button",
	Description: "poll mcp2221 GPIO pins and report button press/release events (active-low: 0 = pressed)",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "product,p", Value: "00dd"},
		&cli.DurationFlag{
			Name:    "interval,i",
			Value:   5 * time.Millisecond,
			Usage:   "polling interval (USB HID round-trip is ~2-3ms; values <5ms may miss frames)",
		},
		&cli.BoolFlag{Name: "verbose,v"},
	},
	Action: func(c *cli.Context) error {
		a, err := mcp2221FromContext(c)
		if err != nil {
			return err
		}
		if err := a.Init(); err != nil {
			return console.Exit(1, "adapter initialization error: %s", console.Red(err))
		}

		ctx, cancel := signal.NotifyContext(
			snsctx.SetVerbose(context.Background(), c.Bool("verbose")),
			os.Interrupt, syscall.SIGTERM,
		)
		defer cancel()

		if err := a.SetGPIOParameters(ctx, adapter.MCP2221GPIOParameters{
			GPIO0Mode: adapter.GPIOModeIn,
			GPIO1Mode: adapter.GPIOModeIn,
			GPIO2Mode: adapter.GPIOModeIn,
			GPIO3Mode: adapter.GPIOModeIn,
		}); err != nil {
			return console.Exit(1, "could not configure GPIO as inputs: %s", console.Red(err))
		}

		// Open a sticky session so the polling loop reuses one HID handle
		// instead of paying USB enumeration on every iteration. Transport
		// errors invalidate the handle internally; the next ReadGPIO will
		// reopen lazily, so the loop just needs to keep retrying.
		if err := a.Open(ctx); err != nil {
			console.Errorf("initial mcp2221 open failed (will retry on first read): %s", console.Red(err))
		}
		defer func() {
			if err := a.Close(); err != nil {
				slog.Debug("mcp2221 close failed", "err", err)
			}
		}()

		interval := c.Duration("interval")
		console.Printf("monitoring GP0..GP3 every %s; press Ctrl-C to stop\n", interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var prev adapter.MCP2221GPIOValues
		first := true
		for {
			select {
			case <-ctx.Done():
				console.Print("stopping button monitor")
				return nil
			case <-ticker.C:
				vals, err := a.ReadGPIO(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					slog.Debug("button read failed; will retry", "err", err)
					continue
				}
				reportButtonChange("GP0", vals.GPIO0Value, prev.GPIO0Value, first)
				reportButtonChange("GP1", vals.GPIO1Value, prev.GPIO1Value, first)
				reportButtonChange("GP2", vals.GPIO2Value, prev.GPIO2Value, first)
				reportButtonChange("GP3", vals.GPIO3Value, prev.GPIO3Value, first)
				prev = vals
				first = false
			}
		}
	},
}

// reportButtonChange prints a colored line when a GPIO pin transitions, or on
// the very first sample to establish initial state. Active-low: 0 = pressed.
func reportButtonChange(name string, cur, prev byte, first bool) {
	if !first && cur == prev {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	if cur == 0 {
		console.Printf("%s %s %s\n", ts, console.Bold(name), console.Red("PRESSED"))
		return
	}
	console.Printf("%s %s %s\n", ts, console.Bold(name), console.Green("RELEASED"))
}
