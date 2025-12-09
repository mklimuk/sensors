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
