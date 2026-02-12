package tiktok

import (
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser"
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
	var p, _ = common.DefaultXrayConfigMap.Load(params.ProxyName)
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
	var p, _ = common.DefaultXrayConfigMap.Load(params.ProxyName)
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
	var p, _ = common.DefaultXrayConfigMap.Load(params.ProxyName)
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
	var p, _ = common.DefaultXrayConfigMap.Load(params.ProxyName)
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
	options.Proxy = common.DefaultProxy
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
	var p, _ = common.DefaultXrayConfigMap.Load(params.ProxyName)
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

	//_browser := common.GBrowser.Peek(false, common.DefaultBrowserTryPeek)
	//if _browser == nil {
	//	return errors.New("browser try peek timeout")
	//}
	//if err := bit.UpdateProxy(
	//	fmt.Sprintf(
	//		"http://%s:%d%s",
	//		common.GBitBrowserOptions.DebugAddr,
	//		common.GBitBrowserOptions.DebugPort,
	//		bit.ApiUpdateProxy,
	//	),
	//	[]string{_browser.ID},
	//	common.DefaultProxy,
	//	request.DefaultRequestOptions(),
	//); err != nil {
	//	return err
	//}
	//if err := _browser.Connect(); err != nil {
	//	return fmt.Errorf("browser connect fail,%w", err)
	//}
	//var page, err = _browser.DefaultPage()
	//if err != nil {
	//	return fmt.Errorf("default page failed,%w", err)
	//}
	//err = page.Load("https://www.tiktok.com")
	//if err != nil {
	//	return err
	//}
	//err = page.WaitLoad()
	//if err != nil {
	//	return fmt.Errorf("wait load failed,%w", err)
	//}
	//var processLoginButton = func() error {
	//	loginEle, err := page.RodPage().Element("#top-right-action-bar-login-button")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return loginEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processSelectLoginEmail = func() error {
	//	emailEle, err := page.RodPage().ElementX("//div[text()='Use phone / email / username']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return emailEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processSwitchLoginEmail = func() error {
	//	userEle, err := page.RodPage().ElementX("//a[text()='Log in with email or username']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return userEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processInputEmail = func(userName string) error {
	//	inputUsrEle, err := page.RodPage().Element("input[placeholder='Email or username']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return inputUsrEle.Input(userName)
	//}
	//var processInputPassword = func(password string) error {
	//	inputPwdEle, err := page.RodPage().Element("input[placeholder='Password']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return inputPwdEle.Input(password)
	//}
	//var processClickLoginButton = func() error {
	//	loginBtnEle, err := page.RodPage().ElementX("//div[@id='loginModalContentContainer']//button[text()='Log in']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return loginBtnEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processCaptcha = func() error {
	//	captchaImgs, err := page.RodPage().Elements("img[alt='Captcha']")
	//	if err != nil {
	//		return err
	//	}
	//	for _, captchaImg := range captchaImgs {
	//		if err := captchaImg.WaitVisible(); err != nil {
	//			return err
	//		}
	//	}
	//	outImg, _ := captchaImgs[0].Attribute("src")
	//	innerImg, _ := captchaImgs[1].Attribute("src")
	//	if outImg == nil || innerImg == nil {
	//		return errors.New("bad captcha image")
	//	}
	//	out := (*outImg)[23:]
	//	inner := (*innerImg)[23:]
	//	_, slide, err := api.GetRotate(out, inner)
	//	if err != nil {
	//		return err
	//	}
	//	slideEle, err := page.RodPage().ElementX("//button[@id='captcha_slide_button']/parent::div")
	//	if err != nil {
	//		return err
	//	}
	//	shape, err := slideEle.Shape()
	//	if err != nil {
	//		return err
	//	}
	//	slide = slide + 36 // add base offset
	//	var slidePage = slideEle.Page()
	//	if err := slidePage.Mouse.MoveTo(proto.Point{X: shape.Box().X + shape.Box().Width/2, Y: shape.Box().Y + shape.Box().Height/2}); err != nil {
	//		return err
	//	}
	//	if err := slidePage.Mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
	//		return err
	//	}
	//	for i := 0; i < slide; i += 5 {
	//		if err := slidePage.Mouse.MoveLinear(proto.Point{X: shape.Box().X + float64(i), Y: shape.Box().Y}, 1); err != nil {
	//			return err
	//		}
	//		time.Sleep(time.Millisecond * time.Duration(10+rand.IntN(50)))
	//	}
	//	if err := slidePage.Mouse.MoveLinear(proto.Point{X: shape.Box().X + float64(slide), Y: shape.Box().Y}, 1); err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return slidePage.Mouse.Up(proto.InputMouseButtonLeft, 1)
	//}
	//var processReadEmailCode = func(email, deviceId, token string) (string, error) {
	//	var start = time.Now().UTC()
	//	body := make(url2.Values)
	//	body.Set("client_id", "dbc8e03a-b00c-46bd-ae65-b683e7707cb0")
	//	body.Set("grant_type", "refresh_token")
	//	body.Set("refresh_token", "M.C535_BL2.0.U.-CnMCwsGZy5Xfzrxx1oFKYDzqtzM7NW7xqrycaNrB6bjVkvqYBCTEWquLvnPCwVSOumHlWtrK*lGC4sv1wLcM!8N4Ti3WR1Wtu2RwTryWWHsR7GRoH1dpydCxbtcSJEcBDd!H!tWMElWlver2NBn8pzZmk135CjwtPKqVTo90Dacfnk7O4BTPNpofUBUh8ON4c70Cn9bm8puEhOZRiu*ak0VszBDMQ4J!1wazlhoM1E1Ax11tBb6rzqNHPramyVLtzWE6u3T9SdlTNLsuEczEPoNv8*mgOjTYlT3gO9VUE5tStFw!EzaN2Bn1pt3r8*Uer4ju05wUpW8CVyjntl2zXwLgyKPlLNl96E1fgmkzfL252LWMvOhRMj*FnvnrZ0U7bg$$")
	//	body.Set("scope", "https://outlook.office.com/IMAP.AccessAsUser.All offline_access")
	//	token, err := api.SyncOutlookToken(body)
	//	if err != nil {
	//		return "", err
	//	}
	//	var ticker = time.NewTicker(time.Second * 15)
	//	var code string
	//	for {
	//		select {
	//		case <-ticker.C:
	//			return "", errors.New("read email code timeout")
	//		default:
	//			codes, err := api.ReadTiktokCodeFromOutlook(
	//				email, token, start)
	//			if err != nil {
	//				return "", err
	//			}
	//			if len(codes) > 0 {
	//				code = codes[0]
	//			}
	//		}
	//		if code != "" {
	//			break
	//		}
	//		time.Sleep(time.Second)
	//	}
	//	return code, nil
	//}
	//var processDigitCode = func(code string) error {
	//	verifyCodeEle, err := page.RodPage().Element("input[placeholder='Enter 6-digit code']")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return verifyCodeEle.Input(code)
	//}
	//var processClickNextButton = func() error {
	//	nextEle, err := page.RodPage().ElementX("//div[text()='Next']/parent::div/parent::div/parent::button")
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return nextEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processClickSendEmail = func() error {
	//	emailVerifyEle, err := page.RodPage().ElementX("//div[@id='idv-modal-container']//h1/following-sibling::div[1]")
	//	if err != nil {
	//		return err
	//	}
	//	err = emailVerifyEle.WaitVisible()
	//	if err != nil {
	//		return err
	//	}
	//	defer time.Sleep(time.Second)
	//	return emailVerifyEle.Click(proto.InputMouseButtonLeft, 1)
	//}
	//var processVerify = func() error {
	//	if err := page.Chains().
	//		//verify email
	//		Selector("#idv-modal-container", func(chain *browser.Chain) error {
	//			var element = chain.Elements().First()
	//			if err := element.WaitVisible(); err != nil {
	//				return err
	//			}
	//			if err := processClickSendEmail(); err != nil {
	//				return err
	//			}
	//			if err := page.Chains().
	//				//verify captcha
	//				Selector("#captcha-verify-container-main-page", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					if err := processCaptcha(); err != nil {
	//						return err
	//					}
	//					chain.ForwardNext()
	//					return nil
	//				}).
	//				XPath("//div[text()='Resend code']", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					code, err := processReadEmailCode("", "", "")
	//					if err != nil {
	//						return err
	//					}
	//					err = processDigitCode(code)
	//					if err != nil {
	//						return err
	//					}
	//					return processClickNextButton()
	//				}).WaitAny(); err != nil {
	//				return err
	//			}
	//			if err := page.Chains().
	//				//verify captcha
	//				Selector("#captcha-verify-container-main-page", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					if err := processCaptcha(); err != nil {
	//						return err
	//					}
	//					chain.ForwardNext()
	//					return nil
	//				}).
	//				Selector(".TUXButton-iconContainer", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					return nil
	//				}).WaitAny(); err != nil {
	//				return err
	//			}
	//			return nil
	//		}).
	//		//verify 2fa
	//		Selector("#loginModalContentContainer", func(chain *browser.Chain) error {
	//			var element = chain.Elements().First()
	//			if err := element.WaitVisible(); err != nil {
	//				return err
	//			}
	//			code, err := api.TwoFA("")
	//			if err != nil {
	//				return err
	//			}
	//			err = processDigitCode(code)
	//			if err != nil {
	//				return err
	//			}
	//			err = processClickNextButton()
	//			if err != nil {
	//				return err
	//			}
	//			if err := page.Chains().
	//				//verify captcha
	//				Selector("#captcha-verify-container-main-page", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					if err := processCaptcha(); err != nil {
	//						return err
	//					}
	//					chain.ForwardNext()
	//					return nil
	//				}).
	//				Selector(".TUXButton-iconContainer", func(chain *browser.Chain) error {
	//					var element = chain.Elements().First()
	//					if err := element.WaitVisible(); err != nil {
	//						return err
	//					}
	//					return nil
	//				}).WaitAny(); err != nil {
	//				return err
	//			}
	//			return nil
	//		}).
	//		//verify captcha
	//		Selector("#captcha-verify-container-main-page", func(chain *browser.Chain) error {
	//			var element = chain.Elements().First()
	//			if err := element.WaitVisible(); err != nil {
	//				return err
	//			}
	//			if err := processCaptcha(); err != nil {
	//				return err
	//			}
	//			chain.Forwards("#idv-modal-container", "#loginModalContentContainer")
	//			return nil
	//
	//		}).WaitAny(); err != nil {
	//		return err
	//	}
	//	return nil
	//}
	return nil
}
func TestBrowser() error {
	_browser := common.GBrowser.Peek(false, common.DefaultBrowserTryPeek)
	if _browser == nil {
		return errors.New("browser try peek timeout")
	}
	if err := bit.UpdateProxy(
		fmt.Sprintf(
			"http://%s:%d%s",
			common.GBitBrowserOptions.DebugAddr,
			common.GBitBrowserOptions.DebugPort,
			bit.ApiUpdateProxy,
		),
		[]string{_browser.ID},
		common.DefaultProxy,
		request.DefaultRequestOptions(),
	); err != nil {
		return err
	}
	if err := _browser.Connect(); err != nil {
		return fmt.Errorf("browser connect fail,%w", err)
	}
	var page, err = _browser.DefaultPage()
	if err != nil {
		return fmt.Errorf("default page failed,%w", err)
	}
	err = page.Load("https://www.tiktok.com")
	if err != nil {
		return err
	}
	err = page.WaitLoad()
	if err != nil {
		return fmt.Errorf("wait load failed,%w", err)
	}
	return page.Chains().
		Ref(browser.Any).
		MaxForward(2).
		XPath("//div[text()='Comedy']", func(chain *browser.Chain) error {
			var first = chain.Elements().First()
			fmt.Println("found Comedy", first)
			fmt.Println(chain.Retied(), chain.Forwarded())
			chain.ForwardNextN(2)
			return nil
		}).
		XPath("//div[text()='Comedy1']", func(chain *browser.Chain) error {
			var first = chain.Elements().First()
			fmt.Println("found Comedy", first)
			return nil
		}).
		XPath("//div[text()='Log in']", func(chain *browser.Chain) error {
			var first = chain.Elements().First()
			fmt.Println("found login", first)
			chain.Forwards("//div[text()='Comedy1']", "//div[text()='Comedy']")
			return nil
		}).Wait()
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
