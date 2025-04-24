package environment

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/mklimuk/sensors"
)

const BH1750AddrHigh = 0b1011100
const BH1750AddrLow = 0b0100011

const (
	opCodeSingleLowResolution = 0b00100011
)

type BH1750 struct {
	transport sensors.I2CBus
	addr      byte
	buf       []byte
}

func NewBH1750(transport sensors.I2CBus, addr byte) *BH1750 {
	return &BH1750{
		addr:      addr,
		transport: transport,
		buf:       make([]byte, 2),
	}
}

func (sensor *BH1750) GetLux(ctx context.Context) (int, error) {
	err := sensor.transport.WriteToAddr(ctx, sensor.addr, []byte{opCodeSingleLowResolution})
	if err != nil {
		return 0, fmt.Errorf("could not write command: %w", err)
	}
	// measurement cycle takes typically 16ms, max time is 24ms, we will wait for 25ms
	time.Sleep(25 * time.Millisecond)
	err = sensor.transport.ReadFromAddr(ctx, sensor.addr, sensor.buf)
	if err != nil {
		return 0, fmt.Errorf("could not read data: %w", err)
	}
	res := float32(binary.BigEndian.Uint16(sensor.buf)) / 1.2
	return int(res), nil
}
