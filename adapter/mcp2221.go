package adapter

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/karalabe/hid"

	"github.com/mklimuk/sensors/cmd/sensors/console"
)

var ErrBusBusy = fmt.Errorf("I2C engine is busy (command not completed)")

const vendorID = 0x04D8
const productID = 0x00DD

type MCP2221 struct {
	mx       sync.Mutex
	request  []byte
	response []byte
}

type MCP2221Status struct {
	I2CDataBufferCounter   int
	I2CSpeedDivider        int
	I2CTimeout             int
	CurrentAddress         string
	LastWriteRequestedSize uint16
	LastWriteSentSize      uint16
	ReadPending            int
}

func NewMCP2221() *MCP2221 {
	return &MCP2221{
		request:  make([]byte, 64),
		response: make([]byte, 64),
	}
}

func (d *MCP2221) WriteToAddr(ctx context.Context, address byte, buffer []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x90
	binary.LittleEndian.PutUint16(d.request[1:3], uint16(len(buffer)))
	d.request[3] = address << 1
	if len(buffer) > 0 {
		copy(d.request[4:], buffer)
	}
	err := d.send(ctx)
	if err != nil {
		return fmt.Errorf("write to %x failed: %w", address, err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		console.Debug("adapter busy")
		return ErrBusBusy
	}
	return nil
}

func (d *MCP2221) ReadFromAddr(ctx context.Context, address byte, buffer []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x91
	binary.LittleEndian.PutUint16(d.request[1:3], uint16(len(buffer)))
	d.request[3] = address<<1 + 1
	err := d.send(ctx)
	// we iterated several times with no result
	if err != nil {
		return fmt.Errorf("bus read from %x failed: %w", address, err)
	}
	d.request[0] = 0x40
	resetBuffer(d.response)
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("error getting read data from adapter: %w", err)
	}
	if d.response[1] == 0x41 {
		return fmt.Errorf("error reading the I2C slave data from the I2C engine")
	}
	if d.response[3] == 127 || int(d.response[3]) != len(buffer) {
		return fmt.Errorf("invalid data size byte; expected %d, got %d", len(buffer), d.response[3])
	}

	copy(buffer, d.response[4:])
	return nil
}

func (d *MCP2221) Status(ctx context.Context) (*MCP2221Status, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x10
	err := d.send(ctx)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	return bufferToStatus(d.response), nil
}

func bufferToStatus(buffer []byte) *MCP2221Status {
	/*
		9: Lower byte (16-bit value) of the requested I2C transfer length
		10: Higher byte (16-bit value) of the requested I2C transfer length
		11:	Lower byte (16-bit value) of the already transferred (through I2C) number of bytes
		12:	Higher byte (16-bit value) of the already transferred (through I2C) number of bytes
		13:	Internal I2C data buffer counter
		14: Current I2C communication speed divider value
		15: Current I2C timeout value
		16:	Lower byte (16-bit value) of the I2C address being used
		17:	Higher byte (16-bit value) of the I2C address being used
	*/
	status := &MCP2221Status{
		I2CDataBufferCounter: int(buffer[13]),
		I2CSpeedDivider:      int(buffer[14]),
		I2CTimeout:           int(buffer[15]),
		ReadPending:          int(buffer[25]),
		CurrentAddress:       hex.EncodeToString(buffer[16:18]),
	}
	status.LastWriteRequestedSize = binary.LittleEndian.Uint16(buffer[9:11])
	status.LastWriteSentSize = binary.LittleEndian.Uint16(buffer[11:13])
	return status
}

func (d *MCP2221) ReleaseBus(ctx context.Context) (*MCP2221Status, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	return d.releaseBus(ctx)
}

func (d *MCP2221) releaseBus(ctx context.Context) (*MCP2221Status, error) {
	d.resetBuffers()
	d.request[0] = 0x10
	d.request[2] = 0x10
	err := d.send(ctx)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	return bufferToStatus(d.response), nil
}

func (d *MCP2221) send(ctx context.Context) error {
	devs := hid.Enumerate(vendorID, productID)
	if len(devs) > 1 {
		return fmt.Errorf("ambiguous device identification")
	}
	if len(devs) == 0 {
		return fmt.Errorf("MCP2221 device not found")
	}
	dev, err := devs[0].Open()
	if err != nil {
		return fmt.Errorf("error opening device: %w", err)
	}
	defer dev.Close()
	verbose := console.IsVerbose(ctx)
	if verbose {
		console.Printf("sending message to adapter:\n%s\n", hex.Dump(d.request))
	}
	n, err := dev.Write(d.request)
	if err != nil {
		return fmt.Errorf("could not write request: %w", err)
	}
	if n != 64 {
		return fmt.Errorf("short write: %d", n)
	}
	time.Sleep(50 * time.Millisecond)
	console.Info("reading response from adapter")
	n, err = dev.Read(d.response)
	if err != nil {
		return fmt.Errorf("could not read response: %w", err)
	}
	if n != 64 {
		return fmt.Errorf("short read: %d", n)
	}
	if verbose {
		console.Printf("read message from adapter:\n%s\n", hex.Dump(d.response))
	}
	return nil
}

func (d *MCP2221) resetBuffers() {
	resetBuffer(d.request)
	resetBuffer(d.response)
}

func resetBuffer(buf []byte) {
	for i := 0; i < len(buf)-1; i++ {
		buf[i] = 0x00
	}
}
