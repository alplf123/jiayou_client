package engine

import (
	"context"
	"errors"
	"fmt"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/cron"
	"jiayou_backend_spider/db"
	"jiayou_backend_spider/engine/common"
	"jiayou_backend_spider/eventx"
	"jiayou_backend_spider/logx"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/utils"
	"slices"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-viper/mapstructure/v2"
	"go.uber.org/zap"
)

var ErrNotRunning = errors.New("engine not running")

type Env string

const (
	Dev  = "dev"
	Prod = "prod"
)

type OnOptions func(*Engine) error
type Event struct {
	Engine *Engine
	Ctx    *eventx.EventCtx
	Args   []any
}
type EventFunc func(*Event)
type EventOptions struct {
	Workers int                `json:"workers" validate:"gte=0"`
	Events  []common.EventType `json:"events"`
}
type CronOptions struct {
	LocalOptions      *cron.LOptions       `json:"local" validate:"required" default:""`
	DistributeOptions *cron.DOptions       `json:"distribute" validate:"required" default:""`
	LocalTasks        []*cron.LTaskOptions `json:"local_tasks"`
	DistributeTasks   []*cron.DTaskOptions `json:"distribute_tasks"`
}
type Options struct {
	Env            Env             `json:"env" validate:"oneof=prod dev" default:"dev"`
	Verbose        bool            `json:"verbose" default:"true"`
	DbDSN          string          `json:"db_dsn" validate:"required"`
	DbMigrate      bool            `json:"db_migrate"`
	DbInitialize   bool            `json:"db_initialize"`
	StaticFolder   string          `json:"static_folder" validate:"required" default:"static"`
	Domain         string          `json:"domain" validate:"required"`
	Server         string          `json:"server" validate:"required,hostname_port" default:"localhost:9876"`
	RedisServer    string          `json:"redis_server" validate:"required"`
	TaskConcurrent int             `json:"task_concurrent" validate:"gte=0"`
	Stack          int             `json:"stack" validate:"gte=0"`
	Event          *EventOptions   `json:"event"`
	Cron           *CronOptions    `json:"cron"`
	Db             []*db.Options   `json:"db"`
	LogConfig      *logx.ZapConfig `json:"log" validate:"required" default:""`
}

type Engine struct {
	ctx            context.Context
	cancel         context.CancelFunc
	lck            sync.Mutex
	opts           *Options
	config         *config.Config
	configFile     string
	configData     string
	localCron      *cron.Local
	distributeCron *cron.Distributed
	logger         *logx.Logger
	event          *eventx.EventX
	db             *db.Sessions
	onEvent        EventFunc
	onOptions      OnOptions
	meta           *option.Option
	server         *gin.Engine
}

