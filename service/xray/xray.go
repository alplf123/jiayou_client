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

func generateXrayConfig(proxies map[string]string) error {

	var xrayVLess = map[string]any{}
	var inBounds = make([]any, 0)
	var outBounds = make([]any, 0)
	var rules = make([]any, 0)
	var startPort = common.DefaultXrayPort
	for k, v := range proxies {
		var vLessUrl, err = url.Parse(v)
		if err != nil {
			return model.NewBase().WithTag(model.ErrBadVless).WithError(err)
		}
		startPort++
		var vLessInBounds = map[string]any{
			"tag":      k,
			"port":     startPort,
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
		}
		var vLessOutBounds = map[string]any{}
		var port, _ = strconv.Atoi(vLessUrl.Port())
		vLessOutBounds["tag"] = k
		vLessOutBounds["protocol"] = "vless"
		vLessOutBounds["settings"] = map[string]any{
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
		vLessOutBounds["streamSettings"] = map[string]any{
			"network":  vLessUrl.Query().Get("type"),
			"security": vLessUrl.Query().Get("security"),
			"realitySettings": map[string]any{
				"serverName":  vLessUrl.Query().Get("sni"),
				"fingerprint": vLessUrl.Query().Get("fp"),
				"publicKey":   vLessUrl.Query().Get("pbk"),
			},
		}
		var rule = map[string]any{
			"type":        "field",
			"inboundTag":  []any{k},
			"outboundTag": k,
		}
		outBounds = append(outBounds, vLessOutBounds)
		inBounds = append(inBounds, vLessInBounds)
		rules = append(rules, rule)
		common.DefaultXrayConfigMap.Store(k, fmt.Sprintf("http://%s:%s@%s:%d", common.DefaultXrayUser, common.DefaultXrayPwd, common.DefaultXrayHost, startPort))
	}
	xrayVLess["outbounds"] = outBounds
	xrayVLess["inbounds"] = inBounds
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
func findActiveProxy() (map[string]string, error) {
	var header = request.DefaultRequestOptions()
	header.Header.Set("Authorization", common.GlobalToken)
	var resp, err = request.Post(common.DefaultDomain+"/api/proxy", nil, header)
	if err != nil {
		return nil, model.NewBase().WithTag(model.ErrLoadProxy).WithError(err)
	}
	if resp.Status() != 200 {
		return nil, model.NewStatusError(resp.Status())
	}
	var j = gjson.Parse(resp.Text())
	var code = j.Get("code").Int()
	if code != 0 {
		return nil, model.NewApiError().WithCode(int(code)).WithMessage(j.Get("message").String())
	}
	var r = make(map[string]string)
	for _, item := range j.Get("result.items").Array() {
		var k, v = item.Get("name").String(), item.Get("proxy").String()
		r[k] = v
	}
	return r, nil
}
func StartXray() error {
	exec.Command("taskkill", "/F", "/IM", "xray.exe").Run()
	proxies, err := findActiveProxy()
	if err != nil {
		return err
	}
	if proxies == nil {
		return model.NewBase().WithTag(model.ErrProxyNotConfig)
	}
	err = generateXrayConfig(proxies)
	if err != nil {
		return err
	}
	var p, _ = os.Executable()
	var xray, xrayConfig string
	//p = ""
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
	common.DefaultLogger, _ = app.Log()
	exec.Command("taskkill", "/F", "/IM", "xray.exe").Run()
	return nil
}
