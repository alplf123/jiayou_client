package api

import (
	"jiayou_backend_spider/engine"

	"github.com/gin-gonic/gin"
)

func OK(ctx *gin.Context) {
	ctx.String(200, "ok")
}
func OnInit(app *engine.Engine) error {
	return nil
}
func OnLoad(app *engine.Engine) error {
	serv, err := app.Gin()
	if err != nil {
		return err
	}
	serv.Group("api")
	return nil
}
