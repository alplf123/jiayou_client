package cron

import (
	"context"
	"fmt"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/list"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var levels = map[zapcore.Level]asynq.LogLevel{
	zap.DebugLevel: asynq.DebugLevel,
	zap.InfoLevel:  asynq.InfoLevel,
	zap.WarnLevel:  asynq.WarnLevel,
	zap.ErrorLevel: asynq.ErrorLevel,
	zap.FatalLevel: asynq.FatalLevel,
}

type DFilterFunc func(*DCore) bool

type DServerOptions struct {
	Logger                   *zap.Logger
	ErrorHandler             ErrorHandlerFunc
	RetryDelayFunc           RetryDelayFunc
	ShouldRetry              func(error) bool
	HealthCheckFunc          func(error)
	IsFailure                func(error) bool
	Concurrency              int            `json:"concurrency"`
	TaskCheckInterval        time.Duration  `json:"task_check_interval"`
	Queues                   map[string]int `json:"queues"`
	StrictPriority           bool           `json:"strict_priority"`
	ShutdownTimeout          time.Duration  `json:"shutdown_timeout"`
	HealthCheckInterval      time.Duration  `json:"health_check_interval"`
	DelayedTaskCheckInterval time.Duration  `json:"delayed_task_check_interval"`
	GroupGracePeriod         time.Duration  `json:"group_grace_period"`
	GroupMaxDelay            time.Duration  `json:"group_max_delay"`
	GroupMaxSize             int            `json:"group_max_size"`
	JanitorInterval          time.Duration  `json:"janitor_interval"`
	JanitorBatchSize         int            `json:"janitor_batch_size"`
}

func (option *DServerOptions) Translate() (asynq.Config, error) {
	var conf = asynq.Config{
		HealthCheckFunc:          option.HealthCheckFunc,
		Concurrency:              option.Concurrency,
		IsFailure:                option.IsFailure,
		TaskCheckInterval:        option.TaskCheckInterval,
		Queues:                   option.Queues,
		StrictPriority:           option.StrictPriority,
		ShutdownTimeout:          option.ShutdownTimeout,
		HealthCheckInterval:      option.HealthCheckInterval,
		DelayedTaskCheckInterval: option.DelayedTaskCheckInterval,
		GroupGracePeriod:         option.GroupGracePeriod,
		GroupMaxDelay:            option.GroupMaxDelay,
		GroupMaxSize:             option.GroupMaxSize,
		JanitorInterval:          option.JanitorInterval,
		JanitorBatchSize:         option.JanitorBatchSize,
	}
	if option.Logger == nil {
		option.Logger = zap.L()
	}
	conf.Logger = option.Logger.Sugar()
	v, ok := levels[option.Logger.Level()]
	if !ok {
		v = asynq.InfoLevel
	}
	conf.LogLevel = v
	if option.ErrorHandler != nil {
		conf.ErrorHandler = asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			option.ErrorHandler(ctx, &Task{task: task}, err)
		})
	}
	if option.RetryDelayFunc != nil {
		conf.RetryDelayFunc = func(n int, e error, t *asynq.Task) time.Duration {
			return option.RetryDelayFunc(n, e, &Task{task: t})
		}
	}
	return conf, nil
}

type DScheduler struct {
	id   string
	Spec string `json:"spec" validate:"required"`
	Task *DTask `json:"task" validate:"required"`
}

func (scheduler *DScheduler) Id() string {
	return scheduler.id
}
func (scheduler *DScheduler) Name() string {
	return scheduler.Task.Name
}

type DSchedulerOptions struct {
	Logger            *zap.Logger
	PreEnqueueFunc    func(task *Task)
	PostEnqueueFunc   func(info *DTaskInfo, err error)
	PayloadLimit      int64         `json:"payload_Limit" validate:"gt=0"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval" validate:"gte=0" default:"10s"`
	Location          string        `json:"location" validate:"required" default:"Local"`
}

