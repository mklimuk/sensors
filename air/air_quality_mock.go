package air

import (
	"context"
)

// TVOCBehaviorFunc defines the function signature for TVOC behavior.
// It returns TVOC in parts-per-billion (ppb) or an error.
type TVOCBehaviorFunc func(ctx context.Context) (uint32, error)

// MockAirQualitySensor is a mock implementation of an air quality (TVOC) sensor
// that uses a behavior function to produce results without requiring hardware.
// This can be used to mock sensors like AGS02MA.
type MockAirQualitySensor struct {
	behavior TVOCBehaviorFunc
}

// NewMockAirQualitySensor creates a new mock air quality sensor with the given behavior function.
// The behavior function is called whenever GetTVOC is invoked.
//
// Example usage:
//
//	sensor := NewMockAirQualitySensor(func(ctx context.Context) (uint32, error) { return 750, nil })
func NewMockAirQualitySensor(behavior TVOCBehaviorFunc) *MockAirQualitySensor {
	return &MockAirQualitySensor{behavior: behavior}
}

// GetTVOC returns the TVOC reading (in ppb) by calling the behavior function.
func (m *MockAirQualitySensor) GetTVOC(ctx context.Context) (uint32, error) {
	return m.behavior(ctx)
}

// NewMockAGS02MA creates a new mock AGS02MA sensor (alias for NewMockAirQualitySensor for backward compatibility).
func NewMockAGS02MA(behavior TVOCBehaviorFunc) *MockAirQualitySensor {
	return NewMockAirQualitySensor(behavior)
}
