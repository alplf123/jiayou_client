package api

import (
	"jiayou_backend_spider/engine"
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(ctx *gin.Context) {}
func OnInit(app *engine.Engine) error {
	return nil
}
func OnLoad(app *engine.Engine) error {
	serv, err := app.Gin()
	if err != nil {
		return err
	}
	var api = serv.Group("api")
	api.Handle(http.MethodGet, "/ok", OK)
	return nil
}
