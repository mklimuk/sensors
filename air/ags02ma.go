package air

import (
	"context"
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
	regTVOC byte = 0x00
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
	return &AGS02MA{transport: transport, addr: ags02maAddress, buf: make([]byte, 4)}
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
	time.Sleep(5 * time.Millisecond)
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
