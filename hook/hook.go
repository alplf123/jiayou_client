package hook

import (
	"errors"
	"sync"
	"time"
)

type Hook struct {
	s chan struct{}
}

func (ctx *Hook) Reset() {
	if ctx.s != nil {
		close(ctx.s)
	}
	ctx.s = make(chan struct{}, 1)
}
func (ctx *Hook) Stop() {
	if ctx.s == nil {
		return
	}
	select {
	case ctx.s <- struct{}{}:
		return
	default:
	}
}
func (ctx *Hook) Wait(timeout time.Duration) error {
	if ctx.s == nil {
		return nil
	}
	if timeout <= 0 {
		<-ctx.s
		return nil
	}
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		return errors.New("hook timeout")
	case <-ctx.s:
	}
	return nil
}

type Hooks struct {
	hooks map[string]*Hook
	lck   sync.Mutex
}

func (hmap *Hooks) Get(name string) *Hook {
	hmap.lck.Lock()
	defer hmap.lck.Unlock()
	if val, ok := hmap.hooks[name]; ok {
		return val
	}
	return nil
}
func (hmap *Hooks) New(name string) *Hook {
	hmap.lck.Lock()
	defer hmap.lck.Unlock()
	hmap.hooks[name] = &Hook{s: make(chan struct{}, 1)}
	return hmap.hooks[name]
}
func (hmap *Hooks) Clear() {
	hmap.lck.Lock()
	defer hmap.lck.Unlock()
	for k := range hmap.hooks {
		delete(hmap.hooks, k)
	}
}

func NewHooks() *Hooks {
	return &Hooks{hooks: make(map[string]*Hook)}
}
