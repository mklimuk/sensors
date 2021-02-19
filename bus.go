package sensors

import (
	"context"
	"fmt"
)

var ErrBusBusy = fmt.Errorf("I2C engine is busy (command not completed)")

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
