package xray

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/model"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func generateXrayConfig(proxy string) error {
	var vLessUrl, err = url.Parse(proxy)
	if err != nil {
		return fmt.Errorf("bad vless url,%w", err)
	}
	var xrayVLess = map[string]any{
		"inbounds": []any{
			map[string]any{
				"tag":      "socks",
				"port":     common.DefaultXrayPort,
				"listen":   common.DefaultXrayHost,
				"protocol": "mixed",
				"sniffing": map[string]any{
					"enabled": true,
					"destOverride": []any{
						"http",
						"tls",
					},
					"routeOnly": false,
				},
				"settings": map[string]any{
					"auth": "password",
					"accounts": []any{
						map[string]any{
							"user": common.DefaultXrayUser,
							"pass": common.DefaultXrayPwd,
						},
					},
					"udp":              true,
					"allowTransparent": false,
				},
			},
		},
		"outbounds": []any{}}
	var outBounds = make([]any, 1)
	var vLess = map[string]any{}
	var port, _ = strconv.Atoi(vLessUrl.Port())
	vLess["protocol"] = "vless"
	vLess["settings"] = map[string]any{
		"vnext": []any{
			map[string]any{
				"address": vLessUrl.Hostname(),
				"port":    port,
				"users": []any{
					map[string]any{
						"id":         vLessUrl.User.Username(),
						"flow":       vLessUrl.Query().Get("flow"),
						"encryption": vLessUrl.Query().Get("encryption"),
						"level":      0,
					},
				},
			},
		},
	}
	vLess["streamSettings"] = map[string]any{
		"network":  vLessUrl.Query().Get("type"),
		"security": vLessUrl.Query().Get("security"),
		"realitySettings": map[string]any{
			"serverName":  vLessUrl.Query().Get("sni"),
			"fingerprint": vLessUrl.Query().Get("fp"),
			"publicKey":   vLessUrl.Query().Get("pbk"),
		},
	}
	outBounds[0] = vLess
	xrayVLess["outbounds"] = outBounds
	dat, err := json.Marshal(xrayVLess)
	if err != nil {
		return fmt.Errorf("marshal vless json,%w", err)
	}
	err = os.WriteFile(common.DefaultXrayConfig, dat, 0644)
	if err != nil {
		return fmt.Errorf("write vless json,%w", err)
	}
	return nil
}
func findActiveProxy() (string, error) {
	var header = request.DefaultRequestOptions()
	header.Header.Set("Authorization", common.GlobalToken)
	var resp, err = request.Post(common.DefaultDomain+"/api/user/setting", nil, header)
	if err != nil {
		return "", fmt.Errorf("get user setting,%w", err)
	}
	if resp.Status() != 200 {
		return "", model.NewStatusError(resp.Status())
	}
	var j = gjson.Parse(resp.Text())
	var code = j.Get("code").Int()
	if code != 0 {
		return "", model.NewApiError().WithCode(int(code)).WithMessage(j.Get("message").String())
	}
	var p = j.Get("result.proxy").String()
	if p == "" {
		return "", errors.New("proxy need")
	}
	return p, nil
}
func StartXray() error {
	exec.Command("taskkill", "/F", "/IM", "xray.exe").Run()
	proxy, err := findActiveProxy()
	if err != nil {
		return err
	}
	err = generateXrayConfig(proxy)
	if err != nil {
		return err
	}
	var p, _ = os.Executable()
	var xray, xrayConfig string
	if p == "" {
		xray = "./" + common.DefaultXrayPath
		xrayConfig = "./" + common.DefaultXrayConfig
	} else {
		xray = filepath.Join(filepath.Dir(p), common.DefaultXrayPath)
		xrayConfig = filepath.Join(filepath.Dir(p), common.DefaultXrayConfig)
	}
	var cmd = exec.Command(xray, "run", "-c", xrayConfig)
	var out = new(bytes.Buffer)
	cmd.Stdout = out
	if err := cmd.Start(); err != nil {
		return err
	}
	<-time.NewTicker(time.Second).C
	if !strings.Contains(out.String(), "started") {
		return errors.New(out.String())
	}
	common.XrayProcess = cmd.Process
	common.DefaultXrayPath = xray
	common.DefaultXrayConfig = xrayConfig
	return nil
}

func OnLoad(app *engine.Engine) error {
	common.DefaultDomain = app.Options().Domain
	return nil
}
