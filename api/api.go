package api

import (
	"fmt"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/request"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pquerna/otp/totp"
	"github.com/tgulacsi/imapclient/xoauth2"
	"github.com/tidwall/gjson"
)

const tokenYM = "YFvopSTPpNfLge5GiNHZEa_OPDNN9EiIejtouWmmHrU"
const apiYM = "http://api.jfbym.com/api/YmServer/customApi"
const apiOutlookMeMessage = "https://outlook.office365.com/v1.0/me/messages"
const apiOutlookSyncToken = "https://login.microsoftonline.com/common/oauth2/v2.0/token"

func GetRotate(outImage, innerImage string) (rotate, slide int, err error) {
	var payload = map[string]any{
		"out_ring_image":     outImage,
		"inner_circle_image": innerImage,
		"token":              tokenYM,
		"type":               90004,
	}
	resp, err := request.PostJson(apiYM, payload, request.DefaultRequestOptions())
	if err != nil {
		return
	}
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Header().StatusCode())
		return
	}
	json := gjson.Parse(resp.Text())
	if json.Get("code").Int() != 10000 {
		err = errorx.Slient().New("bad api request").WithField("url", apiYM).WithField("msg", json.Get("msg").String())
		return
	}
	return int(json.Get("data.data.rotate_angle").Int()), int(json.Get("data.data.slide_px").Int()), nil
}
func connectIMAPWithOAuth(email, server, accessToken string) (*client.Client, error) {
	auth := xoauth2.NewXOAuth2Client(&xoauth2.XOAuth2Options{
		Username:    email,
		AccessToken: accessToken,
	})
	c, err := client.DialTLS(server, nil)
	if err != nil {
		return nil, fmt.Errorf("connect email server failed,%w", err)
	}
	if err := c.Authenticate(auth); err != nil {
		return nil, fmt.Errorf("auth email server failed,%w", err)
	}
	return c, nil
}
func ReadEmailFromImap(user, accessToken string, server, from string, startAt time.Time) ([]string, error) {
	c, err := connectIMAPWithOAuth(user, server, accessToken)
	if err != nil {
		return nil, err
	}
	criteria := imap.NewSearchCriteria()
	if from != "" {
		criteria.Header.Set("From", from)
	}
	_, err = c.Select("inbox", true)
	if err != nil {
		return nil, fmt.Errorf("select inbox failed,%w", err)
	}
	ids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search email failed,%w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids[len(ids)-1])
	messages := make(chan *imap.Message, 1)
	if err := c.Fetch(seqset,
		[]imap.FetchItem{
			imap.FetchEnvelope,
		},
		messages); err != nil {
		return nil, fmt.Errorf("fetch email failed,%w", err)
	}
	var titles []string
	for msg := range messages {
		//fmt.Println(msg.Envelope.Date, startAt, msg.Envelope.Subject)
		if msg.Envelope.Date.After(startAt) {
			titles = append(titles, msg.Envelope.Subject)
		}
	}
	return titles, nil
}
func ReadTiktokCodeFromOutlook(user, accessToken string, startAt time.Time) ([]string, error) {
	titles, err := ReadEmailFromImap(user, accessToken, "outlook.office365.com:993", "noreply@account.tiktok.com", startAt)
	if err != nil {
		return nil, err
	}
	var codes []string
	for _, title := range titles {
		if strings.Contains(title, "is your verification code") {
			var fields = strings.Fields(title)
			if len(fields) > 0 {
				codes = append(codes, strings.TrimSpace(fields[0]))
			}
		}
	}
	return codes, nil
}
func SyncOutlookToken(body url.Values) (string, error) {
	var opts = request.DefaultRequestOptions()
	resp, err := request.PostForm(apiOutlookSyncToken, body, opts)
	if err != nil {
		return "", err
	}
	if resp.Status() != 200 {
		return "", errorx.Slient().New("bad status").WithField("status", resp.Header().StatusCode())
	}
	json := gjson.Parse(resp.Text())
	return json.Get("access_token").String(), nil
}
func TwoFA(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		return "", err
	}
	return code, nil
}
