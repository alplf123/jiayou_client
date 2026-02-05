package tiktok

import (
	"bytes"
	rand2 "crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/model"
	"jiayou_backend_spider/utils"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/dop251/goja"
	"github.com/juju/ratelimit"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

//go:embed crypto.js
var crypto string
var cryptoLck sync.Mutex

//go:embed webmssdk.js
var webmssdk string

const MaxUploadVideoSlice int64 = 15 * 1024 * 1024
const defaultUploadRetry = 5

type Comment struct {
	AwemeId           string   `json:"aweme_id"`
	CId               string   `json:"cid"`
	CreateTime        int64    `json:"create_time"`
	DiggCount         int64    `json:"digg_count"`
	Text              string   `json:"text"`
	ReplyCommentTotal int64    `json:"reply_comment_total"`
	CommentLanguage   string   `json:"comment_language"`
	IsAuthorDigged    bool     `json:"is_author_digged"`
	Nickname          string   `json:"nickname"`
	ImageList         []string `json:"image_list"`
}
type DetailBitRateInfo struct {
	BitRate     int    `json:"bit_rate"`
	QualityType int    `json:"quality_type"`
	BitRateFPS  int    `json:"bit_rate_fps"`
	CodecType   string `json:"codec_type"`
	Format      string `json:"format"`
	DataSize    int64  `json:"data_size"`
	Url         string `json:"url"`
	FileHash    string `json:"file_hash"`
	GearName    string `json:"gear_name"`
}
type DetailVideo struct {
	Height       int                 `json:"height"`
	Width        int                 `json:"width"`
	Duration     int                 `json:"duration"`
	Cover        string              `json:"cover"`
	BitrateInfos []DetailBitRateInfo `json:"bitrate_infos"`
}
type DetailAuthor struct {
	Id         string `json:"id"`
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	SecUid     string `json:"sec_uid"`
	CreateTime int64  `json:"create_time"`
	UniqueId   string `json:"unique_id"`
}
type DetailStats struct {
	DiggCount    int64 `json:"digg_count"`
	ShareCount   int64 `json:"share_count"`
	CommentCount int64 `json:"comment_count"`
	PlayCount    int64 `json:"play_count"`
	CollectCount int64 `json:"collect_count"`
}
type Detail struct {
	Id         string       `json:"id"`
	CreateTime int64        `json:"create_time"`
	Desc       string       `json:"desc"`
	Author     DetailAuthor `json:"author"`
	Video      DetailVideo  `json:"video"`
	Stat       DetailStats  `json:"stat"`
}

type UploadStore struct {
	Uri        string `json:"uri"`
	Auth       string `json:"auth"`
	UploadId   string `json:"upload_id"`
	UploadHost string `json:"upload_host"`
	Vid        string `json:"vid"`
	SessionKey string `json:"session_key"`
}
type UploadAuth struct {
	Auth            string `json:"auth"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	AccessKeyId     string `json:"access_key_id"`
}
type PostUploadResult struct {
	Vid      string  `json:"vid"`
	Uri      string  `json:"uri"`
	Height   int     `json:"height"`
	Width    int     `json:"width"`
	Duration float64 `json:"duration"`
	Bitrate  int     `json:"bitrate"`
	Format   string  `json:"format"`
	Size     int64   `json:"size"`
	FileType string  `json:"file_type"`
	Codec    string  `json:"codec"`
}

type VisibilityType int

const (
	Everyone VisibilityType = iota
	Friends
	OnlyYou
)

type UploadVideoParams struct {
	Text           string         `json:"text"`
	AllowComment   bool           `json:"allow_comment"`
	ScheduleTime   int64          `json:"schedule_time"`
	VisibilityType VisibilityType `json:"visibility_type"`
}
type CommentLevel int

const (
	TopCommentLevel CommentLevel = iota
	SecondCommentLevel
)

var encrypt func(url, body, userAgent string) string

func getStore(route string, options *request.Options) (*UploadStore, *UploadAuth, error) {
	var _url = "https://www.tiktok.com/api/v1/video/upload/auth/?aid=1988"
	auth, err := uploadAuth(_url, options)
	if err != nil {
		return nil, nil, err
	}
	uploadAuthReqOptions := options.Clone()
	var date = time.Now().UTC()
	var currentDataFormat = date.Format("20060102T150405Z")
	var signKey = common.Hmac256([]byte(currentDataFormat[:8]), []byte("AWS4"+auth.SecretAccessKey))
	signKey = common.Hmac256([]byte("US-TTP"), signKey)
	signKey = common.Hmac256([]byte("vod"), signKey)
	signKey = common.Hmac256([]byte("aws4_request"), signKey)
	var params = strings.Split(route, "&")
	sort.Strings(params)
	var canonicalBody = []string{
		"GET",
		"/top/v1",
		strings.Join(params, "&"),
		"x-amz-date:" + currentDataFormat + "\n" +
			"x-amz-security-token:" + auth.SessionToken + "\n",
		"x-amz-date;x-amz-security-token",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	var signBody = []string{
		"AWS4-HMAC-SHA256",
		currentDataFormat,
		currentDataFormat[:8] + "/US-TTP/vod/aws4_request",
		hex.EncodeToString(common.Sha256([]byte(strings.Join(canonicalBody, "\n")))),
	}
	var authorization = fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s, SignedHeaders=%s, Signature=%s",
		auth.AccessKeyId+"/"+currentDataFormat[:8]+"/US-TTP/vod/aws4_request",
		"x-amz-date;x-amz-security-token",
		hex.EncodeToString(common.Hmac256([]byte(strings.Join(signBody, "\n")), signKey)),
	)
	uploadAuthReqOptions.Header.Set("x-amz-security-token", auth.SessionToken)
	uploadAuthReqOptions.Header.Set("x-amz-date", currentDataFormat)
	uploadAuthReqOptions.Header.Set("authorization", authorization)
	pngUploadStore, err := applyUploadInner("https://www.tiktok.com/top/v1?"+route, uploadAuthReqOptions)
	if err != nil {
		return nil, nil, err
	}
	return pngUploadStore, auth, nil
}
func applyUploadInner(url string, reqOpt *request.Options) (*UploadStore, error) {
	resp, err := request.Get(url, reqOpt)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	if !resp.Success() {
		return nil, model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return nil, model.NewBadResponseError()
	}
	if !gjson.Valid(body) {
		return nil, errorx.New("bad json video data")
	}
	var j = gjson.Parse(body)
	var code = j.Get("ResponseMetadata.Error.CodeN").Int()
	var message = j.Get("ResponseMetadata.Error.Message").String()
	if code != 0 {
		return nil, errorx.New("api error").WithField("code", code).WithField("message", message)
	}
	var node = j.Get("Result.InnerUploadAddress.UploadNodes.0")
	if !node.Exists() {
		return nil, errorx.New("node not existed")
	}
	var storeInfo = node.Get("StoreInfos.0")
	var store = new(UploadStore)
	store.UploadId = storeInfo.Get("UploadID").String()
	store.Auth = storeInfo.Get("Auth").String()
	store.Uri = storeInfo.Get("StoreUri").String()
	store.SessionKey = node.Get("SessionKey").String()
	store.Vid = node.Get("Vid").String()
	store.UploadHost = node.Get("UploadHost").String()
	return store, nil
}
func uploadAuth(url string, reqOption *request.Options) (*UploadAuth, error) {
	var resp, err = request.Get(url, reqOption)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	if !resp.Success() {
		return nil, model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return nil, model.NewBadResponseError()
	}
	if !gjson.Valid(body) {
		return nil, errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return nil, model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	var videoTokenV5 = data.Get("video_token_v5")
	var uploadAuth = new(UploadAuth)
	uploadAuth.AccessKeyId = videoTokenV5.Get("access_key_id").String()
	uploadAuth.SessionToken = videoTokenV5.Get("session_token").String()
	uploadAuth.SecretAccessKey = videoTokenV5.Get("secret_acess_key").String()
	uploadAuth.Auth = data.Get("auth").String()
	return uploadAuth, nil
}
func upload(url string, payload any, reqOption *request.Options) (*gjson.Result, error) {
	var resp, err = request.Execute(url, "POST", payload, reqOption)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	if !resp.Success() {
		return nil, model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return nil, model.NewBadResponseError()
	}
	if !gjson.Valid(body) {
		return nil, errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("code").Int())
	if code != 2000 {
		return nil, model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	var r = data.Get("data")
	return &r, nil
}
func uploadData(url string, body any, options *request.Options) (string, error) {
	var r, err = upload(url, body, options)
	if err != nil {
		return "", err
	}
	var crc32 = r.Get("crc32").String()
	var partNumber = r.Get("part_number").String()
	return partNumber + ":" + crc32, nil
}
func uploadFileDone(url string, body map[string]any, reqOption *request.Options) (*PostUploadResult, error) {
	var r, err = upload(url, body, reqOption)
	if err != nil {
		return nil, err
	}
	var data = r.Get("post_upload_resp.results.0")
	var meta = data.Get("video_meta")
	var result = new(PostUploadResult)
	result.Vid = data.Get("vid").String()
	result.Uri = meta.Get("Uri").String()
	result.Height = int(meta.Get("Height").Int())
	result.Width = int(meta.Get("Width").Int())
	result.Duration = meta.Get("Duration").Float()
	result.Size = meta.Get("Size").Int()
	result.Format = meta.Get("Format").String()
	result.Bitrate = int(meta.Get("Bitrate").Int())
	result.FileType = meta.Get("FileType").String()
	result.Codec = meta.Get("Codec").String()
	return result, nil

}
func postVideo(url string, payload any, options *request.Options) error {
	var resp, err = request.Post(url, payload, options)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	if !resp.Success() {
		return model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return model.NewBadResponseError()
	}
	if !gjson.Valid(body) {
		return errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	return nil
}

func Encrypt(url, body, userAgent string) string {
	cryptoLck.Lock()
	defer cryptoLck.Unlock()
	return encrypt(url, body, userAgent)
}
func RandS(length int) string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.IntN(len(charset))]
	}
	return string(result)
}
func RandCreateId(len int) string {
	if len == 0 {
		return ""
	}
	var charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-"
	randomBytes := make([]byte, len)
	if _, err := rand2.Read(randomBytes); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, b := range randomBytes {
		index := b & 63
		sb.WriteByte(charset[index])
	}
	return sb.String()
}
func RandS13() string {
	length := 10 + rand.IntN(3)
	return RandS(length)
}

func WebVideoComment(url string, reqOption *request.Options) ([]*Comment, error) {
	var headers = make(fhttp.Header)
	headers.Set("user-agent", reqOption.Header.UserAgent())
	headers.Set("cookie", reqOption.Header.CookiesText())
	options := []tlsclient.HttpClientOption{
		tlsclient.WithClientProfile(profiles.Chrome_133),
		tlsclient.WithNotFollowRedirects(),
		tlsclient.WithDefaultHeaders(headers),
		tlsclient.WithProxyUrl(reqOption.Proxy),
	}
	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), options...)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	req, err := fhttp.NewRequest(fhttp.MethodGet, url, nil)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, model.NewStatusError(resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var body = string(raw)
	if body == "" {
		return nil, model.NewBase().WithTag(model.ErrBadResponse)
	}
	if !gjson.Valid(body) {
		return nil, errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return nil, model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	var commentsEle = data.Get("comments").Array()
	var comments = make([]*Comment, len(commentsEle))
	for i, comment := range commentsEle {
		comments[i] = &Comment{
			AwemeId:           comment.Get("aweme_id").String(),
			CId:               comment.Get("cid").String(),
			CreateTime:        comment.Get("create_time").Int(),
			DiggCount:         comment.Get("digg_count").Int(),
			Text:              comment.Get("text").String(),
			ReplyCommentTotal: comment.Get("reply_comment_total").Int(),
			CommentLanguage:   comment.Get("comment_language").String(),
			IsAuthorDigged:    comment.Get("is_author_digged").Bool(),
			Nickname:          comment.Get("user.nickname").String(),
		}
		var imageList = comment.Get("image_list").Array()
		if len(imageList) > 0 {
			comments[i].ImageList = make([]string, len(imageList))
			for j, image := range imageList {
				comments[i].ImageList[j] = image.Get("crop_url.ur_list.0").String()
			}
		}
	}
	return comments, nil
}
func WebAddVideoComment(url string, reqOption *request.Options) error {
	var headers = make(fhttp.Header)
	headers.Set("user-agent", reqOption.Header.UserAgent())
	headers.Set("cookie", reqOption.Header.CookiesText())
	options := []tlsclient.HttpClientOption{
		tlsclient.WithClientProfile(profiles.Chrome_104),
		tlsclient.WithNotFollowRedirects(),
		tlsclient.WithDefaultHeaders(headers),
		tlsclient.WithProxyUrl(reqOption.Proxy),
	}
	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), options...)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	req, err := fhttp.NewRequest(fhttp.MethodPost, url, nil)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewStatusError(resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var body = string(raw)
	if body == "" {
		return model.NewBase().WithTag(model.ErrBadResponse)
	}
	if !gjson.Valid(body) {
		return errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	return nil
}
func WebVideoDiggLike(url string, reqOption *request.Options) error {
	var headers = make(fhttp.Header)
	headers.Set("user-agent", reqOption.Header.UserAgent())
	headers.Set("cookie", reqOption.Header.CookiesText())
	options := []tlsclient.HttpClientOption{
		tlsclient.WithClientProfile(profiles.Chrome_104),
		tlsclient.WithNotFollowRedirects(),
		tlsclient.WithDefaultHeaders(headers),
		tlsclient.WithProxyUrl(reqOption.Proxy),
	}
	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), options...)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	req, err := fhttp.NewRequest(fhttp.MethodPost, url, nil)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return model.NewNetError().WithError(err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewStatusError(resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var body = string(raw)
	if body == "" {
		return model.NewBase().WithTag(model.ErrBadResponse)
	}
	if !gjson.Valid(body) {
		return errorx.New("bad json video data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	return nil
}
func WebVideoDetail(url string, reqOption *request.Options) (*Detail, error) {
	var resp, err = request.Get(url, reqOption)
	if err != nil {
		return nil, model.NewNetError().WithError(err)
	}
	if !resp.Success() {
		return nil, model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return nil, model.NoRetry(nil).WithTag(model.ErrBadResponse)
	}
	var prefix = `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">`
	var subfix = "</script>"
	var j = utils.TextBetween(body, prefix, subfix)
	if j == "" || !gjson.Valid(j) {
		return nil, errorx.New("bad json video data")
	}
	var data = gjson.Parse(j)
	var item = data.Get(`webapp\.video-detail.itemInfo.itemStruct`)
	var video = item.Get("video")
	var author = item.Get("author")
	var stats = item.Get("stats")
	var detail = new(Detail)
	detail.Stat.DiggCount = stats.Get("diggCount").Int()
	detail.Stat.ShareCount = stats.Get("shareCount").Int()
	detail.Stat.CommentCount = stats.Get("commentCount").Int()
	detail.Stat.PlayCount = stats.Get("playCount").Int()
	detail.Stat.CollectCount = stats.Get("collectCount").Int()
	detail.Author.Id = author.Get("id").String()
	detail.Author.Nickname = author.Get("nickname").String()
	detail.Author.Avatar = author.Get("avatarThumb").String()
	detail.Author.SecUid = author.Get("secUid").String()
	detail.Author.CreateTime = data.Get("createTime").Int()
	detail.Author.UniqueId = data.Get("uniqueId").String()
	detail.Id = item.Get("id").String()
	detail.CreateTime = item.Get("createTime").Int()
	detail.Desc = item.Get("desc").String()
	detail.Video.Width = int(video.Get("width").Int())
	detail.Video.Height = int(video.Get("height").Int())
	detail.Video.Duration = int(video.Get("duration").Int())
	detail.Video.Cover = video.Get("cover").String()
	for _, bitrate := range video.Get("bitrateInfo").Array() {
		detail.Video.BitrateInfos = append(detail.Video.BitrateInfos, DetailBitRateInfo{
			BitRate:     int(bitrate.Get("Bitrate").Int()),
			QualityType: int(bitrate.Get("QualityType").Int()),
			BitRateFPS:  int(bitrate.Get("BitrateFPS").Int()),
			GearName:    bitrate.Get("GearName").String(),
			CodecType:   bitrate.Get("CodecType").String(),
			Format:      bitrate.Get("Format").String(),
			DataSize:    bitrate.Get("PlayAddr.DataSize").Int(),
			FileHash:    bitrate.Get("PlayAddr.FileHash").String(),
			Url:         bitrate.Get("PlayAddr.UrlList|@reverse|0").String(),
		})
	}
	return detail, nil
}
func WebVideoPublish(file string, params UploadVideoParams, reqOption *request.Options) error {
	fileSize, err := common.GetFileSize(file)
	if err != nil {
		return model.NewBase().WithTag(model.ErrVideoFile).WithError(fmt.Errorf("get file size failed: %w", err))
	}
	if fileSize <= 100*1024 {
		return model.NewBase().WithTag(model.ErrVideoSoSmall).WithError(fmt.Errorf("file size is too small. %d bytes", fileSize))
	}
	firstFrame, err := common.GenerateVideoFirstFrame(file)
	if err != nil {
		return model.NewBase().WithTag(model.ErrFfmpeg).WithError(err)
	}
	var defaultRateLimiter = ratelimit.NewBucketWithRate(1*1024*1024, 1*1024*1024)
	var pngRoute = fmt.Sprintf("Action=ApplyUploadInner&Version=2020-11-19&SpaceName=tiktok&FileType=image&IsInner=1&FileSize=%d&s=%s&Scene=poster&device_platform=web",
		len(firstFrame),
		RandS13(),
	)
	pngUploadStore, _, err := getStore(pngRoute, reqOption)
	if err != nil {
		return fmt.Errorf("get image store failed: %w", err)
	}
	var videoRoute = fmt.Sprintf("Action=ApplyUploadInner&Version=2020-11-19&SpaceName=tiktok&FileType=video&IsInner=1&ClientBestHosts=%s&FileSize=%d&X-Amz-Expires=604800&s=%s&device_platform=web",
		pngUploadStore.UploadHost,
		fileSize,
		RandS13(),
	)
	videoUploadStore, videoUploadAuth, err := getStore(videoRoute, reqOption)
	if err != nil {
		return fmt.Errorf("get video store failed: %w", err)
	}
	pngCrc32, err := common.GenerateCrc32(firstFrame)
	if err != nil {
		return fmt.Errorf("png crc32 failed,%w", err)
	}
	var pngReqOptions = reqOption.Clone()
	pngReqOptions.Header.SetContentType("application/octet-stream")
	pngReqOptions.Header.Set("Authorization", pngUploadStore.Auth)
	pngReqOptions.Header.Set("Content-CRC32", fmt.Sprintf("%x", pngCrc32))
	pngUrl := fmt.Sprintf("https://%s/upload/v1/%s", pngUploadStore.UploadHost, pngUploadStore.Uri)
	_, err = uploadData(pngUrl, ratelimit.Reader(bytes.NewReader(firstFrame), defaultRateLimiter), pngReqOptions)
	if err != nil {
		return fmt.Errorf("upload png failed,%w", err)
	}
	var videoReqOptions = reqOption.Clone()
	videoReqOptions.Header.SetContentType("application/octet-stream")
	videoReqOptions.Header.Set("Authorization", videoUploadStore.Auth)
	videoReqOptions.Header.Set("X-Enable-Upload-Mode", "stream")
	videoReqOptions.Header.Set("X-Enable-Omit-Init-Upload", "1")
	videoReqOptions.Header.Set("X-Phase", "transfer")
	videoReqOptions.Header.Set("Content-CRC32", "Ignore")
	videoReqOptions.Header.Set("X-Offset", "0")
	var offset int64
	var countBlock int = 1
	var tempOptions []*request.Options
	for i := 0; i < int(fileSize/MaxUploadVideoSlice); i++ {
		var tempOption = videoReqOptions.Clone()
		tempOption.Header.Set("X-Part-Number", fmt.Sprintf("%d", i+1))
		tempOption.Header.Set("X-Size", fmt.Sprintf("%d", MaxUploadVideoSlice))
		tempOption.Header.Set("X-Part-Offset", fmt.Sprintf("%d", offset))
		tempOptions = append(tempOptions, tempOption)
		offset += MaxUploadVideoSlice
		countBlock = i + 1
	}
	var lastPart = fileSize % MaxUploadVideoSlice
	if lastPart > 0 {
		if len(tempOptions) == 0 {
			countBlock = 0
		}
		var tempOption = videoReqOptions.Clone()
		tempOption.Header.Set("X-Part-Number", fmt.Sprintf("%d", countBlock+1))
		tempOption.Header.Set("X-Size", fmt.Sprintf("%d", lastPart))
		tempOption.Header.Set("X-Part-Offset", fmt.Sprintf("%d", offset))
		tempOptions = append(tempOptions, tempOption)
	}
	var videoUid = utils.MustDefaultUUID()
	var videoUrl = fmt.Sprintf("https://%s/upload/v1/%s?uploadid=%s&device_platform=web", videoUploadStore.UploadHost, videoUploadStore.Uri, videoUid)
	var waitG = new(errgroup.Group)
	var parts []string
	var partLck sync.Mutex
	var waitGroup = new(sync.WaitGroup)
	var concurrent = 2
	var cancel bool
	for index, opt := range tempOptions {
		if index%concurrent == 0 {
			waitGroup.Wait()
		}
		if cancel {
			break
		}
		waitGroup.Add(1)
		waitG.Go(func() error {
			defer waitGroup.Done()
			var run = func() error {
				fp, err := os.Open(file)
				if err != nil {
					return model.NewBase().WithError(fmt.Errorf("open video file %s failed", file)).WithTag(model.ErrVideoFile)
				}
				defer fp.Close()
				var size, _ = strconv.Atoi(opt.Header.Get("X-Size").First())
				var offset, _ = strconv.Atoi(opt.Header.Get("X-Part-Offset").First())
				if _, err := fp.Seek(int64(offset), 0); err != nil {
					return fmt.Errorf("seek video file %s failed,%w", file, err)
				}
				//var reader = ratelimit.Reader(, defaultRateLimiter)
				part, err := uploadData(videoUrl, io.LimitReader(fp, int64(size)), opt)
				if err != nil {
					return fmt.Errorf("video upload failed at part %d,%w", index, err)
				}
				partLck.Lock()
				parts = append(parts, part)
				partLck.Unlock()
				return nil
			}
			var lastErr error
			for i := 0; i < defaultUploadRetry; i++ {
				if err := run(); err != nil {
					lastErr = err
					common.DefaultLogger.Warn("upload part failed", zap.Int("retry", i+1), zap.Error(err))
				} else {
					break
				}
			}
			cancel = lastErr != nil
			return lastErr
		})
	}
	waitGroup.Wait()
	err = waitG.Wait()
	if err != nil {
		return err
	}
	slices.SortFunc(parts, func(a, b string) int {
		var part1, _ = strconv.Atoi(strings.Split(a, ":")[0])
		var part2, _ = strconv.Atoi(strings.Split(b, ":")[0])
		return part1 - part2
	})
	var uploadDoneBody = map[string]any{
		"parts_crc": strings.Join(parts, ","),
		"post_upload_param": map[string]any{
			"sts2_token":  videoUploadAuth.SessionToken,
			"sts2_secret": videoUploadAuth.SecretAccessKey,
			"session_key": videoUploadStore.SessionKey},
		"functions": []any{},
	}
	var fileDoneOptions = reqOption.Clone()
	fileDoneOptions.Header.Set("Authorization", videoUploadStore.Auth)
	fileDoneOptions.Header.Set("X-Phase", "finish")
	fileDoneOptions.Header.Set("X-Size", fmt.Sprintf("%d", fileSize))
	fileDoneOptions.Header.Set("X-Upload-With-PostUpload", "1")
	fileDoneOptions.Header.SetContentType("application/json")
	postUploadResult, err := uploadFileDone(videoUrl, uploadDoneBody, fileDoneOptions)
	if err != nil {
		return fmt.Errorf("upload post upload result failed,%w", err)
	}
	allowCommon := 1
	if !params.AllowComment {
		allowCommon = 0
	}
	var uploadBody = map[string]any{
		"post_common_info": map[string]any{
			"creation_id":          RandCreateId(21),
			"enter_post_page_from": 2,
			"post_type":            3,
		},
		"feature_common_info_list": []any{
			map[string]any{
				"schedule_time":      params.ScheduleTime,
				"geofencing_regions": []any{},
				"playlist_name":      "",
				"playlist_id":        "",
				"tcm_params":         "{\"commerce_toggle_info\":{}}",
				"sound_exemption":    0,
				"anchors":            []any{},
				"vedit_common_info": map[string]any{
					"draft":    "",
					"video_id": videoUploadStore.Vid,
				},
				"privacy_setting_info": map[string]any{
					"visibility_type":     params.VisibilityType,
					"allow_duet":          0,
					"allow_stitch":        0,
					"allow_comment":       allowCommon,
					"allow_content_reuse": 0,
				},
			},
		},
		"single_post_req_list": []any{
			map[string]any{
				"batch_index":   0,
				"video_id":      videoUploadStore.Vid,
				"is_long_video": 1,
				"single_post_feature_info": map[string]any{
					"text":        params.Text,
					"text_extra":  []any{},
					"markup_text": params.Text,
					"music_info":  map[string]any{},
					"cover_info": map[string]any{
						"frame_duration":           0,
						"blob_url":                 "blob:https://www.tiktok.com/" + utils.MustDefaultUUID(),
						"cover_width":              480,
						"cover_height":             640,
						"cover_type":               2,
						"crop_type":                2,
						"isAutoCropFirstFrame":     true,
						"cover_uri":                pngUploadStore.Uri,
						"isProcessingInitialCover": false,
					},
					"poster_delay":                   0,
					"cloud_edit_video_height":        postUploadResult.Height,
					"cloud_edit_video_width":         postUploadResult.Width,
					"cloud_edit_is_use_video_canvas": false,
				},
			},
		},
	}
	reqOption.Header.SetContentType("application/json")
	reqOption.Header.SetReferer("https://www.tiktok.com/tiktokstudio/upload?from=creator_center")
	reqOption.Header.Set("origin", "https://www.tiktok.com")
	var postVideoUrl = "https://www.tiktok.com/tiktok/web/project/post/v1/?app_name=tiktok_web&channel=tiktok_web&device_platform=web&tz_name=America%2FLos_Angeles&aid=1988"
	err = postVideo(postVideoUrl, uploadBody, fileDoneOptions)
	if err != nil {
		return fmt.Errorf("post video failed,%w", err)
	}
	return nil

}
func WebSync(url string, reqOption *request.Options) error {
	var resp, err = request.Get(url, reqOption)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return model.NewStatusError(resp.Status())
	}
	var body = resp.Text()
	if body == "" {
		return model.NewBadResponseError()
	}
	if !gjson.Valid(body) {
		return errorx.New("bad json data")
	}
	var data = gjson.Parse(body)
	var code = int(data.Get("status_code").Int())
	if code != 0 {
		return model.NewApiError().WithCode(code).WithMessage(data.Get("status_msg").String())
	}
	return nil
}
func WebMsToken(url string, reqOption *request.Options) (string, error) {
	var headers = make(fhttp.Header)
	headers.Set("user-agent", reqOption.Header.UserAgent())
	//headers.Set("cookie", reqOption.Header.CookiesText())
	options := []tlsclient.HttpClientOption{
		tlsclient.WithClientProfile(profiles.Chrome_104),
		tlsclient.WithNotFollowRedirects(),
		tlsclient.WithDefaultHeaders(headers),
		tlsclient.WithProxyUrl(reqOption.Proxy),
	}
	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), options...)
	if err != nil {
		return "", model.NewNetError().WithError(err)
	}
	req, err := fhttp.NewRequest(fhttp.MethodGet, url, nil)
	if err != nil {
		return "", model.NewNetError().WithError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", model.NewNetError().WithError(err)
	}
	var msToken = resp.Header.Get("x-ms-token")
	if msToken == "" {
		msToken = resp.Header.Get("X-Ms-Token")
	}
	return "msToken=" + msToken, nil
}
func init() {
	var ctx = goja.New()
	ctx.RunString(crypto)
	var encryptFunc = ctx.Get("Encrypt")
	if err := ctx.ExportTo(encryptFunc, &encrypt); err != nil {
		panic(fmt.Errorf("tiktok encrypt map failed,%w", err))
	}
}
