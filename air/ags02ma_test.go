package air

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockI2CBus is a mock implementation of sensors.I2CBus using testify/mock
type MockI2CBus struct {
	mock.Mock
	concurrentOps int64 // tracks concurrent operations
	maxConcurrent int64 // maximum concurrent operations observed
	mu            sync.Mutex
}

func (m *MockI2CBus) WriteToAddr(ctx context.Context, address byte, buffer []byte) error {
	m.mu.Lock()
	concurrent := atomic.AddInt64(&m.concurrentOps, 1)
	if concurrent > atomic.LoadInt64(&m.maxConcurrent) {
		atomic.StoreInt64(&m.maxConcurrent, concurrent)
	}
	m.mu.Unlock()

	args := m.Called(ctx, address, buffer)

	m.mu.Lock()
	atomic.AddInt64(&m.concurrentOps, -1)
	m.mu.Unlock()

	return args.Error(0)
}

func (m *MockI2CBus) ReadFromAddr(ctx context.Context, address byte, buffer []byte) error {
	m.mu.Lock()
	concurrent := atomic.AddInt64(&m.concurrentOps, 1)
	if concurrent > atomic.LoadInt64(&m.maxConcurrent) {
		atomic.StoreInt64(&m.maxConcurrent, concurrent)
	}
	m.mu.Unlock()

	args := m.Called(ctx, address, buffer)
	if args.Get(0) != nil {
		// Copy mock data to buffer if provided
		if data, ok := args.Get(0).([]byte); ok && len(data) <= len(buffer) {
			copy(buffer, data)
		}
	}

	m.mu.Lock()
	atomic.AddInt64(&m.concurrentOps, -1)
	m.mu.Unlock()

	return args.Error(1)
}

func (m *MockI2CBus) Release(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockI2CBus) reset() {
	atomic.StoreInt64(&m.concurrentOps, 0)
	atomic.StoreInt64(&m.maxConcurrent, 0)
	m.ExpectedCalls = nil
	m.Calls = nil
}

// Helper to create valid TVOC response data
func validTVOCResponse(ppb uint32) []byte {
	buf := make([]byte, 5)
	buf[0] = 0x00 // status: ready
	buf[1] = byte((ppb >> 16) & 0xFF)
	buf[2] = byte((ppb >> 8) & 0xFF)
	buf[3] = byte(ppb & 0xFF)
	buf[4] = checkCRC(buf[:4])
	return buf
}

// Helper to create version response data
func validVersionResponse(version int) []byte {
	buf := make([]byte, 5)
	buf[0] = 0x00
	buf[1] = 0x00
	buf[2] = 0x00
	buf[3] = byte(version)
	buf[4] = checkCRC(buf[:4])
	return buf
}

// Helper to create resistance response data
func validResistanceResponse(resistance int) []byte {
	buf := make([]byte, 5)
	buf[0] = 0x00
	buf[1] = 0x00
	buf[2] = 0x00
	buf[3] = byte(resistance)
	buf[4] = checkCRC(buf[:4])
	return buf
}

