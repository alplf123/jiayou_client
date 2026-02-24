package xray

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/xtls/xray-core/app/proxyman"
	handlerService "github.com/xtls/xray-core/app/proxyman/command"
	routingService "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/proxy/vless"
	vlessOutbound "github.com/xtls/xray-core/proxy/vless/outbound"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/reality"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gvisor.dev/gvisor/runsc/cmd"

	gcommon "jiayou_backend_spider/common"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/common"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/utils"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed xray.exe
var xray []byte

const DefaultXrayHost = "127.0.0.1"
const DefaultXrayPort = 9878

var Xray *Service

type Service struct {
	c        map[string]string
	pid      int
	lck      sync.Mutex
	port     int
	hsClient handlerService.HandlerServiceClient
	rsClient routingService.RoutingServiceClient
	Host     string
	Port     int
	Path     string
	Config   string
}

func (xray *Service) generateXrayConfig() error {
	var xrayJson = map[string]any{
		"api": map[string]any{
			"tag":    "api",
			"listen": fmt.Sprintf("%s:%d", xray.Host, xray.Port),
			"services": []any{
				"HandlerService",
				"RoutingService",
			}},
	}
	dat, err := json.Marshal(xrayJson)
	if err != nil {
		return fmt.Errorf("marshal xray conf,%w", err)
	}
	err = os.WriteFile(xray.Config, dat, 0644)
	if err != nil {
		return fmt.Errorf("write xray conf,%w", err)
	}
	return nil
}
func (xray *Service) kill() {
	exec.Command("taskkill", "/F", "/IM", "xray.exe").Run()
}
func (xray *Service) vlessOutBoundConfig(proxy *url.URL) *core.OutboundHandlerConfig {
	var port, _ = strconv.Atoi(proxy.Port())
	return &core.OutboundHandlerConfig{
		SenderSettings: serial.ToTypedMessage(
			&proxyman.SenderConfig{
				StreamSettings: &internet.StreamConfig{
					ProtocolName: "tcp",
					SecurityType: serial.GetMessageType(&reality.Config{}),
					SecuritySettings: []*serial.TypedMessage{serial.ToTypedMessage(&reality.Config{
						ServerName:  proxy.Query().Get("sni"),
						Fingerprint: proxy.Query().Get("fp"),
						PublicKey:   []byte(proxy.Query().Get("pbk")),
					})},
				},
			},
		),
		ProxySettings: serial.ToTypedMessage(
			&vlessOutbound.Config{
				Vnext: &protocol.ServerEndpoint{
					Address: net.NewIPOrDomain(net.ParseAddress(proxy.Hostname())),
					Port:    uint32(port),
					User: &protocol.User{
						Account: serial.ToTypedMessage(&vless.Account{
							Id:         proxy.User.Username(),
							Flow:       proxy.Query().Get("flow"),
							Encryption: proxy.Query().Get("encryption"),
						}),
					},
				},
			},
		),
	}
}
func (xray *Service) removeInbound() {}
func (xray *Service) addInbound() {
}
func (xray *Service) removeOutbound() {}
func (xray *Service) addOutbound() error {
}

func (xray *Service) Get(tag string, outbound string) (string, error) {
	xray.lck.Lock()
	defer xray.lck.Unlock()
	if p, ok := xray.c[tag]; ok {
		return p, nil
	}
	var bound, err = url.Parse(outbound)
	if err != nil {
		return "", model.NoRetry(err).WithTag(model.ErrBadProxy)
	}
	if bound.Scheme != "vless" {
		return "", model.NoRetry(nil).WithTag(model.ErrVlessOnly)
	}

}
func (xray *Service) Start() error {
	xray.lck.Lock()
	defer xray.lck.Unlock()
	xray.kill()
	if _, err := os.FindProcess(xray.pid); xray.pid == 0 || err != nil {
		var c = exec.Command(xray.Path, "run", "-c", xray.Config)
		var out = new(bytes.Buffer)
		c.Stdout = out
		if err := c.Start(); err != nil {
			return err
		}
		<-time.NewTicker(time.Second).C
		if !strings.Contains(out.String(), "started") {
			return errors.New(out.String())
		}
		xray.pid = c.Process.Pid
		cmdConn, err := grpc.NewClient(fmt.Sprintf("%s:%d", xray.Host, xray.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		xray.rsClient = routingService.NewRoutingServiceClient(cmdConn)
		xray.hsClient = handlerService.NewHandlerServiceClient(cmdConn)
	}
	return nil
}

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
	var resp, err = request.Post(gcommon.DefaultDomain+"/api/proxy", nil, header)
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

	var xray, xrayConfig string
	//p = ""
	if p == "" {
		xray = "./" + common.DefaultXrayPath
		xrayConfig = "./" + common.DefaultXrayConfig
	} else {
		xray = filepath.Join(filepath.Dir(p), common.DefaultXrayPath)
		xrayConfig = filepath.Join(filepath.Dir(p), common.DefaultXrayConfig)
	}

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
	var xrayPath = filepath.Join(utils.MustGetDefaultHomeDir("xray"), common.DefaultXrayPath)
	_, err := os.Stat(xrayPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.WriteFile(xrayPath, xray, 0644); err != nil {
				common.GLogger.Error("write xray failed", zap.String("xray", xrayPath), zap.Error(err))
				return err
			}
		} else {
			common.GLogger.Error("stat ffmpeg path failed", zap.String("xray", xrayPath), zap.Error(err))
			return err
		}
	}
	common.GLogger.Info("xray loaded", zap.String("xray", xrayPath))
	common.DefaultXrayPath, Xray.Path = xrayPath, xrayPath
	Xray.Config = filepath.Join(utils.MustGetDefaultHomeDir("xray"), common.DefaultXrayConfigPath)
	return nil
}

func init() {
	Xray = &Service{
		c:      make(map[string]string),
		port:   9900,
		Host:   DefaultXrayHost,
		Port:   DefaultXrayPort,
		Path:   common.DefaultXrayPath,
		Config: common.DefaultXrayConfigPath,
	}
}
