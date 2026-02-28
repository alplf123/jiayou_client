package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser"
	"jiayou_backend_spider/browser/bit"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/model"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

var DefaultLogger *zap.Logger

const DefaultBrowserTryPeekTimeout = 60 * time.Second
const DefaultBrowserStepTimeout = 15 * time.Second
const DefaultBrowserHookLoginTimeout = 30 * time.Second
const DefaultBrowserPageTimeout = time.Minute
const DefaultProxy = "http://127.0.0.1:9878"

var DefaultDomain = "http://0.0.0.0:9876"

var DefaultFFmpegPath = "ffmpeg.exe"
var DefaultXrayPath = "xray.exe"
var DefaultXrayConfigPath = "xray.json"

var DefaultNodePath = "node.exe"

var GlobalDevice string = "default"
var GLogger *zap.Logger
var GBrowser *browser.App
var GBitBrowserOptions *bit.Options

func RunJsCtx(ctx context.Context, js string, call string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var _args []string
	for _, arg := range args {
		_args = append(_args, "'"+arg+"'")
	}
	var cmd = exec.CommandContext(ctx, DefaultNodePath)
	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stdin = strings.NewReader(
		fmt.Sprintf("%s\r\n%s", js,
			fmt.Sprintf("console.log(%s(%s))", call, strings.Join(_args, ","))))
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdOut.String()), nil
}
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
	if app.Options().Domain != "" {
		DefaultDomain = app.Options().Domain
	}
	return nil
}
