// Package eeprom provides a Gobot driver for the Microchip 25AA1024 1‑Mbit SPI EEPROM.
// It supports basic read & write operations with automatic page handling and status‑polling.
//
// Datasheet reference: Microchip 25AA1024 Serial EEPROM (Table 3‑1 Instruction Set, page size 256 bytes).
// available in `spec/25aa1024.pdf`
//
// Tested on NanoPi using the Gobot sysfs SPI adaptor, but should work on any board that
// exposes a compliant spi.Connection.
//
// Example usage:
//
//	adaptor := sysfs.NewAdaptor()
//	e := eeprom.New(adaptor, 0, 0) // bus 0, chip‑select 0
//	if err := e.Start(); err != nil { log.Fatal(err) }
//	data, _ := e.Read(0x0000, 16)
//	fmt.Printf("First 16 bytes: %x\n", data)
//
//	err := e.Write(0x1000, []byte("gobot‑rocks"))
//	if err != nil { log.Fatal(err) }
//
//	_ = e.Halt() // optional on shutdown
package eeprom

import (
	"fmt"
	"time"

	"gobot.io/x/gobot/v2/drivers/spi"
)

// --- device constants (datasheet Table 3‑1) ---
const (
	cmdRead  = 0x03 // READ
	cmdWrite = 0x02 // WRITE
	cmdWREN  = 0x06 // WREN (Write‑Enable Latch set)
	cmdWRDI  = 0x04 // WRDI (Write‑Enable Latch clear)
	cmdRDSR  = 0x05 // Read STATUS Register
	cmdWRSR  = 0x01 // Write STATUS Register
	cmdPE    = 0x42 // Page Erase
	cmdSE    = 0xD8 // Sector Erase
	cmdCE    = 0xC7 // Chip Erase

	statusWIP = 0x01 // STATUS bit 0 – Write‑In‑Progress

	pageSize = 256    // bytes per page
	capacity = 131072 // 1 Mbit = 128 KiB total bytes
)

// EEPROM25AA1024 implements gobot.Driver for the 25AA1024 device.
type EEPROM25AA1024 struct {
	*spi.Driver
}

// New returns a new driver bound to a Gobot SPI adaptor. bus and cs are the SPI bus
// number and chip‑select line, matching the board’s numbering.
// Additional driver options (e.g. speed) may be supplied as in other Gobot SPI drivers.
func New(adaptor spi.Connector, bus string, cs byte, opts ...func(spi.Config)) *EEPROM25AA1024 {
	d := spi.NewDriver(adaptor, bus, opts...)

	// Ensure we comply with datasheet limits: mode 0 (CPOL=0, CPHA=0) up to 20 MHz.
	d.SetMode(0)

	if d.GetSpeedOrDefault(0) == 0 {
		d.SetSpeed(5_000_000) // conservative default 5 MHz
	}

	return &EEPROM25AA1024{Driver: d}
}

// Start establishes the SPI bus. Required by Gobot.Driver interface.
func (e *EEPROM25AA1024) Start() error { return e.Driver.Start() }

// Halt releases the bus. Optional.
func (e *EEPROM25AA1024) Halt() error { return e.Driver.Halt() }