func (opt *DSchedulerOptions) Translate() (*asynq.SchedulerOpts, error) {
	var location, err = time.LoadLocation(opt.Location)
	if err != nil {
		return nil, err
	}
	if opt.Logger == nil {
		opt.Logger = zap.L()
	}
	var schedulerOpts = &asynq.SchedulerOpts{
		HeartbeatInterval: opt.HeartbeatInterval,
		Logger:            opt.Logger.Sugar(),
		Location:          location,
	}
	if v, ok := levels[opt.Logger.Level()]; !ok {
		return nil, fmt.Errorf("scheduler unsupported level[%s]", opt.Logger.Level().String())
	} else {
		schedulerOpts.LogLevel = v
	}
	if opt.PreEnqueueFunc != nil {
		schedulerOpts.PreEnqueueFunc = func(task *asynq.Task, opts []asynq.Option) {
			opt.PreEnqueueFunc(&Task{task: task})
		}
	}
	if opt.PostEnqueueFunc != nil {
		schedulerOpts.PostEnqueueFunc = func(info *asynq.TaskInfo, err error) {
			var _task DTaskInfo
			if err := copier.Copy(&_task, info); err != nil {
				schedulerOpts.Logger.Error(fmt.Errorf("the task[%s][%s] copy failed,%w", info.ID, info.Type, err))
			}
			opt.PostEnqueueFunc(&_task, err)
		}
	}
	return schedulerOpts, nil
}

type DTask struct {
	Name         string       `json:"name" validate:"required"`
	Payload      any          `json:"payload"`
	PayloadLimit int64        `json:"payload_Limit" validate:"gte=0"`
	Options      *DTaskOption `json:"options"`
}

type DTaskOption struct {
	MaxRetry  int           `json:"max_retry"`
	Queue     string        `json:"queue"`
	TaskID    string        `json:"task_id"`
	Timeout   time.Duration `json:"timeout"`
	Deadline  time.Time     `json:"deadline"`
	Unique    time.Duration `json:"unique"`
	ProcessAt time.Time     `json:"process_at"`
	ProcessIn time.Duration `json:"process_in"`
	Retention time.Duration `json:"retention"`
	Group     string        `json:"group"`
}

func (opt *DTaskOption) UnTranslate(opts []asynq.Option) {
	for _, _opt := range opts {
		switch _opt.Type() {
		case asynq.MaxRetryOpt:
			opt.MaxRetry = _opt.Value().(int)
		case asynq.QueueOpt:
			opt.Queue = _opt.Value().(string)
		case asynq.TaskIDOpt:
			opt.TaskID = _opt.Value().(string)
		case asynq.TimeoutOpt:
			opt.Timeout = _opt.Value().(time.Duration)
		case asynq.DeadlineOpt:
			opt.Deadline = _opt.Value().(time.Time)
		case asynq.UniqueOpt:
			opt.Unique = _opt.Value().(time.Duration)
		case asynq.ProcessAtOpt:
			opt.ProcessAt = _opt.Value().(time.Time)
		case asynq.ProcessInOpt:
			opt.ProcessIn = _opt.Value().(time.Duration)
		case asynq.RetentionOpt:
			opt.Retention = _opt.Value().(time.Duration)
		case asynq.GroupOpt:
			opt.Group = _opt.Value().(string)
		}

	}
}

func (opt *DTaskOption) Translate() ([]asynq.Option, error) {
	var opts []asynq.Option
	opts = append(opts, asynq.MaxRetry(opt.MaxRetry))
	if opt.Queue != "" {
		opts = append(opts, asynq.Queue(opt.Queue))
	}
	if opt.TaskID != "" {
		opts = append(opts, asynq.TaskID(opt.TaskID))
	}
	if opt.Timeout > 0 {
		opts = append(opts, asynq.Timeout(opt.Timeout))
	}
	if !opt.Deadline.IsZero() {
		opts = append(opts, asynq.Deadline(opt.Deadline))
	}
	if opt.Unique > 0 {
		opts = append(opts, asynq.Unique(opt.Unique))
	}
	if !opt.ProcessAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(opt.ProcessAt))
	}
	if opt.ProcessIn > 0 {
		opts = append(opts, asynq.ProcessIn(opt.ProcessIn))
	}
	if opt.Retention > 0 {
		opts = append(opts, asynq.Retention(opt.Retention))
	}
	if opt.Group != "" {
		opts = append(opts, asynq.Group(opt.Group))
	}
	return opts, nil
}

type DOptions struct {
	Ctx    context.Context
	Logger *zap.Logger
}

type DTaskOptions struct {
	Ctx              context.Context
	Uri              string             `json:"uri" validate:"required"`
	Name             string             `json:"name" validate:"required"`
	Tasks            []*DTask           `json:"tasks"`
	Schedulers       []*DScheduler      `json:"schedulers"`
	ServerOptions    *DServerOptions    `json:"server_options"`
	SchedulerOptions *DSchedulerOptions `json:"scheduler_options"`
}