func TestAGS02MA_DelayMechanism(t *testing.T) {
	tests := []struct {
		name      string
		readDelay time.Duration
	}{
		{
			name:      "short delay",
			readDelay: 50 * time.Millisecond,
		},
		{
			name:      "medium delay",
			readDelay: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithReadDelay(tt.readDelay))
			ctx := context.Background()

			// First operation - should return quickly
			bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
				Return(validTVOCResponse(1000), nil).Once()

			start := time.Now()
			ppb, err := sensor.GetTVOC(ctx)
			firstDuration := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, uint32(1000), ppb)
			assert.Less(t, firstDuration, tt.readDelay/2, "First operation should return quickly (delay runs async)")

			// Second operation - should wait for delay
			bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
				Return(validTVOCResponse(2000), nil).Once()

			start = time.Now()
			ppb, err = sensor.GetTVOC(ctx)
			secondDuration := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, uint32(2000), ppb)
			assert.GreaterOrEqual(t, secondDuration, tt.readDelay-time.Millisecond*10, "Second operation should wait for delay")

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_MutexProtection(t *testing.T) {
	bus := new(MockI2CBus)
	delay := 20 * time.Millisecond
	sensor := NewAGS02MA(bus, WithReadDelay(delay))
	ctx := context.Background()

	// First operation to establish baseline
	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	_, err := sensor.GetTVOC(ctx)
	assert.NoError(t, err)

	// Wait a bit to ensure delay is active
	time.Sleep(5 * time.Millisecond)

	// Reset counters
	bus.reset()

	// Run multiple concurrent operations
	const numOps = 5
	var wg sync.WaitGroup
	wg.Add(numOps)

	// Setup expectations for concurrent calls
	for i := 0; i < numOps; i++ {
		ppb := uint32(1000 + i)
		bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
			Return(validTVOCResponse(ppb), nil).Once()
	}

	for i := 0; i < numOps; i++ {
		go func() {
			defer wg.Done()
			_, err := sensor.GetTVOC(ctx)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	// Verify no concurrent bus operations occurred
	assert.LessOrEqual(t, atomic.LoadInt64(&bus.maxConcurrent), int64(1), "Mutex should serialize operations")
	bus.AssertExpectations(t)
}

func TestAGS02MA_GetTVOC_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "read error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil, errors.New("i2c read failed")).Once()
			},
			expectedError: "ags02ma: read failed: i2c read failed",
		},
		{
			name: "not ready status",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = statusBitRDY // Not ready
				buf[1] = 0x00
				buf[2] = 0x03
				buf[3] = 0xE8
				buf[4] = checkCRC(buf[:4])
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: ErrNotReady.Error(),
		},
		{
			name: "context cancelled during delay wait",
			setupMock: func(bus *MockI2CBus) {
				// First call succeeds to trigger delay
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validTVOCResponse(1000), nil).Once()
			},
			expectedError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithReadDelay(10*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			if tt.name == "context cancelled during delay wait" {
				// First call to trigger delay
				_, err := sensor.GetTVOC(ctx)
				assert.NoError(t, err)

				// Second call with cancelled context
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()
				_, err = sensor.GetTVOC(cancelledCtx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				_, err := sensor.GetTVOC(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_GetTVOCWithRegisterWrite_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "write error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(errors.New("i2c write failed")).Once()
			},
			expectedError: "ags02ma: write reg 0x00 failed: i2c write failed",
		},
		{
			name: "read error after write",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil, errors.New("i2c read failed")).Once()
			},
			expectedError: "ags02ma: read failed: i2c read failed",
		},
		{
			name: "crc mismatch",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = 0x00
				buf[1] = 0x00
				buf[2] = 0x03
				buf[3] = 0xE8
				buf[4] = 0xFF // Wrong CRC
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: "ags02ma: crc mismatch",
		},
		{
			name: "not ready status",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = statusBitRDY // Not ready
				buf[1] = 0x00
				buf[2] = 0x03
				buf[3] = 0xE8
				buf[4] = checkCRC(buf[:4])
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: ErrNotReady.Error(),
		},
		{
			name: "context cancelled during tx delay",
			setupMock: func(bus *MockI2CBus) {
				// Write happens before tx delay, but context might be cancelled before write
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Maybe()
			},
			expectedError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithReadDelay(10*time.Millisecond), WithTxDelay(50*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			if tt.name == "context cancelled during tx delay" {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()
				_, err := sensor.GetTVOCWithRegisterWrite(cancelledCtx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				_, err := sensor.GetTVOCWithRegisterWrite(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_Configure_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "write error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(errors.New("i2c write failed")).Once()
			},
			expectedError: "ags02ma: configuration write failed: i2c write failed",
		},
		{
			name: "context cancelled during delay wait",
			setupMock: func(bus *MockI2CBus) {
				// First call succeeds to trigger delay
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
			},
			expectedError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithConfigureDelay(10*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			if tt.name == "context cancelled during delay wait" {
				// First call to trigger delay
				err := sensor.Configure(ctx)
				assert.NoError(t, err)

				// Second call with cancelled context
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()
				err = sensor.Configure(cancelledCtx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				err := sensor.Configure(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_ReadVersion_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "write error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(errors.New("i2c write failed")).Once()
			},
			expectedError: "ags02ma: write reg 0x11 failed: i2c write failed",
		},
		{
			name: "read error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil, errors.New("i2c read failed")).Once()
			},
			expectedError: "ags02ma: read failed: i2c read failed",
		},
		{
			name: "crc mismatch",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = 0x00
				buf[1] = 0x00
				buf[2] = 0x00
				buf[3] = 0x01
				buf[4] = 0xFF // Wrong CRC
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: "ags02ma: crc mismatch",
		},
		{
			name: "context cancelled during tx delay",
			setupMock: func(bus *MockI2CBus) {
				// Write happens before tx delay, but context might be cancelled before write
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Maybe()
			},
			expectedError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithTxDelay(50*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			if tt.name == "context cancelled during tx delay" {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()
				_, err := sensor.ReadVersion(cancelledCtx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				_, err := sensor.ReadVersion(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_ReadResistance_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "write error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(errors.New("i2c write failed")).Once()
			},
			expectedError: "ags02ma: write reg 0x20 failed: i2c write failed",
		},
		{
			name: "read error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil, errors.New("i2c read failed")).Once()
			},
			expectedError: "ags02ma: read failed: i2c read failed",
		},
		{
			name: "crc mismatch",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = 0x00
				buf[1] = 0x00
				buf[2] = 0x00
				buf[3] = 0x64
				buf[4] = 0xFF // Wrong CRC
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: "ags02ma: crc mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithReadDelay(10*time.Millisecond), WithTxDelay(10*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			_, err := sensor.ReadResistance(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_Calibrate_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockI2CBus)
		expectedError string
	}{
		{
			name: "write error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(errors.New("i2c write failed")).Once()
			},
			expectedError: "ags02ma: write reg 0x01 failed: i2c write failed",
		},
		{
			name: "read error",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil, errors.New("i2c read failed")).Once()
			},
			expectedError: "ags02ma: read failed: i2c read failed",
		},
		{
			name: "crc mismatch",
			setupMock: func(bus *MockI2CBus) {
				buf := make([]byte, 5)
				buf[0] = 0x00
				buf[1] = 0x00
				buf[2] = 0x00
				buf[3] = 0x00
				buf[4] = 0xFF // Wrong CRC
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(buf, nil).Once()
			},
			expectedError: "ags02ma: crc mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus, WithReadDelay(10*time.Millisecond), WithTxDelay(10*time.Millisecond))
			ctx := context.Background()

			tt.setupMock(bus)

			err := sensor.Calibrate(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_SuccessCases(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockI2CBus)
		testFunc  func(*testing.T, *AGS02MA, context.Context) (interface{}, error)
		expected  interface{}
	}{
		{
			name: "GetTVOC success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validTVOCResponse(1000), nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return s.GetTVOC(ctx)
			},
			expected: uint32(1000),
		},
		{
			name: "GetTVOCWithRegisterWrite success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validTVOCResponse(2000), nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return s.GetTVOCWithRegisterWrite(ctx)
			},
			expected: uint32(2000),
		},
		{
			name: "ReadVersion success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validVersionResponse(42), nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return s.ReadVersion(ctx)
			},
			expected: 42,
		},
		{
			name: "ReadResistance success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validResistanceResponse(100), nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return s.ReadResistance(ctx)
			},
			expected: 100,
		},
		{
			name: "Configure success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return nil, s.Configure(ctx)
			},
			expected: nil,
		},
		{
			name: "Calibrate success",
			setupMock: func(bus *MockI2CBus) {
				bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(nil).Once()
				bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
					Return(validResistanceResponse(0), nil).Once()
			},
			testFunc: func(t *testing.T, s *AGS02MA, ctx context.Context) (interface{}, error) {
				return nil, s.Calibrate(ctx)
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := new(MockI2CBus)
			sensor := NewAGS02MA(bus,
				WithReadDelay(10*time.Millisecond),
				WithTxDelay(10*time.Millisecond),
				WithConfigureDelay(10*time.Millisecond),
			)
			ctx := context.Background()

			tt.setupMock(bus)

			result, err := tt.testFunc(t, sensor, ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			bus.AssertExpectations(t)
		})
	}
}

func TestAGS02MA_AsynchronousDelay(t *testing.T) {
	bus := new(MockI2CBus)
	delay := 100 * time.Millisecond
	sensor := NewAGS02MA(bus, WithReadDelay(delay))
	ctx := context.Background()

	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	// First operation should return immediately, not wait for delay
	start := time.Now()
	ppb, err := sensor.GetTVOC(ctx)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, uint32(1000), ppb)
	assert.Less(t, duration, delay/2, "Operation should return quickly (delay runs asynchronously)")

	// Wait a bit to ensure delay goroutine has started
	time.Sleep(10 * time.Millisecond)

	// Verify delay is still running (second operation should wait)
	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(2000), nil).Once()

	start = time.Now()
	ppb, err = sensor.GetTVOC(ctx)
	secondDuration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, uint32(2000), ppb)
	assert.GreaterOrEqual(t, secondDuration, delay-time.Millisecond*20, "Should wait for async delay")

	bus.AssertExpectations(t)
}

