package tiktok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/api"
	"jiayou_backend_spider/browser"
	"jiayou_backend_spider/browser/bit"
	gcommon "jiayou_backend_spider/common"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/common"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/service/xray"
	"jiayou_backend_spider/utils"
	"math/rand/v2"
	url2 "net/url"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/gjson"
)

const (
	PatternVideoComment    = "pattern_add_video_comment"
	PatternVideoPublish    = "pattern_video_publish"
	PatternVideoDetail     = "pattern_video_detail"
	PatternVideoDiggLike   = "pattern_digg_like"
	PatternUpdateAvatar    = "pattern_update_avatar"
	PatternDeviceSync      = "pattern_device_sync"
	PatternDeviceHeartbeat = "pattern_device_heartbeat"
)

func fetchToken(query string, reqOpts *request.Options) (string, error) {
	msTokenUrl, err := utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/explore/item_list/?"+query,
		reqOpts.Header.UserAgent(),
	)
	if err != nil {
		return "", model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	msToken, err := WebMsToken(msTokenUrl, reqOpts)
	if err != nil {
		return "", err
	}
	return msToken, nil
}

func apiDeviceSync(info map[string]any) error {
	resp, err := request.PostJson(common.DefaultDomain+"/api/tiktok/devices/sync", info, request.DefaultRequestOptions())
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	if resp.Status() != 200 {
		return model.NewStatusError(resp.Status())
	}
	var data = gjson.Parse(resp.Text())
	var code = data.Get("code").Int()
	if code != 0 {
		return model.NewApiError().WithCode(int(code)).WithMessage(data.Get("message").String())
	}
	return nil
}
func fetchCommentReplyUserId(url string, reply string, level int, params model.TiktokWebTaskArg, reqOptions *request.Options) (string, error) {
	videoId, err := fetchVideoId(url)
	if err != nil {
		return "", err
	}
	var _url = "aweme_id=" + videoId + "&count=20&cursor=0&aid=1988&" + params.Query()
	_url, err = utils.RunJsCtx(
		context.Background(),
		webmssdk,
		"encrypt",
		"https://www.tiktok.com/api/comment/list/?"+_url,
		reqOptions.Header.UserAgent(),
	)
	if err != nil {
		return "", model.NoRetry(err).WithTag(model.ErrRunJs)
	}
	msToken, err := fetchToken(params.Query(), reqOptions)
	if err != nil {
		return "", err
	}
	reqOptions.Header.SetCookieText(msToken)
	comments, err := WebVideoComment(
		_url,
		reqOptions,
	)
	if err != nil {
		return "", err
	}
	if level == 1 {
		for _, comment := range comments {
			if comment.Nickname == reply {
				return comment.CId, nil
			}
		}
	}
	return "", model.NoRetry(nil).WithTag(model.ErrTaskCommentUserNotFound)

}

