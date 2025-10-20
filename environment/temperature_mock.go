package environment

import (
	"context"
)

// MockTemperatureSensor is a mock implementation of a temperature-only sensor that uses a behavior function
// to produce results without requiring any hardware.
// This can be used to mock any temperature sensor like TC74.
type MockTemperatureSensor struct {
	behavior TemperatureBehaviorFunc
}

// NewMockTemperatureSensor creates a new mock temperature sensor with the given behavior function.
// The behavior function is called whenever GetTemperature is invoked.
//
// Example usage:
//
//	sensor := NewMockTemperatureSensor(func(ctx context.Context) (float32, error) { return 25.0, nil })
func NewMockTemperatureSensor(behavior TemperatureBehaviorFunc) *MockTemperatureSensor {
	return &MockTemperatureSensor{behavior: behavior}
}

// GetTemperature returns the temperature by calling the behavior function.
func (m *MockTemperatureSensor) GetTemperature(ctx context.Context) (float32, error) {
	return m.behavior(ctx)
}

// GetHumidity is not supported by temperature-only sensors; returns 0, nil.
func (m *MockTemperatureSensor) GetHumidity(ctx context.Context) (float32, error) {
	return 0, nil
}

// GetTempAndHum returns temperature from the behavior and 0 for humidity.
// If temperature behavior returns an error, it is propagated.
func (m *MockTemperatureSensor) GetTempAndHum(ctx context.Context) (float32, float32, error) {
	t, err := m.behavior(ctx)
	if err != nil {
		return 0, 0, err
	}
	return t, 0, nil
}

// NewMockTC74 creates a new mock TC74 sensor (alias for NewMockTemperatureSensor for backward compatibility).
func NewMockTC74(behavior TemperatureBehaviorFunc) *MockTemperatureSensor {
	return NewMockTemperatureSensor(behavior)
}
