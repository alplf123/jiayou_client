package wait

import (
	"context"
	"jiayou_backend_spider/utils"
	"time"
)

type Option struct {
	ctx         context.Context
	maxAttempt  int64
	timeout     time.Duration
	backoffFunc utils.BackoffFunc
}

type OptionFunc func(*Option)
type Predicate[T any] func(*Context) (T, bool)

type Context struct {
	err      error
	ctx      context.Context
	abort    bool
	attempt  int64
	interval time.Duration
}

func (r *Context) Attempt() int64 {
	return r.attempt
}
func (r *Context) Interval() time.Duration {
	return r.interval
}
func (r *Context) Abort(err error) {
	r.abort = true
	r.err = err
}

func defaultOption() Option {
	return Option{
		ctx:         context.Background(),
		backoffFunc: utils.FixedDuration(utils.DefaultFixBackoffDuration),
	}
}
func wait(ctx context.Context, interval time.Duration) error {
	var ticker = time.NewTicker(interval)
	select {
	case <-ticker.C:
	case <-ctx.Done():
		return context.DeadlineExceeded
	}
	return nil
}

func WithCtx(ctx context.Context) OptionFunc {
	return func(o *Option) {
		o.ctx = ctx
	}
}
func WithTimeout(timeout time.Duration) OptionFunc {
	return func(o *Option) {
		o.timeout = timeout
	}
}

func WithBackoffFunc(f utils.BackoffFunc) OptionFunc {
	return func(o *Option) {
		if f != nil {
			o.backoffFunc = f
		}
	}
}

func Wait[T any](_func Predicate[T], opts ...OptionFunc) (t T, err error) {
	var defaultOpt = defaultOption()
	for _, opt := range opts {
		opt(&defaultOpt)
	}
	var ctx = defaultOpt.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if defaultOpt.timeout > 0 {
		_ctx, cancel := context.WithTimeout(ctx, defaultOpt.timeout)
		defer cancel()
		ctx = _ctx
	}
	var c = &Context{ctx: ctx}
	for {
		c.attempt++
		if defaultOpt.backoffFunc != nil {
			c.interval = defaultOpt.backoffFunc(c.attempt)
		}
		if v, ok := _func(c); c.abort || ok {
			return v, c.err
		}
		if c.interval > 0 {
			err = wait(ctx, c.interval)
			if err != nil {
				return
			}
		}
	}
}
func Once(opts ...OptionFunc) {
	Wait(func(r *Context) (any, bool) { return nil, true }, opts...)
}
func Forever() {
	Wait(func(r *Context) (any, bool) { return nil, false })
}
