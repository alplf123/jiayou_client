package browser

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"jiayou_backend_spider/browser/impl"
	"jiayou_backend_spider/hook"
	"jiayou_backend_spider/list"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/utils/wait"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/exp/maps"
)

var ErrBrowserConnectFirst = errors.New("connect first")
var ErrChainSkip = errors.New("skip chain")

type Type string

const (
	Native Type = "native"
	Bit    Type = "bit"
)

type Ctx struct {
	Ctx     context.Context
	Browser *Browser
	Hooks   *hook.Hooks
	Meta    *option.Option
}

type HookEvent interface {
	OnConnect(*Ctx)
	OnLoad(*Page, *Ctx)
	OnRequest(*rod.Hijack, *Ctx)
	OnClose(*Ctx) bool
}

type Interceptor struct {
	Pattern string                    `json:"pattern"`
	Network proto.NetworkResourceType `json:"network"`
}
type FilterFunc func(*Browser) bool

func ById(id string) FilterFunc {
	return func(b *Browser) bool {
		return b.ID == id
	}
}
func ByName(name string) FilterFunc {
	return func(b *Browser) bool {
		return b.Name == name
	}
}

type App struct {
	lck      sync.Mutex
	ctx      context.Context
	browsers *list.List[*Browser]
	waits    map[*Browser]chan struct{}
	workers  int
	cursor   impl.BrowserCursor
}

func (app *App) release(browser *Browser, reset bool) {
	app.lck.Lock()
	defer app.lck.Unlock()
	if w, ok := app.waits[browser]; ok {
		w <- struct{}{}
		delete(app.waits, browser)
		if reset || browser.removed {
			browser.reset()
		} else {
			app.browsers.Push(browser)
		}
	}
}
func (app *App) LoadBrowser() error {
	if app.cursor == nil {
		return errors.New("browser cursor not found")
	}
	var cursor = app.cursor
	err := app.cursor.Prepare()
	if err != nil {
		return err
	}
	for i := 0; i < app.workers; i++ {
		if app.cursor.Next() {
			app.browsers.Push(&Browser{
				ID:       cursor.Current().Id(),
				Name:     cursor.Current().Name(),
				Meta:     option.New(nil),
				CreateAt: time.Now(),
				app:      app,
				hookMap:  hook.NewHooks(),
				browser:  cursor.Current(),
				ctx:      app.ctx,
				launcher: launcher.New(),
			})
		} else {
			if cursor.Err() != nil {
				return cursor.Err()
			}
		}
	}
	return cursor.Close()
}
func (app *App) TryPeek(filters ...FilterFunc) *Browser {
	return app.Peek(false, 0, filters...)
}
func (app *App) Peek(block bool, timeout time.Duration, filters ...FilterFunc) (browser *Browser) {
	defer func() {
		if browser != nil {
			browser.Used++
			app.lck.Lock()
			app.waits[browser] = make(chan struct{}, 1)
			app.lck.Unlock()
		}
	}()
	var popFunc = func() *Browser {
		if filters == nil {
			return app.browsers.PopLeft()
		}
		app.lck.Lock()
		var b *Browser
		app.browsers.Iter(func(i int, browser *Browser) bool {
			for _, filter := range filters {
				if filter(browser) {
					b = browser
					return true
				}
			}
			return false
		})
		if b != nil {
			app.browsers.Remove(b)
		}
		app.lck.Unlock()
		return b
	}
	if !block && timeout <= 0 {
		browser = popFunc()
		return
	}
	if block {
		var r, _ = wait.Wait(func(_ *wait.Context) (*Browser, bool) {
			b := popFunc()
			if b != nil {
				return b, true
			}
			return nil, false
		}, wait.WithCtx(app.ctx))
		return r
	}
	var r, _ = wait.Wait(func(_ *wait.Context) (*Browser, bool) {
		b := popFunc()
		if b != nil {
			return b, true
		}
		return nil, false
	}, wait.WithCtx(app.ctx), wait.WithTimeout(timeout))
	return r
}
func (app *App) Running() int {
	app.lck.Lock()
	defer app.lck.Unlock()
	return len(app.waits)
}
func (app *App) Cap() int { return app.browsers.Size() + len(app.waits) }
func (app *App) Size() int {
	return app.browsers.Size()
}
func (app *App) Contain(browser *Browser) bool {
	return app.browsers.Contain(browser)
}
func (app *App) ContainF(f FilterFunc) bool {
	if f == nil {
		return false
	}
	return app.browsers.ContainF(func(browser *Browser) bool {
		if f(browser) {
			return true
		}
		return false
	})
}
func (app *App) Remove(browser *Browser) {
	app.RemoveF(func(b *Browser) bool {
		return browser == b
	})
}
func (app *App) RemoveF(f FilterFunc) {
	if f == nil {
		return
	}
	app.lck.Lock()
	for s, _ := range app.waits {
		if f(s) {
			s.removed = true
		}
	}
	app.lck.Unlock()
	app.browsers.Iter(func(i int, browser *Browser) bool {
		if f(browser) {
			browser.removed = true
		}
		return false
	})
}
func (app *App) Done() {
	app.lck.Lock()
	defer app.lck.Unlock()
	for c, w := range app.waits {
		c.reset()
		close(w)
	}
	app.browsers.Clear()
	maps.Clear(app.waits)
}

