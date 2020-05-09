package environment

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHIH6021_ConvertHum(t *testing.T) {
	tests := []struct {
		given    []byte
		expected float32
	}{
		{[]byte{0x00, 0x00}, 0.0},
		{[]byte{0x3F, 0xFF}, 100.0},
		{[]byte{0x17, 0x8B}, 36.79038},
	}
	for _, test := range tests {
		t.Run(hex.EncodeToString(test.given), func(t *testing.T) {
			assert.Equal(t, test.expected, convertHumidity(test.given))
		})
	}
}

func TestHIH6021_ConvertTemp(t *testing.T) {
	tests := []struct {
		given    []byte
		expected float32
	}{
		{[]byte{0x00, 0x00}, -40.0},
		{[]byte{0xFF, 0xFC}, 125.01007},
		{[]byte{0x65, 0xB8}, 25.568916},
	}
	for _, test := range tests {
		t.Run(hex.EncodeToString(test.given), func(t *testing.T) {
			assert.Equal(t, test.expected, convertTemperature(test.given))
		})
	}
}
