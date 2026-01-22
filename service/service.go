package service

import (
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/api"
	"jiayou_backend_spider/service/cron"
	"jiayou_backend_spider/service/ffmpeg"
	"jiayou_backend_spider/service/xray"
)

func OnInit(app *engine.Engine) error {
	if err := api.OnInit(app); err != nil {
		return err
	}
	if err := cron.OnInit(app); err != nil {
		return err
	}
	return nil
}
func OnLoad(app *engine.Engine) error {
	if err := ffmpeg.OnLoad(app); err != nil {
		return err
	}
	if err := xray.OnLoad(app); err != nil {
		return err
	}
	if err := api.OnLoad(app); err != nil {
		return err
	}
	if err := cron.OnLoad(app); err != nil {
		return err
	}
	return nil
}
