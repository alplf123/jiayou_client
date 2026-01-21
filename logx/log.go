package logx

import (
	"errors"
	"fmt"
	"io"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/utils"
	"os"
	"path"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type WriterKind string

const (
	StdOut    WriterKind = "std_out"
	StdErr               = "std_err"
	FixedFile            = "fixed_file"
	DatedFile            = "dated_file"
)

type datedFile struct {
	dir, ext string
	cur      time.Time
	fp       *os.File
}

func (file *datedFile) Write(raw []byte) (int, error) {
	var cur = time.Now()
	if file.fp == nil || file.cur.Format(time.DateOnly) != cur.Format(time.DateOnly) {
		var p = path.Join(file.dir, cur.Format(time.DateOnly)+"."+file.ext)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
		if err == nil {
			file.fp = f
		}
	}
	return file.fp.Write(raw)
}

type Logger struct {
	Config *ZapConfig
	Writer io.Writer
	Zap    *zap.Logger
}
type ZapConfig struct {
	writers      []io.Writer
	Dev          bool         `json:"dev"`
	WriterKinds  []WriterKind `json:"writer_kinds"`
	FixedFile    string       `json:"fixed_file"`
	DatedDir     string       `json:"dated_dir"`
	DefaultExt   string       `json:"default_ext" validate:"required" default:"log"`
	Level        string       `json:"level" validate:"oneof=debug info warn error" default:"info"`
	TimeLayout   string       `json:"time_layout"`
	OutputJson   bool         `json:"output_json"`
	EnableCaller bool         `json:"enable_caller"`
	Initial      int          `json:"initial" validate:"gte=0"`
	Thereafter   int          `json:"thereafter" validate:"gte=0"`
}

func (config *ZapConfig) parseWriter(kind WriterKind) (io.Writer, error) {
	switch kind {
	default:
		return nil, fmt.Errorf("the writer[%s] kind unsupported", kind)
	case StdOut:
		return os.Stdout, nil
	case StdErr:
		return os.Stderr, nil
	case FixedFile:
		return parseFixedFile(config.FixedFile, config.DefaultExt)
	case DatedFile:
		return parseDatedFile(config.DatedDir, config.DefaultExt)
	}
}
func (config *ZapConfig) WithDevelop() *ZapConfig {
	config.Dev = true
	return config
}
func (config *ZapConfig) WithFixedFile(file string) *ZapConfig {
	config.FixedFile = file
	return config
}
func (config *ZapConfig) WithDatedDir(dir string) *ZapConfig {
	config.DatedDir = dir
	return config
}
func (config *ZapConfig) WithDefaultExt(ext string) *ZapConfig {
	config.DefaultExt = ext
	return config
}
func (config *ZapConfig) WithLevel(level string) *ZapConfig {
	config.Level = level
	return config
}
func (config *ZapConfig) WithWriter(writer io.Writer) *ZapConfig {
	config.writers = append(config.writers, writer)
	return config
}
func (config *ZapConfig) WithWriterKind(kind WriterKind) *ZapConfig {
	config.WriterKinds = append(config.WriterKinds, kind)
	return config
}
func (config *ZapConfig) WithTimeLayout(layout string) *ZapConfig {
	config.TimeLayout = layout
	return config
}
func (config *ZapConfig) WithJson() *ZapConfig {
	config.OutputJson = true
	return config
}
func (config *ZapConfig) WithCaller() *ZapConfig {
	config.EnableCaller = true
	return config
}
func (config *ZapConfig) WithSampling(initial, thereafter int) *ZapConfig {
	config.Initial, config.Thereafter = initial, thereafter
	return config
}
func (config *ZapConfig) Build() (*Logger, error) {
	var conf = zap.NewDevelopmentEncoderConfig()
	var opts []zap.Option
	if config.Level == "" {
		config.Level = "info"
	}
	if !config.Dev {
		conf = zap.NewProductionEncoderConfig()
	} else {
		opts = append(opts, zap.Development())
	}
	if config.EnableCaller {
		opts = append(opts, zap.WithCaller(true))
	}
	if config.Initial > 0 {
		opts = append(opts, zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				c,
				time.Second,
				config.Initial,
				config.Thereafter,
			)
		}))
	}
	var layout = config.TimeLayout
	if layout == "" {
		layout = time.DateTime + ".000"
	}
	conf.EncodeTime = zapcore.TimeEncoderOfLayout(layout)
	conf.EncodeDuration = zapcore.StringDurationEncoder
	var writers []zapcore.WriteSyncer
	if len(config.WriterKinds) == 0 {
		config.WriterKinds = append(config.WriterKinds, StdErr)
	}
	for _, kind := range config.WriterKinds {
		if writer, err := config.parseWriter(kind); err != nil {
			return nil, fmt.Errorf("parse kind writer failed,%w", err)
		} else {
			writers = append(writers, zapcore.AddSync(writer))
		}
	}
	for _, writer := range config.writers {
		switch v := writer.(type) {
		case zapcore.WriteSyncer:
			writers = append(writers, v)
		case io.Writer:
			writers = append(writers, zapcore.AddSync(v))
		}
	}
	var level, err = zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	var encoder = zapcore.NewConsoleEncoder(conf)
	if config.OutputJson {
		encoder = zapcore.NewJSONEncoder(conf)
	}
	var syncWriter = zapcore.NewMultiWriteSyncer(writers...)
	return &Logger{
		Zap:    zap.New(zapcore.NewCore(encoder, syncWriter, level), opts...),
		Writer: syncWriter,
		Config: config}, nil
}

func parseFixedFile(file, defaultExt string) (io.Writer, error) {
	if file == "" {
		return nil, errors.New("bad fixed file")
	}
	if filepath.Ext(file) == "" {
		file += "." + defaultExt
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
func parseDatedFile(dir, defaultExt string) (io.Writer, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0644)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &datedFile{dir: dir, ext: defaultExt}, nil
}

func New() *ZapConfig {
	return config.TryValidate(&ZapConfig{DatedDir: utils.GetDefaultLogDir()})
}
