package i2c

import (
	"context"
	"fmt"

	"github.com/mklimuk/sensors"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

var _ sensors.I2CBus = &GenericBus{}

type GenericBus struct {
	bus i2c.BusCloser
}

func NewGenericBus(dev string) (*GenericBus, error) {
	state, err := host.Init()
	if err != nil {
		return nil, fmt.Errorf("could not init host: %w", err)
	}
	for _, driver := range state.Loaded {
		fmt.Println(driver.String())
	}
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
	return nil
}

func (b *GenericBus) WriteToAddr(ctx context.Context, address byte, buffer []byte) error {
	err := b.bus.Tx(uint16(address), buffer, nil)
	if err != nil {
		return fmt.Errorf("could not write to i2c bus %x: %w", address, err)
	}
	return nil
}

func (b *GenericBus) Release(ctx context.Context) error {
	return nil
}

func (b *GenericBus) Close() error {
	return b.bus.Close()
}