func fetchVideoId(_url string) (string, error) {
	var url, err = url2.Parse(_url)
	if err != nil {
		return "", model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var lastFlagPos = strings.LastIndex(url.Path, "/")
	if lastFlagPos <= 0 {
		return "", model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var videoId = url.Path[lastFlagPos+1:]
	if !strings.HasPrefix(videoId, "7") {
		return "", model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	return videoId, nil
}

func DyAddVideoComment(ctx context.Context, task *cron.Task) error {
	var params model.WebAddCommentTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = xray.Xray.Get(params.ProxyName, params.ProxyValue)
	lineComment, err := common.GetCommentLine(params.File)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrComment)
	}
	videoId, err := fetchVideoId(params.Url)
	if err != nil {
		return err
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p
	msToken, err := fetchToken(params.Query(), reqOptions)
	if err != nil {
		return err
	}
	reqOptions.Header.SetCookieText(msToken)
	replyUserId, err := fetchCommentReplyUserId(params.Url, params.ReplyUser, params.Level, params.TiktokWebTaskArg, reqOptions)
	if err != nil {
		return err
	}
	var _url string
	if params.Level == 0 {
		_url = "aid=1988&aweme_id=" + videoId + "&text=" + url2.QueryEscape(lineComment) + "&text_extra=[]&" + params.TiktokWebTaskArg.Query()
	} else if params.Level == 1 {
		_url = "aid=1988&aweme_id=" + videoId + "&text=" + url2.QueryEscape(lineComment) + "&text_extra=[]&reply_id=" + replyUserId + "&reply_to_reply_id=0&" + params.TiktokWebTaskArg.Query()
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
	return task.Write(model.WebAddCommentTaskResult{VideoId: videoId, ReplyText: lineComment})
}
func DyVideoPublish(ctx context.Context, task *cron.Task) error {
	var params model.WebPublicVideoTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = xray.Xray.Get(params.ProxyName, params.ProxyValue)
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p
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
func DyVideoDetail(ctx context.Context, task *cron.Task) error {
	return nil
}
func DyVideoDiggLike(ctx context.Context, task *cron.Task) error {
	var params model.WebDiggLikeTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = xray.Xray.Get(params.ProxyName, params.ProxyValue)
	videoId, err := fetchVideoId(params.Url)
	if err != nil {
		return err
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p
	msToken, err := fetchToken(params.Query(), reqOptions)
	if err != nil {
		return err
	}
	reqOptions.Header.SetCookieText(msToken)
	replyUserId, err := fetchCommentReplyUserId(params.Url, params.ReplyUser, 1, params.TiktokWebTaskArg, reqOptions)
	if err != nil {
		return err
	}
	var _url = "aid=1988&aweme_id=" + videoId + "&cid=" + replyUserId + "&digg_type=1&" + params.Query()
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
	var p, _ = xray.Xray.Get(params.ProxyName, params.ProxyValue)
	options.Header.SetUserAgent(gcommon.DefaultUserAgent)
	options.Proxy = p
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
func DyDeviceHeartbeat(ctx context.Context, task *cron.Task) error {
	var params model.TiktokWebTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var p, _ = xray.Xray.Get(params.ProxyName, params.ProxyValue)
	var sidGuard = params.ReqHeaders().Cookie("sid_guard")
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = p
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
	var taskArg model.TaskArg
	if err := task.Payload().As(&taskArg); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var twoFa model.BindDeviceVerify2FA
	var email model.BindDeviceVerifyEmail
	var outlook model.BindEmailArgsOutlookOauth
	switch taskArg.VerifyType {
	case model.Email:
		json.Unmarshal([]byte(taskArg.VerifyArgs), &email)
		if email.Type == model.OutLook {
			utils.MS(email.Args, &outlook, nil)
		}
	case model.TwoFA:
		json.Unmarshal([]byte(taskArg.VerifyArgs), &twoFa)
	}
	var p, _ = xray.Xray.Get(taskArg.ProxyName, taskArg.ProxyValue)
	_browser := common.GBrowser.Peek(false, common.DefaultBrowserTryPeekTimeout)
	if _browser == nil {
		return model.Retry(nil).WithTag(model.ErrBrowserPeekTimeout)
	}
	defer func() {
		_browser.Reset()
	}()
	if err := bit.UpdateProxy(
		fmt.Sprintf(
			"http://%s:%d%s",
			common.GBitBrowserOptions.DebugAddr,
			common.GBitBrowserOptions.DebugPort,
			bit.ApiUpdateProxy,
		),
		[]string{_browser.ID},
		p,
		request.DefaultRequestOptions(),
	); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserUpdateProxy)
	}
	_browser.AddInterceptor("", "")
	_browser.OnRequest(onRequest)
	var loginHook = _browser.Hook(loginHook)
	if err := _browser.Connect(); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserConnect)
	}
	var page, err = _browser.Blank()
	if err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserLoadPage)
	}
	page.Timeout(common.DefaultBrowserPageTimeout)
	err = page.Load("https://www.tiktok.com")
	if err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserLoadUrl)
	}
	err = page.WaitLoad()
	if err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserWaitLoad)
	}
	var processLoginButton = func() error {
		loginEle, err := page.RodPage().Element("#top-right-action-bar-login-button")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return loginEle.Click(proto.InputMouseButtonLeft, 1)
	}
	var processSelectLoginEmail = func() error {
		emailEle, err := page.RodPage().ElementsX("//div[text()='Use phone / email / username']")
		if err != nil {
			return err
		}
		emailEle1, err := page.RodPage().ElementsX("//div[text()='Use phone or email']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		if !emailEle.Empty() {
			return emailEle.First().Click(proto.InputMouseButtonLeft, 1)
		} else if !emailEle1.Empty() {
			return emailEle1.First().Click(proto.InputMouseButtonLeft, 1)
		}
		return nil
	}
	var processSwitchLoginEmail = func() error {
		userEle, err := page.RodPage().ElementsX("//a[text()='Log in with email or username']")
		if err != nil {
			return err
		}
		userEle1, err := page.RodPage().ElementsX("//a[text()='Use email or username']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		if !userEle.Empty() {
			return userEle.First().Click(proto.InputMouseButtonLeft, 1)
		} else if !userEle1.Empty() {
			return userEle1.First().Click(proto.InputMouseButtonLeft, 1)
		}
		return nil
	}
	var processInputUser = func(userName string) error {
		inputUsrEle, err := page.RodPage().Element("input[placeholder='Email or username']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return inputUsrEle.Input(userName)
	}
	var processInputPassword = func(password string) error {
		inputPwdEle, err := page.RodPage().Element("input[placeholder='Password']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return inputPwdEle.Input(password)
	}
	var processClickLoginButton = func() error {
		loginBtnEle, err := page.RodPage().ElementX("//div[@id='loginModalContentContainer']//button[text()='Log in']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return loginBtnEle.Click(proto.InputMouseButtonLeft, 1)
	}
	var processCaptcha = func() error {
		for i := 0; i < 5; i++ {
			captchaImgs, err := page.RodPage().Elements("img[alt='Captcha']")
			if err != nil {
				return err
			}
			if len(captchaImgs) < 2 {
				time.Sleep(time.Second)
				continue
			}
			for _, captchaImg := range captchaImgs {
				if err := captchaImg.WaitVisible(); err != nil {
					return err
				}
			}
			outImg, _ := captchaImgs[0].Attribute("src")
			innerImg, _ := captchaImgs[1].Attribute("src")
			if outImg == nil || innerImg == nil {
				return errors.New("bad captcha image")
			}
			out := (*outImg)[23:]
			inner := (*innerImg)[23:]
			_, slide, err := api.GetRotate(out, inner)
			if err != nil {
				return err
			}
			slideEle, err := page.RodPage().ElementX("//button[@id='captcha_slide_button']/parent::div")
			if err != nil {
				return err
			}
			shape, err := slideEle.Shape()
			if err != nil {
				return err
			}
			slide = slide + 36 // add base offset
			var slidePage = slideEle.Page()
			if err := slidePage.Mouse.MoveTo(proto.Point{X: shape.Box().X + shape.Box().Width/2, Y: shape.Box().Y + shape.Box().Height/2}); err != nil {
				return err
			}
			if err := slidePage.Mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
				return err
			}
			for i := 0; i < slide; i += 5 {
				if err := slidePage.Mouse.MoveLinear(proto.Point{X: shape.Box().X + float64(i), Y: shape.Box().Y}, 1); err != nil {
					return err
				}
				time.Sleep(time.Millisecond * time.Duration(10+rand.IntN(50)))
			}
			if err := slidePage.Mouse.MoveLinear(proto.Point{X: shape.Box().X + float64(slide), Y: shape.Box().Y}, 1); err != nil {
				return err
			}
			time.Sleep(time.Second)
			if err := slidePage.Mouse.Up(proto.InputMouseButtonLeft, 1); err != nil {
				return err
			}
			time.Sleep(time.Second * 5)
			maximumAttemptsErr, _ := page.RodPage().ElementsX("//span[text()='Maximum number of attempts reached. Try again later.']")
			if !maximumAttemptsErr.Empty() {
				return model.NoRetry(nil).WithTag(model.ErrAccountMaximium)
			}
			captchaImgs, _ = page.RodPage().Elements("img[alt='Captcha']")
			if len(captchaImgs) > 0 {
				continue
			}
			return nil
		}
		return errors.New("captcha verify out of max retied")
	}
	var processReadEmailCode = func(email, deviceId, token string, start time.Time) (string, error) {
		body := make(url2.Values)
		body.Set("client_id", deviceId)
		body.Set("grant_type", "refresh_token")
		body.Set("refresh_token", token)
		body.Set("scope", "https://outlook.office.com/IMAP.AccessAsUser.All offline_access")
		token, err := api.SyncOutlookToken(body)
		if err != nil {
			return "", err
		}
		var ticker = time.NewTicker(time.Second * 30)
		var code string
		for {
			select {
			case <-ticker.C:
				return "", errors.New("read email code timeout")
			default:
				codes, err := api.ReadTiktokCodeFromOutlook(
					email, token, start)
				if err != nil {
					return "", err
				}
				if len(codes) > 0 {
					code = codes[0]
				}
			}
			if code != "" {
				break
			}
			time.Sleep(time.Second)
		}
		return code, nil
	}
	var processDigitCode = func(code string) error {
		verifyCodeEle, err := page.RodPage().Element("input[placeholder='Enter 6-digit code']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return verifyCodeEle.Input(code)
	}
	var processClickEmailNextButton = func() error {
		nextEle, err := page.RodPage().ElementX("//div[text()='Next']/parent::div/parent::div/parent::button")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return nextEle.Click(proto.InputMouseButtonLeft, 1)
	}
	var processClick2FANextButton = func() error {
		nextEle, err := page.RodPage().ElementX("//button[text()='Next']")
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return nextEle.Click(proto.InputMouseButtonLeft, 1)
	}
	var processClickSendEmail = func() error {
		emailVerifyEle, err := page.RodPage().ElementX("//div[@id='idv-modal-container']//h1/following-sibling::div[1]")
		if err != nil {
			return err
		}
		err = emailVerifyEle.WaitVisible()
		if err != nil {
			return err
		}
		defer time.Sleep(time.Second)
		return emailVerifyEle.Click(proto.InputMouseButtonLeft, 1)
	}
	var processVerify = func() error {
		if err := page.Chains().
			//verify email
			XPath("//h1[text()='Verify it’s really you']", func(chain *browser.Chain) error {
				var element = chain.Elements().First()
				if err := element.WaitVisible(); err != nil {
					return err
				}
				var sendStart = time.Now().UTC()
				if err := processClickSendEmail(); err != nil {
					return model.NoRetry(err).WithTag(model.ErrBrowserClickSendEmail)
				}
				if err := page.Chains().
					//verify captcha
					XPath("//span[text()='Drag the slider to fit the puzzle']", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						if err := processCaptcha(); err != nil {
							return model.Retry(nil).WithTag(model.ErrProcessCaptcha).WithError(err)
						}
						chain.ForwardNext()
						return nil
					}).
					Selector(".pc-password-resend-timer-fpxreE", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						code, err := processReadEmailCode(
							email.Email,
							outlook.DeviceId,
							outlook.AccessToken,
							sendStart,
						)
						if err != nil {
							return model.Retry(err).WithTag(model.ErrEmailReadCode)
						}
						err = processDigitCode(code)
						if err != nil {
							return model.Retry(err).WithTag(model.ErrBrowserInputEmailCode)
						}
						err = processClickEmailNextButton()
						return model.Retry(err).WithTag(model.ErrBrowserClickEmailNext)
					}).WaitAny(); err != nil {
					return err
				}
				if err := page.Chains().
					//verify captcha
					XPath("//span[text()='Drag the slider to fit the puzzle']", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						if err := processCaptcha(); err != nil {
							return model.Retry(nil).WithTag(model.ErrProcessCaptcha).WithError(err)
						}
						chain.ForwardNext()
						return nil
					}).
					Selector(".TUXButton-iconContainer img", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						return nil
					}).WaitAny(); err != nil {
					return err
				}
				return nil
			}).
			//verify 2fa
			XPath("//div[text()='2-step verification']", func(chain *browser.Chain) error {
				var element = chain.Elements().First()
				if err := element.WaitVisible(); err != nil {
					return err
				}
				code, err := api.TwoFA(twoFa.Secret)
				if err != nil {
					return model.Retry(err).WithTag(model.ErrTwoFA)
				}
				err = processDigitCode(code)
				if err != nil {
					return model.Retry(err).WithTag(model.ErrBrowserInput2FACode)
				}
				err = processClick2FANextButton()
				if err != nil {
					return model.Retry(err).WithTag(model.ErrBrowserClick2FANext)
				}
				if err := page.Chains().
					//verify captcha
					XPath("//span[text()='Drag the slider to fit the puzzle']", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						if err := processCaptcha(); err != nil {
							return model.Retry(err).WithTag(model.ErrProcessCaptcha)
						}
						chain.ForwardNext()
						return nil
					}).
					Selector(".TUXButton-iconContainer img", func(chain *browser.Chain) error {
						var element = chain.Elements().First()
						if err := element.WaitVisible(); err != nil {
							return err
						}
						return nil
					}).WaitAny(); err != nil {
					return err
				}
				return nil
			}).
			XPath(`//span[text()="Account doesn't exist"]`, func(chain *browser.Chain) error {
				return model.NoRetry(nil).WithTag(model.ErrAccountNotExist)
			}).
			//verify captcha
			XPath("//span[text()='Drag the slider to fit the puzzle']", func(chain *browser.Chain) error {
				var element = chain.Elements().First()
				if err := element.WaitVisible(); err != nil {
					return err
				}
				if err := processCaptcha(); err != nil {
					return model.Retry(nil).WithTag(model.ErrProcessCaptcha).WithError(err)
				}
				chain.ForwardPrevN(1, 2, 3)
				return nil

			}).
			XPath("//span[text()='Maximum number of attempts reached. Try again later.']", func(chain *browser.Chain) error {
				return model.NoRetry(nil).WithTag(model.ErrAccountMaximium)
			}).
			WaitAny(); err != nil {
			return err
		}
		return nil
	}
	if err := processLoginButton(); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserClickLoginButton)
	}
	if err := processSelectLoginEmail(); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserClickEmailButton)
	}
	if err := processSwitchLoginEmail(); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserSwitchEmailButton)
	}
	if err := processInputUser(taskArg.BindName); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserInputUser)
	}
	if err := processInputPassword(taskArg.BindPwd); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserInputPwd)
	}
	if err := processClickLoginButton(); err != nil {
		return model.Retry(err).WithTag(model.ErrBrowserLogin)
	}
	if err := processVerify(); err != nil {
		return model.Retry(nil).WithTag(model.ErrBrowserProcessVerify).WithError(err)
	}
	//r, err := page.RodPage().Eval(`()=>document.querySelector("#__UNIVERSAL_DATA_FOR_REHYDRATION__").innerHTML`)
	//if err != nil {
	//	return model.Retry(err).WithTag(model.ErrBrowserUserDataNotFound)
	//}
	//var data = gjson.Parse(r.Value.JSON("", ""))
	//var user = data.Get("__DEFAULT_SCOPE__.webapp.app-context.user")
	//if !user.Exists() {
	//	return model.Retry(nil).WithTag(model.ErrBrowserUserDataNotFound)
	//}
	//var avatar = user.Get("avatarUri.0")
	//if err := loginHook.Wait(common.DefaultBrowserHookLoginTimeout); err != nil {
	//	return model.Retry(err).WithTag(model.ErrBrowserHookUserInfoTimeout)
	//}
	if err := loginHook.Wait(time.Second * 15); err != nil {
		return model.Retry(err).WithTag(model.ErrWaitTimeout)
	}
	var info map[string]string
	var headers = map[string]string{}
	var cookies, _ = _browser.Meta.AsString("cookies")
	_browser.Meta.AsObject("queries", &info)
	_browser.Meta.AsObject("headers", &headers)
	_info, _ := json.Marshal(info)
	_headers, _ := json.Marshal(headers)
	return apiDeviceSync(map[string]any{
		"name":    taskArg.BindName,
		"info":    string(_info),
		"headers": string(_headers),
		"cookie":  cookies,
	})
}

func Register(server *cron.Server) error {
	server.HandleFunc(PatternVideoComment, DyAddVideoComment)
	server.HandleFunc(PatternVideoPublish, DyVideoPublish)
	server.HandleFunc(PatternVideoDetail, DyVideoDetail)
	server.HandleFunc(PatternUpdateAvatar, DyUpdateAvatar)
	server.HandleFunc(PatternVideoDiggLike, DyVideoDiggLike)
	server.HandleFunc(PatternDeviceSync, DyDeviceSync)
	server.HandleFunc(PatternDeviceHeartbeat, DyDeviceHeartbeat)
	return nil
}
