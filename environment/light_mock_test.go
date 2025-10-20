package environment

import (
	"context"
	"fmt"
	"testing"
)

func TestMockLightSensor_StaticValue(t *testing.T) {
	// Create a mock that always returns 500 lux
	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
		return 500, nil
	})

	ctx := context.Background()
	lux, err := sensor.GetLux(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lux != 500 {
		t.Errorf("expected 500 lux, got %d", lux)
	}
}

func TestMockLightSensor_DynamicBehavior(t *testing.T) {
	callCount := 0

	// Create a mock that returns different values on each call
	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
		callCount++
		return callCount * 100, nil
	})

	ctx := context.Background()

	// First call should return 100
	lux1, err := sensor.GetLux(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lux1 != 100 {
		t.Errorf("first call: expected 100 lux, got %d", lux1)
	}

	// Second call should return 200
	lux2, err := sensor.GetLux(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lux2 != 200 {
		t.Errorf("second call: expected 200 lux, got %d", lux2)
	}
}

func TestMockLightSensor_ErrorHandling(t *testing.T) {
	// Create a mock that returns an error
	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
		return 0, fmt.Errorf("sensor malfunction")
	})

	ctx := context.Background()
	_, err := sensor.GetLux(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "sensor malfunction" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMockLightSensor_ContextUsage(t *testing.T) {
	// Verify that context is passed through
	var receivedCtx context.Context

	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
		receivedCtx = ctx
		return 1000, nil
	})

	type contextKey string
	key := contextKey("test")
	ctx := context.WithValue(context.Background(), key, "test-value")

	_, err := sensor.GetLux(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedCtx.Value(key) != "test-value" {
		t.Error("context was not passed through correctly")
	}
}

func TestMockLightSensor_EnvironmentalSimulation(t *testing.T) {
	// Simulate varying light conditions throughout a day
	hourOfDay := 0

	sensor := NewMockLightSensor(func(ctx context.Context) (int, error) {
		// Simulate light levels changing throughout the day
		switch {
		case hourOfDay >= 6 && hourOfDay < 8: // Dawn
			return 250, nil
		case hourOfDay >= 8 && hourOfDay < 18: // Daylight
			return 10000, nil
		case hourOfDay >= 18 && hourOfDay < 20: // Dusk
			return 250, nil
		default: // Night
			return 10, nil
		}
	})

	ctx := context.Background()

	testCases := []struct {
		hour     int
		expected int
	}{
		{0, 10},     // Night
		{7, 250},    // Dawn
		{12, 10000}, // Noon
		{19, 250},   // Dusk
		{23, 10},    // Night
	}

	for _, tc := range testCases {
		hourOfDay = tc.hour
		lux, err := sensor.GetLux(ctx)
		if err != nil {
			t.Fatalf("hour %d: unexpected error: %v", tc.hour, err)
		}
		if lux != tc.expected {
			t.Errorf("hour %d: expected %d lux, got %d", tc.hour, tc.expected, lux)
		}
	}
}
