package environment

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
)

func TestMockTemperatureAndHumiditySensor_StaticValues(t *testing.T) {
	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) { return 22.5, nil },
		func(ctx context.Context) (float32, error) { return 45.0, nil },
	)

	ctx := context.Background()

	temp, err := sensor.GetTemperature(ctx)
	if err != nil {
		t.Fatalf("GetTemperature: unexpected error: %v", err)
	}
	if temp != 22.5 {
		t.Errorf("expected temperature 22.5, got %f", temp)
	}

	hum, err := sensor.GetHumidity(ctx)
	if err != nil {
		t.Fatalf("GetHumidity: unexpected error: %v", err)
	}
	if hum != 45.0 {
		t.Errorf("expected humidity 45.0, got %f", hum)
	}

	temp2, hum2, err := sensor.GetTempAndHum(ctx)
	if err != nil {
		t.Fatalf("GetTempAndHum: unexpected error: %v", err)
	}
	if temp2 != 22.5 {
		t.Errorf("expected temperature 22.5, got %f", temp2)
	}
	if hum2 != 45.0 {
		t.Errorf("expected humidity 45.0, got %f", hum2)
	}
}

func TestMockTemperatureAndHumiditySensor_DynamicBehavior(t *testing.T) {
	currentTemp := float32(20.0)
	currentHum := float32(50.0)

	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) { return currentTemp, nil },
		func(ctx context.Context) (float32, error) { return currentHum, nil },
	)

	ctx := context.Background()

	// Initial reading
	temp, hum, err := sensor.GetTempAndHum(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if temp != 20.0 || hum != 50.0 {
		t.Errorf("expected 20.0/50.0, got %f/%f", temp, hum)
	}

	// Change values
	currentTemp = 25.0
	currentHum = 60.0

	temp, hum, err = sensor.GetTempAndHum(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if temp != 25.0 || hum != 60.0 {
		t.Errorf("expected 25.0/60.0, got %f/%f", temp, hum)
	}
}

func TestMockTemperatureAndHumiditySensor_IndependentBehaviors(t *testing.T) {
	tempCalls := 0
	humCalls := 0

	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) {
			tempCalls++
			return 20.0, nil
		},
		func(ctx context.Context) (float32, error) {
			humCalls++
			return 50.0, nil
		},
	)

	ctx := context.Background()

	_, _ = sensor.GetTemperature(ctx)
	if tempCalls != 1 || humCalls != 0 {
		t.Errorf("GetTemperature: unexpected call counts: temp=%d, hum=%d", tempCalls, humCalls)
	}

	_, _ = sensor.GetHumidity(ctx)
	if tempCalls != 1 || humCalls != 1 {
		t.Errorf("GetHumidity: unexpected call counts: temp=%d, hum=%d", tempCalls, humCalls)
	}

	_, _, _ = sensor.GetTempAndHum(ctx)
	if tempCalls != 2 || humCalls != 2 {
		t.Errorf("GetTempAndHum: unexpected call counts: temp=%d, hum=%d (expected 2, 2)", tempCalls, humCalls)
	}
}

func TestMockTemperatureAndHumiditySensor_ErrorHandling(t *testing.T) {
	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) {
			return 0, fmt.Errorf("temperature sensor error")
		},
		func(ctx context.Context) (float32, error) {
			return 0, fmt.Errorf("humidity sensor error")
		},
	)

	ctx := context.Background()

	_, err := sensor.GetTemperature(ctx)
	if err == nil || err.Error() != "temperature sensor error" {
		t.Errorf("GetTemperature: expected specific error, got %v", err)
	}

	_, err = sensor.GetHumidity(ctx)
	if err == nil || err.Error() != "humidity sensor error" {
		t.Errorf("GetHumidity: expected specific error, got %v", err)
	}

	_, _, err = sensor.GetTempAndHum(ctx)
	if err == nil || err.Error() != "temperature sensor error" {
		t.Errorf("GetTempAndHum: expected temperature sensor error, got %v", err)
	}
}

