package command

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/urfave/cli/v2"
	"gobot.io/x/gobot/v2/drivers/i2c"
	"gobot.io/x/gobot/v2/platforms/friendlyelec/nanopi"
)

// MCP4661-103 I2C addresses (example, update as needed)
var mcp4661Addresses = []uint8{0b0101000, 0b0101001, 0b0101010, 0b0101011, 0b0101100, 0b0101101}

var PotentiometerGetCmd = &cli.Command{
	Name:  "get",
	Usage: "get potentiometer values",
	Action: func(c *cli.Context) error {
		npi := nanopi.NewNeoAdaptor()
		err := npi.I2cBusAdaptor.Connect()
		if err != nil {
			return fmt.Errorf("adaptor connect error: %w", err)
		}
		defer npi.I2cBusAdaptor.Finalize()
		for i, addr := range mcp4661Addresses {
			val, err := getKnobValue(npi, addr)
			if err != nil {
				slog.Error("knob read error", "knob", i, "addr", addr, "error", err)
				continue
			}
			fmt.Printf("knob %s (addr %s) value: %s\n", console.White(i), console.White(fmt.Sprintf("%#x", addr)), console.White(val))
		}
		return nil
	},
}

func getKnobValue(adaptor i2c.Connector, addr uint8) (uint16, error) {
	board := i2c.NewGenericDriver(adaptor, "mcp4661", int(addr), func(c i2c.Config) {
		c.SetBus(2)
	})
	err := board.Start()
	if err != nil {
		return 0, fmt.Errorf("start error: %v", err)
	}
	defer func() { _ = board.Halt() }()
	data := make([]byte, 2)
	err = board.Read(data)
	if err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	// Combine to get 10-bit result
	wiper := (uint16(data[0]) << 8) | uint16(data[1])
	wiper &= 0x03FF // mask to 10 bits
	return wiper, nil
}

// MCP446x commands
const (
	OpWrite     byte = 0x00
	OpIncrement      = 0x01
	OpDecrement      = 0x02
	OpRead           = 0x03
)

// Address map
const (
	AddrVolatileWiper0  byte = 0x00
	AddrVolatileWiper1  byte = 0x01
	AddrPermanentWiper0 byte = 0x02
	AddrPermanentWiper1 byte = 0x03
	AddrVolatileTCON0   byte = 0x04
	AddrStatus          byte = 0x05
	DataEEPROM0         byte = 0x06
	DataEEPROM1         byte = 0x07
	DataEEPROM2         byte = 0x08
	DataEEPROM3         byte = 0x09
	DataEEPROM4         byte = 0x0A
	DataEEPROM5         byte = 0x0B
	DataEEPROM6         byte = 0x0C
	DataEEPROM7         byte = 0x0D
	DataEEPROM8         byte = 0x0E
	DataEEPROM9         byte = 0x0F
)

var PotentiometerSetCmd = &cli.Command{
	Name:  "set",
	Usage: "set amplifier knob values",
	Action: func(c *cli.Context) error {
		if c.NArg() < 2 {
			fmt.Println("Usage: haectl amp knobs set <knob_index 0-5> <value 0-255>")
			return nil
		}
		knobIdx, err := strconv.Atoi(c.Args().Get(0))
		if err != nil || knobIdx < 0 || knobIdx >= len(mcp4661Addresses) {
			return fmt.Errorf("invalid knob index: %d", knobIdx)
		}
		val, err := strconv.Atoi(c.Args().Get(1))
		if err != nil || val < 0 || val > 255 {
			return fmt.Errorf("invalid value: %d", val)
		}
		npi := nanopi.NewNeoAdaptor()
		err = npi.I2cBusAdaptor.Connect()
		if err != nil {
			return fmt.Errorf("adaptor connect error: %w", err)
		}
		defer npi.I2cBusAdaptor.Finalize()

		addr := mcp4661Addresses[knobIdx]
		board := i2c.NewGenericDriver(npi, "mcp4661", int(addr), func(c i2c.Config) {
			c.SetBus(2)
		})
		err = board.Start()
		if err != nil {
			return fmt.Errorf("knob %d (addr %#x) start error: %v", knobIdx, addr, err)
		}

		registry := byte(0x00)
		op := OpWrite
		dataHigh := byte((val >> 8) & 0x03) // 2 bits
		dataLow := byte(val & 0xFF)         // 8 bits
		// registry address on oldest 4 bits, command on bits 3:2, dataHigh on bits 1:0
		control := ((registry << 4) | (op << 2) | dataHigh) // 8 bits
		command := []byte{control, dataLow}

		fmt.Println(hex.Dump(command))
		// write command to board
		err = board.Write(command)
		if err != nil {
			return fmt.Errorf("knob %d (addr %#x) write error: %v", knobIdx, addr, err)
		}

		defer func() { _ = board.Halt() }()
		err = board.WriteByteData(0x00, byte(val))
		if err != nil {
			return fmt.Errorf("knob %d (addr %#x) write error: %v", knobIdx, addr, err)
		}
		fmt.Printf("knob %s (addr %s) set to %s\n", console.White(knobIdx), console.White(fmt.Sprintf("%#x", addr)), console.White(val))
		return nil
	},
}

var PotentiometerCmd = &cli.Command{
	Name:    "potentiometer",
	Aliases: []string{"pot"},
	Usage:   "control potentiometer",
	Subcommands: []*cli.Command{
		PotentiometerGetCmd,
		PotentiometerSetCmd,
	},
}

// Helper to parse hex string to bytes
func hexStringToBytes(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex string must have even length")
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		v, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		b[i] = byte(v)
	}
	return b, nil
}
