package cron

import (
	"bytes"
	"context"
	"encoding"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/list"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jinzhu/copier"
	"github.com/tidwall/gjson"
)

type skipWrapper struct {
	error
}

func (b *skipWrapper) Unwrap() error {
	return asynq.SkipRetry
}

type SchedulerFilterFunc func(*DScheduler) bool
type RetryDelayFunc func(n int, e error, t *Task) time.Duration

type ErrorHandlerFunc func(ctx context.Context, task *Task, err error)

type HandlerFunc func(context.Context, *Task) error

type Server struct {
	opts            *DTaskOptions
	server          *asynq.Server
	mux             *asynq.ServeMux
	beforeTask      HandlerFunc
	afterTask       HandlerFunc
	notFoundHandler HandlerFunc
}

func (server *Server) Start() error {
	server.mux.Use(server.handle)
	return server.server.Start(server.mux)
}
func (server *Server) shutdown() {
	server.server.Shutdown()
}
func (server *Server) handle(handler asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {

		if server.beforeTask != nil {
			if err := server.beforeTask(ctx, &Task{task: task}); err != nil {
				return &skipWrapper{err}
			}
		}
		if err := handler.ProcessTask(ctx, task); err != nil {
			// handler not found for task
			if errors.Is(err, asynq.NotFound(ctx, task)) {
				if server.notFoundHandler != nil {
					if err := server.notFoundHandler(ctx, &Task{task: task}); err != nil {
						return err
					}
				}
				return err
			}
			if !errors.Is(err, asynq.SkipRetry) {
				var opts = server.opts.ServerOptions
				if opts != nil && opts.ShouldRetry != nil && !opts.ShouldRetry(err) {
					return &skipWrapper{err}
				}
				select {
				case <-ctx.Done():
					return &skipWrapper{err}
				default:
				}
			}
			return err
		}
		if server.afterTask != nil {
			if err := server.afterTask(ctx, &Task{task: task}); err != nil {
				return err
			}
		}
		return nil
	})
}
func (server *Server) NotFound(f HandlerFunc) {
	server.notFoundHandler = f
}
func (server *Server) BeforeTask(f HandlerFunc) {
	server.beforeTask = f
}
func (server *Server) AfterTask(f HandlerFunc) {
	server.afterTask = f
}
func (server *Server) HandleFunc(pattern string, handler func(context.Context, *Task) error) {
	server.mux.HandleFunc(pattern, func(ctx context.Context, task *asynq.Task) error {
		return handler(ctx, &Task{task: task})
	})
}

func (server *Server) Ping() error {
	return server.server.Ping()
}

type Client struct {
	opts      *DTaskOptions
	client    *asynq.Client
	inspector *Inspector
}

func (client *Client) close() error {
	return client.client.Close()
}
func (client *Client) parallel(tasks []*DTask, opts *ParallelOptions) (*Parallel, error) {
	if len(tasks) == 0 {
		return nil, errors.New("task required")
	}
	if opts == nil {
		opts = DefaultParallelOptions()
	} else {
		if _, err := config.Validate[*ParallelOptions](opts); err != nil {
			return nil, err
		}
	}
	for _, task := range tasks {
		if _, err := config.Validate(task); err != nil {
			return nil, err
		}
	}
	if opts.Ctx == nil {
		opts.Ctx = client.opts.Ctx
	}
	return &Parallel{client: client, tasks: tasks, opts: opts}, nil
}
func (client *Client) enqueue(ctx context.Context, task *DTask) (*DTaskInfo, error) {
	if task == nil {
		return nil, errors.New("the dtask required")
	}
	t, opts, err := newTask(task)
	if err != nil {
		return nil, err
	}
	info, err := client.client.EnqueueContext(ctx, t, opts...)
	if err != nil {
		return nil, err
	}
	var _info DTaskInfo
	if err := copier.Copy(&_info, info); err != nil {
		return nil, fmt.Errorf("copy task info failed,%w", err)
	}
	_info.Payload = &TaskPayload{Payload: info.Payload}
	_info.Result = &TaskResult{Payload: info.Result}
	return &_info, nil
}
func (client *Client) Put(ctx context.Context, task *DTask) (*DTaskInfo, error) {
	return client.enqueue(ctx, task)
}
func (client *Client) PutAsync(task *DTask, clone int, opts *ParallelOptions) (*Parallel, error) {
	if task == nil {
		return nil, errors.New("task required")
	}
	var tasks = []*DTask{task}
	for i := 0; i < clone; i++ {
		tasks = append(tasks, task)
	}
	return client.parallel(tasks, opts)
}
func (client *Client) PutOneAsync(task *DTask, opts *ParallelOptions) (*Parallel, error) {
	return client.PutAsync(task, 0, opts)
}
func (client *Client) Ping() error {
	return client.client.Ping()
}

