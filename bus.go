package sensors

import (
	"context"
	"fmt"
)

var ErrBusBusy = fmt.Errorf("I2C engine is busy (command not completed)")

type BusReader interface {
	Read(ctx context.Context, buffer []byte) error
}

type BusWriter interface {
	Write(ctx context.Context, buffer []byte) error
}

type AddressableReader interface {
	ReadFromAddr(ctx context.Context, address byte, buffer []byte) error
}

type AddressableWriter interface {
	WriteToAddr(ctx context.Context, address byte, buffer []byte) error
	Release(ctx context.Context) error
}

type I2CBus interface {
	AddressableReader
	AddressableWriter
}

type I2CDevice interface {
	BusReader
	BusWriter
}

// AddressLocker provides per-address mutual exclusion for multi-step I2C
// sequences (e.g. trigger-write → delay → data-read) that must not be
// interleaved by another goroutine targeting the same device address.
// Callers must pair every LockAddr with a corresponding UnlockAddr.
type AddressLocker interface {
	LockAddr(addr byte)
	UnlockAddr(addr byte)
}
