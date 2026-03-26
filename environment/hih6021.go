package environment

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/mklimuk/sensors"
)

const defaultAddress = 0x27

// minReadInterval avoids re-triggering a measurement before the HIH6021 has
// latched fresh data; rapid reads return ErrStaleData ("state data" / status bit).
const minReadInterval = time.Second

var divider = float32(1<<14 - 2)

var ErrStaleData = fmt.Errorf("stale data")
var ErrCommandMode = fmt.Errorf("device in command mode")

// HIH6021 represents Honywell HumidIcon™ Digital Humidity/Temperature sensor
type HIH6021 struct {
	mu         sync.Mutex
	transport  sensors.I2CBus
	lastTemp   float32
	lastHum    float32
	lastReadAt time.Time
	hasReading bool
}

func NewHIH6021(trans sensors.I2CBus) *HIH6021 {
	return &HIH6021{transport: trans}
}

func (sensor *HIH6021) GetTemperature(ctx context.Context) (float32, error) {
	err := sensor.measure(ctx)
	return sensor.lastTemp, err
}

func (sensor *HIH6021) GetHumidity(ctx context.Context) (float32, error) {
	err := sensor.measure(ctx)
	return sensor.lastHum, err
}

func (sensor *HIH6021) GetTempAndHum(ctx context.Context) (float32, float32, error) {
	err := sensor.measure(ctx)
	return sensor.lastTemp, sensor.lastHum, err
}

func (sensor *HIH6021) measure(ctx context.Context) error {
	sensor.mu.Lock()
	defer sensor.mu.Unlock()

	if sensor.hasReading && time.Since(sensor.lastReadAt) < minReadInterval {
		return nil
	}

	err := sensor.transport.WriteToAddr(ctx, defaultAddress, []byte{})
	if err != nil {
		return fmt.Errorf("could not write measurement request to device: %w", err)
	}
	// measurement cycle takes typically 36.65ms
	time.Sleep(50 * time.Millisecond)
	resp := make([]byte, 4)
	err = sensor.transport.ReadFromAddr(ctx, defaultAddress, resp)
	if err != nil {
		return fmt.Errorf("could not write measurement request to device: %w", err)
	}
	// check the oldest bit
	if resp[0]&0x80 > 0 {
		return ErrCommandMode
	}
	// check the second oldest bit
	if resp[0]&0x40 > 0 {
		// data has already been fetched since last measurement ot data fetched before the first measurement
		// has been completed
		return ErrStaleData
	}
	sensor.lastHum = convertHumidity(resp[0:2])
	sensor.lastTemp = convertTemperature(resp[2:4])
	sensor.lastReadAt = time.Now()
	sensor.hasReading = true
	return nil
}

func convertHumidity(resp []byte) float32 {
	hum := float32(binary.BigEndian.Uint16(resp)) / divider * 100
	if hum > 100.00 {
		return 100.00
	}
	return hum
}

func convertTemperature(resp []byte) float32 {
	shift := resp[0] & 0x03
	shift <<= 6
	lsb := (resp[1] >> 2) | shift
	msb := resp[0] >> 2
	return float32(binary.BigEndian.Uint16([]byte{msb, lsb}))/divider*165 - 40
}
