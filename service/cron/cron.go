package cron

import (
	"fmt"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/cron/tiktok"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/utils"
	"net"
	"time"
)

func ShouldRetry(err error) bool {
	switch v := err.(type) {
	case *net.OpError:
		if v.Timeout() || v.Temporary() {
			return true
		}
	case model.Retry:
		return v.ShouldRetry()
	case interface{ Unwrap() error }:
		return ShouldRetry(v.Unwrap())
	}
	return false
}

var DelayFunc = utils.DefaultBackoffDuration(time.Second, time.Second*15)

func OnInit(app *engine.Engine) error {
	for _, opt := range app.Options().Cron.DistributeTasks {
		opt.ServerOptions.ShouldRetry = ShouldRetry
		opt.ServerOptions.RetryDelayFunc = func(n int, e error, t *cron.Task) time.Duration {
			return DelayFunc(int64(n))
		}
	}
	return nil
}
func OnLoad(app *engine.Engine) error {
	distribute, err := app.Distribute()
	if err != nil {
		return err
	}
	if distribute.First() != nil {
		if err := tiktok.Register(distribute.First().Server()); err != nil {
			return fmt.Errorf("douyin register failed,%w", err)
		}
	}
	return nil
}
