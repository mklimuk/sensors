package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/karalabe/hid"
	"github.com/urfave/cli"
)

var usbCmd = cli.Command{
	Name: "usb",
	Subcommands: cli.Commands{
		usbLsCmd,
	},
}

var usbLsCmd = cli.Command{
	Name: "ls",
	Action: func(c *cli.Context) error {
		devices := hid.Enumerate(0, 0)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, '.', tabwriter.AlignRight|tabwriter.Debug)
		for _, d := range devices {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n", d.Path, d.Serial, d.VendorID, d.ProductID, d.Manufacturer, d.Product)
		}
		_ = w.Flush()
		return nil
	},
}
