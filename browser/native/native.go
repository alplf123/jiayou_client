package native

import (
	"errors"
	"fmt"
	"jiayou_backend_spider/browser/impl"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/utils"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

var _ impl.IBrowser = &Browser{}

type Browser struct {
	id, name string
	launch   *launcher.Launcher
	native   *Native
}

func (browser *Browser) Pid() int {
	if browser.launch != nil {
		return browser.launch.PID()
	}
	return -1
}

func (browser *Browser) Cleanup() {
	if browser.launch != nil {
		browser.launch.Cleanup()
	}
}

func (browser *Browser) Kill() {
	if browser.launch != nil {
		browser.launch.Kill()
	}
}

func (browser *Browser) Open() (string, error) {
	if browser.launch == nil {
		return "", errors.New("no launcher available")
	}
	browser.launch.
		Headless(browser.native.opts.HeadLess).
		Leakless(browser.native.opts.LeakLess).
		RemoteDebuggingPort(browser.native.opts.DebugPort).
		UserDataDir(browser.native.opts.UserDataDir).
		Bin(browser.native.opts.ChromeBin)
	for _, arg := range browser.native.opts.Args {
		kv := strings.Split(arg, "=")
		var key = strings.TrimSpace(kv[0])
		if key != "" && strings.HasPrefix(key, "--") {
			if len(kv) == 2 {
				browser.launch.Set(flags.Flag(key), kv[1])
			} else {
				browser.launch.Set(flags.Flag(key))
			}
		}
	}
	return browser.launch.Launch()
}

func (browser *Browser) Id() string {
	return browser.id
}
func (browser *Browser) Name() string {
	return browser.name
}

type Options struct {
	ChromeBin         string   `json:"chrome_bin"`
	UserDataDir       string   `json:"user_data_dir"`
	DebugPort         int      `json:"debug_port" validate:"min=0,max=65535"`
	LeakLess          bool     `json:"leakless"`
	HeadLess          bool     `json:"headless"`
	Args              []string `json:"args"`
	KillZombieProcess bool     `json:"kill_zombie_process"`
	TitlePrefix       string   `json:"title_prefix"`
}

type Native struct {
	opts    *Options
	current impl.IBrowser
	id      int64
}

func (native *Native) Prepare() error {
	if native.opts.KillZombieProcess {
		utils.KillRemoteDebugChromeProcess()
	}
	return nil
}
func (native *Native) Current() impl.IBrowser { return native.current }
func (native *Native) Next() bool {
	defer func() { atomic.AddInt64(&native.id, 1) }()
	var browser = &Browser{native: native, launch: launcher.New()}
	browser.id = strconv.FormatInt(native.id, 10)
	var title = browser.id
	if native.opts.TitlePrefix != "" {
		title = fmt.Sprintf("%s-%s", native.opts.TitlePrefix, title)
	}
	browser.name = title
	native.current = browser
	return true
}
func (native *Native) Err() error   { return nil }
func (native *Native) Close() error { native.current = nil; return nil }

var DefaultNativeOptions = func() *Options {
	return config.TryValidate(&Options{})
}

func New() *Native {
	return &Native{opts: DefaultNativeOptions()}
}
func NewWithOptions(opts *Options) *Native {
	if opts == nil {
		opts = DefaultNativeOptions()
	}
	return &Native{opts: opts}
}
