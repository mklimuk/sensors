package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mklimuk/sensors/adapter"

	"github.com/karalabe/hid"
	"github.com/urfave/cli/v2"
)

var usbCmd = cli.Command{
	Name: "usb",
	Subcommands: cli.Commands{
		&usbLsCmd,
		&usbDetectCmd,
	},
}

var usbLsCmd = cli.Command{
	Name: "ls",
	Action: func(c *cli.Context) error {
		// List all HID devices
		devices := hid.Enumerate(0, 0)

		w := tabwriter.NewWriter(os.Stdout, 24, 0, 1, ' ', 0)
		_, _ = fmt.Fprintf(w, "PATH\tSERIAL\tVENDOR\tPRODUCT ID\tMANUFACTURER\tPRODUCT\n")

		for _, dev := range devices {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%#x\t%#x\t%s\t%s\n",
				dev.Path, dev.Serial, dev.VendorID, dev.ProductID, dev.Manufacturer, dev.Product)
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

		// List all HID devices
		devices := hid.Enumerate(0, 0)

		w := tabwriter.NewWriter(os.Stdout, 24, 0, 1, ' ', 0)
		_, _ = fmt.Fprintf(w, "VENDOR\tPRODUCT\tDEVICE\n")

		for _, dev := range devices {
			for descName, codes := range predefined {
				if codes[0] == dev.VendorID && codes[1] == dev.ProductID {
					_, _ = fmt.Fprintf(w, "%#x\t%#x\t%s\n", dev.VendorID, dev.ProductID, descName)
				}
			}
		}
		_ = w.Flush()
		return nil
	},
}
