package bit

import (
	_ "embed"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser/impl"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/list"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/request"
	"jiayou_backend_spider/utils"
	"jiayou_backend_spider/utils/wait"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/launcher/flags"
)

var _ impl.IBrowser = &Browser{}

type Browser struct {
	bit      *Bit
	id, name string
	pid      int
}

func (browser *Browser) kill(pid int) {
	if pid > 0 {
		CloseWindow(browser.bit.debugHost()+ApiCloseWindow, strconv.Itoa(pid), request.DefaultRequestOptions())
		wait.Wait(func(context *wait.Context) (any, bool) {
			return nil, !utils.AliveProcess(pid)
		}, wait.WithTimeout(5*time.Second), wait.WithBackoffFunc(utils.FixedDuration(time.Second)))
		if utils.AliveProcess(pid) {
			utils.KillProcess(pid)
		}
	}
}
func (browser *Browser) buildArgs() []string {
	var temp []string
	if browser.bit.opts.OriginOption.FingerPrint.LaunchArgs != "" {
		temp = append(temp, strings.Split(browser.bit.opts.OriginOption.FingerPrint.LaunchArgs, ",")...)
	}
	temp = append(temp, browser.bit.opts.Args...)
	var args []string
	for _, arg := range temp {
		kv := strings.Split(arg, "=")
		var key = strings.TrimSpace(kv[0])
		if key != "" && strings.HasPrefix(key, "--") {
			if len(kv) == 2 {
				args = append(args, fmt.Sprintf("%s=%s", flags.Flag(key), kv[1]))
			} else {
				args = append(args, key)
			}
		}
	}
	return args
}
func (browser *Browser) Pid() int {
	return browser.pid
}
func (browser *Browser) Cleanup() {
	if browser.id != "" {
		ClearWindow(browser.bit.debugHost()+ApiCloseWindow, browser.id, request.DefaultRequestOptions())
	}
}
func (browser *Browser) Kill() {
	browser.kill(browser.pid)
	browser.pid = -1
}
func (browser *Browser) Open() (string, error) {
	alive, pid, err := WindowAlive(browser.bit.debugHost()+ApiWindowAlive, browser.id, request.DefaultRequestOptions())
	if err != nil {
		return "", err
	}
	if alive && pid > 0 {
		browser.kill(pid)
	}
	time.Sleep(time.Second)
	ws, pid, err := OpenWindow(browser.bit.debugHost()+ApiOpenWindow,
		browser.id,
		browser.buildArgs(),
		true,
		request.DefaultRequestOptions())
	if err != nil {
		return "", err
	}
	browser.pid = pid
	return ws, nil
}
func (browser *Browser) Id() string {
	return browser.id
}
func (browser *Browser) Name() string {
	return browser.name
}

type Options struct {
	GroupName         string        `json:"group_name" validate:"required"`
	PageSize          int           `json:"page_size" validate:"gt=0" default:"1"`
	DebugAddr         string        `json:"debug_addr" validate:"ipv4" default:"127.0.0.1"`
	DebugPort         int           `json:"debug_port" validate:"min=0,max=65535" default:"54345"`
	OriginOption      *OriginOption `json:"origin_option" validate:"required" default:""`
	Args              []string      `json:"args"`
	KillZombieProcess bool          `json:"kill_zombie_process"`
}

type Bit struct {
	opts        *Options
	groupId     string
	err         error
	current     impl.IBrowser
	browserPage int
	browsers    *list.List[impl.IBrowser]
}

func (bit *Bit) buildOption() *option.Option {
	if !bit.opts.OriginOption.RandomFingerprint {
		switch bit.opts.OriginOption.FingerPrint.Ostype {
		case "Windows":
			bit.opts.OriginOption.FingerPrint.Os = "Win32"
		case "Linux":
			bit.opts.OriginOption.FingerPrint.Os = "Linux x86_64"
		case "macOS":
			bit.opts.OriginOption.FingerPrint.Os = "MacIntel"
		case "Android":
			bit.opts.OriginOption.FingerPrint.Os = "Linux armv81"
		case "IOS":
			bit.opts.OriginOption.FingerPrint.Os = "iPhone"
		}
	}
	o, _ := option.FromS(bit.opts.OriginOption)
	return o
}
func (bit *Bit) debugHost() string {
	return fmt.Sprintf("http://%s:%d", bit.opts.DebugAddr, bit.opts.DebugPort)
}
func (bit *Bit) Prepare() error {
	if bit.opts.GroupName == "" {
		return errors.New("bit group name is empty")
	}
	if bit.opts.KillZombieProcess {
		utils.KillProcessByName("BitBrowser.exe")
	}
	var groupPage int
	var groupRespList = list.New[RespMap]()
	for {
		groups, err := ListGroup(bit.debugHost()+ApiListGroup, groupPage, request.DefaultRequestOptions())
		if err != nil {
			return err
		}
		if groups == nil {
			break
		}
		groupRespList.PushR(groups...)
		groupPage++
	}
	var groups = groupRespList.Filter(func(rm RespMap) bool {
		return rm["name"] == bit.opts.GroupName
	})
	if len(groups) == 0 {
		//create group
		g, err := AddGroup(bit.debugHost()+ApiAddGroup, bit.opts.GroupName, request.DefaultRequestOptions())
		if err != nil {
			return err
		}
		bit.groupId = g["id"]
	} else {
		bit.groupId = groups[0]["id"]
	}
	return nil

}
func (bit *Bit) Next() bool {
	if bit.groupId == "" {
		bit.err = errors.New("group required")
		return false
	}
	if bit.browsers.Size() == 0 {
		browsers, total, err := ListBrowser(bit.debugHost()+ApiListBrowser, bit.browserPage, bit.opts.PageSize, bit.groupId, request.DefaultRequestOptions())
		if err != nil {
			bit.err = err
			return false
		}
		if len(browsers) == 0 {
			var opt = bit.buildOption()
			opt.Set("groupId", bit.groupId)
			opt.Set("name", fmt.Sprintf("%s-%d", bit.opts.GroupName, total))
			browser, err := CreateWindow(bit.debugHost()+ApiCreateWindow, opt.String(), request.DefaultRequestOptions())
			if err != nil {
				bit.err = err
				return false
			}
			bit.browsers.PushR(&Browser{
				bit:  bit,
				id:   browser["id"],
				name: browser["name"],
				pid:  -1,
			})
		} else {
			for _, browser := range browsers {
				bit.browsers.PushR(&Browser{
					bit:  bit,
					id:   browser["id"],
					name: browser["name"],
					pid:  -1,
				})
			}
		}
		bit.browserPage++
	}
	bit.current = bit.browsers.PopL()
	return true
}
func (bit *Bit) Current() impl.IBrowser {
	return bit.current
}
func (bit *Bit) Err() error {
	return bit.err
}

func (bit *Bit) Close() error {
	bit.groupId = ""
	bit.browsers = nil
	bit.current = nil
	bit.err = nil
	bit.browserPage = 0
	return nil
}

var DefaultBitOptions = func() *Options {
	return config.TryValidate(&Options{})
}

func New() *Bit {
	var bit = &Bit{
		opts:     DefaultBitOptions(),
		browsers: list.New[impl.IBrowser](),
	}
	return bit
}
func NewWithOptions(opts *Options) *Bit {
	if opts == nil {
		opts = DefaultBitOptions()
	}
	var bit = &Bit{
		opts:     opts,
		browsers: list.New[impl.IBrowser](),
	}
	return bit
}
