package tiktok

import (
	"context"
	"errors"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/request"
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
	PatternUpdateAvatar    = "pattern_update_avatar"
	PatternSync            = "pattern_sync"
)

func DyAddVideoComment(ctx context.Context, task *cron.Task) error {
	var params model.WebAddCommentTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = common.DefaultProxy
	var _url string
	if params.Level == 0 {
		_url = "aid=1988&aweme_id=" + params.VideoId + "&text=" + params.Text + "&text_extra=[]&" + params.TiktokWebTaskArg.Query()
	} else if params.Level == 1 {
		_url = "aid=1988&aweme_id=" + params.VideoId + "&text=" + params.Text + "&text_extra=[]&reply_id=" + params.ReplyId + "&reply_to_reply_id=0&" + params.TiktokWebTaskArg.Query()
	} else {
		return model.NoRetry(nil).WithTag(model.ErrTaskCommentLevelUnSupported)
	}
	_url, err := utils.RunJsCtx(
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
		return err
	}
	return nil
}
func DyVideoPublish(ctx context.Context, task *cron.Task) error {
	var params model.WebPublicVideoTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = common.DefaultProxy
	reqOptions.Timeout = time.Minute * 30
	reqOptions.ReadTimeout = time.Minute * 30
	reqOptions.WriteTimeout = time.Minute * 30
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
	var url, err = url2.Parse(params.Url)
	if err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var lastFlagPos = strings.LastIndex(url.Path, "/")
	if lastFlagPos <= 0 {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var videoId = url.Path[lastFlagPos+1:]
	if !strings.HasPrefix(videoId, "75") {
		return model.NoRetry(err).WithTag(model.ErrTaskBadVideoUrl)
	}
	var result model.WebCommentTaskArgResult
	result.VideoId = videoId
	if params.Level == 0 {
		return task.Write(result)
	}
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = common.DefaultProxy
	var _url = "aweme_id=" + videoId + "&count=20&cursor=0&aid=1988&" + params.TiktokWebTaskArg.Query()
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
func DyUpdateAvatar(ctx context.Context, task *cron.Task) error {
	var params model.WebUpdateAvatar
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	if params.Avatar == "" {
		return model.NoRetry(errors.New("bad avatar url"))
	}
	var options = request.DefaultRequestOptions()
	options.Header.SetUserAgent(common.DefaultUserAgent)
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
func DySync(ctx context.Context, task *cron.Task) error {
	var params model.TiktokWebTaskArg
	if err := task.Payload().As(&params); err != nil {
		return model.NoRetry(err).WithTag(model.ErrTaskArgs)
	}
	var sidGuard = params.ReqHeaders().Cookie("sid_guard")
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header = params.ReqHeaders()
	reqOptions.Proxy = common.DefaultProxy
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

func Register(server *cron.Server) error {
	server.HandleFunc(PatternAddVideoComment, DyAddVideoComment)
	server.HandleFunc(PatternVideoPublish, DyVideoPublish)
	server.HandleFunc(PatternVideoComment, DyVideoComment)
	server.HandleFunc(PatternVideoDetail, DyVideoDetail)
	server.HandleFunc(PatternUpdateAvatar, DyUpdateAvatar)
	server.HandleFunc(PatternSync, DySync)
	return nil
}
