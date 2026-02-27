package tiktok

import (
	"jiayou_backend_spider/browser"
	"net/url"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

var loginHook = "loginHook"

func onRequest(hijack *rod.Hijack, ctx *browser.Ctx) {
	hijack.ContinueRequest(&proto.FetchContinueRequest{})
	var target = hijack.Request.URL().String()
	if strings.Contains(target, "/passport/web/account/info") {
		var cookies = hijack.Request.Header("Cookie")
		var query = hijack.Request.URL().Query()
		var queries = map[string]string{}
		var headers = map[string]string{
			"user-agent": hijack.Request.Header("User-Agent"),
		}
		for k, v := range query {
			unescape, _ := url.QueryUnescape(v[0])
			if unescape != "" {
				queries[k] = unescape
			} else {
				queries[k] = v[0]
			}
		}
		ctx.Meta.Set("cookies", cookies)
		ctx.Meta.Set("queries", queries)
		ctx.Meta.Set("headers", headers)
		ctx.Hooks.Get(loginHook).Stop()
	}
}
