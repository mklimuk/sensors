package i2c

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/mklimuk/sensors"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

var _ sensors.I2CBus = &GenericBus{}

type GenericBus struct {
	bus i2c.BusCloser
}

func NewGenericBus(dev string) (*GenericBus, error) {
	_, err := host.Init()
	if err != nil {
		return nil, fmt.Errorf("could not init host: %w", err)
	}
	slog.Debug("opening i2c bus", "device", dev)
	bus, err := i2creg.Open(dev)
	if err != nil {
		return nil, fmt.Errorf("could not open i2c bus: %w", err)
	}
	return &GenericBus{
		bus: bus,
	}, nil
}

func (b *GenericBus) ReadFromAddr(ctx context.Context, address byte, buffer []byte) error {
	err := b.bus.Tx(uint16(address), nil, buffer)
	if err != nil {
		return fmt.Errorf("could not read from i2c bus %x: %w", address, err)
	}
	slog.Debug("i2c read completed", "address", address, "buffer", hex.Dump(buffer))
	return nil
}

func (b *GenericBus) WriteToAddr(ctx context.Context, address byte, buffer []byte) error {
	slog.Debug("writing to i2c bus", "address", address, "buffer", hex.Dump(buffer))
	err := b.bus.Tx(uint16(address), buffer, nil)
	if err != nil {
		return fmt.Errorf("could not write to i2c bus %x: %w", address, err)
	}
	return nil
}

// SetSpeed sets the speed of the I2C bus in kHz.
//
// Example:
//
//	bus.SetSpeed(100) // 100 kHz
//	bus.SetSpeed(1000) // 1 MHz
//	bus.SetSpeed(10000) // 10 MHz
//	bus.SetSpeed(100000) // 100 MHz
//	bus.SetSpeed(1000000) // 1 GHz
func (b *GenericBus) SetSpeed(speed int) error {
	var freq physic.Frequency
	err := freq.Set(fmt.Sprintf("%dkHz", speed))
	if err != nil {
		return fmt.Errorf("could not set speed: %w; %s", err, freq.String())
	}
	fmt.Println("setting speed to", freq.String())
	err = b.bus.SetSpeed(freq)
	if err != nil {
		return fmt.Errorf("could not set speed: %w", err)
	}
	return nil
}

func (b *GenericBus) Release(ctx context.Context) error {
	return nil
}

func (b *GenericBus) Close() error {
	return b.bus.Close()
}
