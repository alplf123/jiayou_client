package xray

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/engine"
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

	"github.com/xtls/xray-core/app/proxyman"
	handlerService "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/app/router"
	routingService "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	httpInbound "github.com/xtls/xray-core/proxy/http"
	"github.com/xtls/xray-core/proxy/vless"
	vlessOutbound "github.com/xtls/xray-core/proxy/vless/outbound"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/reality"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed xray.exe
var xray []byte

const DefaultXrayHost = "127.0.0.1"
const DefaultXrayPort = 8080
const DefaultXrayUser = "jiayou"
const DefaultXrayPwd = "jiayou"
const DefaultInboundPort = 9900
const DefaultInboundHost = "127.0.0.1"

var Xray *Service

type Service struct {
	c           map[string]string
	pid         int
	lck         sync.Mutex
	user, pwd   string
	inBoundHost string
	inBoundPort int
	hsClient    handlerService.HandlerServiceClient
	rsClient    routingService.RoutingServiceClient
	Host        string
	Port        int
	Path        string
	Config      string
}

func (xray *Service) generateXrayConfig() error {
	var xrayJson = map[string]any{
		"api": map[string]any{
			"tag":    "api",
			"listen": fmt.Sprintf("%s:%d", xray.Host, xray.Port),
			"services": []any{
				"HandlerService",
				"RoutingService",
				"StatsService",
				"LoggerService",
			}},
		"routing": map[string]any{
			"rules": []any{},
		},
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
func (xray *Service) vlessOutBoundConfig(tag string, proxy *url.URL) *core.OutboundHandlerConfig {
	var port, _ = strconv.Atoi(proxy.Port())
	var pbk, _ = base64.RawURLEncoding.DecodeString(proxy.Query().Get("pbk"))
	return &core.OutboundHandlerConfig{
		Tag: tag,
		SenderSettings: serial.ToTypedMessage(
			&proxyman.SenderConfig{
				StreamSettings: &internet.StreamConfig{
					ProtocolName: proxy.Query().Get("type"),
					SecurityType: serial.GetMessageType(&reality.Config{}),
					SecuritySettings: []*serial.TypedMessage{serial.ToTypedMessage(&reality.Config{
						ServerName:  proxy.Query().Get("sni"),
						Fingerprint: proxy.Query().Get("fp"),
						PublicKey:   pbk,
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
func (xray *Service) removeInbound(tag string) error {
	_, err := xray.hsClient.RemoveInbound(context.Background(), &handlerService.RemoveInboundRequest{Tag: tag})
	return err
}
func (xray *Service) addInbound(tag string) error {
	var httpInBoundConf = &core.InboundHandlerConfig{
		Tag: tag,
		ReceiverSettings: serial.ToTypedMessage(
			&proxyman.ReceiverConfig{
				Listen:   net.NewIPOrDomain(net.ParseAddress(xray.inBoundHost)),
				PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(net.Port(xray.inBoundPort))}},
				SniffingSettings: &proxyman.SniffingConfig{
					Enabled:             true,
					DestinationOverride: []string{"http", "tls"},
				},
			},
		),
		ProxySettings: serial.ToTypedMessage(
			&httpInbound.ServerConfig{
				Accounts: map[string]string{
					xray.user: xray.pwd,
				},
			}),
	}
	r, err := xray.hsClient.AddInbound(context.Background(), &handlerService.AddInboundRequest{Inbound: httpInBoundConf})
	fmt.Println(r, err)
	return err
}
func (xray *Service) removeOutbound(tag string) error {
	_, err := xray.hsClient.RemoveOutbound(context.Background(), &handlerService.RemoveOutboundRequest{Tag: tag})
	return err
}
func (xray *Service) addOutbound(tag string, vless *url.URL) error {
	r, err := xray.hsClient.AddOutbound(
		context.Background(),
		&handlerService.AddOutboundRequest{Outbound: xray.vlessOutBoundConfig(tag, vless)},
	)
	fmt.Println(r, err)
	return err
}
func (xray *Service) addRule(ruleTag, inTag, outTag string) error {
	r, err := xray.rsClient.AddRule(context.Background(),
		&routingService.AddRuleRequest{
			Config: serial.ToTypedMessage(&router.Config{
				DomainStrategy: router.Config_AsIs,
				Rule: []*router.RoutingRule{
					{
						RuleTag:    ruleTag,
						InboundTag: []string{inTag},
						TargetTag:  &router.RoutingRule_Tag{Tag: outTag},
					},
				}}),
			ShouldAppend: false,
		})
	fmt.Println(r, err)
	return err
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
	var inBoundTag = tag + "_inbound"
	var outBoundTag = tag + "_outbound"
	var ruleTag = tag + "_rule"
	if err := xray.addOutbound(outBoundTag, bound); err != nil {
		return "", model.NoRetry(nil).WithTag(model.ErrXrayOutBound)
	}
	if err := xray.addInbound(inBoundTag); err != nil {
		if err1 := xray.removeOutbound(outBoundTag); err1 != nil {
			common.GLogger.Error("remove outbound", zap.String("tag", tag), zap.Error(err1))
		}
		return "", model.NoRetry(err).WithTag(model.ErrXrayInBound)
	}
	if err := xray.addRule(ruleTag, inBoundTag, outBoundTag); err != nil {
		if err1 := xray.removeOutbound(outBoundTag); err1 != nil {
			common.GLogger.Error("remove outbound", zap.String("tag", tag), zap.Error(err1))
		}
		if err2 := xray.removeInbound(inBoundTag); err2 != nil {
			common.GLogger.Error("remove inbound", zap.String("tag", tag), zap.Error(err2))
		}
		return "", model.NoRetry(err).WithTag(model.ErrXrayRule)
	}
	xray.c[tag] = fmt.Sprintf("http://%s:%d", xray.inBoundHost, xray.inBoundPort)
	xray.inBoundPort++
	return xray.c[tag], nil

}
func (xray *Service) Start() error {
	xray.lck.Lock()
	defer xray.lck.Unlock()
	xray.kill()
	if _, err := os.FindProcess(xray.pid); xray.pid == 0 || err != nil {
		if err := xray.generateXrayConfig(); err != nil {
			return err
		}
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
		c:           make(map[string]string),
		user:        DefaultXrayUser,
		pwd:         DefaultXrayPwd,
		inBoundHost: DefaultInboundHost,
		inBoundPort: DefaultInboundPort,
		Host:        DefaultXrayHost,
		Port:        DefaultXrayPort,
		Path:        common.DefaultXrayPath,
		Config:      common.DefaultXrayConfigPath,
	}
}
