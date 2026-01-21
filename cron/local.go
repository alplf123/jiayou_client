package cron

import (
	"context"
	"fmt"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/utils"
	"jiayou_backend_spider/utils/retry"
	"slices"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type cronLogger struct {
	logger *zap.Logger
}
type LFilterFunc func(*LTask) bool

func (cron *cronLogger) formatField(values ...any) []zap.Field {
	var fields []zap.Field
	for i := 0; i < len(values); i += 2 {
		if v, ok := values[i].(string); ok {
			fields = append(fields, zap.Any(v, values[i+1]))
		}
	}
	return fields
}
func (cron *cronLogger) Info(msg string, keysAndValues ...any) {
	cron.logger.Info(msg, cron.formatField(keysAndValues...)...)
}

func (cron *cronLogger) Error(err error, msg string, keysAndValues ...any) {
	cron.logger.Error(msg, slices.Insert(cron.formatField(keysAndValues...), 0, zap.Error(err))...)
}

type LOptions struct {
	Ctx             context.Context
	Logger          *zap.Logger
	Location        string `json:"location" default:"Local"`
	EnableSeconds   bool   `json:"enable_seconds" default:"true"`
	SkipRunningTask bool   `json:"skip_running_task"`
}

type LTaskOptions struct {
	Func          func(ctx context.Context) error
	Name          string        `json:"name" validate:"required"`
	Spec          string        `json:"spec" validate:"required"`
	Retry         int64         `json:"retry" validate:"gte=0"`
	RetryInterval time.Duration `json:"retry_interval" validate:"gte=0" default:"1s"`
}
type LTask struct {
	context   context.Context
	opts      *LTaskOptions
	running   bool
	completed int64
	err       error
	ctx       context.Context
	entry     cron.Entry
	call      func(context.Context) error
}

func (task *LTask) Options() *LTaskOptions {
	return task.opts
}
func (task *LTask) Running() bool {
	return task.running
}
func (task *LTask) Completed() int64 {
	return task.completed
}
func (task *LTask) Id() string {
	return strconv.Itoa(int(task.entry.ID))
}
func (task *LTask) Valid() bool {
	return task.entry.Valid()
}
func (task *LTask) LastErr() error {
	return task.err
}
func (task *LTask) Next() time.Time {
	return task.entry.Next
}
func (task *LTask) Prev() time.Time {
	return task.entry.Prev
}
func (task *LTask) Run() {
	task.running = true
	defer func() { task.running = false }()
	if task.opts.Retry > 0 && task.opts.RetryInterval > 0 {
		_, err := retry.Retry(func(c *retry.Context) (any, error) {
			var err = task.call(task.ctx)
			return nil, err
		}, retry.WithCtx(task.ctx),
			retry.WithMaxRetry(task.opts.Retry),
			retry.WithBackoffFunc(utils.FixedDuration(task.opts.RetryInterval)),
		)
		task.err = err
	} else {
		task.err = task.call(task.ctx)
	}
	if task.err == nil {
		task.completed++
	}
}

type Local struct {
	opts *LOptions
	cron *cron.Cron
}

func (local *Local) Add(tasks ...*LTaskOptions) error {
	for _, t := range tasks {
		if _, err := config.Validate(t); err != nil {
			return err
		}
		if local.ContainF(func(task *LTask) bool {
			return task.opts.Name == t.Name
		}) {
			return fmt.Errorf("the local task[%s] existed", t.Name)
		}
		var f = t.Func
		if f == nil {
			f = func(ctx context.Context) error {
				local.opts.Logger.Warn(fmt.Sprintf("the local[%s], it`s a default local task function, please replace it", t.Name))
				return nil
			}
		}
		var task = &LTask{
			ctx:  local.opts.Ctx,
			call: f,
			opts: t,
		}
		entryId, err := local.cron.AddJob(t.Spec, task)
		if err != nil {
			return err
		}
		task.entry = local.cron.Entry(entryId)
	}
	return nil
}
func (local *Local) Get(name string) (*LTask, error) {
	for _, entry := range local.cron.Entries() {
		if entry.Valid() {
			var task = entry.Job.(*LTask)
			if task.opts.Name == name {
				return task, nil
			}
		}
	}
	return nil, fmt.Errorf("local[%s] task not existed", name)
}
func (local *Local) Locals() []*LTask {
	var tasks []*LTask
	for _, entry := range local.cron.Entries() {
		var task = entry.Job.(*LTask)
		tasks = append(tasks, task)
	}
	return tasks
}
func (local *Local) Contain(task *LTask) bool {
	if task == nil || !task.Valid() {
		return false
	}
	return local.ContainF(func(t *LTask) bool {
		return task == t
	})
}
func (local *Local) ContainF(filter LFilterFunc) bool {
	for _, entry := range local.cron.Entries() {
		if entry.Valid() {
			var task = entry.Job.(*LTask)
			if filter(task) {
				return true
			}
		}
	}
	return false
}
func (local *Local) Remove(task *LTask) {
	if task == nil || !task.Valid() {
		return
	}
	local.RemoveF(func(t *LTask) bool {
		return task == t
	})
}
func (local *Local) RemoveF(filter LFilterFunc) {
	var removeId []cron.EntryID
	for _, entry := range local.cron.Entries() {
		var task = entry.Job.(*LTask)
		if filter(task) {
			removeId = append(removeId, entry.ID)
		}
	}
	for _, id := range removeId {
		local.cron.Remove(id)
	}
}
func (local *Local) Size() int {
	return len(local.cron.Entries())
}
func (local *Local) Run() {
	local.cron.Run()
}
func (local *Local) Start() {
	local.cron.Start()
}
func (local *Local) Stop() {
	local.cron.Stop()
}

func DefaultLOptions() *LOptions {
	return config.TryValidate(&LOptions{})
}
func DefaultLTaskOptions() *LTaskOptions {
	return config.TryValidate(&LTaskOptions{})
}

func NewL(opts *LOptions) (*Local, error) {
	if opts == nil {
		opts = DefaultLOptions()
	}
	var cronOpts []cron.Option
	location, err := time.LoadLocation(opts.Location)
	if err != nil {
		return nil, err
	}
	cronOpts = append(cronOpts, cron.WithLocation(location))
	if opts.Logger == nil {
		opts.Logger = zap.L()
	}
	var logger = &cronLogger{opts.Logger}
	cronOpts = append(cronOpts, cron.WithLogger(logger))
	if opts.EnableSeconds {
		cronOpts = append(cronOpts, cron.WithSeconds())
	}
	var jobWrappers = []cron.JobWrapper{cron.Recover(logger)}
	if opts.SkipRunningTask {
		jobWrappers = append(jobWrappers, cron.SkipIfStillRunning(logger))
	} else {
		jobWrappers = append(jobWrappers, cron.DelayIfStillRunning(logger))
	}
	cronOpts = append(cronOpts, cron.WithChain(jobWrappers...))
	return &Local{cron: cron.New(cronOpts...), opts: opts}, nil
}