type Browser struct {
	ctx      context.Context
	app      *App
	router   *rod.HijackRouter
	hookMap  *hook.Hooks
	hookEvt  HookEvent
	rod      *rod.Browser
	launcher *launcher.Launcher
	browser  impl.IBrowser
	removed  bool
	lck      sync.Mutex
	ID       string
	Name     string
	Used     int64
	CreateAt time.Time
	Meta     *option.Option
}

type ChainType string

const (
	ChainJS       ChainType = "js"
	ChainXPath    ChainType = "xpath"
	ChainSelector ChainType = "selector"
)

type ChainRef string

const (
	Any ChainRef = "any"
	All ChainRef = "all"
)

type ChainFunc func(*Chain)

type ChainCallback func(*Chain) error

func Js(expr string, args []any, callback ChainCallback) ChainFunc {
	return func(chain *Chain) {
		chain.callback = callback
		chain.Expr = expr
		chain.Args = args
		chain.Type = ChainJS
	}
}
func Selector(expr string, callback ChainCallback) ChainFunc {
	return func(chain *Chain) {
		chain.callback = callback
		chain.Expr = expr
		chain.Type = ChainSelector
	}
}
func XPath(expr string, callback ChainCallback) ChainFunc {
	return func(chain *Chain) {
		chain.callback = callback
		chain.Expr = expr
		chain.Type = ChainXPath

	}
}

type Chain struct {
	prev, next *Chain
	forwards   []*Chain
	friend     func() (func(), bool)
	index      int
	forwarded  int
	page       *rod.Page
	callback   ChainCallback
	opts       *option.Option
	elements   rod.Elements
	Args       []any
	Expr       string
	Type       ChainType
}

func (chain *Chain) waitFriend(c chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
	}
	return false
}
func (chain *Chain) forward(chs ...*Chain) {
	if len(chs) > 1 {
		var makeFriend = make(chan struct{})
		var lck sync.Mutex
		var closed bool
		for _, ch := range chs {
			ch.friend = func() (func(), bool) {
				return func() {
					lck.Lock()
					defer lck.Unlock()
					if !closed {
						close(makeFriend)
						closed = true
					}
				}, chain.waitFriend(makeFriend)
			}
		}
	}
	chain.forwards = chain.forwards[:0]
	chain.forwards = append(chain.forwards, chs...)
}
func (chain *Chain) call() error {
	var elements rod.Elements
	var err error
	switch chain.Type {
	default:
		err = fmt.Errorf("chain type not supported: %v", chain.Type)
	case ChainJS:
		elements, err = chain.page.ElementsByJS(rod.Eval(chain.Expr, chain.Args...))
	case ChainSelector:
		elements, err = chain.page.Elements(chain.Expr)
	case ChainXPath:
		elements, err = chain.page.ElementsX(chain.Expr)
	}
	if err != nil {
		return fmt.Errorf("%s('%s') find elements falied,%w",
			chain.Type,
			chain.Expr,
			err,
		)
	}
	if len(elements) == 0 {
		return ErrChainSkip
	}
	chain.elements = elements
	if err := chain.callback(chain); err != nil {
		return fmt.Errorf("%s('%s') callback falied,%w",
			chain.Type,
			chain.Expr,
			err,
		)
	}
	return nil
}
func (chain *Chain) Elements() rod.Elements {
	return chain.elements
}
func (chain *Chain) Options() *option.Option {
	return chain.opts
}
func (chain *Chain) Forwards(exprs ...string) *Chain {
	var chs []*Chain
	var prev = chain.prev
	for prev != nil {
		chs = append(chs, prev)
		prev = prev.prev
	}
	var next = chain.next
	for next != nil {
		chs = append(chs, next)
		next = next.next
	}
	var cMap = make(map[*Chain]struct{})
	for _, ch := range chs {
		if slices.Contains(exprs, ch.Expr) {
			cMap[ch] = struct{}{}
		}
	}
	chain.forward(maps.Keys(cMap)...)
	return chain
}
func (chain *Chain) Forwarded() int {
	return chain.forwarded
}
func (chain *Chain) ForwardNext() {
	if chain.next != nil {
		chain.forward(chain.next)
	}
}
func (chain *Chain) ForwardPrev() {
	if chain.prev != nil {
		chain.forward(chain.prev)
	}
}
func (chain *Chain) ForwardNextN(n int) {
	var next = chain.next
	for n > 0 {
		if next != nil {
			next = next.next
		}
		n--
	}
	if next != nil {
		chain.forward(next)
	}
}
func (chain *Chain) ForwardPrevN(n int) {
	var prev = chain.prev
	for n > 0 {
		if prev != nil {
			prev = prev.prev
		}
		n--
	}
	if prev != nil {
		chain.forward(prev)
	}
}
func (chain *Chain) ForwardTop() {
	temp, prev := chain.prev, chain.prev
	for prev != nil {
		temp, prev = prev, prev.prev
	}
	if temp != nil {
		chain.forward(temp)
	}
}
func (chain *Chain) ForwardLast() {
	temp, next := chain.next, chain.next
	for next != nil {
		temp, next = next, next.next
	}
	if temp != nil {
		chain.forward(temp)
	}
}

