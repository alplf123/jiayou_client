package test

import (
	"context"
	"fmt"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/service/cron/tiktok"
	"jiayou_backend_spider/utils"
	"os"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
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
	reqOptions.Header.SetCookieText("tt_csrf_token=ouCkwnO-bp4dbwZ3G9dg0kGif7jAQCTtSpQ; tt_chain_token=fcV2sBSqJh8HTggY4zaqjA==; tiktok_webapp_theme_source=auto; tiktok_webapp_theme=dark; passport_csrf_token=667ee57963c2393d54d9cc4ab40e014c; passport_csrf_token_default=667ee57963c2393d54d9cc4ab40e014c; s_v_web_id=verify_mkfbkouh_TVEebQqK_NwtN_4Ia5_8imi_ZjuVghkbuYtJ; d_ticket=d90074f2930beadeed8db24421dd4dd64c706; multi_sids=7516175768040260654%3A9f0ef818f255445c2bb2f7f5b7726bbe; cmpl_token=AgQQAPNSF-RO0rhwPK5rQp0X8wlde3nff4_ZYKIn9g; sid_guard=9f0ef818f255445c2bb2f7f5b7726bbe%7C1768473724%7C15552000%7CTue%2C+14-Jul-2026+10%3A42%3A04+GMT; uid_tt=b16fb1ecf2abed4f86f218f1a2b685d9916d6298298e82864346a4def4b7232d; uid_tt_ss=b16fb1ecf2abed4f86f218f1a2b685d9916d6298298e82864346a4def4b7232d; sid_tt=9f0ef818f255445c2bb2f7f5b7726bbe; sessionid=9f0ef818f255445c2bb2f7f5b7726bbe; sessionid_ss=9f0ef818f255445c2bb2f7f5b7726bbe; tt_session_tlb_tag=sttt%7C3%7Cnw74GPJVRFwrsvf1t3Jrvv_________G-QiJlSlXqrG43c3_MSLDJB255k2yMgloM8340J8Pufg%3D; sid_ucp_v1=1.0.1-KDY4ZDY3M2MwOGVkYjc2YTk2MjJlNzQ4NDAxMjQ1MWJmMTliNDczNjkKIgiuiMGbnsG0p2gQ_IijywYYswsgDDDMpLvCBjgEQOoHSAQQBBoHdXNlYXN0NSIgOWYwZWY4MThmMjU1NDQ1YzJiYjJmN2Y1Yjc3MjZiYmUyTgog5bkiFRch_K9MsMsSNoU91qVJzo1jplLzW6E32yUs3IUSIJRckZS6tR11iz7kJ3AbL6tUhyRGnnDB_Xn2bMAuMuszGAEiBnRpa3Rvaw; ssid_ucp_v1=1.0.1-KDY4ZDY3M2MwOGVkYjc2YTk2MjJlNzQ4NDAxMjQ1MWJmMTliNDczNjkKIgiuiMGbnsG0p2gQ_IijywYYswsgDDDMpLvCBjgEQOoHSAQQBBoHdXNlYXN0NSIgOWYwZWY4MThmMjU1NDQ1YzJiYjJmN2Y1Yjc3MjZiYmUyTgog5bkiFRch_K9MsMsSNoU91qVJzo1jplLzW6E32yUs3IUSIJRckZS6tR11iz7kJ3AbL6tUhyRGnnDB_Xn2bMAuMuszGAEiBnRpa3Rvaw; store-idc=useast5; store-country-code=us; store-country-code-src=uid; tt-target-idc=useast5; tt-target-idc-sign=fxF3Kn63Oa_kFVlX5mDayDVctyOrVhvcMVI3o_wO8AXnsdovrOJZGuz-kZ33_MB2CkakzV1WcWEEmHio9eNq2ycl0pFf62RfW05dqxxGea4j1nQ0OQ2xAGzx-hwuxGkm137FREZtLqD4PTR2uZXTv5rJbAFFO5vRbG_rsJtS3_05SmOW7F_712vkjiIHlPedSZq9yx9_pw6d0MfXKyU00GVPHMPwWLH1XNwEV5rs6ISA3yw39SN0NFbuCIUl_8SLrIgtZZJjvlLeZ1jctwKBlpEw3GRMi05oSSW2KkAC_GXuXq3cj0hqgk8GwolnJirzlI_xIXX9EoC4zDzq-jSpN6zjcLhm2WQeaNpZp9abXQ62GDeiWn40yUXWWYgB1pFQRT5ZfxR8yLUaBPjs5nkd_4EWtbGhn_yBZRRBa8HhCI7IhQbvcUhFV3IZvX7ssqQE1chKXCio74ZXLRmpUBd7o3x5BF7axdrhi00RN8w6j3qhJhCW6cFlxzbF1JqbGO5K; last_login_method=handle; store-country-sign=MEIEDFMdmNuLbQehYbWrkAQgEtLxxj7_vdUTktvhPq9HZWS8XliShaqYYo9RIzovIGQEELeOP34xh_Cb1UDhp9hAE38; passport_fe_beating_status=true; ttwid=1%7CYsrdiqh13fW8mp7fqn17Z1_Ht5m9tl9gtZXqiihq1iE%7C1768473729%7C1aef0402f6cb3ef6c7055d97666142296afddb0496010acbabf8683dbd5ff51c; msToken=dUI8nA0tm8Rl-gHwABfUKHpsGnTdUFAyOLuwLNxHb1hY2sZUSbPIfaunutipvcgex0WSrUs70hSQp8Rl0w09vetOnndj5falUvg2aFVCmUHjwXWokR9iDnVa4QKXtF4KrITsH0IZA43PzyVyr5p1Ytk=; odin_tt=f1f51381bbccc4113a1db808403c6ce61e6f62ae701e8039025a0078c2a64f41bb031975ff24b1dea30bb7bab893730e8900373f722984b7a74d1fc2cb068abc6465f5dd517d25f9be03647191e77754")
	var url = "https://www.tiktok.com/api/comment/list/?aweme_id=7575663473602956574&count=20&cursor=0&aid=1988&WebIdLastTime=1768473681&app_language=en&app_name=tiktok_web&browser_language=en-US&browser_name=Mozilla&browser_online=true&browser_platform=Win32&browser_version=5.0%20(Windows%20NT%2010.0;%20Win64;%20x64)%20AppleWebKit/537.36%20(KHTML,%20like%20Gecko)%20Chrome/143.0.0.0%20Safari/537.36&channel=tiktok_web&cookie_enabled=true&data_collection_enabled=true&device_id=7595536588671567373&device_platform=web_pc&focus_state=true&from_page=&is_fullscreen=false&is_page_visible=true&odinId=7516175768040260654&os=windows&priority_region=US&referer=https://www.tiktok.com/explore&region=US&root_referer=https://www.tiktok.com/explore&screen_height=1080&screen_width=1920&tz_name=America/Los_Angeles&user_is_login=true&webcast_language=en&msToken=&X-Bogus=DFSzs5VOHXsANCJRCzy5BjtTeBcD&X-Gnarly=MC7wMD0SlFNYGRu8BPLMjXiYZBQzZ6knV8kgHYKanpidRI4J5Z40SDMP3xlstMA97yogiiem1HPYEtD0S0LOcktul9t2IWbEzwCIhdAKtTUjH2ZijTK2llGvNK6oA1kHuVdA6VK67xun7-ND27rFQha1ZGAec/fRELs1jZWG-ysivqYIkXBbh-iWrzQ98zG1DlKQv5n0rc5fI3WY0t-9aUrKuqxkZMRaiqccE5hABDsqe6L1mDu1sPa2cCd0MyiPeiuwvvT3DP5Sbun34PQuYpPR4hqNA6sqDQbYGUsykTa0Qh5pkqAOFHF/G7sBvR3OQvA="
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
