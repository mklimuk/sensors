package accel

import (
	"context"
	"fmt"

	"github.com/mklimuk/sensors"
)

const (
	regRange         = 0x22
	regLatch         = 0x1C
	regSlopeSettings = 0x12
	regSlopeDet      = 0x1A
	regWatchdog      = 0x2E
	regInterrupts    = 0x18
)

const addr = 0x0A

// BMA220 represents Bosh BMA220 accelerometer
type BMA220 struct {
	transport sensors.I2CBus
}

func NewBMA220(trans sensors.I2CBus) *BMA220 {
	return &BMA220{transport: trans}
}

/*
en_slope_x (0x1A.5) enable slope detection on x-axis
en_slope_y (0x1A.4) enable slope detection on y-axis
en_slope_z (0x1A.3) enable slope detection on z-axis
slope_th (0x12[5:2]) define the threshold level of the slope 1 LSB threshold is 1 LSB of acc_data
slope_dur (0x12[1:0]) define the number of consecutive slope data points above slope_th which are required to set the interrupt (“00” = 1,”01” = 2,”10” = 3, “11” = 4)
slope_filt (0x12.6) defines whether filtered or unfiltered acceleration data should be used (evaluated) (‘0’=unfiltered, ‘1’=filtered)
slope_int (0x0C.0) whetherslopeinterrupthasbeentriggered
slope_first_x whether x-axis has triggered the interrupt (0=no, 1=yes)
slope_first_y whether y-axis has triggered the interrupt (0=no, 1=yes)
slope_first_z whether z-axis has triggered the interrupt (0=no, 1=yes)
slope_sign global register bit for all interrupts define the slope sign of the triggering signal (0=positive slope, 1=negative slope)
*/
func (b *BMA220) InitMotionDetection(ctx context.Context) error {
	// set sensitivity
	err := b.transport.WriteToAddr(ctx, addr, []byte{regRange, 0x03})
	if err != nil {
		return fmt.Errorf("could not set detection sensitivity: %w", err)
	}
	// set permanent interrupt latch lat_int[2:0] = 111
	err = b.transport.WriteToAddr(ctx, addr, []byte{regLatch, 0b01110000})
	if err != nil {
		return fmt.Errorf("could not set interrupt settings: %w", err)
	}
	// enable slope detection
	err = b.transport.WriteToAddr(ctx, addr, []byte{regSlopeDet, 0b00111000})
	if err != nil {
		return fmt.Errorf("could not enable slope detection: %w", err)
	}
	// set slope detection parameters (default 0x45)
	err = b.transport.WriteToAddr(ctx, addr, []byte{regSlopeSettings, 0x45})
	if err != nil {
		return fmt.Errorf("could not set stope detection settings: %w", err)
	}
	// enable watchdog
	err = b.transport.WriteToAddr(ctx, addr, []byte{regWatchdog, 0x06})
	if err != nil {
		return fmt.Errorf("could not set watchdog settings: %w", err)
	}
	return nil
}

func (b *BMA220) CheckMotionInterrupt(ctx context.Context) (int, error) {
	err := b.transport.WriteToAddr(ctx, addr, []byte{regInterrupts})
	if err != nil {
		return 0, fmt.Errorf("could not set registry pointer: %w", err)
	}
	buf := []byte{0x00}
	err = b.transport.ReadFromAddr(ctx, addr, buf)
	if err != nil {
		return 0, fmt.Errorf("could not read registry content: %w", err)
	}
	// slope detection is on bit 0
	return int(buf[0] & 0x01), nil
}

func (b *BMA220) ResetMotionInterrupt(ctx context.Context) error {
	err := b.transport.WriteToAddr(ctx, addr, []byte{regLatch, 0b11110000})
	if err != nil {
		return fmt.Errorf("could not set interrupt settings: %w", err)
	}
	return nil
}
