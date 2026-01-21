package eventx

import (
	"errors"
	"fmt"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/errorx"
	"jiayou_backend_spider/list"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/utils/wait"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/panjf2000/ants/v2"
)

var defaultCallback = func(*EventCtx, ...any) {}

type handler struct {
	runOnce, ret bool
	evt          *Event
	meta         *option.Option
	call         any
}
type Options struct {
	Concurrent int `json:"concurrent" validator:"gt=0"`
}
type EventId int64
type EventCtx struct {
	topic             string
	args              []any
	call              func(*EventCtx) []reflect.Value
	aborted, finished bool
	evt               *Event
	meta              *option.Option
}

func (ctx *EventCtx) Topic() string        { return ctx.topic }
func (ctx *EventCtx) Abort()               { ctx.aborted = true }
func (ctx *EventCtx) Meta() *option.Option { return ctx.meta }

type EventFunc any
type OnPanicFunc func(ctx *EventCtx, err error)
type Event struct {
	id EventId
	c  chan EventValues
}

func (evt *Event) Id() EventId {
	return evt.id
}
func (evt *Event) Wait() EventValues {
	return <-evt.c
}
func (evt *Event) TryWait() EventValues {
	select {
	case e := <-evt.c:
		return e
	default:
	}
	return nil
}

type EventValues []reflect.Value

func (values EventValues) ValueOf(i int) (any, error) {
	if i < 0 || len(values)-1 < i {
		return nil, fmt.Errorf("value out of range index[%d]", i)
	}
	return values[i].Interface(), nil
}
func (values EventValues) MustValueOf(i int) any {
	if v, err := values.ValueOf(i); err != nil {
		panic(err)
	} else {
		return v
	}
}
func (values EventValues) Size() int {
	return len(values)
}

type EventX struct {
	opts      *Options
	lck       sync.Mutex
	handlers  map[string]*list.List[handler]
	panicFunc OnPanicFunc
	evtPool   *ants.PoolWithFuncGeneric[*EventCtx]
	id        int64
}

