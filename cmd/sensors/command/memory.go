package command

import (
	"encoding/hex"
	"fmt"

	"github.com/urfave/cli/v2"
	"gobot.io/x/gobot/v2/drivers/spi"
	"gobot.io/x/gobot/v2/platforms/friendlyelec/nanopi"
)

var MemoryReadCmd = &cli.Command{
	Name:  "read",
	Usage: "read amplifier DSP memory",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "device", Usage: "memory device name", Value: "25AA1024"},
		&cli.IntFlag{Name: "address", Usage: "memory address to read", Required: true},
		&cli.IntFlag{Name: "length", Usage: "number of bytes to read", Value: 16},
	},
	Action: func(c *cli.Context) error {
		adaptor := nanopi.NewNeoAdaptor()
		spiDev := spi.NewDriver(adaptor, "spi")
		err := spiDev.Start()
		if err != nil {
			return fmt.Errorf("SPI device start error: %w", err)
		}
		addr := c.Int("address")
		length := c.Int("length")
		if addr < 0 || addr > 0x1FFFF {
			fmt.Println("Address out of range (0-0x1FFFF)")
			return nil
		}
		if length <= 0 || length > 256 {
			return fmt.Errorf("length out of range: %d", length)
		}
		// 25AA1024 read: 0x03 [addr_hi] [addr_mid] [addr_lo]
		cmd := []byte{0x03, byte(addr >> 16), byte(addr >> 8), byte(addr)}
		fmt.Println(hex.Dump(cmd))
		return nil
	},
}

var MemoryWriteCmd = &cli.Command{
	Name:  "write",
	Usage: "write amplifier DSP memory",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "device", Usage: "memory device name", Value: "25AA1024"},
		&cli.IntFlag{Name: "address", Usage: "memory address to write", Required: true},
		&cli.StringFlag{Name: "data", Usage: "hex bytes to write (e.g. '01FF23')", Required: true},
	},
	Action: func(c *cli.Context) error {
		adaptor := nanopi.NewNeoAdaptor()
		spiDev := spi.NewDriver(adaptor, "spi")
		err := spiDev.Start()
		if err != nil {
			return fmt.Errorf("SPI device start error: %w", err)
		}
		addr := c.Int("address")
		if addr < 0 || addr > 0x1FFFF {
			return fmt.Errorf("address out of range: %d", addr)
		}
		data, err := hexStringToBytes(c.String("data"))
		if err != nil {
			return fmt.Errorf("invalid data hex string: %w", err)
		}
		// 25AA1024 write enable: 0x06
		/*err = spiDev.Write([]byte{0x06})
		if err != nil {
			return fmt.Errorf("SPI write enable error: %w", err)
		}
		// 25AA1024 write: 0x02 [addr_hi] [addr_mid] [addr_lo] [data...]
		cmd := []byte{0x02, byte(addr >> 16), byte(addr >> 8), byte(addr)}
		cmd = append(cmd, data...)
		if err := spiDev.Write(cmd); err != nil {
			return fmt.Errorf("SPI write error: %w", err)
		}
		fmt.Printf("Wrote %d bytes to DSP memory 0x%05X: % X\n", len(data), addr, data)
		return nil*/
		fmt.Println(hex.Dump(data))
		return nil
	},
}

var MemoryCmd = &cli.Command{
	Name:    "memory",
	Aliases: []string{"mem"},
	Usage:   "memory-related operations",
	Subcommands: []*cli.Command{
		MemoryReadCmd,
		MemoryWriteCmd,
	},
}
