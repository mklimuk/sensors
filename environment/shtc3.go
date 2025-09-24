package environment

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/mklimuk/sensors"
)

// SHTC3 I2C address (7-bit)
const shtc3Address = 0x70

// Commands (Big Endian on the wire)
const (
	shtc3CmdWake  uint16 = 0x3517
	shtc3CmdSleep uint16 = 0xB098

	// Normal power, clock stretching disabled
	// Measure T first, then RH
	shtc3CmdMeasureTFirstNoCS uint16 = 0x7866
)

// SHTC3 represents Sensirion SHTC3 Temperature/Humidity sensor
// Typical usage:
//
//	s := NewSHTC3(bus)
//	t, h, err := s.GetTempAndHum(ctx)
type SHTC3 struct {
	transport sensors.I2CBus
	lastTemp  float32
	lastHum   float32
}

func NewSHTC3(trans sensors.I2CBus) *SHTC3 {
	return &SHTC3{transport: trans}
}

// GetTemperature performs a single measurement and returns temperature in Celsius.
func (s *SHTC3) GetTemperature(ctx context.Context) (float32, error) {
	if err := s.measure(ctx); err != nil {
		return 0, err
	}
	return s.lastTemp, nil
}

// GetHumidity performs a single measurement and returns relative humidity in %RH.
func (s *SHTC3) GetHumidity(ctx context.Context) (float32, error) {
	if err := s.measure(ctx); err != nil {
		return 0, err
	}
	return s.lastHum, nil
}

// GetTempAndHum performs a single measurement and returns temperature and humidity.
func (s *SHTC3) GetTempAndHum(ctx context.Context) (float32, float32, error) {
	if err := s.measure(ctx); err != nil {
		return 0, 0, err
	}
	return s.lastTemp, s.lastHum, nil
}

func (s *SHTC3) measure(ctx context.Context) error {
	// Wake up from sleep
	if err := s.writeCmd(ctx, shtc3CmdWake); err != nil {
		return fmt.Errorf("shtc3: wake failed: %w", err)
	}
	// Typical wake time is very short (< 240us), small delay to be safe
	time.Sleep(1 * time.Millisecond)

	// Trigger measurement (normal power, no clock stretching, T first)
	if err := s.writeCmd(ctx, shtc3CmdMeasureTFirstNoCS); err != nil {
		return fmt.Errorf("shtc3: measure command failed: %w", err)
	}
	// Typical measurement time ~12.1 ms (normal mode). Wait conservatively.
	time.Sleep(15 * time.Millisecond)

	// Read 6 bytes: T[0:2], CRC, RH[3:5]
	buf := make([]byte, 6)
	if err := s.transport.ReadFromAddr(ctx, shtc3Address, buf); err != nil {
		return fmt.Errorf("shtc3: read failed: %w", err)
	}

	// Verify CRC for temperature and humidity words
	if !shtCRC8Check(buf[0:2], buf[2]) {
		return fmt.Errorf("shtc3: temperature CRC mismatch")
	}
	if !shtCRC8Check(buf[3:5], buf[5]) {
		return fmt.Errorf("shtc3: humidity CRC mismatch")
	}

	rawT := binary.BigEndian.Uint16(buf[0:2])
	rawRH := binary.BigEndian.Uint16(buf[3:5])

	// Conversion formulas from datasheet
	// T(C) = -45 + 175 * rawT / 65535
	// RH(%) = 100 * rawRH / 65535
	s.lastTemp = -45.0 + (175.0 * float32(rawT) / 65535.0)
	s.lastHum = 100.0 * float32(rawRH) / 65535.0

	// Go back to sleep to save power
	if err := s.writeCmd(ctx, shtc3CmdSleep); err != nil {
		// Not fatal for reading, but report so caller knows
		return fmt.Errorf("shtc3: sleep failed: %w", err)
	}
	return nil
}

func (s *SHTC3) writeCmd(ctx context.Context, cmd uint16) error {
	var out [2]byte
	binary.BigEndian.PutUint16(out[:], cmd)
	return s.transport.WriteToAddr(ctx, shtc3Address, out[:])
}

// Sensirion CRC-8, polynomial 0x31, init 0xFF
func shtCRC8(data []byte) byte {
	var crc byte = 0xFF
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if (crc & 0x80) != 0 {
				crc = (crc << 1) ^ 0x31
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func shtCRC8Check(data []byte, expected byte) bool {
	return shtCRC8(data) == expected
}
