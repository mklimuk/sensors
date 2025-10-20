package environment

import (
	"context"
)

// LightBehaviorFunc defines the function signature for light sensor behavior.
// It returns the lux value or an error.
type LightBehaviorFunc func(ctx context.Context) (int, error)

// MockLightSensor is a mock implementation of a light sensor that uses a behavior function
// to produce results without requiring any hardware.
// This can be used to mock any light sensor like BH1750.
type MockLightSensor struct {
	behavior LightBehaviorFunc
}

// NewMockLightSensor creates a new mock light sensor with the given behavior function.
// The behavior function is called whenever GetLux is invoked.
//
// Example usage:
//
//	// Static value
//	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
//		return 500, nil
//	})
//
//	// Dynamic behavior
//	counter := 0
//	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
//		counter++
//		return counter * 100, nil
//	})
//
//	// Error simulation
//	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
//		return 0, fmt.Errorf("sensor malfunction")
//	})
func NewMockLightSensor(behavior LightBehaviorFunc) *MockLightSensor {
	return &MockLightSensor{
		behavior: behavior,
	}
}

// GetLux returns the lux value by calling the behavior function.
func (m *MockLightSensor) GetLux(ctx context.Context) (int, error) {
	return m.behavior(ctx)
}

// NewMockBH1750 creates a new mock BH1750 sensor (alias for NewMockLightSensor for backward compatibility).
func NewMockBH1750(behavior LightBehaviorFunc) *MockLightSensor {
	return NewMockLightSensor(behavior)
}
