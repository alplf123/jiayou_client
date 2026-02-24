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
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/go-rod/rod"
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

type OnRequest func(*rod.Hijack, *Ctx)
type OnLoad func(*Page, *Ctx)
type OnConnect func(*Ctx)
type OnClose func(*Ctx)

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
			id, name := cursor.Current().Id(), cursor.Current().Name()
			if id == "" || name == "" {
				return fmt.Errorf("browser cursor id or name is empty")
			}
			app.browsers.Push(&Browser{
				app:          app,
				hookMap:      hook.NewHooks(),
				browser:      cursor.Current(),
				ctx:          app.ctx,
				interceptors: make(map[string]proto.NetworkResourceType),
				ID:           cursor.Current().Id(),
				Name:         cursor.Current().Name(),
				Meta:         option.New(nil),
				CreateAt:     time.Now(),
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
			return app.browsers.PopL()
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
	ctx          context.Context
	app          *App
	router       *rod.HijackRouter
	hookMap      *hook.Hooks
	rod          *rod.Browser
	browser      impl.IBrowser
	removed      bool
	interceptors map[string]proto.NetworkResourceType
	onRequests   []OnRequest
	onCloses     []OnClose
	onloads      []OnLoad
	onConnects   []OnConnect
	lck          sync.Mutex
	ID           string
	Name         string
	Used         int64
	CreateAt     time.Time
	Meta         *option.Option
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

type ChainErr struct {
	error
	Expr string
	Type ChainType
}

func (chainErr *ChainErr) Error() string {
	return fmt.Sprintf("%s('%s'),%s", chainErr.Type, chainErr.Expr, chainErr.error)
}

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
	retried    int
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
		return &ChainErr{err, chain.Expr, chain.Type}
	}
	if len(elements) == 0 {
		return ErrChainSkip
	}
	chain.elements = elements
	if err := chain.callback(chain); err != nil {
		var e *ChainErr
		if errors.As(err, &e) {
			return e
		}
		return &ChainErr{err, chain.Expr, chain.Type}
	}
	return nil
}
func (chain *Chain) Elements() rod.Elements {
	return chain.elements
}
func (chain *Chain) Options() *option.Option {
	return chain.opts
}
func (chain *Chain) Forward(expr string) {
	chain.Forwards(expr)
}
func (chain *Chain) Forwards(exprs ...string) {
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
}
func (chain *Chain) Forwarded() int {
	return chain.forwarded
}
func (chain *Chain) Retied() int { return chain.retried }
func (chain *Chain) ForwardNext() {
	chain.ForwardNextN(1)
}
func (chain *Chain) ForwardNextN(ns ...int) {
	var nextChains []*Chain
	for _, n := range ns {
		var next = chain.next
		for n > 1 {
			if next != nil {
				next = next.next
			}
			n--
		}
		if next != nil {
			nextChains = append(nextChains, next)
		}
	}
	chain.forward(nextChains...)
}
func (chain *Chain) ForwardPrev() {
	chain.ForwardPrevN(1)
}
func (chain *Chain) ForwardPrevN(ns ...int) {
	var prevChains []*Chain
	for _, n := range ns {
		var prev = chain.prev
		for n > 1 {
			if prev != nil {
				prev = prev.prev
			}
			n--
		}
		if prev != nil {
			prevChains = append(prevChains, prev)
		}
	}
	chain.forward(prevChains...)
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
	errors     *list.List[error]
	opts       *option.Option
	lck        sync.Mutex
}

