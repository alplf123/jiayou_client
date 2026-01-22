package api

import (
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/status"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func OK(ctx *gin.Context) {
	_, err := os.Stat(common.DefaultXrayPath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusOK, status.ErrXrayNotFound)
			return
		}
		ctx.JSON(http.StatusOK, status.ErrXrayFile)
		return
	}
	_, err = os.Stat(common.DefaultXrayConfig)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusOK, status.ErrXrayConfigNotFound)
			return
		}
		ctx.JSON(http.StatusOK, status.ErrXrayConfig)
		return
	}
	if common.XrayProcess == nil {
		ctx.JSON(http.StatusOK, status.ErrXray)
		return
	}
	_, err = os.FindProcess(common.XrayProcess.Pid)
	if err != nil {
		ctx.JSON(http.StatusOK, status.ErrXray)
		return
	}
	ctx.JSON(http.StatusOK, status.Success)
}
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
