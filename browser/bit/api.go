package bit

import (
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/request"
	"net/url"
	"strconv"

	"github.com/tidwall/gjson"
)

const ApiCreateWindow = "/browser/update"
const ApiHealth = "/health"
const ApiDeleteWindow = "/browser/delete/ids"
const ApiWindowAlive = "/browser/pids/alive"
const ApiCloseWindow = "/browser/close"
const ApiOpenWindow = "/browser/open"
const ApiListGroup = "/group/list"
const ApiAddGroup = "/group/add"
const ApiListBrowser = "/browser/list"
const ApiUpdateBrowser = "/browser/update/partial"
const ApiListPids = "/browser/pids"
const ApiUpdateProxy = "/browser/proxy/update"

type RespMap map[string]string

func UpdateProxy(api string, ids []string, proxy string, opts *request.Options) error {
	var p, err = url.Parse(proxy)
	if err != nil {
		return err
	}
	var user, pwd = "", ""
	if p.User != nil {
		user = p.User.Username()
		pwd, _ = p.User.Password()
	}
	var payload = map[string]any{
		"ids":            ids,
		"ipCheckService": "",
		"proxyMethod":    2,
		"proxyType":      p.Scheme,
		"host":           p.Hostname(),
		"port":           p.Port(),
		"proxyUserName":  user,
		"proxyPassword":  pwd,
	}
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return err
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Header().StatusCode())
		return err
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return err
	}
	return nil
}
func CreateWindow(api string, windowConfig string, opts *request.Options) (RespMap, error) {
	resp, err := request.PostJson(api, windowConfig, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Header().StatusCode())
		return nil, err
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return nil, err
	}
	var r = make(RespMap)
	r["id"] = json.Get("data.id").String()
	r["name"] = json.Get("data.name").String()
	return r, nil
}
func WindowAlive(api string, id string, opts *request.Options) (ok bool, err error) {
	var payload = `{"ids": ["` + id + `"]}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	return json.Get("data." + id).Exists(), nil
}
func FindPid(api string, ids []string, opts *request.Options) (pids map[string]int, err error) {
	var data = option.New(nil)
	data.Set("ids", ids)
	resp, err := request.PostJson(api, data.String(), opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	var pidMap map[string]int = make(map[string]int)
	for _, id := range ids {
		if json.Get("data." + id).Exists() {
			pidMap[id] = int(json.Get("data." + id).Int())
		}
	}
	return pidMap, nil
}
func CloseWindow(api string, id string, opts *request.Options) (ok bool, err error) {
	var payload = `{"id":"` + id + `"}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	return true, nil
}
func OpenWindow(api string, id string, args []string, sync bool, opts *request.Options) (ws string, pid int, err error) {
	var data = option.New(nil)
	data.Set("id", id)
	data.Set("queue", sync)
	data.Set("args", args)
	resp, err := request.PostJson(api, data.String(), opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	ws = json.Get("data.ws").String()
	pid = int(json.Get("data.pid").Int())
	return
}
func Health(api string, opts *request.Options) (ok bool, err error) {
	resp, err := request.PostJson(api, "", opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	return true, nil
}

func ListGroup(api string, page int, opts *request.Options) (groups []RespMap, err error) {
	var payload = `{"page":` + strconv.Itoa(page) + `,"pageSize":10,"all": true,"sortDirection": "asc","sortProperties": "sortNum"}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	for _, g := range json.Get("data.list").Array() {
		group := make(RespMap)
		group["id"] = g.Get("id").String()
		group["name"] = g.Get("groupName").String()
		group["belong"] = g.Get("belongName").String()
		groups = append(groups, group)
	}
	return
}

func AddGroup(api string, name string, opts *request.Options) (groups RespMap, err error) {
	var payload = `{"groupName":"` + name + `","sortNum": 1}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	group := make(RespMap)
	group["id"] = json.Get("data.id").String()
	group["name"] = json.Get("data.groupName").String()
	group["belong"] = json.Get("data.belongName").String()
	return group, nil
}
func ListBrowser(api string, page int, size int, groupId string, opts *request.Options) (browsers []RespMap, total int64, err error) {
	var payload = `{"page":` + strconv.Itoa(page) + `,"pageSize":` + strconv.Itoa(size) + `,"groupId":"` + groupId + `"}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	for _, g := range json.Get("data.list").Array() {
		if groupId == g.Get("groupId").String() {
			browser := make(RespMap)
			browser["id"] = g.Get("id").String()
			browser["name"] = g.Get("name").String()
			browsers = append(browsers, browser)
		}
	}
	total = json.Get("data.totalNum").Int()
	return
}
func ListBrowserCount(api string, page int, groupId string, opts *request.Options) (c int64, err error) {
	var payload = `{"page":` + strconv.Itoa(page) + `,"pageSize":10,"groupId":"` + groupId + `"}`
	resp, err := request.PostJson(api, payload, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	c = json.Get("data.totalNum").Int()
	return
}
func UpdateBrowser(api string, conf string, opts *request.Options) (err error) {
	resp, err := request.PostJson(api, conf, opts)
	if err != nil {
		return
	}
	defer resp.Close()
	if resp.Status() != 200 {
		err = errorx.Slient().New("bad status").WithField("status", resp.Status())
		return
	}
	json := gjson.Parse(resp.Text())
	if !json.Get("success").Bool() {
		err = errorx.Slient().New("bad api request").WithField("url", api).WithField("message", json.Get("msg").String())
		return
	}
	return nil
}