func TestAGS02MA_ContextCancellation(t *testing.T) {
	bus := new(MockI2CBus)
	delay := 100 * time.Millisecond
	sensor := NewAGS02MA(bus, WithReadDelay(delay))
	ctx := context.Background()

	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	// First operation to trigger delay
	_, err := sensor.GetTVOC(ctx)
	assert.NoError(t, err)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Operation with cancelled context should fail immediately
	start := time.Now()
	_, err = sensor.GetTVOC(cancelledCtx)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Less(t, duration, 50*time.Millisecond, "Cancelled operation should fail quickly")

	bus.AssertExpectations(t)
}

func TestAGS02MA_MixedOperations(t *testing.T) {
	bus := new(MockI2CBus)
	configureDelay := 30 * time.Millisecond
	readDelay := 20 * time.Millisecond
	txDelay := 10 * time.Millisecond
	sensor := NewAGS02MA(bus,
		WithConfigureDelay(configureDelay),
		WithReadDelay(readDelay),
		WithTxDelay(txDelay),
	)
	ctx := context.Background()

	bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(nil).Once()

	// Configure should schedule its delay
	err := sensor.Configure(ctx)
	assert.NoError(t, err)

	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	// GetTVOC should wait for configure delay, then schedule its own
	start := time.Now()
	_, err = sensor.GetTVOC(ctx)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, configureDelay-time.Millisecond*10, "Should wait for configure delay")

	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(2000), nil).Once()

	// Next operation should wait for read delay
	start = time.Now()
	_, err = sensor.GetTVOC(ctx)
	secondDuration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, secondDuration, readDelay-time.Millisecond*10, "Should wait for read delay")

	bus.AssertExpectations(t)
}

