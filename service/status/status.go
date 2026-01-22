package status

import "fmt"

type Code int

type Response struct {
	error   `json:",inline"`
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Result  any    `json:"result,omitempty"`
}

func (response *Response) Error() string {
	if response.error != nil {
		return fmt.Sprintf("Response[Code:%d Message:%s],wrapped err:%s", response.Code, response.Message, response.error)
	}
	return fmt.Sprintf("Response[Code:%d Message:%s]", response.Code, response.Message)
}
func (response *Response) WrapData(data any) *Response {
	if response.Result == nil {
		var copyResp = *response
		copyResp.Result = data
		return &copyResp
	}
	response.Result = data
	return response
}
func (response *Response) WrapError(err error) *Response {
	if response.Result == nil {
		var copyResp = *response
		copyResp.error = err
		return &copyResp
	}
	response.error = err
	return response
}

func (response *Response) Unwrap() error {
	return response.error
}

var (
	Success               = New(0, "ok")
	ErrXrayNotFound       = New(20001, "xray不存在")
	ErrXray               = New(20002, "xray服务异常")
	ErrXrayConfig         = New(20003, "xray配置异常")
	ErrXrayConfigNotFound = New(20003, "xray配置不存在")
	ErrXrayFile           = New(20004, "xray文件异常")
)

func New(code Code, Message string) *Response {
	return &Response{Code: code, Message: Message}
}
