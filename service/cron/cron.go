package cron

import (
	"context"
	"fmt"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/cron/tiktok"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/service/xray"
	"jiayou_backend_spider/utils"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
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

var pLocker sync.Mutex

func OnInit(app *engine.Engine) error {
	for _, opt := range app.Options().Cron.DistributeTasks {
		opt.ServerOptions.ShouldRetry = ShouldRetry
		opt.ServerOptions.Queues = map[string]int{common.GlobalDevice: 1}
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
	logger, err := app.Log()
	if err != nil {
		return err
	}
	if distribute.First() != nil {
		var server = distribute.First().Server()
		server.BeforeTask(func(ctx context.Context, task *cron.Task) error {
			pLocker.Lock()
			defer pLocker.Unlock()
			if common.XrayProcess == nil {
				if err := xray.StartXray(); err != nil {
					logger.Error("xray start failed", zap.Error(err))
					return model.NewBase().WithTag(model.ErrXray).WithError(err)
				}
				logger.Info("xray loaded", zap.String("xray", common.DefaultXrayPath))
				logger.Info("xray config loaded", zap.String("xray", common.DefaultXrayConfig))
			}
			return nil
		})
		if err := tiktok.Register(server); err != nil {
			return fmt.Errorf("douyin register failed,%w", err)
		}
	}
	return nil
}