func TestAGS02MA_ConcurrentOperations(t *testing.T) {
	bus := new(MockI2CBus)
	configureDelay := 50 * time.Millisecond
	readDelay := 30 * time.Millisecond
	sensor := NewAGS02MA(bus,
		WithConfigureDelay(configureDelay),
		WithReadDelay(readDelay),
	)
	ctx := context.Background()

	bus.On("WriteToAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(nil).Once()
	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	var wg sync.WaitGroup
	wg.Add(2)

	configureDone := make(chan error, 1)
	readDone := make(chan error, 1)

	go func() {
		defer wg.Done()
		err := sensor.Configure(ctx)
		configureDone <- err
	}()

	go func() {
		defer wg.Done()
		_, err := sensor.GetTVOC(ctx)
		readDone <- err
	}()

	wg.Wait()

	assert.NoError(t, <-configureDone)
	assert.NoError(t, <-readDone)

	// Verify no concurrent bus operations
	assert.LessOrEqual(t, atomic.LoadInt64(&bus.maxConcurrent), int64(1), "Operations should be serialized")
	bus.AssertExpectations(t)
}

func TestAGS02MA_CloseWaitsForDelay(t *testing.T) {
	bus := new(MockI2CBus)
	delay := 50 * time.Millisecond
	sensor := NewAGS02MA(bus, WithReadDelay(delay))
	ctx := context.Background()

	bus.On("ReadFromAddr", mock.Anything, byte(ags02maAddress), mock.Anything).
		Return(validTVOCResponse(1000), nil).Once()

	// Trigger a delay
	_, err := sensor.GetTVOC(ctx)
	assert.NoError(t, err)

	// Close should wait for delay to complete
	start := time.Now()
	sensor.Close(ctx)
	duration := time.Since(start)

	assert.GreaterOrEqual(t, duration, delay-time.Millisecond*10, "Close should wait for delay")

	bus.AssertExpectations(t)
}
