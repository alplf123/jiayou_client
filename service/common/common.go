package common

import (
	"errors"
	"fmt"
	"jiayou_backend_spider/browser"
	"jiayou_backend_spider/browser/bit"
	"jiayou_backend_spider/service/model"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const DefaultDeviceSyncQueue = "__device_sync_queue__"

var DefaultLogger *zap.Logger

var GlobalDevice = ""
var GlobalToken = ""

var XrayProcess *os.Process

var DefaultXrayConfig = "xray.json"
var DefaultXrayPath = "xray.exe"

var DefaultXrayConfigMap = new(sync.Map)

const DefaultXrayHost = "127.0.0.1"
const DefaultXrayPort = 9878
const DefaultXrayUser = "jiayou"
const DefaultXrayPwd = "jiayou"

var DefaultProxy = fmt.Sprintf("http://%s:%s@%s:%d", DefaultXrayUser, DefaultXrayPwd, DefaultXrayHost, DefaultXrayPort)

const DefaultBrowserTryPeek = 60 * time.Second

var GBrowser *browser.App
var GBitBrowserOptions *bit.Options

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
