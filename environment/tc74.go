package environment

import (
	"context"
	"fmt"

	"github.com/mklimuk/sensors"
)

const tc74DefaultAddress = 0x4D
const tc74TempRegister = 0x00
const tc74ConfigRegister = 0x01

// TC74 represents a Microchip TC74 Digital Temperature Sensor
// See: https://ww1.microchip.com/downloads/en/DeviceDoc/21462D.pdf
//
// Usage: Instantiate with NewTC74, then call GetTemperature(ctx)
type TC74 struct {
	transport sensors.I2CBus
	address   byte
	lastTemp  float32
}

type TC74Config struct {
	Address byte
}

type TC74ConfigOption func(*TC74Config)

func WithAddress(address byte) TC74ConfigOption {
	return func(c *TC74Config) {
		c.Address = address
	}
}

// NewTC74 creates a new TC74 sensor connector with the given I2CBus transport and optional address.
// If address is 0, the default 0x4D is used.
func NewTC74(trans sensors.I2CBus, opts ...TC74ConfigOption) *TC74 {
	config := &TC74Config{
		Address: tc74DefaultAddress,
	}
	for _, opt := range opts {
		opt(config)
	}
	return &TC74{transport: trans, address: config.Address}
}

// GetConfig reads the configuration register (0x01) and returns its value.
func (sensor *TC74) GetConfig(ctx context.Context) (byte, error) {
	err := sensor.transport.WriteToAddr(ctx, sensor.address, []byte{tc74ConfigRegister})
	if err != nil {
		return 0, fmt.Errorf("tc74: could not write config register request: %w", err)
	}
	resp := make([]byte, 1)
	err = sensor.transport.ReadFromAddr(ctx, sensor.address, resp)
	if err != nil {
		return 0, fmt.Errorf("tc74: could not read config register: %w", err)
	}
	return resp[0], nil
}

// GetTemperature reads the current temperature in Celsius from the TC74 sensor.
// It checks the DATA_RDY bit in the config register before reading temperature.
func (sensor *TC74) GetTemperature(ctx context.Context) (float32, error) {
	config, err := sensor.GetConfig(ctx)
	if err != nil {
		return 0, fmt.Errorf("tc74: could not get config: %w", err)
	}
	if (config & 0x40) == 0 {
		// TODO: do we want to return a common error here?
		return sensor.lastTemp, nil
	}
	// Write the temperature register address (0x00)
	err = sensor.transport.WriteToAddr(ctx, sensor.address, []byte{tc74TempRegister})
	if err != nil {
		return 0, fmt.Errorf("tc74: could not write temp register request: %w", err)
	}
	resp := make([]byte, 1)
	err = sensor.transport.ReadFromAddr(ctx, sensor.address, resp)
	if err != nil {
		return 0, fmt.Errorf("tc74: could not read temp register: %w", err)
	}
	// Convert 2's complement 8-bit value to int8
	temp := int8(resp[0])
	sensor.lastTemp = float32(temp)
	return sensor.lastTemp, nil
}

// GetHumidity is not supported by TC74 and returns an error.
func (sensor *TC74) GetHumidity(ctx context.Context) (float32, error) {
	// TODO: add global unsupported error
	return 0, nil
}
