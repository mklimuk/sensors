package air

import (
	"context"
	"fmt"
	"sync"
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

const (
	TVOCModeDirectRead    byte = 0x00
	TVOCModeRegisterWrite byte = 0x01
)

type AGS02MAOpts struct {
	ConfigureDelay time.Duration
	ReadDelay      time.Duration
	TxDelay        time.Duration
	TVOCMode       byte
}

type AGS02MAOpt func(*AGS02MAOpts)

func WithConfigureDelay(delay time.Duration) AGS02MAOpt {
	return func(o *AGS02MAOpts) {
		o.ConfigureDelay = delay
	}
}

func WithReadDelay(delay time.Duration) AGS02MAOpt {
	return func(o *AGS02MAOpts) {
		o.ReadDelay = delay
	}
}

func WithTxDelay(delay time.Duration) AGS02MAOpt {
	return func(o *AGS02MAOpts) {
		o.TxDelay = delay
	}
}

func WithTVOCMode(mode byte) AGS02MAOpt {
	return func(o *AGS02MAOpts) {
		o.TVOCMode = mode
	}
}

// AGS02MA represents Aosong AGS02MA TVOC sensor.
// Typical usage:
//
//	s := NewAGS02MA(bus)
//	v, err := s.GetTVOC(ctx)
//
// Value is returned in parts-per-billion (ppb) as integer.
// Note: The sensor requires a slow I2C clock (<= 30 kHz). Ensure adapter supports it.
type AGS02MA struct {
	mx        sync.Mutex
	delayDone chan struct{} // closed when delay after last operation completes
	delayMx   sync.Mutex    // protects delayDone channel

	config AGS02MAOpts

	transport sensors.I2CBus
	addr      byte
	buf       []byte
}

func NewAGS02MA(transport sensors.I2CBus, opts ...AGS02MAOpt) *AGS02MA {
	config := AGS02MAOpts{
		ConfigureDelay: 2 * time.Second,
		ReadDelay:      1500 * time.Millisecond,
		TxDelay:        100 * time.Millisecond,
		TVOCMode:       TVOCModeRegisterWrite,
	}
	for _, opt := range opts {
		opt(&config)
	}
	// Create a closed channel so first operation can proceed immediately
	ch := make(chan struct{})
	close(ch)
	return &AGS02MA{
		config:    config,
		transport: transport,
		addr:      ags02maAddress,
		buf:       make([]byte, 5),
		delayDone: ch, // initially ready (closed channel)
	}
}

// waitForDelay waits for any pending delay from previous operations to complete.
func (s *AGS02MA) waitForDelay(ctx context.Context) error {
	s.delayMx.Lock()
	ch := s.delayDone
	s.delayMx.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// scheduleDelay schedules a delay in a goroutine and updates delayDone channel when complete.
func (s *AGS02MA) scheduleDelay(ctx context.Context, duration time.Duration) {
	s.delayMx.Lock()
	// Create new channel for this delay
	ch := make(chan struct{})
	s.delayDone = ch
	s.delayMx.Unlock()

	go func() {
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-timer.C:
			close(ch)
		case <-ctx.Done():
			close(ch)
		}
	}()
}

func (s *AGS02MA) Close(ctx context.Context) {
	// Wait for any pending delay to complete
	_ = s.waitForDelay(ctx)
}

func (s *AGS02MA) Configure(ctx context.Context) error {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return err
	}

	s.mx.Lock()
	err := s.transport.WriteToAddr(ctx, s.addr, []byte{regTVOC, 0x00, 0xFF, 0x00, 0xFF, 0x30})
	s.mx.Unlock()

	if err != nil {
		return fmt.Errorf("ags02ma: configuration write failed: %w", err)
	}
	// Recommended 2 second delay after configuration (runs asynchronously)
	s.scheduleDelay(ctx, s.config.ConfigureDelay)
	return nil
}

// GetTVOC performs a "master direct read" or "register write" as described in the datasheet.
// The mode is determined by the TVOCMode configuration option.
func (s *AGS02MA) GetTVOC(ctx context.Context) (uint32, error) {
	if s.config.TVOCMode == TVOCModeDirectRead {
		return s.GetTVOCDirectRead(ctx)
	}
	return s.GetTVOCWithRegisterWrite(ctx)
}

// GetTVOCDirectRead performs a "master direct read" as described in the datasheet.
// This does NOT write the register first and simply reads 4 bytes.
// The first byte is status; the remaining three make a 24-bit big-endian ppb value.
func (s *AGS02MA) GetTVOCDirectRead(ctx context.Context) (uint32, error) {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return 0, err
	}

	s.mx.Lock()
	err := s.transport.ReadFromAddr(ctx, s.addr, s.buf)
	s.mx.Unlock()

	if err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	status := s.buf[0]
	if status&statusBitRDY != 0 {
		return 0, ErrNotReady
	}
	ppb := (uint32(s.buf[1]) << 16) | (uint32(s.buf[2]) << 8) | uint32(s.buf[3])
	// Recommended 1.5 second delay after TVOC read (runs asynchronously)
	s.scheduleDelay(ctx, s.config.ReadDelay)
	return ppb, nil
}

