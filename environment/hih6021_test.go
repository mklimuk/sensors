package environment

import (
	"context"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// lockableBus is a test I2CBus that also implements AddressLocker.
// It tracks whether write→read sequences overlap across goroutines.
type lockableBus struct {
	addrMu      sync.Mutex
	addrLocks   map[byte]*sync.Mutex
	inSequence  atomic.Int32
	maxOverlap  atomic.Int32
	writeDelay  time.Duration
}

func newLockableBus(writeDelay time.Duration) *lockableBus {
	return &lockableBus{
		addrLocks:  make(map[byte]*sync.Mutex),
		writeDelay: writeDelay,
	}
}

func (b *lockableBus) addrMutex(addr byte) *sync.Mutex {
	b.addrMu.Lock()
	defer b.addrMu.Unlock()
	mu, ok := b.addrLocks[addr]
	if !ok {
		mu = &sync.Mutex{}
		b.addrLocks[addr] = mu
	}
	return mu
}

func (b *lockableBus) LockAddr(addr byte)   { b.addrMutex(addr).Lock() }
func (b *lockableBus) UnlockAddr(addr byte) { b.addrMutex(addr).Unlock() }

func (b *lockableBus) WriteToAddr(_ context.Context, _ byte, _ []byte) error {
	n := b.inSequence.Add(1)
	for {
		cur := b.maxOverlap.Load()
		if n <= cur || b.maxOverlap.CompareAndSwap(cur, n) {
			break
		}
	}
	if b.writeDelay > 0 {
		time.Sleep(b.writeDelay)
	}
	return nil
}

func (b *lockableBus) ReadFromAddr(_ context.Context, _ byte, buf []byte) error {
	defer b.inSequence.Add(-1)
	// valid HIH6021 response: status OK, ~37% humidity, ~25 °C
	copy(buf, []byte{0x17, 0x8B, 0x65, 0xB8})
	return nil
}

func (b *lockableBus) Release(_ context.Context) error { return nil }

func TestHIH6021_AddressLockPreventsInterleaving(t *testing.T) {
	bus := newLockableBus(5 * time.Millisecond)

	const n = 4
	sensors := make([]*HIH6021, n)
	for i := range sensors {
		sensors[i] = NewHIH6021(bus)
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(s *HIH6021) {
			defer wg.Done()
			_, _, err := s.GetTempAndHum(context.Background())
			assert.NoError(t, err)
		}(sensors[i])
	}
	wg.Wait()

	assert.Equal(t, int32(1), bus.maxOverlap.Load(),
		"address lock should serialize write→read sequences; observed %d concurrent", bus.maxOverlap.Load())
}
