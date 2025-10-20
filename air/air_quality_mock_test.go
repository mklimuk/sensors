package air

import (
	"context"
	"fmt"
	"testing"
)

func TestMockAirQualitySensor_StaticValue(t *testing.T) {
	s := NewMockAirQualitySensor(func(ctx context.Context) (uint32, error) { return 500, nil })
	ctx := context.Background()
	v, err := s.GetTVOC(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 500 {
		t.Errorf("expected 500, got %d", v)
	}
}

func TestMockAirQualitySensor_Dynamic(t *testing.T) {
	val := uint32(100)
	s := NewMockAirQualitySensor(func(ctx context.Context) (uint32, error) { return val, nil })
	ctx := context.Background()

	v1, _ := s.GetTVOC(ctx)
	if v1 != 100 {
		t.Errorf("expected 100, got %d", v1)
	}
	val = 250
	v2, _ := s.GetTVOC(ctx)
	if v2 != 250 {
		t.Errorf("expected 250, got %d", v2)
	}
}

func TestMockAirQualitySensor_Error(t *testing.T) {
	s := NewMockAirQualitySensor(func(ctx context.Context) (uint32, error) { return 0, fmt.Errorf("sensor error") })
	ctx := context.Background()
	_, err := s.GetTVOC(ctx)
	if err == nil || err.Error() != "sensor error" {
		t.Errorf("expected sensor error, got %v", err)
	}
}

func TestMockAirQualitySensor_ContextPropagation(t *testing.T) {
	var received context.Context
	s := NewMockAirQualitySensor(func(ctx context.Context) (uint32, error) { received = ctx; return 42, nil })
	type ctxKey string
	key := ctxKey("k")
	ctx := context.WithValue(context.Background(), key, "v")
	_, _ = s.GetTVOC(ctx)
	if received.Value(key) != "v" {
		t.Error("context was not propagated")
	}
}