func (chains *Chains) wait() error {
	var ticker = time.NewTicker(chains.interval)
	var wg sync.WaitGroup
	var lck sync.Mutex
	var forwards = list.New[*Chain]()
	for {
		var waitCtx = context.Background()
		var waitCancel context.CancelFunc
		var finished bool
		var done = make(chan struct{})
		if chains.timeout > 0 {
			waitCtx, waitCancel = context.WithTimeout(waitCtx, chains.timeout)
		}
		for _, chain := range chains.chains.ToArray() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					case <-waitCtx.Done():
						if waitCtx.Err() != nil {
							chains.errors.Push(
								fmt.Errorf("%s('%s') %w",
									chain.Type,
									chain.Expr,
									waitCtx.Err()),
							)
						}
						return
					case <-ticker.C:
					default:
						if chain.friend != nil {
							if _, ok := chain.friend(); ok {
								chain.friend = nil
							}
						}
					}
					lck.Lock()
					if finished {
						lck.Unlock()
						break
					}
					chain.retried++
					err := chain.call()
					if err != nil {
						if errors.Is(err, ErrChainSkip) {
							lck.Unlock()
							continue
						}
						chains.errors.PushL(err)
						//callback error can finish chain when found elements
						if len(chain.elements) == 0 {
							lck.Unlock()
							break
						}
					}
					if chains.ref == Any {
						if !finished {
							finished = true
							close(done)
						}
					}
					lck.Unlock()
					if len(chain.forwards) > 0 {
						if chain.forwarded >= chains.maxForward {
							chain.forwarded = 0
						} else {
							forwards.PushR(chain.forwards...)
						}
					}
					if chain.friend != nil {
						if c, ok := chain.friend(); !ok {
							c()
							chain.friend = nil
						}
					}
					break
				}
			}()
			if !chains.async {
				wg.Wait()
			}
		}
		wg.Wait()
		waitCancel()
		if forwards.Size() == 0 {
			break
		}
		chains.errors.Clear()
		chains.chains.Clear()
		for _, forward := range forwards.ToArray() {
			if !chains.chains.Contain(forward) {
				forward.forwarded++
				chains.chains.Push(forward)
			}
		}
		forwards.Clear()
	}
	if chains.errors.Size() > 0 {
		return chains.errors.PopL()
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

func (bitPage *Page) WaitSelector(expr string, callback ChainCallback) error {
	return bitPage.Chains().Selector(expr, callback).Wait()
}
func (bitPage *Page) WaitXPath(expr string, callback ChainCallback) error {
	return bitPage.Chains().XPath(expr, callback).Wait()
}
func (bitPage *Page) WaitJs(expr string, args []any, callback ChainCallback) error {
	return bitPage.Chains().Js(expr, args, callback).Wait()
}
func (bitPage *Page) Chains() *Chains {
	return &Chains{
		page:       bitPage.page,
		interval:   time.Second,
		timeout:    time.Minute,
		maxForward: 1,
		async:      true,
		ref:        Any,
		chains:     list.New[*Chain](),
		errors:     list.New[error](),
		opts:       option.New(nil),
	}
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
func (bitPage *Page) WaitLoad() error {
	err := bitPage.page.WaitLoad()
	if err == nil {
		bitPage.browser.dispatchOnLoadHookEvt(bitPage, bitPage.browser.makeCtx())
	}
	return err
}
func (bitPage *Page) Close() error {
	return bitPage.page.Close()
}
func (bitPage *Page) Timeout(timeout time.Duration) {
	if timeout > 0 {
		bitPage.page = bitPage.page.Timeout(timeout)
	}
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
	for _, conn := range browser.onConnects {
		conn(ctx)
	}
}
func (browser *Browser) dispatchOnLoadHookEvt(page *Page, ctx *Ctx) {
	for _, onload := range browser.onloads {
		onload(page, ctx)
	}
}
func (browser *Browser) dispatchOnRequestHookEvt(req *rod.Hijack, ctx *Ctx) {
	for _, onReq := range browser.onRequests {
		onReq(req, ctx)
	}
}
func (browser *Browser) dispatchOnCloseHookEvt(ctx *Ctx) {
	for _, onClose := range browser.onCloses {
		onClose(ctx)
	}

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
			err = errors.New("there is no default page")
		} else {
			defaultPage = pages.First()
		}
	}
	if err != nil {
		return nil, err
	}
	return defaultPage.Context(browser.ctx), defaultPage.Navigate(source)
}
func (browser *Browser) reset() {
	if browser.rod != nil {
		browser.dispatchOnCloseHookEvt(browser.makeCtx())
		browser.hookMap.Clear()
		browser.Meta.Clear()
		if browser.router != nil {
			browser.router.Stop()
			browser.router = nil
		}
		if browser.browser != nil {
			browser.browser.Kill()
		}
		browser.browser = nil
		browser.rod = nil
	}
}
func (browser *Browser) OnRequest(req ...OnRequest) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.onRequests = append(browser.onRequests, req...)
}
func (browser *Browser) OnConnect(conn ...OnConnect) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.onConnects = append(browser.onConnects, conn...)
}
func (browser *Browser) OnClose(close ...OnClose) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.onCloses = append(browser.onCloses, close...)
}
func (browser *Browser) OnLoad(load ...OnLoad) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.onloads = append(browser.onloads, load...)
}
func (browser *Browser) AddInterceptor(pattern string, network proto.NetworkResourceType) {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	browser.interceptors[pattern] = network
}
func (browser *Browser) Ready() bool {
	browser.lck.Lock()
	defer browser.lck.Unlock()
	return browser.rod != nil
}
func (browser *Browser) Pid() int {
	if browser.browser != nil {
		return browser.browser.Pid()
	}
	return -1
}
func (browser *Browser) Cleanup() {
	if browser.browser != nil {
		browser.browser.Cleanup()
	}
}
func (browser *Browser) Kill() {
	if browser.browser != nil {
		browser.browser.Kill()
	}
}
func (browser *Browser) Hook(name string) *hook.Hook {
	return browser.hookMap.New(name)
}
func (browser *Browser) Connect() error {
	if browser.rod == nil {
		r := rod.New().NoDefaultDevice()
		c, err := browser.browser.Open()
		if err != nil {
			return err
		}
		err = r.ControlURL(c).Connect()
		if err != nil {
			return err
		}
		browser.rod = r
		if len(browser.interceptors) > 0 {
			browser.router = browser.rod.HijackRequests()
			for pattern, network := range browser.interceptors {
				err = browser.router.Add(pattern, network, browser.dispatchRequest)
				if err != nil {
					return err
				}
			}
			go browser.router.Run()
		}
		browser.dispatchOnConnectHookEvt(browser.makeCtx())
	}
	return nil
}
func (browser *Browser) Blank() (*Page, error) {
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
