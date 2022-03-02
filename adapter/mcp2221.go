package adapter

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mklimuk/sensors"

	"github.com/karalabe/hid"

	"github.com/mklimuk/sensors/cmd/sensors/console"
)

const VendorID = 0x04D8
const ProductID = 0x00DD

var ErrCommandUnsupported = errors.New("unsupported command")
var ErrCommandFailed = errors.New("command failed")

type MCP2221 struct {
	mx           sync.Mutex
	request      []byte
	response     []byte
	responseWait time.Duration
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

type GPIOMode byte

const (
	GPIOModeOut         GPIOMode = 0b00000000
	GPIOModeIn          GPIOMode = 0b00001000
	GPIOModeNoOperation GPIOMode = 0xEF
)

func (m GPIOMode) String() string {
	switch m {
	case GPIOModeIn:
		return "INPUT"
	case GPIOModeOut:
		return "OUTPUT"
	default:
		return "NOOP"
	}
}

type GPIODesignation byte

const (
	GPIOOperation GPIODesignation = 0b00000000
	// This is alternate function of GPIO0
	GPIO0LedUartRx GPIODesignation = 0b00000001
	// This is the dedicated function operation of GPIO0
	GPIO0SSPND GPIODesignation = 0b00000010
	// This is the dedicated function of GPIO1
	GPIO1ClockOutput GPIODesignation = 0b00000001
	// This is the alternate function 0 of GPIO1
	GPIO1ADC1 GPIODesignation = 0b00000010
	// This is the alternate function 1 of GPIO1
	GPIO1LedUartTx GPIODesignation = 0b00000011
	// This is the alternate function 2 of GPIO1
	GPIO1InterruptDetection GPIODesignation = 0b00000100
	// This is the dedicated function of GPIO2
	GPIO2ClockOutput GPIODesignation = 0b00000001
	// This is the alternate function 0 of GPIO2
	GPIO2ADC2 GPIODesignation = 0b00000010
	// This is the alternate function 1 of GPIO2
	GPIO2DAC1 GPIODesignation = 0b00000011
	// This is the dedicated function of GPIO3
	GPIO3LEDI2C GPIODesignation = 0b00000001
	// This is the alternate function 0 of GPIO3
	GPIO3ADC3 GPIODesignation = 0b00000010
	// This is the alternate function 1 of GPIO3
	GPIO3DAC2 GPIODesignation = 0b00000011
)

const gpioModeMask = 0b00001000
const gpioOperationMask = 0b00000111

type MCP2221GPIOValues struct {
	GPIO0Mode  GPIOMode `yaml:"GP0_mode"`
	GPIO0Value byte     `yaml:"GPIO0"`
	GPIO1Mode  GPIOMode `yaml:"GP1_mode"`
	GPIO1Value byte     `yaml:"GPIO1"`
	GPIO2Mode  GPIOMode `yaml:"GP2_mode"`
	GPIO2Value byte     `yaml:"GPIO2"`
	GPIO3Mode  GPIOMode `yaml:"GP3_mode"`
	GPIO3Value byte     `yaml:"GPIO3"`
}

type MCP2221GPIOParameters struct {
	GPIO0Mode        GPIOMode        `yaml:"GP0_mode"`
	GPIO0Designation GPIODesignation `yaml:"GP0_designation"`
	GPIO1Mode        GPIOMode        `yaml:"GP1_mode"`
	GPIO1Designation GPIODesignation `yaml:"GP1_designation"`
	GPIO2Mode        GPIOMode        `yaml:"GP2_mode"`
	GPIO2Designation GPIODesignation `yaml:"GP2_designation"`
	GPIO3Mode        GPIOMode        `yaml:"GP3_mode"`
	GPIO3Designation GPIODesignation `yaml:"GP3_designation"`
}

func NewMCP2221() *MCP2221 {
	return &MCP2221{
		request:      make([]byte, 64),
		response:     make([]byte, 64),
		responseWait: 50 * time.Millisecond,
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
	err := d.send(ctx, true)
	if err != nil {
		return fmt.Errorf("write to %x failed: %w", address, err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		console.Debug("adapter busy")
		return sensors.ErrBusBusy
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
	err := d.send(ctx, true)
	// we iterated several times with no result
	if err != nil {
		return fmt.Errorf("bus read from %x failed: %w", address, err)
	}
	d.request[0] = 0x40
	resetBuffer(d.response)
	err = d.send(ctx, true)
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

func (d *MCP2221) SetGPIOParameters(ctx context.Context, params MCP2221GPIOParameters) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0xB1
	d.request[1] = 0x01
	d.request[2] = byte(params.GPIO0Designation) | byte(params.GPIO0Mode)
	d.request[3] = byte(params.GPIO1Designation) | byte(params.GPIO1Mode)
	d.request[4] = byte(params.GPIO2Designation) | byte(params.GPIO2Mode)
	d.request[5] = byte(params.GPIO3Designation) | byte(params.GPIO3Mode)
	err := d.send(ctx, true)
	if err != nil {
		return fmt.Errorf("set GP parameters command write failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return ErrCommandFailed
	}
	return nil
}

func (d *MCP2221) Read(ctx context.Context, id ...int) ([]byte, error) {
	res, err := d.ReadGPIO(ctx, id...)
	if err != nil {
		return nil, err
	}
	return []byte{res.GPIO0Value, res.GPIO1Value, res.GPIO2Value, res.GPIO3Value}, nil
}

func (d *MCP2221) ReadGPIO(ctx context.Context, id ...int) (MCP2221GPIOValues, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x51
	err := d.send(ctx, true, id...)
	var res MCP2221GPIOValues
	if err != nil {
		return res, fmt.Errorf("read GPIO values command write failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return res, ErrCommandFailed
	}
	res.GPIO0Mode = GPIOModeNoOperation
	res.GPIO0Value = d.response[2]
	if d.response[3] != byte(GPIOModeNoOperation) {
		res.GPIO0Mode = GPIOMode(d.response[3] << 3)
	}
	res.GPIO1Mode = GPIOModeNoOperation
	res.GPIO1Value = d.response[4]
	if d.response[5] != byte(GPIOModeNoOperation) {
		res.GPIO1Mode = GPIOMode(d.response[5] << 3)
	}
	res.GPIO2Mode = GPIOModeNoOperation
	res.GPIO2Value = d.response[6]
	if d.response[7] != byte(GPIOModeNoOperation) {
		res.GPIO2Mode = GPIOMode(d.response[7] << 3)
	}
	res.GPIO3Mode = GPIOModeNoOperation
	res.GPIO3Value = d.response[8]
	if d.response[9] != byte(GPIOModeNoOperation) {
		res.GPIO3Mode = GPIOMode(d.response[9] << 3)
	}
	return res, nil
}

func (d *MCP2221) GetGPIOParameters(ctx context.Context) (MCP2221GPIOParameters, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0xB0
	d.request[1] = 0x01
	err := d.send(ctx, true)
	if err != nil {
		return MCP2221GPIOParameters{}, fmt.Errorf("get GP parameters command write failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return MCP2221GPIOParameters{}, ErrCommandUnsupported
	}
	return MCP2221GPIOParameters{
		GPIO0Mode:        GPIOMode(d.response[4] & gpioModeMask),
		GPIO0Designation: GPIODesignation(d.response[4] & gpioOperationMask),
		GPIO1Mode:        GPIOMode(d.response[5] & gpioModeMask),
		GPIO1Designation: GPIODesignation(d.response[5] & gpioOperationMask),
		GPIO2Mode:        GPIOMode(d.response[6] & gpioModeMask),
		GPIO2Designation: GPIODesignation(d.response[6] & gpioOperationMask),
		GPIO3Mode:        GPIOMode(d.response[7] & gpioModeMask),
		GPIO3Designation: GPIODesignation(d.response[7] & gpioOperationMask),
	}, nil
}

func (d *MCP2221) Status(ctx context.Context) (*MCP2221Status, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x10
	err := d.send(ctx, true)
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

func (d *MCP2221) Release(ctx context.Context) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	_, err := d.releaseBus(ctx)
	return err
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
	err := d.send(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	return bufferToStatus(d.response), nil
}

func (d *MCP2221) send(ctx context.Context, response bool, id ...int) error {
	devs := hid.Enumerate(VendorID, ProductID)
	if len(devs) > 1 && len(id) == 0 {
		return fmt.Errorf("ambiguous device identification")
	}
	if len(devs) == 0 {
		return fmt.Errorf("MCP2221 device not found")
	}
	var dev *hid.Device
	var err error
	if len(id) == 0 {
		dev, err = devs[0].Open()
		if err != nil {
			return fmt.Errorf("error opening device: %w", err)
		}
	} else {
		for d := range devs {
			if d == id[0] {
				dev, err = devs[0].Open()
				if err != nil {
					return fmt.Errorf("error opening device: %w", err)
				}
			}
		}
		if dev == nil {
			return fmt.Errorf("no device with id %d", id[0])
		}
	}
	defer func() {
		err := dev.Close()
		if err != nil {
		}
	}()
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
	if !response {
		return nil
	}
	time.Sleep(d.responseWait)
	console.Debug("reading response from adapter")
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
