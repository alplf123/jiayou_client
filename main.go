package main

import (
	_ "embed"
	"fmt"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service"
	"os"
)

func main() {
	args := os.Args[1:]
	var config string = "config.yaml"
	if len(args) > 0 {
		common.GlobalDevice = args[0]
		common.GlobalToken = args[1]

	}

	var app = engine.FromConfigFile(config)
	app.OnOptions(func(engine *engine.Engine) error {
		if err := service.OnInit(engine); err != nil {
			return err
		}
		return nil
	})
	if err := app.Start(); err != nil {
		panic(err)
	}
	if err := service.OnLoad(app); err != nil {
		panic(fmt.Errorf("service register failed,%w", err))
	}
	serv, _ := app.Gin()
	logger, _ := app.Log()
	opts := app.Options()
	logger.Info(fmt.Sprintf("gin server listen on [%s]", opts.Server))
	if err := serv.Run(app.Options().Server); err != nil {
		panic(fmt.Errorf("gin server run failed,%w", err))
	}
}