type ClusterNode struct {
	ID   string
	Addr string
}
type QueueInfo struct {
	Queue          string
	MemoryUsage    int64
	Latency        time.Duration
	Size           int
	Groups         int
	Pending        int
	Active         int
	Scheduled      int
	Retry          int
	Archived       int
	Completed      int
	Aggregating    int
	Processed      int
	Failed         int
	ProcessedTotal int
	FailedTotal    int
	Paused         bool
	Timestamp      time.Time
}

type ListOption interface{}

func Page(n int) ListOption {
	return asynq.Page(n)
}
func PageSize(n int) ListOption {
	return asynq.PageSize(n)
}

type GroupInfo struct {
	Group string
	Size  int
}

type DailyStats struct {
	Queue     string
	Processed int
	Failed    int
	Date      time.Time
}
type SchedulerEnqueueEvent struct {
	TaskID     string
	EnqueuedAt time.Time
}

type WorkerInfo struct {
	TaskID      string
	TaskType    string
	TaskPayload []byte
	Queue       string
	Started     time.Time
	Deadline    time.Time
}
type ServerInfo struct {
	ID             string
	Host           string
	PID            int
	Concurrency    int
	Queues         map[string]int
	StrictPriority bool
	Started        time.Time
	Status         string
	ActiveWorkers  []*WorkerInfo
}
type SchedulerEntry struct {
	ID   string
	Spec string
	Task *Task
	Opts *DTaskOption
	Next time.Time
	Prev time.Time
}
type Inspector struct {
	inspector *asynq.Inspector
	opts      *DTaskOptions
}

