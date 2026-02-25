package common

import (
	"bytes"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser"
	"jiayou_backend_spider/browser/bit"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/model"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

var DefaultLogger *zap.Logger

const DefaultBrowserTryPeekTimeout = 60 * time.Second
const DefaultBrowserHookLoginTimeout = 30 * time.Second
const DefaultBrowserPageTimeout = 30 * time.Second

const DefaultProxy = "http://127.0.0.1:9878"

var DefaultFFmpegPath = "ffmpeg.exe"
var DefaultXrayPath = "xray.exe"
var DefaultXrayConfigPath = "xray.json"

var GlobalDevice string
var GLogger *zap.Logger
var GBrowser *browser.App
var GBitBrowserOptions *bit.Options

func GenerateVideoFirstFrame(file string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := ffmpeg.Input(file).
		Silent(true).
		Output("pipe:", ffmpeg.KwArgs{
			"ss":      "00:00:00",
			"vframes": 1,
			"vcodec":  "png",
			"f":       "image2pipe",
			"pix_fmt": "rgb24",
		}).
		SetFfmpegPath(DefaultFFmpegPath).
		WithOutput(buf).
		Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg run failed,%w", err)
	}
	return buf.Bytes(), nil

}
func GetCommentLine(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	var items = strings.Split(string(data), "\r\n")
	if len(items) == 0 {
		return "", errors.New(model.ErrCommentInvalidLine)
	}
	return items[rand.IntN(len(items))], nil
}

func OnLoad(app *engine.Engine) error {
	GBrowser, _ = app.Browser()
	GLogger, _ = app.Log()
	GBitBrowserOptions = app.Options().Browser.BitOptions
	if app.Options().Device != "" {
		GlobalDevice = app.Options().Device
	}
	return nil
}
