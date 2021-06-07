package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mklimuk/sensors/adapter"

	"github.com/karalabe/hid"
	"github.com/urfave/cli"
)

var usbCmd = cli.Command{
	Name: "usb",
	Subcommands: cli.Commands{
		usbLsCmd,
		usbDetectCmd,
	},
}

var usbLsCmd = cli.Command{
	Name: "ls",
	Action: func(c *cli.Context) error {
		devices := hid.Enumerate(0, 0)
		w := tabwriter.NewWriter(os.Stdout, 24, 0, 1, ' ', 0)
		_, _ = fmt.Fprintf(w, "PATH\tSERIAL\tVENDOR\tPRODUCT ID\tMANUFACTURER\tPRODUCT\n")
		for _, d := range devices {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%#x\t%#x\t%s\t%s\n", d.Path, d.Serial, d.VendorID, d.ProductID, d.Manufacturer, d.Product)
		}
		_ = w.Flush()
		return nil
	},
}

var usbDetectCmd = cli.Command{
	Name: "detect",
	Action: func(c *cli.Context) error {
		predefined := map[string][]uint16{
			"MCP2221": {adapter.VendorID, adapter.ProductID},
		}
		devices := hid.Enumerate(0, 0)
		w := tabwriter.NewWriter(os.Stdout, 24, 0, 1, ' ', 0)
		_, _ = fmt.Fprintf(w, "VENDOR\tPRODUCT\tDEVICE\n")
		for _, d := range devices {
			for desc, codes := range predefined {
				if codes[0] == d.VendorID && codes[1] == d.ProductID {
					_, _ = fmt.Fprintf(w, "%#x\t%#x\t%s\n", d.VendorID, d.ProductID, desc)
				}
			}
		}
		_ = w.Flush()
		return nil
	},
}
