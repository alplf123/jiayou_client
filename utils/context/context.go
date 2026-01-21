package context

import "context"

type CancelContext struct {
	parent context.Context
	ctx    context.Context
	cancel context.CancelFunc
}

func (ctx *CancelContext) Ctx() context.Context {
	if ctx.ctx != nil {
		return ctx.ctx
	}
	return ctx.parent
}
func (ctx *CancelContext) Cancel() {
	if ctx.cancel != nil {
		ctx.cancel()
		ctx.ctx = nil
		ctx.cancel = nil
	}
}
func (ctx *CancelContext) Canceled() bool {
	return ctx.cancel == nil || ctx.ctx == nil
}
func New(ctx context.Context) *CancelContext {
	var _ctx = &CancelContext{parent: ctx}
	if _ctx.parent == nil {
		_ctx.parent = context.Background()
		_ctx.ctx, _ctx.cancel = context.WithCancel(_ctx.parent)
	} else {
		_ctx.ctx, _ctx.cancel = context.WithCancel(_ctx.parent)
	}
	return _ctx
}