func (event *EventX) deleteHandler(topic string, evtId EventId) {
	event.lck.Lock()
	defer event.lck.Unlock()
	if v, ok := event.handlers[topic]; ok {
		v.RemoveF(func(h handler) bool {
			return h.evt.id == evtId
		})
	}
}
func (event *EventX) dispatchEvent(ctx *EventCtx) {
	var retValues = ctx.call(ctx)
	ctx.finished = true
	if ctx.aborted {
		event.deleteHandler(ctx.topic, ctx.evt.id)
	}
	select {
	case ctx.evt.c <- EventValues(retValues):
	default:
	}
}
func (event *EventX) call(ctx *EventCtx, f any) []reflect.Value {
	var ff = reflect.TypeOf(f)
	var values = []reflect.Value{reflect.ValueOf(ctx)}
	var args = ctx.args
	var typ reflect.Type
	if !ff.IsVariadic() {
		var argNum = ff.NumIn() - 1
		if len(args) > argNum {
			args = args[:argNum]
		}
	} else {
		typ = ff.In(ff.NumIn() - 1).Elem()
	}
	for i, arg := range args {
		if arg == nil {
			if typ != nil {
				values = append(values, reflect.New(typ).Elem())
			} else {
				values = append(values, reflect.New(ff.In(i+1)).Elem())
			}
		} else {
			values = append(values, reflect.ValueOf(arg))
		}
	}
	return reflect.ValueOf(f).Call(values)
}
func (event *EventX) panic(ctx *EventCtx, err error) {
	event.lck.Lock()
	defer event.lck.Unlock()
	if event.panicFunc != nil {
		event.panicFunc(ctx, err)
	}
}
func (event *EventX) handlePanic(f any) func(*EventCtx) []reflect.Value {
	return func(ctx *EventCtx) []reflect.Value {
		defer func() {
			v := recover()
			if v != nil {
				var msg error
				switch vv := v.(type) {
				case error:
					msg = fmt.Errorf("event[%s] call[0x%x] exception, %w",
						ctx.Topic(),
						uintptr(unsafe.Pointer(&f)),
						errorx.Verbose().Error(vv))
				default:
					msg = fmt.Errorf("event[%s] call[0x%x] exception, %#v",
						ctx.Topic(),
						uintptr(unsafe.Pointer(&f)),
						vv)
				}
				event.panic(ctx, msg)
			}
		}()
		return event.call(ctx, f)
	}
}
func (event *EventX) doPublish(topic string, waitable bool, timeout time.Duration, args ...any) error {
	event.lck.Lock()
	defer event.lck.Unlock()
	if handlers, ok := event.handlers[topic]; ok && handlers.Size() > 0 {
		var needDeletedHandlers []int
		var needWaitCtxs []*EventCtx
		for i := 0; i < handlers.Size(); i++ {
			handler := handlers.Index(i)
			ctx := &EventCtx{
				topic:    topic,
				args:     args,
				call:     event.handlePanic(handler.call),
				finished: false,
				evt:      handler.evt,
				meta:     handler.meta,
			}
			if err := event.evtPool.Invoke(ctx); err != nil {
				return fmt.Errorf("dispatch event[%s] handlers[%d] err,%w",
					topic,
					uintptr(unsafe.Pointer(&handler.call)), err)
			} else {
				if handler.runOnce {
					needDeletedHandlers = append(needDeletedHandlers, i)
				}
				if waitable {
					needWaitCtxs = append(needWaitCtxs, ctx)
				}
			}
		}
		for _, ctx := range needWaitCtxs {
			if _, err := wait.Wait(func(r *wait.Context) (any, bool) {
				return nil, ctx.finished
			}, wait.WithTimeout(timeout)); err != nil {
				return err
			}
		}
		for _, index := range needDeletedHandlers {
			handlers.RemoveI(index)
		}
	}
	return nil
}
func (event *EventX) doSubscribe(topic string, f EventFunc, once bool) (*Event, error) {
	event.lck.Lock()
	defer event.lck.Unlock()
	if f == nil {
		f = defaultCallback
	}
	var fType = reflect.TypeOf(f)
	if fType.Kind() != reflect.Func {
		return nil, errors.New("bad subscribe function")
	}
	if fType.NumIn() == 0 || fType.In(0) != reflect.TypeOf(&EventCtx{}) {
		return nil, errors.New("the first parameter of the subscribe function must be *EventCtx")
	}
	if _, ok := event.handlers[topic]; !ok {
		event.handlers[topic] = list.New[handler]()
	}
	var handler = handler{
		runOnce: once,
		evt:     &Event{id: EventId(atomic.AddInt64(&event.id, 1)), c: make(chan EventValues, 1)},
		meta:    option.New(nil),
		call:    f,
	}
	event.handlers[topic].Push(handler)
	return handler.evt, nil
}
func (event *EventX) Subscribe(topic string, f EventFunc) (*Event, error) {
	return event.doSubscribe(topic, f, false)
}
func (event *EventX) SubscribeOnce(topic string, f EventFunc) (*Event, error) {
	return event.doSubscribe(topic, f, true)
}
func (event *EventX) Publish(topic string, args ...any) error {
	return event.doPublish(topic, false, 0, args...)
}
func (event *EventX) UnSubscribe(topic string, id EventId) {
	event.deleteHandler(topic, id)
}
func (event *EventX) Clear() {
	event.lck.Lock()
	defer event.lck.Unlock()
	for topic, handlers := range event.handlers {
		handlers.Clear()
		delete(event.handlers, topic)
	}
}
func (event *EventX) OnPanic(f OnPanicFunc) {
	if f == nil {
		return
	}
	event.lck.Lock()
	defer event.lck.Unlock()
	event.panicFunc = f
}
func (event *EventX) MustPublish(topic string, timeout time.Duration, args ...any) {
	if err := event.doPublish(topic, true, timeout, args...); err != nil {
		panic(err)
	}
}
func (event *EventX) TryPublish(topic string, args ...any) {
	event.doPublish(topic, false, 0, args...)
}
func (event *EventX) MustSubscribe(topic string, f EventFunc) *Event {
	if evt, err := event.Subscribe(topic, f); err != nil {
		panic(err)
	} else {
		return evt
	}
}
func (event *EventX) MustSubscribeOnce(topic string, f EventFunc) *Event {
	if evt, err := event.SubscribeOnce(topic, f); err != nil {
		panic(err)
	} else {
		return evt
	}
}
func DefaultOptions() *Options {
	return config.TryValidate(&Options{Concurrent: runtime.NumCPU()})
}
func New(opts *Options) *EventX {
	if opts == nil {
		opts = DefaultOptions()
	}
	var evt = &EventX{
		opts:     opts,
		handlers: make(map[string]*list.List[handler]),
	}
	evt.evtPool, _ = ants.NewPoolWithFuncGeneric[*EventCtx](opts.Concurrent, evt.dispatchEvent)
	return evt
}
