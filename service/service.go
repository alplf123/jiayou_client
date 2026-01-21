package service

import (
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/api"
	"jiayou_backend_spider/service/cron"
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
	if err := api.OnLoad(app); err != nil {
		return err
	}
	if err := cron.OnLoad(app); err != nil {
		return err
	}
	return nil
}
