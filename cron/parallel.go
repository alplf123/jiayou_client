package cron

import (
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/config"
	"slices"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/panjf2000/ants/v2"
)

type ParallelFilter func(info *DTaskInfo) bool

type ParallelOptions struct {
	Ctx          context.Context
	Ref          ParallelRef   `json:"ref" default:"any" validate:"oneof=any all"`
	WaitTimeout  time.Duration `json:"wait_timeout" validate:"gte=0"`
	EachInterval time.Duration `json:"each_interval" default:"100ms" validate:"gt=0"`
	Concurrent   int           `json:"concurrent" default:"1" validate:"gt=0"`
}
type ParallelRef string

const (
	Any ParallelRef = "any"
	All ParallelRef = "all"
)

type ParallelResult struct {
	errs  []error
	infos []*DTaskInfo
}

func (result *ParallelResult) pushErr(err error) {
	if err != nil {
		result.errs = append(result.errs, err)
	}
}
func (result *ParallelResult) Size() int {
	return len(result.infos)
}
func (result *ParallelResult) First() *DTaskInfo {
	if len(result.infos) > 0 {
		return result.infos[0]
	}
	return nil
}
func (result *ParallelResult) Last() *DTaskInfo {
	if len(result.infos) > 0 {
		return result.infos[len(result.infos)-1]
	}
	return nil
}
func (result *ParallelResult) Filter(filter ParallelFilter) *ParallelResult {
	var _infos = new(ParallelResult)
	for _, info := range result.infos {
		if filter(info) {
			_infos.infos = append(_infos.infos, info)
		}
	}
	_infos.errs = result.errs
	return _infos
}
func (result *ParallelResult) Completed() *ParallelResult {
	return result.Filter(func(info *DTaskInfo) bool {
		return info.State == TaskStateCompleted
	})
}
func (result *ParallelResult) Archived() *ParallelResult {
	return result.Filter(func(info *DTaskInfo) bool {
		return info.State == TaskStateArchived
	})
}
func (result *ParallelResult) Err() error {
	if len(result.errs) > 0 {
		return result.errs[len(result.errs)-1]
	}
	return nil
}

type Parallel struct {
	opts   *ParallelOptions
	client *Client
	tasks  []*DTask
}

func (parallel *Parallel) wait(filter ParallelFilter) ParallelResult {
	var opts = parallel.opts
	var ctx = opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	var result = ParallelResult{}
	var pool, err = ants.NewPool(opts.Concurrent)
	if err != nil {
		result.pushErr(err)
		return result
	}
	var infos []*DTaskInfo
	for i := 0; i < len(parallel.tasks); i++ {
		info, err := parallel.client.enqueue(ctx, parallel.tasks[i])
		if err != nil {
			result.pushErr(err)
			return result
		}
		infos = append(infos, info)
	}
	var done = make(chan struct{})
	var finished bool
	var ticker = time.NewTicker(opts.EachInterval)
	var wg sync.WaitGroup
	var lck sync.Mutex
	for _, info := range infos {
		wg.Add(1)
		err := pool.Submit(func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				var i, err = parallel.client.inspector.GetTaskInfo(info.Queue, info.ID)
				if err != nil {
					if errors.Is(err, asynq.ErrTaskNotFound) {
						return
					}
					result.pushErr(fmt.Errorf("inspector failed,%w", err))
					return
				}
				if filter(i) {
					lck.Lock()
					result.infos = append(result.infos, i)
					if opts.Ref == Any {
						if !finished {
							close(done)
							finished = true
						}
					}
					lck.Unlock()
					return
				}
			}
		})
		if err != nil {
			result.pushErr(err)
			return result
		}
	}
	if opts.WaitTimeout > 0 {
		var waiter = time.NewTicker(opts.WaitTimeout)
		select {
		case <-done:
		case <-waiter.C:
			cancel()
			return result
		}
	}
	wg.Wait()
	return result
}
func (parallel *Parallel) Ref(ref ParallelRef) {
	if parallel.opts != nil {
		parallel.opts.Ref = ref
	}
}
func (parallel *Parallel) Wait(filter ParallelFilter) ParallelResult {
	if filter == nil {
		return ParallelResult{}
	}
	return parallel.wait(filter)
}
func (parallel *Parallel) WaitDone() ParallelResult {
	return parallel.WaitState(TaskStateCompleted, TaskStateArchived)
}
func (parallel *Parallel) WaitState(states ...TaskState) ParallelResult {
	return parallel.wait(func(task *DTaskInfo) bool {
		return slices.Contains(states, task.State)
	})
}
func (parallel *Parallel) WaitCompleted() ParallelResult {
	return parallel.WaitState(TaskStateCompleted)
}
func (parallel *Parallel) WaitArchived() ParallelResult {
	return parallel.WaitState(TaskStateArchived)
}

func DefaultParallelOptions() *ParallelOptions {
	return config.TryValidate(&ParallelOptions{})
}