// Transfer performs a full‑duplex SPI transaction.
//
// Semantics follow the typical SPI rules:
//   - If rx is non‑nil, it must be at least the length of tx. Received bytes are
//     written into rx.
//   - If rx is nil, received bytes are discarded.
//   - If tx is nil and rx is non‑nil, zeros are clocked out while reading (if the
//     underlying connection supports it).
//
// This simply delegates to the underlying Gobot SPI driver connection.
func (e *EEPROM25AA1024) Transfer(tx []byte, rx []byte) error {
	if e == nil || e.Driver == nil {
		return fmt.Errorf("spi driver not initialized")
	}
	// Access the underlying SPI connection via Gobot driver.
	conn := e.Driver.Connection()
	// Define the subset of operations we need from the SPI connection.
	type spiOps interface {
		ReadCommandData(command []byte, data []byte) error
		WriteBytes(data []byte) error
	}
	ops, ok := conn.(spiOps)
	if !ok {
		return fmt.Errorf("spi connection does not support required operations")
	}

	// Write-only transaction
	if rx == nil || len(rx) == 0 {
		if len(tx) == 0 {
			return nil
		}
		return ops.WriteBytes(tx)
	}

	// Read transaction with command header and dummy bytes.
	if len(tx) != len(rx) {
		return fmt.Errorf("tx/rx length mismatch: %d != %d", len(tx), len(rx))
	}

	// Heuristics based on 25AA1024 protocol to split header and data lengths
	// so we can use ReadCommandData(command, data).
	headerLen := 1
	switch tx[0] {
	case cmdRead:
		// READ: opcode + 24-bit address
		headerLen = 4
	case cmdRDSR:
		headerLen = 1
	default:
		// Reasonable default for single-byte commands that expect a response.
		headerLen = 1
	}
	if headerLen > len(tx) {
		headerLen = len(tx)
	}
	dataLen := len(tx) - headerLen
	tmp := make([]byte, dataLen)
	if err := ops.ReadCommandData(tx[:headerLen], tmp); err != nil {
		return err
	}
	// Emulate full-duplex buffer: prefix (header echo) is undefined; callers
	// skip it anyway. Place received data after the header area.
	copy(rx[:headerLen], make([]byte, headerLen))
	copy(rx[headerLen:], tmp)
	return nil
}

// Read returns length bytes starting at address. Reads that exceed the device’s
// capacity return an error.
func (e *EEPROM25AA1024) Read(address uint32, length int) ([]byte, error) {
	if address+uint32(length) > capacity {
		return nil, fmt.Errorf("read out of range")
	}
	// Build command + 24‑bit address (only A16..A0 used, seven MSB are “don’t care”).
	header := []byte{cmdRead, byte(address >> 16), byte(address >> 8), byte(address)}

	tx := append(header, make([]byte, length)...) // dummy bytes clock out data
	rx := make([]byte, len(tx))

	if err := e.Transfer(tx, rx); err != nil {
		return nil, err
	}

	return rx[4:], nil // skip echoed header
}

// Write writes data at the given start address. It automatically pages data into
// <=256‑byte chunks, as required by the device, and polls the STATUS register
// until each internal write cycle completes.
func (e *EEPROM25AA1024) Write(address uint32, data []byte) error {
	if address+uint32(len(data)) > capacity {
		return fmt.Errorf("write out of range")
	}

	offset := 0
	for offset < len(data) {
		pageOffset := address % pageSize
		space := pageSize - pageOffset
		chunk := data[offset:]
		if len(chunk) > int(space) {
			chunk = chunk[:space]
		}

		if err := e.pageWrite(address, chunk); err != nil {
			return err
		}
		offset += len(chunk)
		address += uint32(len(chunk))
	}
	return nil
}

// --- helpers ---
func (e *EEPROM25AA1024) writeEnable() error {
	return e.Transfer([]byte{cmdWREN}, nil)
}

func (e *EEPROM25AA1024) readStatus() (byte, error) {
	rx := make([]byte, 2)
	if err := e.Transfer([]byte{cmdRDSR, 0x00}, rx); err != nil {
		return 0, err
	}
	return rx[1], nil
}

func (e *EEPROM25AA1024) waitUntilReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := e.readStatus()
		if err != nil {
			return err
		}
		if st&statusWIP == 0 {
			return nil // ready
		}
		time.Sleep(500 * time.Microsecond)
	}
	return fmt.Errorf("timeout waiting for write completion")
}

func (e *EEPROM25AA1024) pageWrite(address uint32, data []byte) error {
	if len(data) == 0 || len(data) > pageSize {
		return fmt.Errorf("invalid page size")
	}
	if err := e.writeEnable(); err != nil {
		return err
	}

	header := []byte{cmdWrite, byte(address >> 16), byte(address >> 8), byte(address)}
	tx := append(header, data...)

	if err := e.Transfer(tx, nil); err != nil {
		return err
	}

	// Internal write cycle (max 6 ms per datasheet). Poll STATUS.WIP.
	return e.waitUntilReady(10 * time.Millisecond)
}