func (chain *Chain) Index() int {
	return chain.index
}
func (chain *Chain) IsTop() bool {
	return chain.prev == nil
}
func (chain *Chain) IsLast() bool {
	return chain.next == nil
}

type Chains struct {
	page       *rod.Page
	ref        ChainRef
	timeout    time.Duration
	interval   time.Duration
	maxForward int
	async      bool
	chains     *list.List[*Chain]
	errors     []error
	opts       *option.Option
	lck        sync.Mutex
}

func (chains *Chains) wait() error {
	var ticker = time.NewTicker(chains.interval)
	var wg sync.WaitGroup
	var lck sync.Mutex
	var forwards = list.New[*Chain]()
	for {
		var done = make(chan struct{})
		var finished bool
		var _wait = func() error {
			if chains.timeout > 0 {
				var waiter = time.NewTicker(chains.timeout)
				select {
				case <-done:
				case <-waiter.C:
					return errors.New("wait timeout")
				}
			}
			return nil
		}
		for _, chain := range chains.chains.ToArray() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
					default:
						if chain.friend != nil {
							if _, ok := chain.friend(); ok {
								return
							}
						}
					}
					err := chain.call()
					if err != nil {
						if errors.Is(err, ErrChainSkip) {
							continue
						}
					}
					if len(chain.forwards) > 0 {
						if err != nil && chain.forwarded >= chains.maxForward {
							err = fmt.Errorf("out of max forward,%w", err)
						} else {
							forwards.PushRange(chain.forwards...)
							return
						}
					}
					if err != nil {
						lck.Lock()
						chains.errors = append(
							chains.errors,
							err,
						)
						lck.Unlock()
						return
					}
					if chain.friend != nil {
						if c, ok := chain.friend(); !ok {
							c()
						}
					}
					lck.Lock()
					if chains.ref == Any {
						if !finished {
							close(done)
							finished = true
						}
					}
					lck.Unlock()
				}
			}()
			if chains.async {
				if err := _wait(); err != nil {
					return err
				}
				wg.Wait()
			}
		}
		if err := _wait(); err != nil {
			return err
		}
		wg.Wait()
		if forwards.Size() == 0 {
			break
		}
		for _, forward := range forwards.ToArray() {
			if !chains.chains.Contain(forward) {
				forward.forwarded++
				chains.chains.Push(forward)
			}
		}
		forwards.Clear()
	}
	if len(chains.errors) > 0 {
		return chains.errors[0]
	}
	return nil
}
func (chains *Chains) then(chs ...ChainFunc) *Chains {
	chains.lck.Lock()
	defer chains.lck.Unlock()
	var lastChain = chains.chains.Last()
	for _, ch := range chs {
		if ch != nil {
			var chain Chain
			ch(&chain)
			if chain.Expr != "" && chain.callback != nil && !chains.chains.ContainF(func(c *Chain) bool {
				return chain.Expr == c.Expr
			},
			) {
				if lastChain != nil {
					lastChain.next = &chain
					chain.prev = lastChain
				}
				chain.page = chains.page
				chain.elements = make(rod.Elements, 0)
				chain.index = chains.chains.Size()
				chains.chains.Push(&chain)
			}
		}
	}
	return chains
}
func (chains *Chains) Interval(interval time.Duration) *Chains {
	if interval > 0 {
		chains.interval = interval
	}
	return chains
}
func (chains *Chains) Timeout(timeout time.Duration) *Chains {
	if timeout > 0 {
		chains.timeout = timeout
	}
	return chains
}
func (chains *Chains) MaxForward(maxForward int) *Chains {
	if maxForward > 0 {
		chains.maxForward = maxForward
	}
	return chains
}
func (chains *Chains) Async(async bool) *Chains {
	chains.async = async
	return chains
}
func (chains *Chains) XPath(expr string, callback ChainCallback) *Chains {
	return chains.then(XPath(expr, callback))
}
func (chains *Chains) Selector(expr string, callback ChainCallback) *Chains {
	return chains.then(Selector(expr, callback))
}
func (chains *Chains) Js(expr string, args []any, callback ChainCallback) *Chains {
	return chains.then(Js(expr, args, callback))
}
func (chains *Chains) Then(chs ...ChainFunc) *Chains {
	return chains.then(chs...)
}
func (chains *Chains) Ref(ref ChainRef) *Chains {
	switch ref {
	case Any:
		chains.ref = Any
	case All:
		chains.ref = All
	}
	return chains
}
func (chains *Chains) WaitAny() error {
	chains.ref = Any
	return chains.wait()
}
func (chains *Chains) WaitAll() error {
	chains.ref = All
	return chains.wait()
}
func (chains *Chains) Wait() error {
	return chains.wait()
}

