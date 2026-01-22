package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/option"
	"net"
	"strings"
)

var ErrTaskCanceled = "主动取消"
var ErrTaskSystem = "系统异常"
var ErrTaskArgs = "任务参数错误"
var ErrTaskBadVideoUrl = "无效的视频链接"
var ErrTaskCommentUserNotFound = "未找到评论用户"
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
	b.err = err
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

type Retry interface {
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

func (err *NetError) WithError(e error) *NetError {
	var v *net.OpError
	if errors.As(e, &v) {
		if v.Timeout() || v.Temporary() {
			err.retry = true
		}
	}
	_ = err.ErrBase.WithError(e)
	return err
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

func NoRetry(err error) *ErrBase {
	return NewBase().WithRetry(false).WithError(err)
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
	return &ErrBase{fields: option.New(nil), retry: true}
}
func NewSystemError() *ErrBase {
	return NewBase().WithTag(ErrTaskSystem)
}
func NewApiError() *ApiError {
	var apiErr = &ApiError{
		ErrBase: NewBase(),
	}
	apiErr.WithTag(ErrApi)
	return apiErr
}
func NewStatusError(code int) *StatusError {
	var statusErr = &StatusError{
		ErrBase: NewBase(),
	}
	statusErr.WithTag(ErrStatus)
	statusErr.WithStatus(code)
	return statusErr
}
func NewBadResponseError() *ErrBase {
	return NewBase().WithTag(ErrBadResponse)
}
func NewNetError() *NetError {
	var netErr = &NetError{
		ErrBase: NewBase(),
	}
	netErr.WithTag(ErrNet)
	return netErr
}