func (inspector *Inspector) copyTaskInfos(infos []*asynq.TaskInfo) ([]*DTaskInfo, error) {
	var _infos []*DTaskInfo
	for _, info := range infos {
		var _info DTaskInfo
		if err := copier.Copy(&_info, info); err != nil {
			return nil, fmt.Errorf("copy info failed,%w", err)
		}
		_info.Result = &TaskResult{Payload: info.Result}
		_info.Payload = &TaskPayload{Payload: info.Payload}
		_infos = append(_infos, &_info)
	}
	return _infos, nil
}
func (inspector *Inspector) Close() error {
	return inspector.inspector.Close()
}
func (inspector *Inspector) GetTaskInfo(queue, id string) (*DTaskInfo, error) {
	info, err := inspector.inspector.GetTaskInfo(queue, id)
	if err != nil {
		return nil, err
	}
	var _info DTaskInfo
	if err := copier.Copy(&_info, info); err != nil {
		return nil, fmt.Errorf("copy task info failed,%w", err)
	}
	_info.Result = &TaskResult{Payload: info.Result}
	_info.Payload = &TaskPayload{Payload: info.Payload}
	return &_info, nil
}
func (inspector *Inspector) ArchiveAllAggregatingTasks(queue, group string) (int, error) {
	return inspector.inspector.ArchiveAllAggregatingTasks(queue, group)
}
func (inspector *Inspector) ArchiveAllPendingTasks(queue string) (int, error) {
	return inspector.inspector.ArchiveAllPendingTasks(queue)
}
func (inspector *Inspector) ArchiveAllRetryTasks(queue string) (int, error) {
	return inspector.inspector.ArchiveAllRetryTasks(queue)
}
func (inspector *Inspector) ArchiveAllScheduledTasks(queue string) (int, error) {
	return inspector.inspector.ArchiveAllScheduledTasks(queue)
}
func (inspector *Inspector) ArchiveTask(queue, id string) error {
	return inspector.inspector.ArchiveTask(queue, id)
}
func (inspector *Inspector) CancelProcessing(id string) error {
	return inspector.inspector.CancelProcessing(id)
}
func (inspector *Inspector) ClusterKeySlot(queue string) (int64, error) {
	return inspector.inspector.ClusterKeySlot(queue)
}
func (inspector *Inspector) ClusterNodes(queue string) ([]*ClusterNode, error) {
	nodes, err := inspector.inspector.ClusterNodes(queue)
	if err != nil {
		return nil, err
	}
	var _nodes []*ClusterNode
	err = copier.Copy(&_nodes, nodes)
	if err != nil {
		return nil, fmt.Errorf("copy cluster node failed,%w", err)
	}
	return _nodes, nil
}
func (inspector *Inspector) DeleteAllAggregatingTasks(queue, group string) (int, error) {
	return inspector.inspector.DeleteAllAggregatingTasks(queue, group)
}
func (inspector *Inspector) DeleteAllArchivedTasks(queue string) (int, error) {
	return inspector.inspector.DeleteAllArchivedTasks(queue)
}
func (inspector *Inspector) DeleteAllCompletedTasks(queue string) (int, error) {
	return inspector.inspector.DeleteAllCompletedTasks(queue)
}
func (inspector *Inspector) DeleteAllPendingTasks(queue string) (int, error) {
	return inspector.inspector.DeleteAllPendingTasks(queue)
}
func (inspector *Inspector) DeleteAllRetryTasks(queue string) (int, error) {
	return inspector.inspector.DeleteAllRetryTasks(queue)
}
func (inspector *Inspector) DeleteAllScheduledTasks(queue string) (int, error) {
	return inspector.inspector.DeleteAllScheduledTasks(queue)
}
func (inspector *Inspector) DeleteQueue(queue string, force bool) error {
	return inspector.inspector.DeleteQueue(queue, force)
}
func (inspector *Inspector) DeleteTask(queue, id string) error {
	return inspector.inspector.DeleteTask(queue, id)
}
func (inspector *Inspector) GetQueueInfo(queue string) (*QueueInfo, error) {
	q, err := inspector.inspector.GetQueueInfo(queue)
	if err != nil {
		return nil, err
	}
	var queueInfo QueueInfo
	if err := copier.Copy(&queueInfo, q); err != nil {
		return nil, fmt.Errorf("copy queue info failed,%w", err)
	}
	return &queueInfo, nil
}
func (inspector *Inspector) Groups(queue string) ([]*GroupInfo, error) {
	groups, err := inspector.inspector.Groups(queue)
	if err != nil {
		return nil, err
	}
	var _groups []*GroupInfo
	if err := copier.Copy(&_groups, groups); err != nil {
		return nil, fmt.Errorf("copy group failed,%w", err)
	}
	return _groups, nil
}
func (inspector *Inspector) History(queue string, n int) ([]*DailyStats, error) {
	var stats, err = inspector.inspector.History(queue, n)
	if err != nil {
		return nil, err
	}
	var _stats []*DailyStats
	if err := copier.Copy(&_stats, stats); err != nil {
		return nil, fmt.Errorf("copy history failed,%w", err)
	}
	return _stats, nil
}
func (inspector *Inspector) ListActiveTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListActiveTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListAggregatingTasks(queue, group string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListAggregatingTasks(queue, group, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListArchivedTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListArchivedTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListCompletedTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListCompletedTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListPendingTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListPendingTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListRetryTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListRetryTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListScheduledTasks(queue string, opts ...ListOption) ([]*DTaskInfo, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var infos, err = inspector.inspector.ListScheduledTasks(queue, _opts...)
	if err != nil {
		return nil, err
	}
	return inspector.copyTaskInfos(infos)
}
func (inspector *Inspector) ListSchedulerEnqueueEvents(entryID string, opts ...ListOption) ([]*SchedulerEnqueueEvent, error) {
	var _opts []asynq.ListOption
	for _, o := range opts {
		_opts = append(_opts, asynq.ListOption(o))
	}
	var evts, err = inspector.inspector.ListSchedulerEnqueueEvents(entryID, _opts...)
	if err != nil {
		return nil, err
	}
	var _evts []*SchedulerEnqueueEvent
	if err := copier.Copy(&_evts, evts); err != nil {
		return nil, fmt.Errorf("copy event failed,%w", err)
	}
	return _evts, nil
}
func (inspector *Inspector) PauseQueue(queue string) error {
	return inspector.inspector.PauseQueue(queue)
}
func (inspector *Inspector) Queues() ([]string, error) {
	return inspector.inspector.Queues()
}
func (inspector *Inspector) RunAllAggregatingTasks(queue, group string) (int, error) {
	return inspector.inspector.RunAllAggregatingTasks(queue, group)
}
func (inspector *Inspector) RunAllArchivedTasks(queue string) (int, error) {
	return inspector.inspector.RunAllArchivedTasks(queue)
}
func (inspector *Inspector) RunAllRetryTasks(queue string) (int, error) {
	return inspector.inspector.RunAllRetryTasks(queue)
}
func (inspector *Inspector) RunAllScheduledTasks(queue string) (int, error) {
	return inspector.inspector.RunAllScheduledTasks(queue)
}
func (inspector *Inspector) RunTask(queue, id string) error {
	return inspector.inspector.RunTask(queue, id)
}
func (inspector *Inspector) SchedulerEntries() ([]*SchedulerEntry, error) {
	var entries, err = inspector.inspector.SchedulerEntries()
	if err != nil {
		return nil, err
	}
	var _entries []*SchedulerEntry
	for _, entry := range entries {
		var _entry SchedulerEntry
		if err := copier.Copy(&_entry, entry); err != nil {
			return nil, fmt.Errorf("copy entry failed,%w", err)
		}
		var opts DTaskOption
		opts.UnTranslate(entry.Opts)
		_entry.Task = &Task{task: entry.Task}
		_entry.Opts = &opts
		_entries = append(_entries, &_entry)
	}
	return _entries, nil
}
func (inspector *Inspector) Servers() ([]*ServerInfo, error) {
	var infos, err = inspector.inspector.Servers()
	if err != nil {
		return nil, err
	}
	var servers []*ServerInfo
	if err := copier.Copy(&servers, infos); err != nil {
		return nil, fmt.Errorf("copy servers failed,%w", err)
	}
	return servers, nil
}
func (inspector *Inspector) UnpauseQueue(queue string) error {
	return inspector.inspector.UnpauseQueue(queue)
}

type Scheduler struct {
	scheduler     *asynq.Scheduler
	opts          *DTaskOptions
	schedulerOpts *DSchedulerOptions
	tasks         *list.List[*DScheduler]
}

func (s *Scheduler) start() error {
	return s.scheduler.Start()
}
func (s *Scheduler) shutdown() {
	s.scheduler.Shutdown()
}
func (s *Scheduler) Get(name string) (*DScheduler, error) {
	schedulers := s.tasks.Filter(func(d *DScheduler) bool {
		return d.Name() == name
	})
	if len(schedulers) == 0 {
		return nil, fmt.Errorf("scheduler[%s] not existed", name)
	}
	return schedulers[0], nil
}
func (s *Scheduler) Add(tasks ...*DScheduler) error {
	for _, task := range tasks {
		if _, err := config.Validate(task); err != nil {
			return err
		}
		for _, task := range tasks {
			if s.tasks.ContainF(func(d *DScheduler) bool {
				return d.Task.Name == task.Task.Name
			}) {
				return fmt.Errorf("the task[%s] existed", task.Name())
			}
			t, opts, err := newTask(task.Task)
			if err != nil {
				return err
			}
			id, err := s.scheduler.Register(task.Spec, t, opts...)
			if err != nil {
				return err
			}
			task.id = id
		}
	}
	return nil
}
func (s *Scheduler) Schedulers() []*DScheduler {
	return s.tasks.ToArray()
}
func (s *Scheduler) Remove(scheduler *DScheduler) error {
	if scheduler == nil {
		return nil
	}
	if scheduler.id == "" {
		return fmt.Errorf("scheduler not registed")
	}
	s.tasks.Remove(scheduler)
	return s.scheduler.Unregister(scheduler.id)
}
func (s *Scheduler) RemoveF(filter SchedulerFilterFunc) error {
	if filter == nil {
		return nil
	}
	var ss = s.tasks.Filter(func(d *DScheduler) bool {
		return filter(d)
	})
	for _, scheduler := range ss {
		s.tasks.Remove(scheduler)
		if err := s.scheduler.Unregister(scheduler.id); err != nil {
			return err
		}
	}
	return nil
}
func (s *Scheduler) Contain(scheduler *DScheduler) bool {
	if scheduler == nil {
		return false
	}
	return s.tasks.Contain(scheduler)
}
func (s *Scheduler) ContainF(filter SchedulerFilterFunc) bool {
	if filter == nil {
		return false
	}
	return s.tasks.ContainF(func(d *DScheduler) bool { return filter(d) })
}
func (s *Scheduler) Size() int {
	return s.tasks.Size()
}
func (s *Scheduler) Ping() error {
	return s.scheduler.Ping()
}

type TaskPayload struct {
	Payload []byte `json:"payload"`
}

func (task *TaskPayload) Content() []byte {
	var r []byte
	task.As(&r)
	return r
}
func (task *TaskPayload) Text() string {
	var r string
	task.As(&r)
	return r
}
func (task *TaskPayload) Json() map[string]any {
	var r = make(map[string]any)
	task.As(&r)
	return r
}
func (task *TaskPayload) GJson() gjson.Result {
	return gjson.Parse(task.Text())
}
func (task *TaskPayload) As(data any) error {
	return DecodePayload(task.Payload, 0, data)
}

type TaskResult = TaskPayload
type Task struct {
	task            *asynq.Task
	replacedPayload []byte
}

func (task *Task) Name() string {
	return task.task.Type()
}
func (task *Task) Payload() *TaskPayload {
	if len(task.replacedPayload) > 0 {
		return &TaskPayload{Payload: task.replacedPayload}
	}
	return &TaskPayload{Payload: task.task.Payload()}
}
func (task *Task) UpdatePayload(payload []byte) {
	task.replacedPayload = payload
}
func (task *Task) Id() string {
	if task.task.ResultWriter() != nil {
		return task.task.ResultWriter().TaskID()
	}
	return ""
}
func (task *Task) Write(data any) error {
	return task.WriteLimit(data, 0)
}
func (task *Task) WriteLimit(data any, limit int64) error {
	writer := task.task.ResultWriter()
	if writer != nil {
		raw, err := EncodePayload(data, limit)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, bytes.NewReader(raw))
		return err
	}
	return errors.New("writer not ready")
}
func (task *Task) CanWrite() bool { return task.task.ResultWriter() != nil }

type TaskState int

const (
	TaskStateActive TaskState = iota + 1
	TaskStatePending
	TaskStateScheduled
	TaskStateRetry
	TaskStateArchived
	TaskStateCompleted
	TaskStateAggregating
)

func (s TaskState) String() string {
	switch s {
	case TaskStateActive:
		return "active"
	case TaskStatePending:
		return "pending"
	case TaskStateScheduled:
		return "scheduled"
	case TaskStateRetry:
		return "retry"
	case TaskStateArchived:
		return "archived"
	case TaskStateCompleted:
		return "completed"
	case TaskStateAggregating:
		return "aggregating"
	}
	panic("asynq: unknown task state")
}

type DTaskInfo struct {
	ID            string        `json:"id"`
	Queue         string        `json:"queue"`
	Type          string        `json:"type"`
	Payload       *TaskPayload  `json:"payload"`
	State         TaskState     `json:"task_state"`
	MaxRetry      int           `json:"max_retry"`
	Retried       int           `json:"retried"`
	LastErr       string        `json:"last_err"`
	LastFailedAt  time.Time     `json:"last_failed_at"`
	Timeout       time.Duration `json:"timeout"`
	Deadline      time.Time     `json:"deadline"`
	Group         string        `json:"group"`
	NextProcessAt time.Time     `json:"next_process_at"`
	IsOrphaned    bool          `json:"is_orphaned"`
	Retention     time.Duration `json:"retention"`
	CompletedAt   time.Time     `json:"completed_at"`
	Result        *TaskResult   `json:"result"`
}

func DecodePayload(payload []byte, limit int64, out any) error {
	if len(payload) == 0 || out == nil {
		return nil
	}
	var raw = payload
	if limit > 0 {
		_raw, err := io.ReadAll(io.LimitReader(bytes.NewReader(raw), limit))
		if err != nil {
			return err
		}
		raw = _raw
	}
	switch v := out.(type) {
	default:
		var decoder = gob.NewDecoder(bytes.NewReader(payload))
		if err := decoder.Decode(out); err != nil {
			return fmt.Errorf("can`t support[%T],try impl encoding.BinaryMarshaler,%w", out, err)
		}
	case *[]byte:
		*v = raw
	case *string:
		*v = string(raw)
	case io.Writer:
		_, err := v.Write(raw)
		if err != nil {
			return err
		}
	case encoding.BinaryUnmarshaler:
		if err := v.UnmarshalBinary(raw); err != nil {
			return err
		}
	}
	return nil
}
func EncodePayload(payload any, limit int64) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	var reader io.Reader
	switch t := payload.(type) {
	default:
		var buf = new(bytes.Buffer)
		encoder := gob.NewEncoder(buf)
		if err := encoder.Encode(payload); err != nil {
			return nil, fmt.Errorf("can`t support[%T],try impl encoding.BinaryMarshaler,%w", t, err)
		} else {
			reader = buf
		}
	case []byte:
		return t, nil
	case string:
		reader = strings.NewReader(t)
	case encoding.BinaryMarshaler:
		if raw, err := t.MarshalBinary(); err != nil {
			return nil, err
		} else {
			reader = bytes.NewReader(raw)
		}
	case io.Reader:
		if t != nil {
			reader = t
		}
	}
	if reader != nil {
		if limit > 0 {
			reader = io.LimitReader(reader, limit)
		}
		raw, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, nil
}
func newTask(option *DTask) (*asynq.Task, []asynq.Option, error) {
	if option == nil {
		return nil, nil, errors.New("the dtask option required")
	}
	var taskOpts []asynq.Option
	if option.Options != nil {
		if opts, err := option.Options.Translate(); err != nil {
			return nil, nil, err
		} else {
			taskOpts = opts
		}
	}
	var payload, err = EncodePayload(option.Payload, option.PayloadLimit)
	if err != nil {
		return nil, nil, err
	}
	return asynq.NewTask(option.Name, payload, taskOpts...), taskOpts, nil
}
func newScheduler(opt *DTaskOptions) (*Scheduler, error) {
	if opt == nil {
		return nil, errors.New("the dtask option required")
	}
	var conn, err = asynq.ParseRedisURI(opt.Uri)
	if err != nil {
		return nil, err
	}
	var schedulerOpt *asynq.SchedulerOpts
	if opt.SchedulerOptions != nil {
		if o, err := opt.SchedulerOptions.Translate(); err != nil {
			return nil, err
		} else {
			schedulerOpt = o
		}
	}
	var scheduler = &Scheduler{
		tasks:         list.New[*DScheduler](),
		opts:          opt,
		schedulerOpts: opt.SchedulerOptions,
		scheduler:     asynq.NewScheduler(conn, schedulerOpt),
	}
	return scheduler, scheduler.Add(opt.Schedulers...)
}

func newInspector(opt *DTaskOptions) (*Inspector, error) {
	if opt == nil {
		return nil, errors.New("the dtask option required")
	}
	var conn, err = asynq.ParseRedisURI(opt.Uri)
	if err != nil {
		return nil, err
	}
	return &Inspector{inspector: asynq.NewInspector(conn), opts: opt}, nil
}
func newClient(opt *DTaskOptions) (*Client, error) {
	if opt == nil {
		return nil, errors.New("the dtask option required")
	}
	var conn, err = asynq.ParseRedisURI(opt.Uri)
	if err != nil {
		return nil, err
	}
	return &Client{client: asynq.NewClient(conn), opts: opt}, nil
}
func newServer(opt *DTaskOptions) (*Server, error) {
	if opt == nil {
		return nil, errors.New("the dtask option required")
	}
	var conn, err = asynq.ParseRedisURI(opt.Uri)
	if err != nil {
		return nil, err
	}
	var c asynq.Config
	if opt.ServerOptions != nil {
		if conf, err := opt.ServerOptions.Translate(); err != nil {
			return nil, err
		} else {
			c = conf
		}
	}
	return &Server{server: asynq.NewServer(conn, c), mux: asynq.NewServeMux(), opts: opt}, err
}
