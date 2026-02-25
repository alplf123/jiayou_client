package cron

import (
	"context"
	"fmt"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/common"
	"jiayou_backend_spider/service/cron/tiktok"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/service/xray"
	"jiayou_backend_spider/utils"
	"time"

	"go.uber.org/zap"
)

func ShouldRetry(err error) bool {
	switch v := err.(type) {
	case *model.ErrBase:
		return v.ShouldRetry()
	case model.IRetry:
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
		opt.ServerOptions.Queues = map[string]int{common.GlobalDevice: 1}
		opt.ServerOptions.RetryDelayFunc = func(n int, e error, t *cron.Task) time.Duration {
			return DelayFunc(int64(n))
		}
	}
	return nil
}
func OnLoad(app *engine.Engine) error {
	distribute, _ := app.Distribute()
	logger, _ := app.Log()
	if distribute.First() != nil {
		var server = distribute.First().Server()
		server.BeforeTask(func(ctx context.Context, task *cron.Task) error {
			if err := xray.Xray.Start(); err != nil {
				logger.Error("xray start failed", zap.Error(err))
				return model.NoRetry(err).WithTag(model.ErrXray)
			}
			logger.Info("xray loaded", zap.String("xray", common.DefaultXrayPath))
			logger.Info("xray config loaded", zap.String("xray", common.DefaultXrayConfigPath))
			var taskArg model.TaskArg
			if err := task.Payload().As(&taskArg); err == nil {
				if _, err := xray.Xray.Get(taskArg.ProxyName, taskArg.ProxyValue); err != nil {
					return err
				}
			}
			return nil
		})
		if err := tiktok.Register(server); err != nil {
			return fmt.Errorf("douyin register failed,%w", err)
		}
		if err := server.Start(); err != nil {
			logger.Error("server start failed", zap.Error(err))
			return err
		}
	}
	return nil
}
