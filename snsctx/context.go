package snsctx

import "context"

type ctxIndex int

const ctxIndexVerbose ctxIndex = iota

func IsVerbose(ctx context.Context) bool {
	val := ctx.Value(ctxIndexVerbose)
	if val == nil {
		return false
	}
	return val.(bool)
}

func SetVerbose(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, ctxIndexVerbose, value)
}
