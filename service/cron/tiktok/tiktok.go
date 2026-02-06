package tiktok

import (
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser/bit"
	gcommon "jiayou_backend_spider/common"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/common"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/utils"
	url2 "net/url"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

const (
	PatternAddVideoComment = "pattern_add_video_comment"
	PatternVideoPublish    = "pattern_video_publish"
	PatternVideoComment    = "pattern_video_comment"
	PatternVideoDetail     = "pattern_video_detail"
	PatternVideoDiggLike   = "pattern_digg_like"
	PatternUpdateAvatar    = "pattern_update_avatar"
	PatternDeviceSync      = "pattern_device_sync"
)

func DyAddVideoComment(ctx context.Context, task *cron.Task) error {
	var params model.WebAddCommentTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = gcommon.DefaultXrayConfigMap.Load(params.ProxyName)
	lineComment, err := common.GetCommentLine(params.File)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrComment)
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p.(string)
	msTokenUrl, err := utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/explore/item_list/?"+params.Query(),
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	msToken, err := WebMsToken(msTokenUrl, reqOptions)
	if err != nil {
		return err
	}
	reqOptions.Header.SetCookieText(msToken)
	var _url string
	if params.Level == 0 {
		_url = "aid=1988&aweme_id=" + params.VideoId + "&text=" + url2.QueryEscape(lineComment) + "&text_extra=[]&" + params.TiktokWebTaskArg.Query()
	} else if params.Level == 1 {
		_url = "aid=1988&aweme_id=" + params.VideoId + "&text=" + url2.QueryEscape(lineComment) + "&text_extra=[]&reply_id=" + params.ReplyId + "&reply_to_reply_id=0&" + params.TiktokWebTaskArg.Query()
	} else {
		return model.NoRetry(nil).WithTag(model.ErrTaskCommentLevelUnSupported)
	}
	_url, err = utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/comment/publish/?"+_url,
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	err = WebAddVideoComment(_url, reqOptions)
	if err != nil {
		return nil
	}
	return task.Write(model.WebAddCommentTaskResult{VideoId: params.VideoId, ReplyText: lineComment})
}
func DyVideoPublish(ctx context.Context, task *cron.Task) error {
	var params model.WebPublicVideoTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = gcommon.DefaultXrayConfigMap.Load(params.ProxyName)
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p.(string)
	reqOptions.Timeout = time.Hour
	reqOptions.ReadTimeout = time.Hour
	reqOptions.WriteTimeout = time.Hour
	return WebVideoPublish(params.FileName, UploadVideoParams{
		Text:           params.Text,
		AllowComment:   params.AllowComment,
		ScheduleTime:   0,
		VisibilityType: VisibilityType(params.VisibilityType),
	}, reqOptions)
}
func DyVideoComment(ctx context.Context, task *cron.Task) error {
	var params model.WebCommentTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = gcommon.DefaultXrayConfigMap.Load(params.ProxyName)
	var url, err = url2.Parse(params.Url)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var lastFlagPos = strings.LastIndex(url.Path, "/")
	if lastFlagPos <= 0 {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var videoId = url.Path[lastFlagPos+1:]
	if !strings.HasPrefix(videoId, "7") {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var result model.WebCommentTaskArgResult
	result.VideoId = videoId
	if params.Level == 0 {
		return task.Write(result)
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p.(string)
	var _url = "aweme_id=" + videoId + "&count=20&cursor=0&aid=1988&" + params.Query()
	_url, err = utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/comment/list/?"+_url,
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	msTokenUrl, err := utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/explore/item_list/?"+params.Query(),
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NewBase().WithTag(model.ErrRunJs)
	}
	msToken, err := WebMsToken(msTokenUrl, reqOptions)
	if err != nil {
		return err
	}
	reqOptions.Header.SetCookieText(msToken)
	comments, err := WebVideoComment(
		_url,
		reqOptions,
	)
	if err != nil {
		return err
	}
	if params.Level == 1 {
		for _, comment := range comments {
			if comment.Nickname == params.ReplyUser {
				result.ReplyUser = comment.Nickname
				result.ReplyId = comment.CId
				return task.Write(result)
			}
		}
	}
	return model.NoRetry(nil).WithTag(model.ErrTaskCommentUserNotFound)

}
func DyVideoDetail(ctx context.Context, task *cron.Task) error {
	return nil
}
func DyVideoDiggLike(ctx context.Context, task *cron.Task) error {
	var params model.WebDiggLikeTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = gcommon.DefaultXrayConfigMap.Load(params.ProxyName)
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p.(string)
	msTokenUrl, err := utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/explore/item_list/?"+params.Query(),
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	msToken, err := WebMsToken(msTokenUrl, reqOptions)
	if err != nil {
		return err
	}
	reqOptions.Header.SetCookieText(msToken)
	var _url = "aid=1988&aweme_id=" + params.VideoId + "&cid=" + params.ReplyId + "&digg_type=1&" + params.Query()
	_url, err = utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/comment/digg/?"+_url,
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	err = WebVideoDiggLike(_url, reqOptions)
	if err != nil {
		return err
	}
	return nil
}
func DyUpdateAvatar(ctx context.Context, task *cron.Task) error {
	var params model.WebUpdateAvatar
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	if params.Avatar == "" {
		return model.NoRetry(errors.New("bad avatar url"))
	}
	var options = request.DefaultRequestOptions()
	options.Header.SetUserAgent(gcommon.DefaultUserAgent)
	options.Proxy = gcommon.DefaultProxy
	var resp, err = request.Get(params.Avatar, options)
	if err != nil {
		return err
	}
	if resp.Status() != 200 {
		return errorx.New("status error").WithField("status", resp.Status())
	}
	var data = resp.Content()
	if data == nil {
		return model.NewBadResponseError()
	}
	err = task.Write(data)
	if err != nil {
		return err
	}
	return nil
}
func DyExpiredDevice(ctx context.Context, task *cron.Task) error {
	var params model.TiktokWebTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = gcommon.DefaultXrayConfigMap.Load(params.ProxyName)
	var sidGuard = params.ReqHeaders().Cookie("sid_guard")
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p.(string)
	reqOptions.Header.SetCookie("sid_guard", sidGuard)
	var _url = "https://webcast.us.tiktok.com/webcast/room/create_info/?" + params.Query()
	_url += "&X-Gnarly=" + Encrypt(_url, "", reqOptions.Header.UserAgent())
	err := WebSync(
		_url,
		reqOptions,
	)
	if err != nil {
		var apiErr *model.ApiError
		if errors.As(err, &apiErr) {
			return task.Write(apiErr.Code == 20003)
		}
	}
	return nil
}
func DyDeviceSync(ctx context.Context, task *cron.Task) error {
	return nil
}
func TestBrowser() {
	_browser := common.GBrowser.Peek(false, time.Second*15)
	if _browser == nil {
		return
	}
	if err := bit.UpdateProxy(
		fmt.Sprintf(
			"http://%s:%d%s",
			common.GBitBrowserOptions.DebugAddr,
			common.GBitBrowserOptions.DebugPort,
			bit.ApiUpdateProxy,
		),
		[]string{_browser.ID},
		gcommon.DefaultProxy,
		request.DefaultRequestOptions(),
	); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(_browser.Name, _browser.ID)
	if err := _browser.Connect(); err != nil {
		fmt.Println(err)
		return
	}
	var page, err = _browser.DefaultPage()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = page.Load("https://www.tiktok.com")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(page.WaitLoad())
	loginEle, err := page.RodPage().Element("#top-right-action-bar-login-button")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = loginEle.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	emailEle, err := page.RodPage().ElementX("//div[text()='Use phone / email / username']")
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(time.Second)
	err = emailEle.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	userEle, err := page.RodPage().ElementX("//a[text()='Log in with email or username']")
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(time.Second)
	err = userEle.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(time.Second)
	inputUsrEle, err := page.RodPage().Element("input[placeholder='Email or username']")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = inputUsrEle.Input("stephpna5d1")
	if err != nil {
		fmt.Println(err)
		return
	}
	inputPwdEle, err := page.RodPage().Element("input[placeholder='Password']")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = inputPwdEle.Input("Aa112233@@")
	if err != nil {
		fmt.Println(err)
		return
	}
	loginBtnEle, err := page.RodPage().ElementX("//div[@id='loginModalContentContainer']//button[text()='Log in']")
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(time.Second)
	err = loginBtnEle.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	if page.Wait("()=>document.querySelector('#captcha-verify-container-main-page') != null", time.Second*5, false) == nil {
		time.Sleep(time.Second)
		captchaImgs, err := page.RodPage().Elements("img[alt='Captcha']")
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, captchaImg := range captchaImgs {
			v, err := captchaImg.Attribute("src")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(*v)
		}
	}
	if page.Wait("()=>document.querySelector('#idv-modal-container') != null", time.Second*5, false) == nil {
		emailVerifyEle, err := page.RodPage().ElementX("//div[@id='idv-modal-container']//h1/following-sibling::div[1]")
		if err != nil {
			fmt.Println(err)
			return
		}
		emailVerifyEle.WaitVisible()
		err = emailVerifyEle.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			fmt.Println(err)
			return
		}
		if page.Wait("()=>document.querySelector('#captcha-verify-container-main-page') != null", time.Second*5, false) == nil {
			time.Sleep(time.Second)
			captchaImgs, err := page.RodPage().Elements("img[alt='Captcha']")
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, captchaImg := range captchaImgs {
				v, err := captchaImg.Attribute("src")
				if err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(*v)
			}
		}
	}
}

func Register(server *cron.Server) error {
	server.HandleFunc(PatternAddVideoComment, DyAddVideoComment)
	server.HandleFunc(PatternVideoPublish, DyVideoPublish)
	server.HandleFunc(PatternVideoComment, DyVideoComment)
	server.HandleFunc(PatternVideoDetail, DyVideoDetail)
	server.HandleFunc(PatternUpdateAvatar, DyUpdateAvatar)
	server.HandleFunc(PatternVideoDiggLike, DyVideoDiggLike)
	server.HandleFunc(PatternDeviceSync, DyDeviceSync)
	return nil
}