func (engine *Engine) enterLck() {
	engine.lck.Lock()
}
func (engine *Engine) exitLck() {
	engine.lck.Unlock()
}
func (engine *Engine) withConfigFile(file string) *Engine {
	engine.configFile = file
	return engine
}
func (engine *Engine) withConfigData(data string) *Engine {
	engine.configData = data
	return engine
}
func (engine *Engine) withConfig(config *config.Config) *Engine {
	engine.config = config
	return engine
}
func (engine *Engine) init() error {
	if engine.opts == nil {
		var defaultOpts = DefaultOptions()
		if engine.configFile != "" || engine.configData != "" {
			if engine.configFile != "" {
				engine.config.SetConfigFile(engine.configFile)
				if err := engine.config.ReadInConfig(); err != nil {
					return fmt.Errorf("read config file failed,%w", err)
				}
			}
			if engine.configData != "" {
				if err := engine.config.ReadConfig(strings.NewReader(engine.configData)); err != nil {
					return fmt.Errorf("read config data failed,%w", err)
				}
			}
			if err := engine.config.Unmarshal(defaultOpts, func(dc *mapstructure.DecoderConfig) {
				dc.TagName = "json"
			}); err != nil {
				return fmt.Errorf("unmarshal from config err,%w", err)
			}

		}
		engine.opts = defaultOpts
	}
	if _, err := config.Validate(engine.opts); err != nil {
		return fmt.Errorf("config validate failed[%w]", err)
	}
	if engine.onOptions != nil {
		if err := engine.onOptions(engine); err != nil {
			return fmt.Errorf("onOptions failed[%w]", err)
		}
	}
	if err := engine.initEvent(); err != nil {
		return fmt.Errorf("init event err,%w", err)
	}
	if err := engine.initLogger(); err != nil {
		return fmt.Errorf("init logger err,%w", err)
	}
	if err := engine.initCron(); err != nil {
		return fmt.Errorf("init cron err,%w", err)
	}
	if err := engine.initGin(); err != nil {
		return fmt.Errorf("init gin err,%w", err)
	}
	return nil
}
func (engine *Engine) subscribe(evt *eventx.EventCtx, args ...any) {
	if engine.running() {
		var fields []zap.Field
		for i, field := range args {
			fields = append(fields, zap.Any(fmt.Sprintf("arg%d", i), field))
		}
		engine.logger.Zap.Info(fmt.Sprintf("recv event[%s]", evt.Topic()), fields...)
		if engine.onEvent != nil {
			engine.onEvent(&Event{
				Engine: engine,
				Ctx:    evt,
				Args:   args,
			})
		}
	}
}
func (engine *Engine) publicEvt(evt common.EventType, data ...any) {
	if engine.running() {
		if err := engine.event.Publish(string(evt), data...); err != nil {
			engine.logger.Zap.Error(fmt.Sprintf("public event[%s]", evt), zap.Error(err))
		}
	}
}
func (engine *Engine) initDbSessions() error {
	if len(engine.opts.Db) > 0 {
		engine.db = db.NewSessions()
	}
	for _, session := range engine.opts.Db {
		session.Ctx = engine.ctx
		session.OnKeyExpired = func(v db.KVItem) {
			engine.publicEvt(common.EventDbKeyExpired, v)
		}
		sess, err := db.NewSession(session)
		if err != nil {
			return err
		}
		err = engine.db.Add(sess)
		if err != nil {
			return err
		}
	}
	engine.publicEvt(common.EventDbReady)
	return nil
}
func (engine *Engine) initLogger() error {
	if engine.opts.Verbose {
		if engine.opts.LogConfig.DatedDir == "" {
			engine.opts.LogConfig.DatedDir = utils.GetDefaultLogDir()
		}
		logger, err := engine.opts.LogConfig.Build()
		if err != nil {
			return err
		}
		engine.logger = logger
	} else {
		engine.logger = &logx.Logger{Zap: zap.NewNop()}
	}
	engine.publicEvt(common.EventLoggerReady)
	return nil
}
func (engine *Engine) initEvent() error {
	var eventOpts = eventx.DefaultOptions()
	var evts []common.EventType
	if engine.opts.Event != nil {
		eventOpts.Concurrent = engine.opts.Event.Workers
		evts = engine.opts.Event.Events
	}
	engine.event = eventx.New(eventOpts)
	for _, evt := range evts {
		if slices.Contains(common.EventSlices, evt) {
			if _, err := engine.event.Subscribe(string(evt), engine.subscribe); err != nil {
				return err
			}
		}
	}
	engine.publicEvt(common.EventReady)
	return nil
}
func (engine *Engine) initCron() error {
	if engine.opts.Cron != nil {
		if engine.opts.Cron.LocalOptions != nil {
			engine.opts.Cron.LocalOptions.Ctx = engine.ctx
			//engine.opts.Cron.LocalOptions.Logger = engine.logger.Zap
			if local, err := cron.NewL(engine.opts.Cron.LocalOptions); err != nil {
				return err
			} else {
				if err := local.Add(engine.opts.Cron.LocalTasks...); err != nil {
					return err
				}
				local.Start()
				engine.localCron = local
			}
			engine.publicEvt(common.EventLocalCronReady)
		}
		if engine.opts.Cron.DistributeOptions != nil {
			engine.opts.Cron.DistributeOptions.Ctx = engine.ctx
			engine.opts.Cron.DistributeOptions.Logger = engine.logger.Zap
			var distribute = cron.NewD(engine.opts.Cron.DistributeOptions)
			if err := distribute.Add(engine.opts.Cron.DistributeTasks...); err != nil {
				return err
			}
			for _, dcore := range distribute.Distributes() {
				dcore.Server().BeforeTask(func(ctx context.Context, task *cron.Task) error {
					engine.publicEvt(common.EventCronDistributeBeforeTask, task)
					return nil
				})
				dcore.Server().AfterTask(func(ctx context.Context, task *cron.Task) error {
					engine.publicEvt(common.EventCronDistributeAfterTask, task)
					return nil
				})
			}
			engine.distributeCron = distribute
			engine.publicEvt(common.EventDistributeCronReady)
		}
	}
	return nil
}
func (engine *Engine) initGin() error {
	if engine.opts.Env == Prod {
		gin.SetMode(gin.ReleaseMode)
	}
	engine.server = gin.New()
	engine.server.Use(gin.Recovery())
	if engine.logger.Writer != nil {
		engine.server.Use(gin.LoggerWithWriter(engine.logger.Writer))
	}
	engine.publicEvt(common.EventGinServerReady)
	return nil
}
func (engine *Engine) running() bool {
	if engine.ctx == nil {
		return false
	}
	select {
	case <-engine.ctx.Done():
		return false
	default:
		return true
	}
}
func (engine *Engine) Gin() (*gin.Engine, error) {
	if !engine.Running() {
		return nil, ErrNotRunning
	}
	return engine.server, nil
}
func (engine *Engine) Log() (*zap.Logger, error) {
	if !engine.Running() {
		return nil, ErrNotRunning
	}
	return engine.logger.Zap, nil
}
func (engine *Engine) Local() (*cron.Local, error) {
	if !engine.Running() {
		return nil, ErrNotRunning
	}
	if engine.localCron == nil {
		return nil, errors.New("local cron disabled")
	}
	return engine.localCron, nil
}
func (engine *Engine) Distribute() (*cron.Distributed, error) {
	if !engine.Running() {
		return nil, ErrNotRunning
	}
	if engine.distributeCron == nil {
		return nil, errors.New("distributed cron disabled")
	}
	return engine.distributeCron, nil
}
func (engine *Engine) Db() (*db.Sessions, error) {
	if !engine.Running() {
		return nil, ErrNotRunning
	}
	if engine.db == nil {
		return nil, errors.New("db disabled")
	}
	return engine.db, nil
}
func (engine *Engine) OnEvent(f EventFunc) {
	if engine.running() {
		return
	}
	engine.enterLck()
	defer engine.exitLck()
	engine.onEvent = f
}
func (engine *Engine) OnOptions(f OnOptions) {
	if engine.running() {
		return
	}
	engine.enterLck()
	defer engine.exitLck()
	engine.onOptions = f
}

