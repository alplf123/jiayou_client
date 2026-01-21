package retry

import (
	"context"
	"fmt"
	"jiayou_backend_spider/utils"
	"jiayou_backend_spider/utils/wait"
	"time"
)

type Func[T any] func(*Context) (T, error)
type IRetry interface {
	Retry()
}
type ShouldRetry interface {
	Retry() bool
}
type IfRetry func(*Context, error) bool

func AnyRetry() IfRetry {
	return func(c *Context, err error) bool {
		return true
	}
}
func defaultOption() *Option {
	return &Option{
		ctx:         context.Background(),
		backoffFunc: utils.FixedDuration(utils.DefaultFixBackoffDuration),
	}
}

type retryWrapper struct {
	error
}

func (retry retryWrapper) Retry() {}

func WithRetry(err error) error {
	return retryWrapper{error: err}
}
func shouldRetry(err error) bool {
	switch e := err.(type) {
	case IRetry:
		return true
	case ShouldRetry:
		return e.Retry()
	case interface{ Unwrap() error }:
		return shouldRetry(e.Unwrap())
	}
	return false
}

type Option struct {
	ctx         context.Context
	backoffFunc utils.BackoffFunc
	ifRetry     IfRetry
	maxRetry    int64
}
type OptionFunc func(*Option)

type Context struct {
	ctx      context.Context
	attempt  int64
	interval time.Duration
}

func (ctx Context) Ctx() context.Context {
	return ctx.ctx
}
func (ctx Context) Attempt() int64 {
	return ctx.attempt
}
func (ctx Context) Interval() time.Duration {
	return ctx.interval
}

func WithCtx(ctx context.Context) OptionFunc {
	return func(o *Option) {
		o.ctx = ctx
	}
}

func WithBackoffFunc(f utils.BackoffFunc) OptionFunc {
	return func(o *Option) {
		if f != nil {
			o.backoffFunc = f
		}
	}
}
func WithIfRetry(f IfRetry) OptionFunc {
	return func(o *Option) {
		o.ifRetry = f
	}
}
func WithMaxRetry(retry int64) OptionFunc {
	return func(o *Option) {
		o.maxRetry = retry
	}
}
func retry[T any](_func Func[T], opt *Option) (t T, err error) {
	return wait.Wait(func(c *wait.Context) (t T, v bool) {
		var ctx = &Context{
			ctx:      opt.ctx,
			attempt:  c.Attempt(),
			interval: c.Interval(),
		}
		if opt.maxRetry > 0 && ctx.Attempt() > opt.maxRetry {
			c.Abort(fmt.Errorf("out of max retry[%d]", opt.maxRetry))
			v = true
			return
		}
		if r, _err := _func(ctx); _err != nil {
			if shouldRetry(_err) || opt.ifRetry == nil ||
				opt.ifRetry(ctx, _err) {
			} else {
				c.Abort(_err)
				v = true
				return
			}
			select {
			case <-opt.ctx.Done():
				c.Abort(opt.ctx.Err())
				v = true
				return
			default:
			}
		} else {
			return r, true
		}
		return
	}, wait.WithCtx(opt.ctx), wait.WithBackoffFunc(opt.backoffFunc))
}
func Retry[T any](_func Func[T], opts ...OptionFunc) (t T, err error) {
	var defaultOpt = defaultOption()
	for _, opt := range opts {
		opt(defaultOpt)
	}
	if defaultOpt.ctx == nil {
		defaultOpt.ctx = context.Background()
	}
	return retry(_func, defaultOpt)
}
