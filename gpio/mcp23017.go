package gpio

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mklimuk/sensors"
)

type registry int

const DefaultMCP23017Address = 0x21

// BRegistries
const (
	IODIRA registry = iota
	IOPOLA
	GPINTENA
	DEFVALA
	INTCONA
	IOCONA
	GPPUA
	INTFA
	INTCAPA
	GPIOA
	IODIRB
	IOPOLB
	GPINTENB
	DEFVALB
	INTCONB
	IOCONB
	GPPUB
	INTFB
	INTCAPB
	GPIOB
	OLATB
)

var (
	BankAddr = []map[registry]byte{
		{
			IODIRA:   0x00,
			IOPOLA:   0x02,
			GPINTENA: 0x04,
			DEFVALA:  0x06,
			INTCONA:  0x08,
			IOCONA:   0x0A,
			GPPUA:    0x0C,
			INTFA:    0x0E,
			INTCAPA:  0x10,
			GPIOA:    0x12,
			IODIRB:   0x01,
			IOPOLB:   0x03,
			GPINTENB: 0x05,
			DEFVALB:  0x07,
			INTCONB:  0x09,
			IOCONB:   0x0B,
			GPPUB:    0x0D,
			INTFB:    0x0F,
			INTCAPB:  0x11,
			GPIOB:    0x13,
			OLATB:    0x15,
		},
		{
			IODIRA:   0x00,
			IOPOLA:   0x01,
			GPINTENA: 0x02,
			DEFVALA:  0x03,
			INTCONA:  0x04,
			IOCONA:   0x05,
			GPPUA:    0x06,
			INTFA:    0x07,
			INTCAPA:  0x08,
			GPIOA:    0x09,
			IODIRB:   0x10,
			IOPOLB:   0x11,
			GPINTENB: 0x12,
			DEFVALB:  0x13,
			INTCONB:  0x14,
			IOCONB:   0x15,
			GPPUB:    0x16,
			INTFB:    0x17,
			INTCAPB:  0x18,
			GPIOB:    0x19,
			OLATB:    0x1A,
		},
	}
)

/*
	Steps to read GPIO:

1. Set 0xFF to IODIR registry (all inputs) - 0x00(A)/0x01(B)
2. Configure pull-up? 0x06
3. Read port register 0x09
*/
type MCP23017 struct {
	mx         sync.Mutex
	transport  sensors.I2CBus
	bank       int
	address    byte
	retryLimit int
}

func NewMCP23017(bus sensors.I2CBus, address byte) *MCP23017 {
	return &MCP23017{retryLimit: 1, transport: bus, address: address}
}

// InitA sets IODIR registry to inout on I/O pool A
func (m *MCP23017) InitA(ctx context.Context, inout byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][IODIRA], inout})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not initialize gpio A set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not initialize gpio A set (retry limit reached): %w", err)
}

// InitB sets IODIR registry to inout on I/O pool B
func (m *MCP23017) InitB(ctx context.Context, inout byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][IODIRA], inout})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not initialize gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not initialize gpio B set (retry limit reached): %w", err)
}

func (m *MCP23017) readRegistry(ctx context.Context, addr byte) (byte, error) {
	m.mx.Lock()
	defer m.mx.Unlock()
	err := m.transport.WriteToAddr(ctx, m.address, []byte{addr})
	if err != nil {
		return 0x00, fmt.Errorf("could not set I/O registry address: %w", err)
	}
	buf := make([]byte, 1)
	err = m.transport.ReadFromAddr(ctx, m.address, buf)
	if err != nil {
		return 0x00, fmt.Errorf("could not read gpio data: %w", err)
	}
	return buf[0], nil
}

// PullUpA sets up pull up resistors on set A
func (m *MCP23017) PullUpA(ctx context.Context, settings byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][GPPUA], settings})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not set pull-up on gpio A set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not set pull-up on gpio A set (retry limit reached): %w", err)
}

// PullUpB sets up pull up resistors on set B
func (m *MCP23017) PullUpB(ctx context.Context, settings byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][GPPUB], settings})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not set pull-up on gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not set pull-up on gpio B set (retry limit reached): %w", err)
}

func (m *MCP23017) Read(ctx context.Context) ([]byte, error) {
	res := make([]byte, 2)
	var err error
	res[0], err = m.ReadA(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read gpio set A: %w", err)
	}
	res[1], err = m.ReadB(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read gpio set B: %w", err)
	}
	return res, nil
}

// ReadA reads gpio A set values
func (m *MCP23017) ReadA(ctx context.Context) (byte, error) {
	var err error
	var res byte
	for i := m.retryLimit; i > 0; i-- {
		res, err = m.readRegistry(ctx, BankAddr[m.bank][GPIOA])
		if err == nil {
			return res, nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return res, fmt.Errorf("could not read gpio A set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return res, fmt.Errorf("could not read gpio A set (retry limit reached): %w", err)
}

// ReadB reads gpio B set values
func (m *MCP23017) ReadB(ctx context.Context) (byte, error) {
	var err error
	var res byte
	for i := m.retryLimit; i > 0; i-- {
		res, err = m.readRegistry(ctx, BankAddr[m.bank][GPIOB])
		if err == nil {
			return res, nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return res, fmt.Errorf("could not read gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return res, fmt.Errorf("could not read gpio B set (retry limit reached): %w", err)
}

// ReadSettingsA reads contents of IOCON registry
func (m *MCP23017) ReadSettingsA(ctx context.Context) (byte, error) {
	var err error
	var res byte
	for i := m.retryLimit; i > 0; i-- {
		res, err = m.readRegistry(ctx, BankAddr[m.bank][IOCONA])
		if err == nil {
			return res, nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return res, fmt.Errorf("could not read gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return res, fmt.Errorf("could not read gpio B set (retry limit reached): %w", err)
}

// WriteSettingsA sets up pull up resistors on set A
func (m *MCP23017) WriteSettingsA(ctx context.Context, settings byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][IOCONA], settings})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not write settings on gpio A set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not write settings on gpio A set (retry limit reached): %w", err)
}

// ReadSettingsB reads contents of IOCON registry
func (m *MCP23017) ReadSettingsB(ctx context.Context) (byte, error) {
	var err error
	var res byte
	for i := m.retryLimit; i > 0; i-- {
		res, err = m.readRegistry(ctx, BankAddr[m.bank][IOCONB])
		if err == nil {
			return res, nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return res, fmt.Errorf("could not read gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return res, fmt.Errorf("could not read gpio B set (retry limit reached): %w", err)
}

// WriteSettingsA sets up pull up resistors on set A
func (m *MCP23017) WriteSettingsB(ctx context.Context, settings byte) error {
	var err error
	for i := m.retryLimit; i > 0; i-- {
		err = m.transport.WriteToAddr(ctx, m.address, []byte{BankAddr[m.bank][IOCONB], settings})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sensors.ErrBusBusy) {
			return fmt.Errorf("could not write settings on gpio B set: %w", err)
		}
		// try to release the bus
		_ = m.transport.Release(ctx)
	}
	return fmt.Errorf("could not write settings on gpio B set (retry limit reached): %w", err)
}