func (engine *Engine) Options() *Options {
	return engine.opts
}
func (engine *Engine) Running() bool {
	engine.enterLck()
	defer engine.exitLck()
	return engine.running()
}
func (engine *Engine) Ctx() context.Context {
	engine.enterLck()
	defer engine.exitLck()
	return engine.ctx
}
func (engine *Engine) Start() error {
	engine.enterLck()
	if engine.running() {
		engine.exitLck()
		return nil
	}
	engine.ctx, engine.cancel = context.WithCancel(context.Background())
	engine.exitLck()
	if err := engine.init(); err != nil {
		return err
	}
	engine.publicEvt(common.EventEngineStart)
	return nil
}

func (engine *Engine) Stop() {
	engine.enterLck()
	defer engine.exitLck()
	if !engine.running() {
		return
	}
	engine.cancel()
	if engine.localCron != nil {
		engine.localCron.Stop()
		engine.publicEvt(common.EventLocalCronDone)
	}
	if engine.distributeCron != nil {
		engine.distributeCron.Done()
		engine.publicEvt(common.EventDistributeCronDone)
	}
	engine.event.Clear()
	engine.ctx = nil
	engine.publicEvt(common.EventEngineStop)
}
func DefaultOptions() *Options {
	return config.TryValidate(&Options{})
}

func FromConfigFile(file string) *Engine {
	return New(nil).withConfigFile(file)
}
func New(opts *Options) *Engine {
	return &Engine{
		opts:   opts,
		meta:   option.New(nil),
		config: config.WithStructHook(),
	}
}
