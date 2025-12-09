package adapter

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mklimuk/sensors"

	"github.com/karalabe/hid"

	"github.com/mklimuk/sensors/cmd/sensors/console"
	"github.com/mklimuk/sensors/snsctx"
)

/*
https://www.microchip.com/en-us/product/MCP2221

Bus 002 Device 003: ID 04d8:00dd Microchip Technology, Inc. MCP2221(a) UART/I2C Bridge
Device Descriptor:
  bLength                18
  bDescriptorType         1
  bcdUSB               2.00
  bDeviceClass          239 Miscellaneous Device
  bDeviceSubClass         2
  bDeviceProtocol         1 Interface Association
  bMaxPacketSize0         8
  idVendor           0x04d8 Microchip Technology, Inc.
  idProduct          0x00dd MCP2221(a) UART/I2C Bridge
  bcdDevice            1.00
  iManufacturer           1 Microchip Technology Inc.
  iProduct                2 MCP2221 USB-I2C/UART Combo
  iSerial                 0
  bNumConfigurations      1
  Configuration Descriptor:
    bLength                 9
    bDescriptorType         2
    wTotalLength       0x006b
    bNumInterfaces          3
    bConfigurationValue     1
    iConfiguration          0
    bmAttributes         0x80
      (Bus Powered)
    MaxPower              100mA
    Interface Association:
      bLength                 8
      bDescriptorType        11
      bFirstInterface         0
      bInterfaceCount         2
      bFunctionClass          2 Communications
      bFunctionSubClass       2 Abstract (modem)
      bFunctionProtocol       1 AT-commands (v.25ter)
      iFunction               0
    Interface Descriptor:
      bLength                 9
      bDescriptorType         4
      bInterfaceNumber        0
      bAlternateSetting       0
      bNumEndpoints           1
      bInterfaceClass         2 Communications
      bInterfaceSubClass      2 Abstract (modem)
      bInterfaceProtocol      1 AT-commands (v.25ter)
      iInterface              0
      CDC Header:
        bcdCDC               1.10
      CDC ACM:
        bmCapabilities       0x02
          line coding and serial state
      CDC Union:
        bMasterInterface        0
        bSlaveInterface         1
      CDC Call Management:
        bmCapabilities       0x00
        bDataInterface          1
      Endpoint Descriptor:
        bLength                 7
        bDescriptorType         5
        bEndpointAddress     0x81  EP 1 IN
        bmAttributes            3
          Transfer Type            Interrupt
          Synch Type               None
          Usage Type               Data
        wMaxPacketSize     0x0008  1x 8 bytes
        bInterval               2
    Interface Descriptor:
      bLength                 9
      bDescriptorType         4
      bInterfaceNumber        1
      bAlternateSetting       0
      bNumEndpoints           2
      bInterfaceClass        10 CDC Data
      bInterfaceSubClass      0
      bInterfaceProtocol      0
      iInterface              0
      Endpoint Descriptor:
        bLength                 7
        bDescriptorType         5
        bEndpointAddress     0x02  EP 2 OUT
        bmAttributes            2
          Transfer Type            Bulk
          Synch Type               None
          Usage Type               Data
        wMaxPacketSize     0x0010  1x 16 bytes
        bInterval               0
      Endpoint Descriptor:
        bLength                 7
        bDescriptorType         5
        bEndpointAddress     0x82  EP 2 IN
        bmAttributes            2
          Transfer Type            Bulk
          Synch Type               None
          Usage Type               Data
        wMaxPacketSize     0x0010  1x 16 bytes
        bInterval               0
    Interface Descriptor:
      bLength                 9
      bDescriptorType         4
      bInterfaceNumber        2
      bAlternateSetting       0
      bNumEndpoints           2
      bInterfaceClass         3 Human Interface Device
      bInterfaceSubClass      0
      bInterfaceProtocol      0
      iInterface              0
        HID Device Descriptor:
          bLength                 9
          bDescriptorType        33
          bcdHID               1.11
          bCountryCode            0 Not supported
          bNumDescriptors         1
          bDescriptorType        34 Report
          wDescriptorLength      28
         Report Descriptors:
           ** UNAVAILABLE **
      Endpoint Descriptor:
        bLength                 7
        bDescriptorType         5
        bEndpointAddress     0x83  EP 3 IN
        bmAttributes            3
          Transfer Type            Interrupt
          Synch Type               None
          Usage Type               Data
        wMaxPacketSize     0x0040  1x 64 bytes
        bInterval               1
      Endpoint Descriptor:
        bLength                 7
        bDescriptorType         5
        bEndpointAddress     0x03  EP 3 OUT
        bmAttributes            3
          Transfer Type            Interrupt
          Synch Type               None
          Usage Type               Data
        wMaxPacketSize     0x0040  1x 64 bytes
        bInterval               1
Device Status:     0x0000
  (Bus Powered)

MCP2221 uses a dedicated kernel driver:

hid_mcp2221            20480  0
hid                   159744  3 usbhid,hid_generic,hid_mcp2221
*/

