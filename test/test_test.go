package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/api"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/cron/tiktok"
	"jiayou_backend_spider/utils"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func TestCaptcha(t *testing.T) {
	fmt.Println(common.GenerateCaptcha())
}
func TestToken(t *testing.T) {
	fmt.Println(common.GenerateToken(1, "jiayou"))
}
func TestJs(t *testing.T) {
	var ctx = goja.New()
	registry := new(require.Registry)
	registry.Enable(ctx)
	var data, _ = os.ReadFile("webmssdk.js")
	ctx.RunString(string(data))
	v := ctx.Get("Encrypt")
	fmt.Println(v)
	var encrypt func(string, string) string
	ctx.ExportTo(v, &encrypt)
	fmt.Println(encrypt(
		"https://www.tiktok.com/api/comment/list/?WebIdLastTime=0&aid=1988&app_language=en&app_name=tiktok_web&aweme_id=7591085507807874324&browser_language=en-US%2Cen%2Czh-CN%2Czh&browser_name=Mozilla&browser_online=true&browser_platform=Win32&browser_version=5.0%20%28Windows%20NT%2010.0%3B%20Win64%3B%20x64%29%20AppleWebKit%2F537.36%20%28KHTML%2C%20like%20Gecko%29%20Chrome%2F143.0.0.0%20Safari%2F537.36&channel=tiktok_web&cookie_enabled=true&count=20&cursor=0&data_collection_enabled=false&device_id=7595150252313052690&device_platform=web_pc&focus_state=true&from_page=video&history_len=3&is_fullscreen=true&is_page_visible=true&odinId=7595150410640344078&os=windows&priority_region=&referer=https%3A%2F%2Fwww.tiktok.com%2Fexplore&region=US&root_referer=https%3A%2F%2Fwww.tiktok.com%2Fexplore&screen_height=1026&screen_width=1824&tz_name=America%2FChicago&user_is_login=false&verifyFp=verify_mkcawh66_LH1qPUmF_BdjK_4GeX_9U0y_dOg7SKgLrX3j&webcast_language=en",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"))
}
func TestUploadVideo(t *testing.T) {
	//var reqOptions = request.DefaultRequestOptions()
	//reqOptions.Header.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	//reqOptions.Header.SetCookie("sid_guard", "5461cd59da59c1c0928fd1a6da1e1f84%7C1767405803%7C15551985%7CThu%2C+02-Jul-2026+02%3A03%3A08+GMT")
	//reqOptions.Header.SetCookie("ttwid", "1%7CY8gbV7r18MfVTMXYODAFn9bGGyVXkii7kvrtF9rwjCI%7C1767689713%7Cb8f3deae53d0edc4da3180da293935f9efc35e079eae54ae1071113a1c126329")
	//reqOptions.Header.Set("X-Storage-U", "7517037320047608846")
	//reqOptions.Proxy = "http://localhost:7890"
	//fmt.Println(tiktok2.WebVideoPublish("1.mp4", tiktok.UploadParams{
	//	Text:         "testing post video",
	//	AllowComment: true,
	//}, reqOptions))
}
func TestUserDetail(t *testing.T) {
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	reqOptions.Proxy = "http://localhost:7890"
	reqOptions.Header.SetCookieText("msToken=rlM-P41Mldt63DsIb4Uz0dJXBXPbScE2GdO8aVzZWFXGQynPaoLuYICIuo7jm9gZ6F6zHQLiHJNm9fqYt4HFOE_UYksiOqiN8d4cclqh2_0oTn-gSs6KsBKjQN0OoA==")
	var url = "https://www.tiktok.com/api/comment/list/?aweme_id=7587404592787311927&count=20&cursor=0&aid=1988&WebIdLastTime=1768904435&app_language=en&app_name=tiktok_web&browser_language=en-US&browser_name=Mozilla&browser_online=true&browser_platform=Win32&browser_version=5.0%20(Windows%20NT%2010.0;%20Win64;%20x64)%20AppleWebKit/537.36%20(KHTML,%20like%20Gecko)%20Chrome/143.0.0.0%20Safari/537.36&channel=tiktok_web&cookie_enabled=true&data_collection_enabled=true&device_id=7597386681973392926&device_platform=web_pc&focus_state=false&from_page=&is_fullscreen=false&is_page_visible=true&odinId=7586816370102682679&os=windows&priority_region=US&referer=https://www.tiktok.com/explore&region=US&root_referer=https://www.tiktok.com/explore&screen_height=1080&screen_width=1920&tz_name=America/Chicago&user_is_login=true&webcast_language=en&msToken=&X-Bogus=DFSzs5VOJmXANCJRCuKaujtTeBc/&X-Gnarly=MHiD3lLoiDOIxCoRAv4FTynwdLOMD8V9DPWNR4093YICtcExxddfLiQHAYso9akRsXVPrGlJxlCldWz9trg8qKm/vfgI16ToWgHs9QJK-9LtNZ-61J3OIhgVErMNXI/wlQlIqo98eHhaVzK6JzDfiXiCep1QgSmT7a3vS2JULcrG8pnrgNHm45Sl--NTSyHk8034R9xBrOE/fv-ECpk1MtoyqiIs02qJhwHBLU9YRA99Npx2Nsckwme6kqKkwcdLUQbObCGgaFcvotP07/TNT-L8S61t/3o7ObrqTvqySHVxkVNJl1TdGgwqRKW3nrnT-EW="
	var comments, err = tiktok.WebVideoComment(url, reqOptions)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(comments)
}
func TestRunJs(t *testing.T) {
	fmt.Println(utils.RunJsCtx(
		context.Background(),
		"webmssdk.js",
		"encrypt",
		"https://www.tiktok.com/api/comment/list/?WebIdLastTime=0&aid=1988&app_language=en&app_name=tiktok_web&aweme_id=7591085507807874324&browser_language=en-US%2Cen%2Czh-CN%2Czh&browser_name=Mozilla&browser_online=true&browser_platform=Win32&browser_version=5.0%20%28Windows%20NT%2010.0%3B%20Win64%3B%20x64%29%20AppleWebKit%2F537.36%20%28KHTML%2C%20like%20Gecko%29%20Chrome%2F143.0.0.0%20Safari%2F537.36&channel=tiktok_web&cookie_enabled=true&count=20&cursor=0&data_collection_enabled=false&device_id=7595150252313052690&device_platform=web_pc&focus_state=true&from_page=video&history_len=3&is_fullscreen=true&is_page_visible=true&odinId=7595150410640344078&os=windows&priority_region=&referer=https%3A%2F%2Fwww.tiktok.com%2Fexplore&region=US&root_referer=https%3A%2F%2Fwww.tiktok.com%2Fexplore&screen_height=1026&screen_width=1824&tz_name=America%2FChicago&user_is_login=false&verifyFp=verify_mkcawh66_LH1qPUmF_BdjK_4GeX_9U0y_dOg7SKgLrX3j&webcast_language=en",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
	))
}
func TestGoogle(t *testing.T) {
	var reqOptions = request.DefaultRequestOptions()
	reqOptions.Header.SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	reqOptions.Proxy = "socks5://jiayou:jiayou@38.92.13.175:1080"
	var resp, err = request.Get("https://www.google.com", reqOptions)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp.Text())
}

func TestVless(t *testing.T) {
	var u = "vless://051f145e-9255-4310-bf5f-0f8901c23522@38.92.13.175:18888?encryption=none&security=reality&flow=xtls-rprx-vision&type=tcp&sni=www.amazon.com&pbk=WvhPa6rAs5sROaQ2T39Fdwh6lNTDd2lj7eD6AIcdKXU&fp=chrome#233boy-tcp-38.92.13.175"
	var r, _ = url.Parse(u)
	fmt.Println(r)
	fmt.Println(r.User.Username())
	fmt.Println(r.Query())
	fmt.Println(r.Host)
}
func TestFfmpeg(t *testing.T) {
	var buf = new(bytes.Buffer)
	err := ffmpeg.Input("1.mp4").
		Silent(true).
		Output("pipe:", ffmpeg.KwArgs{
			"ss":      "00:00:00",
			"vframes": 1,
			"vcodec":  "png",
			"f":       "image2pipe",
			"pix_fmt": "rgb24",
		}).
		WithOutput(buf).
		SetFfmpegPath("C:/Users/Administrator/.go-cas/ffmpeg/ffmpeg.exe").
		Run(func(s *ffmpeg.Stream, cmd *exec.Cmd) {
			//cmd.Dir = "C:\\Users\\Administrator\\.go-cas\\ffmpeg"
		})
	if err != nil {
		fmt.Println(errors.Is(err, exec.ErrDot))
		fmt.Println(err)
		return
	}
	fmt.Println(buf.Bytes())
}

func TestOutlook(t *testing.T) {
	fmt.Println(
		api.ReadTiktokCodeFromOutlook(
			"JoshuaHansen2791@outlook.com",
			"EwAYBOl3BAAU0wDjFA6usBY8gB2LLZHCr9gRQlcAAVNk8m0qCAb3GKgfW8iquwDRTO6ZaDzarGuaUq+0LplepxLbjQKMGyqAmWdtFTkilxDz4d4lWm15msDWz07skJkL/g/YX2ciOXop/KvLNqzDZGJSC3XzCSIYUnkptJDswfNvm1GRYeF+38NvrUHPIePo9CuXf74TnfFmjjarv9bDOjxrjh4vUXEqBS1P+iCpBcnVgyDkbFWTV9G6o68MeBevWi9T7l3BsQ1zy3HEbLdOarScGBoKlU+ToDKkTPdAb3euY8jr8pux6BeBQsMg22iYXmZqra/C4vurzadvPEVeGEzqyot+RS83SgTmTi1YCPXpUZwEWPrM/qComuf/qyoQZgAAEIwtI21o9AXcatGBdnALP6DgAm+FGRneFywvOkAJBz6xC9njkyyRZhvDivZRp0xAtHwhny7giRswSRXQ98ZVf4ZA0GqWB+F65/okig4+nPfjK4Hl89Trb2IWPTO9UKB22mDOd7l8yLJpdPzY6V5TL+f+sHJITuE+5mrbHCP46Zt+PjQG+520nS40RViM495NC06vOhGx/oD9YYBTqdmQq6YYUfj6OpzIm5397VakogTUe7VfnhraYVX1+T0BKMsJHUBzR769T3hTI7hq95RfJUexucpZ3AR+kE1L1CNhwdc5evZ2xONfW6gN/RE7V0Jk8nSb3w7XYOtBXByH1SoOlVmXzH2etb/qPY7jq6IbGtZsF+Pm/9rBXUUoaNUuSD0XUIMY5FdGBjak0V0CRyNrjjtOg4SVESl98fkPw75n9D7mpL0KTmcflBnA8UGfv481L9Asvq7C0v17rQaiZXlOtkxAsM5Jf77Ld49deilEVNbdaRVCSfOgOFG3exWtRYrOIjUHkCA19IaqmMc+MXKxMbg1rQ8UCOj8KVZbnXCo2JdEpGHGhaL/HKb25nVXWUAUMR+JzsvXh92BoVB1JF1SSE3l+HFAmTXvb9P+5/jwFN/yZnFZSX55W9YbLTdV3TkFKZK0yKLgKaHw3l6NIn/0wERhpGKGCm+iawrsWNtKnUuhMbcyNEoCwu/i7SGgiDfnjWU+3kCX/r4VW0LYlfGbe0POx1GAgagi4+3mUEsmNw6uf6PHyk5FCTYm+eHZDNWps0jENSfflkAel09JmV/qX5o6Z1twZyLBAHJK1WuFXkgTQmxy+Ul+mKKWP0swSUpfCCCvlJBMJ7rcwNby744uqQL6kEmcmyZS7jxFgAcTPJgq1ZCwTdWtn+buHlyGc53jDjNHJ/Rb/jEy9Bqr425ASR01FLpFRIWyR/Iy6/cF/YrY1ZBDaO364gGwgWjge4lZiYZuifwsymNyzQ0cIhOwSb4NkZu1P/nUmDMyYM07aXtNFuYWAw==",
			time.Now().UTC(),
		),
	)
}
func TestTwoFA(t *testing.T) {
	fmt.Println(
		api.TwoFA("GZDQD2WCBNQAALEHVJB4YFL62I5ZDD7I"),
	)
}

func TestSyncOutlookToken(t *testing.T) {
	body := make(url.Values)
	body.Set("client_id", "dbc8e03a-b00c-46bd-ae65-b683e7707cb0")
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", "M.C535_BL2.0.U.-ChmAeJQushvSZx820wOXbGFUjJ7Gpe2uV4MQd80L2M4NnF2z72doEoshUieF0WpbVecQb1rwNTyk6bblsnaNo!nUW!XQ5my2ygvyZcqltKPjtl3dviVc8yTQWBRWY0XRkGd9fhr5inighEj491OLttjtyRn9vK48IJ8sK9uM0PwGeTU!pUwQ5*RnT5!k2y7Z0UxomPmfE3sw7Xpj1pJVKhP3JIDySsX63X4*NkzLXDQJFLiXHx1a4qHi6KatqGLeUGH4XFdhRkAU58RrSIJhXHoRpROQuXQcEObQ4KoNkvl7ckGVWx6hQ8od9CFSfpytuKrAklcKGDpw3Ho3xF6WM9bJSZ8UtqmZBv0XT6LFDfj4rxOxk0fXxYHm3*ro906P5LN4MBc0mIwKeHeMmfJnvoY$")
	body.Set("scope", "https://outlook.office.com/IMAP.AccessAsUser.All offline_access")
	fmt.Println(api.SyncOutlookToken(body))
}