// GetTVOCWithRegisterWrite explicitly writes register 0x00 and then reads.
// Some hosts or sequences may prefer this form. A small wait can be added to
// allow the device to update data, but the RDY bit is authoritative.
func (s *AGS02MA) GetTVOCWithRegisterWrite(ctx context.Context) (uint32, error) {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return 0, err
	}

	s.mx.Lock()
	err := s.transport.WriteToAddr(ctx, s.addr, []byte{regTVOC})
	s.mx.Unlock()

	if err != nil {
		return 0, fmt.Errorf("ags02ma: write reg 0x00 failed: %w", err)
	}

	// Small guard delay; actual readiness is indicated by RDY bit.
	// This is part of the operation sequence, so we wait synchronously.
	timer := time.NewTimer(s.config.TxDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	s.mx.Lock()
	err = s.transport.ReadFromAddr(ctx, s.addr, s.buf)
	s.mx.Unlock()

	if err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	status := s.buf[0]
	if status&statusBitRDY != 0 {
		return 0, ErrNotReady
	}
	ppb := (uint32(s.buf[1]) << 16) | (uint32(s.buf[2]) << 8) | uint32(s.buf[3])
	// Recommended 1.5 second delay after TVOC read (runs asynchronously)
	s.scheduleDelay(ctx, s.config.ReadDelay)
	return ppb, nil
}

func (s *AGS02MA) ReadVersion(ctx context.Context) (int, error) {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return 0, err
	}

	s.mx.Lock()
	err := s.transport.WriteToAddr(ctx, s.addr, []byte{regVersion})
	if err != nil {
		s.mx.Unlock()
		return 0, fmt.Errorf("ags02ma: write reg 0x11 failed: %w", err)
	}
	s.mx.Unlock()

	// Small guard delay; part of the operation sequence, so we wait synchronously.
	timer := time.NewTimer(s.config.TxDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	s.mx.Lock()
	err = s.transport.ReadFromAddr(ctx, s.addr, s.buf)
	s.mx.Unlock()

	if err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	return int(s.buf[3]), nil
}

func (s *AGS02MA) ReadResistance(ctx context.Context) (int, error) {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return 0, err
	}

	s.mx.Lock()
	err := s.transport.WriteToAddr(ctx, s.addr, []byte{regResistance})
	if err != nil {
		s.mx.Unlock()
		return 0, fmt.Errorf("ags02ma: write reg 0x20 failed: %w", err)
	}
	s.mx.Unlock()

	// Small guard delay; part of the operation sequence, so we wait synchronously.
	timer := time.NewTimer(s.config.TxDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	s.mx.Lock()
	err = s.transport.ReadFromAddr(ctx, s.addr, s.buf)
	s.mx.Unlock()

	if err != nil {
		return 0, fmt.Errorf("ags02ma: read failed: %w", err)
	}
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return 0, fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	// Recommended 1.5 second delay after resistance read (runs asynchronously)
	s.scheduleDelay(ctx, s.config.ReadDelay)
	return int(s.buf[3]), nil
}

func (s *AGS02MA) Calibrate(ctx context.Context) error {
	// Wait for any pending delay from previous operations
	if err := s.waitForDelay(ctx); err != nil {
		return err
	}

	s.mx.Lock()
	err := s.transport.WriteToAddr(ctx, s.addr, []byte{regCalibrate})
	if err != nil {
		s.mx.Unlock()
		return fmt.Errorf("ags02ma: write reg 0x01 failed: %w", err)
	}
	s.mx.Unlock()

	// Small guard delay; part of the operation sequence, so we wait synchronously.
	timer := time.NewTimer(s.config.TxDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return ctx.Err()
	}

	s.mx.Lock()
	err = s.transport.ReadFromAddr(ctx, s.addr, s.buf)
	s.mx.Unlock()

	if err != nil {
		return fmt.Errorf("ags02ma: read failed: %w", err)
	}
	crc := checkCRC(s.buf[:4])
	if crc != s.buf[4] {
		return fmt.Errorf("ags02ma: crc mismatch: expected %#x, got %#x", s.buf[4], crc)
	}
	// Recommended 1.5 second delay after calibrate (runs asynchronously)
	s.scheduleDelay(ctx, s.config.ReadDelay)
	return nil
}

// checkCRC calculates CRC8 checksum with initial value 0xFF and polynomial 0x31.
// This implements the algorithm from AGS02MA datasheet (x8 + x5 + x4 + 1).
func checkCRC(data []byte) byte {
	crc := byte(0xFF)
	for _, b := range data {
		crc ^= b
		for range 8 {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x31
			} else {
				crc = crc << 1
			}
		}
	}
	return crc
}