type Page struct {
	browser *Browser
	page    *rod.Page
}

func (bitPage *Page) Load(_url string) error {
	_, err := url.Parse(_url)
	if err != nil {
		return err
	}
	err = bitPage.page.Navigate(_url)
	if err != nil {
		return err
	}
	return nil
}
func (bitPage *Page) Chains() *Chains {
	return &Chains{
		page:     bitPage.page,
		interval: time.Second,
		timeout:  time.Minute,
		async:    true,
		ref:      Any,
		chains:   list.New[*Chain](),
		opts:     option.New(nil),
	}
}
func (bitPage *Page) WaitLoad() error {
	err := bitPage.page.WaitLoad()
	if err == nil {
		bitPage.browser.dispatchOnLoadHookEvt(bitPage, bitPage.browser.makeCtx())
	}
	return err
}
func (bitPage *Page) Timeout(timeout time.Duration) {
	if timeout > 0 {
		bitPage.page = bitPage.page.Timeout(timeout)
	}
}
func (bitPage *Page) Close() error {
	return bitPage.page.Close()
}
func (bitPage *Page) RodPage() *rod.Page {
	return bitPage.page
}

func (browser *Browser) makeCtx() *Ctx {
	return &Ctx{
		Ctx:     browser.ctx,
		Hooks:   browser.hookMap,
		Meta:    browser.Meta,
		Browser: browser,
	}
}
func (browser *Browser) dispatchOnConnectHookEvt(ctx *Ctx) {
	if browser.hookEvt != nil {
		browser.hookEvt.OnConnect(ctx)
	}
}
func (browser *Browser) dispatchOnLoadHookEvt(page *Page, ctx *Ctx) {
	if browser.hookEvt != nil {
		browser.hookEvt.OnLoad(page, ctx)
	}
}
func (browser *Browser) dispatchDefaultRequest(req *rod.Hijack) {
	req.LoadResponse(http.DefaultClient, true)
}
func (browser *Browser) dispatchOnRequestHookEvt(req *rod.Hijack, ctx *Ctx) {
	if browser.hookEvt != nil {
		browser.hookEvt.OnRequest(req, ctx)
		return
	}
	browser.dispatchDefaultRequest(req)
}
func (browser *Browser) dispatchOnCloseHookEvt(ctx *Ctx) bool {
	if browser.hookEvt != nil {
		return browser.hookEvt.OnClose(ctx)
	}
	return true
}
func (browser *Browser) dispatchRequest(handler *rod.Hijack) {
	if !browser.Ready() {
		return
	}
	browser.dispatchOnRequestHookEvt(handler, browser.makeCtx())
}
func (browser *Browser) switchPage(source string, newPage bool) (*rod.Page, error) {
	if browser.rod == nil {
		return nil, ErrBrowserConnectFirst
	}
	pages, err := browser.rod.Pages()
	if err != nil {
		return nil, err
	}
	var defaultPage *rod.Page
	if newPage {
		defaultPage, err = browser.rod.Page(proto.TargetCreateTarget{})
	} else {
		if len(pages) == 0 {
			defaultPage, err = browser.rod.Page(proto.TargetCreateTarget{})
		} else {
			defaultPage = pages.First()
		}
	}
	if defaultPage == nil {
		return nil, errors.New("switch page failed")
	}
	return defaultPage.Context(browser.ctx), defaultPage.Navigate(source)
}
func (browser *Browser) reset() {
	if browser.rod != nil && browser.launcher != nil {
		browser.dispatchOnCloseHookEvt(browser.makeCtx())
		browser.hookEvt = nil
		browser.hookMap.Clear()
		browser.Meta.Clear()
		if browser.router != nil {
			browser.router.Stop()
			browser.router = nil
		}
		if browser.rod != nil {
			browser.rod.Close()
		}
		browser.rod = nil
		browser.launcher = nil
	}
}
func (browser *Browser) AddIntercepts(intercepts ...Interceptor) error {
	if browser.rod != nil {
		if browser.router == nil {
			browser.router = browser.rod.HijackRequests()
			go browser.router.Run()
		}
		for _, interceptor := range intercepts {
			err := browser.router.Add(interceptor.Pattern, interceptor.Network, browser.dispatchRequest)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return ErrBrowserConnectFirst
}
func (browser *Browser) UseEvt(hookEvt HookEvent) {
	if hookEvt != nil {
		browser.hookEvt = hookEvt
	}
}
func (browser *Browser) Ready() bool {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	return browser.rod != nil && browser.launcher != nil
}
func (browser *Browser) SetProxy(proxy string) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.launcher.Proxy(proxy)
}
func (browser *Browser) Pid() int {
	if browser.launcher != nil {
		return browser.launcher.PID()
	}
	return -1
}
func (browser *Browser) Cleanup() {
	if browser.launcher != nil {
		browser.launcher.Cleanup()
	}
}
func (browser *Browser) Kill() {
	if browser.launcher != nil {
		browser.launcher.Kill()
	}
}
func (browser *Browser) NewHook(hookName string) *hook.Hook {
	return browser.hookMap.New(hookName)
}
func (browser *Browser) Connect() error {
	if browser.rod == nil {
		r := rod.New().NoDefaultDevice()
		c, err := browser.browser.OnLaunch(browser.launcher)
		if err != nil {
			return err
		}
		err = r.ControlURL(c).Connect()
		if err != nil {
			return err
		}
		browser.rod = r
		browser.dispatchOnConnectHookEvt(browser.makeCtx())
	}
	return nil
}
func (browser *Browser) DefaultPage() (*Page, error) {
	page, err := browser.switchPage("", false)
	if err != nil {
		return nil, err
	}
	return &Page{page: page, browser: browser}, nil
}
func (browser *Browser) Page(source string, newPage bool) (*Page, error) {
	page, err := browser.switchPage(source, newPage)
	if err != nil {
		return nil, err
	}
	return &Page{page: page, browser: browser}, nil
}
func (browser *Browser) RodBrowser() (*rod.Browser, error) {
	if browser.rod == nil {
		return nil, ErrBrowserConnectFirst
	}
	return browser.rod, nil
}
func (browser *Browser) Reset() {
	browser.app.release(browser, true)
}
func (browser *Browser) Release() {
	browser.app.release(browser, false)
}

type Option func(app *App)

func WithCtx(ctx context.Context) Option {
	return func(app *App) {
		if ctx != nil {
			app.ctx = ctx
		}
	}
}
func WithWorkers(workers int) Option {
	return func(app *App) {
		app.workers = workers
	}
}
func WithCursor(cursor impl.BrowserCursor) Option {
	return func(app *App) {
		app.cursor = cursor
	}
}
func New(opts ...Option) *App {
	var app = &App{
		workers:  1,
		ctx:      context.Background(),
		browsers: list.New[*Browser](),
		waits:    make(map[*Browser]chan struct{}),
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}