type DCore struct {
	distributed *Distributed
	opts        *DTaskOptions
	server      *Server
	client      *Client
	scheduler   *Scheduler
	inspector   *Inspector
	running     bool
}

func (core *DCore) Running() bool {
	return core.running
}
func (core *DCore) Scheduler() *Scheduler {
	return core.scheduler
}
func (core *DCore) Server() *Server {
	return core.server
}
func (core *DCore) Client() *Client {
	return core.client
}
func (core *DCore) Inspector() *Inspector {
	return core.inspector
}
func (core *DCore) Shutdown() {
	if !core.running {
		return
	}
	core.client.close()
	core.scheduler.shutdown()
	core.server.shutdown()
	core.inspector.Close()
	core.running = false
}

type Distributed struct {
	opts  *DOptions
	items *list.List[*DCore]
}

func (distributed *Distributed) newCore(opts *DTaskOptions) (*DCore, error) {
	var core = &DCore{
		distributed: distributed,
		opts:        opts,
	}
	core.opts.Ctx = distributed.opts.Ctx
	if opts.SchedulerOptions != nil {
		opts.SchedulerOptions.Logger = distributed.opts.Logger
	}
	if opts.ServerOptions != nil {
		opts.ServerOptions.Logger = distributed.opts.Logger
	}
	if scheduler, err := newScheduler(opts); err != nil {
		return nil, err
	} else {
		if err := scheduler.start(); err != nil {
			return nil, fmt.Errorf("scheduler start failed[%w]", err)
		}
		core.scheduler = scheduler
	}
	if server, err := newServer(opts); err != nil {
		return nil, err
	} else {
		//if err := server.start(); err != nil {
		//	return nil, fmt.Errorf("server start failed[%w]", err)
		//}
		core.server = server
	}
	if inspector, err := newInspector(opts); err != nil {
		return nil, err
	} else {
		core.inspector = inspector
		if client, err := newClient(opts); err != nil {
			return nil, err
		} else {
			client.inspector = inspector
			core.client = client
		}
	}
	core.running = true
	return core, nil
}
func (distributed *Distributed) Add(tasks ...*DTaskOptions) error {
	for _, t := range tasks {
		if _, err := config.Validate(t); err != nil {
			return err
		}
		if distributed.items.ContainF(func(core *DCore) bool {
			return core.opts.Name == t.Name
		}) {
			return fmt.Errorf("the task core[%s] existed", t.Name)
		}
		if core, err := distributed.newCore(t); err != nil {
			return err
		} else {
			distributed.items.Push(core)
		}
	}
	return nil
}
func (distributed *Distributed) Get(name string) (*DCore, error) {
	var cores = distributed.items.Filter(func(d *DCore) bool {
		return d.opts.Name == name
	})
	if len(cores) == 0 {
		return nil, fmt.Errorf("task core[%s] not existed", name)
	}
	return cores[0], nil
}
func (distributed *Distributed) First() *DCore {
	return distributed.items.Top()
}
func (distributed *Distributed) Last() *DCore {
	return distributed.items.Last()
}
func (distributed *Distributed) Distributes() []*DCore {
	return distributed.items.ToArray()
}
func (distributed *Distributed) Contain(task *DCore) bool { return distributed.Contain(task) }
func (distributed *Distributed) ContainF(filter DFilterFunc) bool {
	if filter == nil {
		return false
	}
	return distributed.ContainF(func(d *DCore) bool { return filter(d) })
}
func (distributed *Distributed) Remove(task *DCore) {
	if task == nil {
		return
	}
	task.Shutdown()
	distributed.items.Remove(task)
}
func (distributed *Distributed) RemoveF(filter DFilterFunc) {
	if filter == nil {
		return
	}
	distributed.items.RemoveF(func(d *DCore) bool {
		if filter(d) {
			d.Shutdown()
			return true
		}
		return false
	})
}
func (distributed *Distributed) Size() int { return distributed.items.Size() }
func (distributed *Distributed) Done() {
	distributed.items.Iter(func(i int, d *DCore) bool {
		d.Shutdown()
		return false
	})
	distributed.items.Clear()
}

func DefaultDOptions() *DOptions {
	return &DOptions{Ctx: context.Background(), Logger: zap.L()}
}
func NewD(opt *DOptions) *Distributed {
	if opt == nil {
		opt = DefaultDOptions()
	}
	return &Distributed{opts: opt, items: &list.List[*DCore]{}}
}
