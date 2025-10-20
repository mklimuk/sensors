package environment

import (
	"context"
)

// TemperatureBehaviorFunc defines the function signature for temperature behavior.
// It returns the temperature in Celsius or an error.
type TemperatureBehaviorFunc func(ctx context.Context) (float32, error)

// HumidityBehaviorFunc defines the function signature for humidity behavior.
// It returns the relative humidity in %RH or an error.
type HumidityBehaviorFunc func(ctx context.Context) (float32, error)

// MockTemperatureAndHumiditySensor is a mock implementation of a temperature and humidity sensor
// that uses behavior functions to produce results without requiring any hardware.
// This can be used to mock any temperature/humidity sensor like SHTC3, HIH6021, etc.
type MockTemperatureAndHumiditySensor struct {
	tempBehavior TemperatureBehaviorFunc
	humBehavior  HumidityBehaviorFunc
}

// NewMockTemperatureAndHumiditySensor creates a new mock temperature/humidity sensor with the given behavior functions.
// The temperature behavior is called by GetTemperature() and GetTempAndHum().
// The humidity behavior is called by GetHumidity() and GetTempAndHum().
//
// Example usage:
//
//	// Simple static values
//	sensor := NewMockTemperatureAndHumiditySensor(
//		func(ctx context.Context) (float32, error) { return 22.5, nil },
//		func(ctx context.Context) (float32, error) { return 45.0, nil },
//	)
//
//	// Dynamic behavior
//	temp := float32(20.0)
//	sensor := NewMockTemperatureAndHumiditySensor(
//		func(ctx context.Context) (float32, error) { return temp, nil },
//		func(ctx context.Context) (float32, error) { return 50.0, nil },
//	)
func NewMockTemperatureAndHumiditySensor(tempBehavior TemperatureBehaviorFunc, humBehavior HumidityBehaviorFunc) *MockTemperatureAndHumiditySensor {
	return &MockTemperatureAndHumiditySensor{
		tempBehavior: tempBehavior,
		humBehavior:  humBehavior,
	}
}

// GetTemperature returns the temperature by calling the temperature behavior function.
func (m *MockTemperatureAndHumiditySensor) GetTemperature(ctx context.Context) (float32, error) {
	return m.tempBehavior(ctx)
}

// GetHumidity returns the humidity by calling the humidity behavior function.
func (m *MockTemperatureAndHumiditySensor) GetHumidity(ctx context.Context) (float32, error) {
	return m.humBehavior(ctx)
}

// GetTempAndHum returns both temperature and humidity by calling both behavior functions.
func (m *MockTemperatureAndHumiditySensor) GetTempAndHum(ctx context.Context) (float32, float32, error) {
	temp, err := m.tempBehavior(ctx)
	if err != nil {
		return 0, 0, err
	}
	hum, err := m.humBehavior(ctx)
	if err != nil {
		return 0, 0, err
	}
	return temp, hum, nil
}

// Backward compatibility aliases for specific sensors
// NewMockSHTC3 creates a new mock SHTC3 sensor (alias for NewMockTemperatureAndHumiditySensor).
func NewMockSHTC3(tempBehavior TemperatureBehaviorFunc, humBehavior HumidityBehaviorFunc) *MockTemperatureAndHumiditySensor {
	return NewMockTemperatureAndHumiditySensor(tempBehavior, humBehavior)
}

// NewMockHIH6021 creates a new mock HIH6021 sensor (alias for NewMockTemperatureAndHumiditySensor).
func NewMockHIH6021(tempBehavior TemperatureBehaviorFunc, humBehavior HumidityBehaviorFunc) *MockTemperatureAndHumiditySensor {
	return NewMockTemperatureAndHumiditySensor(tempBehavior, humBehavior)
}