const VendorID = 0x04D8
const ProductID = 0x00DD

type GPIODesignation byte

const (
	GPIOOperation GPIODesignation = 0b00000000
	// GPIO0LedUartRx This is alternate function of GPIO0
	GPIO0LedUartRx GPIODesignation = 0b00000001
	// GPIO0SSPND This is the dedicated function operation of GPIO0
	GPIO0SSPND GPIODesignation = 0b00000010
	// GPIO1ClockOutput This is the dedicated function of GPIO1
	GPIO1ClockOutput GPIODesignation = 0b00000001
	// GPIO1ADC1 This is the alternate function 0 of GPIO1
	GPIO1ADC1 GPIODesignation = 0b00000010
	// GPIO1LedUartTx This is the alternate function 1 of GPIO1
	GPIO1LedUartTx GPIODesignation = 0b00000011
	// GPIO1InterruptDetection This is the alternate function 2 of GPIO1
	GPIO1InterruptDetection GPIODesignation = 0b00000100
	// GPIO2ClockOutput This is the dedicated function of GPIO2
	GPIO2ClockOutput GPIODesignation = 0b00000001
	// GPIO2ADC2 This is the alternate function 0 of GPIO2
	GPIO2ADC2 GPIODesignation = 0b00000010
	// GPIO2DAC1 This is the alternate function 1 of GPIO2
	GPIO2DAC1 GPIODesignation = 0b00000011
	// GPIO3LEDI2C This is the dedicated function of GPIO3
	GPIO3LEDI2C GPIODesignation = 0b00000001
	// GPIO3ADC3 This is the alternate function 0 of GPIO3
	GPIO3ADC3 GPIODesignation = 0b00000010
	// GPIO3DAC2 This is the alternate function 1 of GPIO3
	GPIO3DAC2 GPIODesignation = 0b00000011
)

var ErrCommandUnsupported = errors.New("unsupported command")
var ErrCommandFailed = errors.New("command failed")
var ErrNotConnected = errors.New("not connected")
var ErrNoReconnectChannel = errors.New("reconnect channel not initialized")
var ErrI2CStatusTimeout = errors.New("i2c status check timeout")
var ErrI2CAddressMismatch = errors.New("i2c address mismatch")

const (
	StatusNew = iota
	StatusInitialized
	StatusConnecting
	StatusConnected
)

var (
	chipDelay = 5 * time.Millisecond
	i2cDelay  = 50 * time.Millisecond
	maxDelay  = 75 * time.Millisecond
)

type MCP2221 struct {
	mx           sync.Mutex
	request      []byte
	response     []byte
	responseWait time.Duration
	vendorID     uint16
	productID    uint16
	device       *hid.Device
}

type MCP2221Status struct {
	I2CDataBufferCounter   int
	I2CSpeedDivider        int
	I2CTimeout             int
	CurrentAddress         string
	I2CAddress             byte
	LastI2CRequestedSize   uint16
	LastI2CTransferredSize uint16
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

const gpioModeMask = 0b00001000
const gpioOperationMask = 0b00000111

type MCP2221GPIOValues struct {
	GPIO0Mode  GPIOMode `json:"gp0_mode" yaml:"GP0_mode"`
	GPIO0Value byte     `json:"gp0" yaml:"GPIO0"`
	GPIO1Mode  GPIOMode `json:"gp1_mode" yaml:"GP1_mode"`
	GPIO1Value byte     `json:"gp1" yaml:"GPIO1"`
	GPIO2Mode  GPIOMode `json:"gp2_mode" yaml:"GP2_mode"`
	GPIO2Value byte     `json:"gp2" yaml:"GPIO2"`
	GPIO3Mode  GPIOMode `json:"gp3_mode" yaml:"GP3_mode"`
	GPIO3Value byte     `json:"gp3" yaml:"GPIO3"`
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
		vendorID:     VendorID,
		productID:    ProductID,
	}
}

func (d *MCP2221) Init() error {
	d.mx.Lock()
	defer d.mx.Unlock()
	// karalabe/hid doesn't need explicit initialization
	return nil
}

func (d *MCP2221) connect() error {
	// Enumerate devices to find the MCP2221
	devices := hid.Enumerate(d.vendorID, d.productID)
	if len(devices) == 0 {
		return fmt.Errorf("could not find hid device vendor: %#x product: %#x", d.vendorID, d.productID)
	}

	// Open the first device found
	device, err := devices[0].Open()
	if err != nil {
		return fmt.Errorf("could not open hid device vendor: %#x product: %#x: %w", d.vendorID, d.productID, err)
	}
	d.device = device
	return nil
}

