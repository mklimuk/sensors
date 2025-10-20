package air

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mklimuk/sensors"
)

// AGS02MA default 7-bit I2C address is 0x1A.
// Datasheet also mentions write/read instructions 0x34/0x35 which are the
// 8-bit bus addresses (0x1A<<1 | 0 for write, | 1 for read) used on the wire.
const ags02maAddress = 0x1A

// Register/command map (per datasheet)
//
//	0x00: TVOC readout (first byte is status, next three bytes are TVOC ppb)
const (
	regTVOC       byte = 0x00
	regVersion    byte = 0x11
	regResistance byte = 0x20
	regCalibrate  byte = 0x01
)

// Status byte bit definitions (Data1):
// Bit0: RDY (0 = ready, 1 = not ready or pre-heat)
// Bit3..1: CI[2:0] data type (000 => TVOC in ppb after power-on)
// Bit7..4: Reserved (0)
const (
	statusBitRDY = 0x01
)

var ErrNotReady = fmt.Errorf("ags02ma: data not ready or sensor in pre-heat stage")

// AGS02MA represents Aosong AGS02MA TVOC sensor.
// Typical usage:
//   s := NewAGS02MA(bus)
//   v, err := s.GetTVOC(ctx)
// Value is returned in parts-per-billion (ppb) as integer.
// Note: The sensor requires a slow I2C clock (<= 30 kHz). Ensure adapter supports it.

type AGS02MA struct {
	transport sensors.I2CBus
	addr      byte
	buf       []byte
}

func NewAGS02MA(transport sensors.I2CBus) *AGS02MA {
	return &AGS02MA{transport: transport, addr: ags02maAddress, buf: make([]byte, 64)}
}

// GetTVOC performs a "master direct read" as described in the datasheet.
// This does NOT write the register first and simply reads 4 bytes.
// The first byte is status; the remaining three make a 24-bit big-endian ppb value.
func (s *AGS02MA) GetTVOC(ctx context.Context) (uint32, error) {
	if err := s.transport.ReadFromAddr(ctx, s.addr, s.buf[:4]); err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	status := s.buf[0]
	if status&statusBitRDY != 0 {
		return 0, ErrNotReady
	}
	ppb := (uint32(s.buf[1]) << 16) | (uint32(s.buf[2]) << 8) | uint32(s.buf[3])
	return ppb, nil
}

// GetTVOCWithRegisterRead explicitly writes register 0x00 and then reads.
// Some hosts or sequences may prefer this form. A small wait can be added to
// allow the device to update data, but the RDY bit is authoritative.
func (s *AGS02MA) GetTVOCWithRegisterRead(ctx context.Context) (uint32, error) {
	if err := s.transport.WriteToAddr(ctx, s.addr, []byte{regTVOC}); err != nil {
		return 0, fmt.Errorf("ags02ma: write reg 0x00 failed: %w", err)
	}
	// Small guard delay; actual readiness is indicated by RDY bit.
	time.Sleep(2 * time.Second)
	if err := s.transport.ReadFromAddr(ctx, s.addr, s.buf[:32]); err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	fmt.Println("buf", hex.Dump(s.buf[:32]))
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	status := s.buf[0]
	if status&statusBitRDY != 0 {
		return 0, ErrNotReady
	}
	ppb := (uint32(s.buf[1]) << 16) | (uint32(s.buf[2]) << 8) | uint32(s.buf[3])
	return ppb, nil
}

func (s *AGS02MA) ReadVersion(ctx context.Context) (int, error) {
	if err := s.transport.WriteToAddr(ctx, s.addr, []byte{regVersion}); err != nil {
		return 0, fmt.Errorf("ags02ma: write reg 0x11 failed: %w", err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := s.transport.ReadFromAddr(ctx, s.addr, s.buf[:5]); err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	fmt.Println("buf", hex.Dump(s.buf[:5]))
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	return int(s.buf[3]), nil
}

func (s *AGS02MA) ReadResistance(ctx context.Context) (int, error) {
	if err := s.transport.WriteToAddr(ctx, s.addr, []byte{regResistance}); err != nil {
		return 0, fmt.Errorf("ags02ma: write reg 0x20 failed: %w", err)
	}
	time.Sleep(2 * time.Second)
	if err := s.transport.ReadFromAddr(ctx, s.addr, s.buf[:5]); err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	fmt.Println("buf", hex.Dump(s.buf[:5]))
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	return int(s.buf[3]), nil
}

func (s *AGS02MA) Calibrate(ctx context.Context) error {
	if err := s.transport.WriteToAddr(ctx, s.addr, []byte{regCalibrate}); err != nil {
		return fmt.Errorf("ags02ma: write reg 0x01 failed: %w", err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := s.transport.ReadFromAddr(ctx, s.addr, s.buf[:5]); err != nil {
		return fmt.Errorf("ags02ma: read failed: %w", err)
	}
	fmt.Println("buf", hex.Dump(s.buf[:5]))
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	return nil
}

// checkCRC calculates CRC8 checksum with initial value 0xFF and polynomial 0x31.
// This implements the algorithm from AGS02MA datasheet (x8 + x5 + x4 + 1).
func checkCRC(data []byte) byte {
	crc := byte(0xFF)
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x31
			} else {
				crc = crc << 1
			}
		}
	}
	return crc
}
