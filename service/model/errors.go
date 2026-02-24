package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/option"
	"net"
	"net/http"
	"strings"
)

var ErrTaskCanceled = "主动取消"
var ErrTaskSystem = "系统异常"
var ErrTaskArgs = "任务参数错误"
var ErrTaskBadVideoUrl = "无效的视频链接"
var ErrTaskCommentUserNotFound = "未找到用户"
var ErrTaskCommentFile = "话术文件异常"
var ErrTaskCommentLevelUnSupported = "评论层级不支持"
var ErrApi = "接口异常"
var ErrStatus = "状态码异常"
var ErrBadResponse = "响应异常"
var ErrRunJs = "运行签名异常"
var ErrNeedProxy = "缺少代理"
var ErrNet = "网络异常"
var ErrFfmpeg = "FFmpeg异常"
var ErrVideoFile = "视频文件异常"
var ErrVideoSoSmall = "视频文件太小了"
var ErrXray = "xray服务异常"
var ErrLoginExpired = "登录过期"
var ErrCommentInvalidLine = "未发现评论行"
var ErrComment = "评论文件异常"
var ErrLoadProxy = "加载代理失败"
var ErrProxyNotConfig = "代理未配置"
var ErrBadVless = "含有无效的vless链接"
var ErrBadProxy = "无效的代理链接"
var ErrVlessOnly = "只支持vless代理"
var ErrProxyNotBound = "代理未绑定，请重启客户端"
var ErrBrowserPeekTimeout = "指纹浏览器加载超时"
var ErrBrowserLoadPage = "指纹浏览器加载页面失败"
var ErrBrowserLoadUrl = "指纹浏览器加载链接失败"
var ErrBrowserWaitLoad = "指纹浏览器等待页面完成加载失败"
var ErrBrowserConnect = "指纹浏览器连接失败"
var ErrBrowserClickLoginButton = "指纹浏览器点击登录失败"
var ErrBrowserClickEmailButton = "指纹浏览器选择邮件登录方式失败"
var ErrBrowserSwitchEmailButton = "指纹浏览器切换邮件登录方式失败"
var ErrBrowserInputUser = "指纹浏览器输入账号失败"
var ErrBrowserInputPwd = "指纹浏览器输入密码失败"
var ErrBrowserLogin = "指纹浏览器登录失败"
var ErrBrowserProcessVerify = "指纹浏览器处理过程失败"
var ErrBrowserUserDataNotFound = "指纹浏览器未发现用户信息"
var ErrBrowserHookUserInfoTimeout = "指纹浏览器拦截用户消息超时"
var ErrBrowserClickSendEmail = "指纹浏览器点击发送邮件失败"
var ErrBrowserInputEmailCode = "指纹浏览器输入邮箱验证码失败"
var ErrBrowserInput2FACode = "指纹浏览器输入2fa验证码失败"
var ErrBrowserClickEmailNext = "指纹浏览器点击邮箱下一步失败"
var ErrBrowserClick2FANext = "指纹浏览器点击2fa下一步失败"
var ErrBrowserUpdateProxy = "指纹浏览器更新代理失败"
var ErrEmailReadCode = "邮件读取验证码失败"
var ErrProcessCaptcha = "验证码处理失败"
var ErrAccountMaximium = "账号登录次数过多"
var ErrTwoFA = "2fa计算失败"

type ErrBase struct {
	tag    string
	err    error
	retry  bool
	fields *option.Option
}

func (b *ErrBase) Tag() string {
	return b.tag
}
func (b *ErrBase) WithTag(tag string) *ErrBase {
	b.tag = tag
	return b
}
func (b *ErrBase) Error() string {
	var err string
	if b.err != nil {
		err = b.err.Error()
	}
	var r, _ = json.Marshal(map[string]any{
		"tag":    b.Tag(),
		"error":  err,
		"fields": b.fields.Raw(),
	})
	return fmt.Sprintf("ErrBase:%s", string(r))
}
func (b *ErrBase) RawError() error {
	return b.err
}
func (b *ErrBase) WithError(err error) *ErrBase {
	var e *ErrBase
	if errors.As(err, &e) {
		*b = *e
		b.fields = e.fields.Clone()
	} else {
		b.err = err
	}
	return b
}
func (b *ErrBase) WithRetry(retry bool) *ErrBase {
	b.retry = retry
	return b
}
func (b *ErrBase) Option() *option.Option {
	return b.fields
}
func (b *ErrBase) ShouldRetry() bool {
	return b.retry
}

type IRetry interface {
	ShouldRetry() bool
}

type ApiError struct {
	*ErrBase
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *ApiError) WithCode(code int) *ApiError {
	err.Code = code
	err.fields.Set("code", code)
	return err
}
func (err *ApiError) WithMessage(message string) *ApiError {
	err.Message = message
	err.fields.Set("message", message)
	return err
}

type NetError struct {
	*ErrBase
}

func (netErr *NetError) ShouldRetry(e error) bool {
	var v *net.OpError
	if errors.As(e, &v) {
		if v.Timeout() || v.Temporary() {
			return true
		}
	}
	return false
}

type StatusError struct {
	*ErrBase
	Status int `json:"code"`
}

func (err *StatusError) WithStatus(status int) *StatusError {
	err.Status = status
	err.fields.Set("status", status)
	return err
}
func (err *StatusError) ShouldRetry() bool {
	return err.Status == http.StatusInternalServerError ||
		err.Status == http.StatusBadGateway ||
		err.Status == http.StatusServiceUnavailable ||
		err.Status == http.StatusGatewayTimeout
}

func Retry(err error) *ErrBase {
	return NewBase().WithRetry(true).WithError(err)
}
func NoRetry(err error) *ErrBase {
	return NewBase().WithError(err)
}
func IsErrBase(err error) bool {
	var errBase *ErrBase
	if errors.As(err, &errBase) {
		return true
	}
	if strings.HasPrefix(err.Error(), "ErrBase:") {
		return true
	}
	return false
}
func ParseErrBase(err string) (*ErrBase, bool) {
	if IsErrBase(errors.New(err)) {
		err = strings.Replace(err, "ErrBase:", "", 1)
		var r, err = option.FromJson(err)
		if err != nil {
			return nil, false
		}
		var tag, _ = r.AsString("tag")
		var e, _ = r.AsString("error")
		var fields, _ = r.AsMap("fields")
		return &ErrBase{tag: tag, err: errors.New(e), fields: option.New(fields)}, true
	}
	return nil, false
}
func NewBase() *ErrBase {
	return &ErrBase{fields: option.New(nil)}
}
func NewApiError() *ApiError {
	return &ApiError{
		ErrBase: NewBase(),
	}
}
func NewStatusError(code int) *StatusError {
	return (&StatusError{
		ErrBase: NewBase().WithTag(ErrStatus),
	}).WithStatus(code)
}
func NewBadResponseError() *ErrBase {
	return NewBase().WithTag(ErrBadResponse)
}
func NewNetError() *NetError {
	return &NetError{
		ErrBase: NewBase().WithTag(ErrNet),
	}
}