func (d *MCP2221) disconnect() error {
	if d.device != nil {
		err := d.device.Close()
		if err != nil {
			return fmt.Errorf("could not close hid device: %w", err)
		}
		d.device = nil
	}
	return nil
}

func (d *MCP2221) SetVendorAndProductID(vendor, product uint16) {
	d.vendorID = vendor
	d.productID = product
}

func (d *MCP2221) WriteToAddr(ctx context.Context, address byte, buffer []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x90
	binary.LittleEndian.PutUint16(d.request[1:3], uint16(len(buffer)))
	addr := address << 1
	d.request[3] = addr
	if len(buffer) > 0 {
		copy(d.request[4:], buffer)
	}
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("i2c write to %x request write failed: %w", address, err)
	}
	err = d.waitAndReceive(ctx, chipDelay)
	if err != nil {
		return fmt.Errorf("i2c write to %x response read failed: %w", address, err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		slog.Debug("i2c bus busy, releasing bus", "state", d.response[2])
		_, err = d.doReleaseBus(ctx)
		if err != nil {
			return fmt.Errorf("%w; could not release bus: %v", sensors.ErrBusBusy, err)
		}
	}
	return nil
}

func (d *MCP2221) ReadFromAddr(ctx context.Context, address byte, buffer []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	// send i2c read request
	d.request[0] = 0x91
	binary.LittleEndian.PutUint16(d.request[1:3], uint16(len(buffer)))
	addr := address<<1 + 1
	d.request[3] = addr
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	// we iterated several times with no result
	if err != nil {
		return fmt.Errorf("i2c read from %x request failed: %w", address, err)
	}
	err = d.receive(ctx)
	if err != nil {
		return fmt.Errorf("i2c read from %x response receive failed: %w", address, err)
	}
	if d.response[1] == 0x01 {
		slog.Debug("i2c bus busy, releasing bus", "state", d.response[2])
		_, err = d.doReleaseBus(ctx)
		if err != nil {
			return fmt.Errorf("%w; could not release bus: %v", sensors.ErrBusBusy, err)
		}
		return sensors.ErrBusBusy
	}
	// read i2c data
	d.request[0] = 0x40
	resetBuffer(d.response)
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("error getting i2c read data from adapter: %w", err)
	}
	err = d.waitAndReceive(ctx, chipDelay)
	if err != nil {
		return fmt.Errorf("i2c read from %x response receive failed: %w", address, err)
	}
	if d.response[1] == 0x41 {
		return fmt.Errorf("error reading the i2c slave data from the i2c engine")
	}
	if d.response[3] == 127 || int(d.response[3]) != len(buffer) {
		return fmt.Errorf("invalid data size byte; expected %d, got %d", len(buffer), d.response[3])
	}

	copy(buffer, d.response[4:])
	return nil
}

func (d *MCP2221) ReadChipSettings(ctx context.Context) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0xB0
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("read chip parameters command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return fmt.Errorf("read chip parameters command response read failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return ErrCommandFailed
	}
	dump("chip settings:", d.response[:14])
	return nil
}

func (d *MCP2221) ReadGPIOSettings(ctx context.Context) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0xB0
	d.request[1] = 0x01
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("read gpio parameters command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return fmt.Errorf("read gpio parameters command response read failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return ErrCommandFailed
	}
	dump("gpio settings:", d.response[:14])
	return nil
}

func dump(description string, value []byte) {
	fmt.Println(description)
	for i, b := range value {
		fmt.Printf("%d | %08b | %#02x\n", i, b, b)
	}
}

func (d *MCP2221) UpdateVendorAndProductID(ctx context.Context, vendor, product []byte, dryrun ...bool) error {
	for len(vendor) < 2 {
		vendor = append(vendor, 0x00)
	}
	for len(product) < 2 {
		product = append(product, 0x00)
	}
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0xB1 // command
	d.request[1] = 0x00 // subcommand
	//d.request[2] = 0xFC // set usb CDC serial number
	d.request[2] = 0x7C // set usb CDC serial number
	d.request[3] = 0x12 // clock divider
	d.request[4] = 0x88 // reference voltage DAC
	d.request[5] = 0x6F // reference voltage ADC
	d.request[6] = vendor[1]
	d.request[7] = vendor[0]
	d.request[8] = product[1]
	d.request[9] = product[0]
	d.request[10] = 0x80 // usb power attributes
	d.request[11] = 0x32 // usb requested mA value

	if len(dryrun) > 0 && dryrun[0] {
		dump("sent chip settings:", d.request[:12])
		return nil
	}
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("write chip parameters command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return fmt.Errorf("write chip parameters command response read failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return ErrCommandFailed
	}
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
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("set GP parameters command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return fmt.Errorf("set GP parameters command response read failed: %w", err)
	}
	// read could not be performed
	if d.response[1] == 0x01 {
		return ErrCommandFailed
	}
	return nil
}

