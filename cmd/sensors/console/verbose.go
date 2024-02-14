package console

import "context"

type ctxIndex int

const ctxIndexVerbose ctxIndex = iota

func SetVerbose(parent context.Context, value bool) context.Context {
	return context.WithValue(parent, ctxIndexVerbose, value)
}

func IsVerbose(ctx context.Context) bool {
	val := ctx.Value(ctxIndexVerbose)
	if val == nil {
		return false
	}
	return val.(bool)
}
