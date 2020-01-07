package sensors

import "context"

type AddressableReader interface{
	ReadFromAddr(ctx context.Context, address byte, buffer []byte) error
}

type AddressableWriter interface{
	WriteToAddr(ctx context.Context, address byte, buffer []byte)  error
}

type I2CBus interface{
	AddressableReader
	AddressableWriter
}