func TestMockTemperatureAndHumiditySensor_ContextUsage(t *testing.T) {
	var receivedTempCtx context.Context
	var receivedHumCtx context.Context

	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) {
			receivedTempCtx = ctx
			return 20.0, nil
		},
		func(ctx context.Context) (float32, error) {
			receivedHumCtx = ctx
			return 50.0, nil
		},
	)

	type contextKey string
	key := contextKey("test")
	ctx := context.WithValue(context.Background(), key, "test-value")

	_, _, err := sensor.GetTempAndHum(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedTempCtx.Value(key) != "test-value" {
		t.Error("context was not passed through to temperature behavior")
	}
	if receivedHumCtx.Value(key) != "test-value" {
		t.Error("context was not passed through to humidity behavior")
	}
}

func TestMockTemperatureAndHumiditySensor_EnvironmentalSimulation(t *testing.T) {
	// Simulate indoor temperature/humidity changes throughout the day
	hourOfDay := 0

	getTempForHour := func() float32 {
		switch {
		case hourOfDay >= 6 && hourOfDay < 12:
			return 18.0
		case hourOfDay >= 12 && hourOfDay < 18:
			return 24.0
		case hourOfDay >= 18 && hourOfDay < 22:
			return 22.0
		default:
			return 16.0
		}
	}

	getHumForHour := func() float32 {
		switch {
		case hourOfDay >= 6 && hourOfDay < 12:
			return 55.0
		case hourOfDay >= 12 && hourOfDay < 18:
			return 45.0
		case hourOfDay >= 18 && hourOfDay < 22:
			return 50.0
		default:
			return 60.0
		}
	}

	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) { return getTempForHour(), nil },
		func(ctx context.Context) (float32, error) { return getHumForHour(), nil },
	)

	ctx := context.Background()

	testCases := []struct {
		hour         int
		expectedTemp float32
		expectedHum  float32
	}{
		{0, 16.0, 60.0},  // Night
		{8, 18.0, 55.0},  // Morning
		{14, 24.0, 45.0}, // Afternoon
		{20, 22.0, 50.0}, // Evening
		{23, 16.0, 60.0}, // Night
	}

	for _, tc := range testCases {
		hourOfDay = tc.hour
		temp, hum, err := sensor.GetTempAndHum(ctx)
		if err != nil {
			t.Fatalf("hour %d: unexpected error: %v", tc.hour, err)
		}
		if temp != tc.expectedTemp || hum != tc.expectedHum {
			t.Errorf("hour %d: expected %f°C/%f%%RH, got %f°C/%f%%RH",
				tc.hour, tc.expectedTemp, tc.expectedHum, temp, hum)
		}
	}
}

func TestMockTemperatureAndHumiditySensor_RandomValues(t *testing.T) {
	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) {
			// Random temperature between 15-30°C
			return 15.0 + rand.Float32()*15.0, nil
		},
		func(ctx context.Context) (float32, error) {
			// Random humidity between 30-70%
			return 30.0 + rand.Float32()*40.0, nil
		},
	)

	ctx := context.Background()

	// Test multiple readings
	for i := 0; i < 10; i++ {
		temp, hum, err := sensor.GetTempAndHum(ctx)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if temp < 15.0 || temp > 30.0 {
			t.Errorf("iteration %d: temperature %f out of range [15, 30]", i, temp)
		}
		if hum < 30.0 || hum > 70.0 {
			t.Errorf("iteration %d: humidity %f out of range [30, 70]", i, hum)
		}
	}
}

func TestMockTemperatureAndHumiditySensor_CounterBehavior(t *testing.T) {
	counter := 0

	sensor := NewMockTemperatureAndHumiditySensor(
		func(ctx context.Context) (float32, error) {
			counter++
			return 20.0 + float32(counter)*0.5, nil
		},
		func(ctx context.Context) (float32, error) {
			return 50.0 - float32(counter)*1.0, nil
		},
	)

	ctx := context.Background()

	// First reading
	temp1, hum1, _ := sensor.GetTempAndHum(ctx)
	if temp1 != 20.5 || hum1 != 49.0 {
		t.Errorf("first reading: expected 20.5/49.0, got %f/%f", temp1, hum1)
	}

	// Second reading
	temp2, hum2, _ := sensor.GetTempAndHum(ctx)
	if temp2 != 21.0 || hum2 != 48.0 {
		t.Errorf("second reading: expected 21.0/48.0, got %f/%f", temp2, hum2)
	}
}