func (d *MCP2221) Read(ctx context.Context) ([]byte, error) {
	res, err := d.ReadGPIO(ctx)
	if err != nil {
		return nil, err
	}
	return []byte{res.GPIO0Value, res.GPIO1Value, res.GPIO2Value, res.GPIO3Value}, nil
}

func (d *MCP2221) ReadGPIO(ctx context.Context) (MCP2221GPIOValues, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x51
	err := d.connect()
	if err != nil {
		return MCP2221GPIOValues{}, fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	var res MCP2221GPIOValues
	if err != nil {
		return res, fmt.Errorf("read GPIO values command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return res, fmt.Errorf("read GPIO values command response read failed: %w", err)
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
	err := d.connect()
	if err != nil {
		return MCP2221GPIOParameters{}, fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return MCP2221GPIOParameters{}, fmt.Errorf("get GP parameters command request write failed: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return MCP2221GPIOParameters{}, fmt.Errorf("get GP parameters command response read failed: %w", err)
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
	return d.doGetStatus(ctx)
}

func (d *MCP2221) doGetStatus(ctx context.Context) (*MCP2221Status, error) {
	d.resetBuffers()
	d.request[0] = 0x10
	err := d.connect()
	if err != nil {
		return nil, fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not send status request: %w", err)
	}
	err = d.receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not receive status: %w", err)
	}
	return bufferToStatus(d.response), nil
}

func (d *MCP2221) Reset(ctx context.Context) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.resetBuffers()
	d.request[0] = 0x70
	d.request[1] = 0xAB
	d.request[2] = 0xCD
	d.request[3] = 0xEF
	err := d.connect()
	if err != nil {
		return fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	err = d.send(ctx)
	if err != nil {
		return fmt.Errorf("reset request failed: %w", err)
	}
	return nil
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
	status.LastI2CRequestedSize = binary.LittleEndian.Uint16(buffer[9:11])
	status.LastI2CTransferredSize = binary.LittleEndian.Uint16(buffer[11:13])
	status.I2CAddress = buffer[16]
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
	err := d.connect()
	if err != nil {
		return nil, fmt.Errorf("could not connect to mcp2221: %w", err)
	}
	defer func() {
		err = d.disconnect()
		if err != nil {
			slog.Error("could not disconnect from mcp2221", "err", err)
		}
	}()
	return d.doReleaseBus(ctx)
}

func (d *MCP2221) doReleaseBus(ctx context.Context) (*MCP2221Status, error) {
	d.resetBuffers()
	d.request[0] = 0x10
	d.request[2] = 0x10
	err := d.send(ctx)
	if err != nil {
		return nil, fmt.Errorf("release request failed: %w", err)
	}
	err = d.waitAndReceive(ctx, chipDelay)
	if err != nil {
		return nil, fmt.Errorf("release response read failed: %w", err)
	}
	return bufferToStatus(d.response), nil
}

func (d *MCP2221) waitForI2CTransfer(ctx context.Context, address byte) error {
	timeout := time.NewTimer(maxDelay)
	defer timeout.Stop()
	tick := time.NewTicker(chipDelay)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			status, err := d.doGetStatus(ctx)
			if err != nil {
				slog.Error("could not get status", "err", err)
			}
			if status.I2CAddress != 0x00 && status.I2CAddress != address {
				return fmt.Errorf("%w; expected %#x, got %#x", ErrI2CAddressMismatch, address, status.I2CAddress)
			}
			if status.LastI2CRequestedSize == status.LastI2CTransferredSize {
				return nil
			}
		case <-timeout.C:
			return ErrI2CStatusTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *MCP2221) waitAndReceive(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	err := d.receive(ctx)
	if err != nil {
		return fmt.Errorf("i2c receive failed: %w", err)
	}
	return nil
}

func (d *MCP2221) send(ctx context.Context) error {
	verbose := snsctx.IsVerbose(ctx)
	if verbose {
		console.Printf("sending message to mcp2221:\n%s\n", hex.Dump(d.request))
	}

	n, err := d.device.Write(d.request)
	if err != nil {
		return fmt.Errorf("could not write request: %w", err)
	}
	if n != 64 {
		return fmt.Errorf("short write: %d", n)
	}
	return nil
}

// receive reads the response from the device
func (d *MCP2221) receive(ctx context.Context) error {
	n, err := d.device.Read(d.response)
	if err != nil {
		return fmt.Errorf("could not read response: %w", err)
	}
	if n != 64 {
		return fmt.Errorf("short read: %d", n)
	}
	verbose := snsctx.IsVerbose(ctx)
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
