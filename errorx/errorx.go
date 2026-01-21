package errorx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var defaultOptions *Options

type StringifyFunc func(x *ErrorX) string

func TextStringify() StringifyFunc {
	return func(x *ErrorX) string {
		var items []string
		items = append(items, x.Err.Error())
		for k, value := range x.Fields {
			items = append(items,
				fmt.Sprintf("%s:%#v", k, value),
			)
		}
		var sb = new(strings.Builder)
		sb.WriteString(strings.Join(items, " "))
		if x.opts.stack && len(x.StackFrame) > 0 {
			sb.WriteString("\n")
			for _, stack := range x.StackFrame {
				fmt.Fprintf(sb, "(%d)%s\n", stack.Depth, stack.FuncName)
				fmt.Fprintf(sb, "\t%s:%d +0x%x\n\t\t%s\n", stack.File, stack.LineN, stack.PC, stack.Line)
			}
		}
		return sb.String()
	}
}
func JsonStringify() StringifyFunc {
	return func(x *ErrorX) string {
		var items map[string]any = make(map[string]any)
		items["error"] = x.Err.Error()
		items["fields"] = x.Fields
		if x.opts.stack && len(x.StackFrame) > 0 {
			items["stacks"] = x.StackFrame
		}
		v, err := json.Marshal(items)
		if err != nil {
			return fmt.Sprintf("errorx marshal: %s", err)
		}
		return string(v)
	}
}

type Options struct {
	stack         bool
	stackDepth    int
	stackSkip     int
	stringifyFunc StringifyFunc
}

func (opts *Options) err(err error) *ErrorX {
	return &ErrorX{
		opts:       opts,
		Err:        err,
		At:         time.Now(),
		Fields:     make(map[string]any),
		StackFrame: stack(opts.stackDepth, opts.stackSkip),
	}
}

func (opts *Options) WithStack(enable bool) *Options {
	opts.stack = enable
	return opts
}
func (opts *Options) WithStackDepth(n int) *Options {
	opts.stackDepth = n
	return opts
}
func (opts *Options) WithStackSkip(n int) *Options {
	opts.stackSkip = n
	return opts
}
func (opts *Options) WithStringify(f StringifyFunc) *Options {
	opts.stringifyFunc = f
	return opts
}
func (opts *Options) New(err string) *ErrorX {
	return opts.err(errors.New(err))
}
func (opts *Options) Newf(f string, args ...any) *ErrorX {
	return opts.New(fmt.Sprintf(f, args...))
}
func (opts *Options) Error(err error) *ErrorX {
	return opts.err(err)
}
func (opts *Options) ErrorF(f string, args ...any) *ErrorX {
	return opts.err(fmt.Errorf(f, args...))
}

type StackFrame struct {
	File     string  `json:"file"`
	Line     string  `json:"line"`
	LineN    int     `json:"lineN"`
	Depth    int     `json:"depth"`
	PC       uintptr `json:"pc"`
	FuncName string  `json:"funcName"`
}

type ErrorX struct {
	opts       *Options
	lck        sync.Mutex
	At         time.Time
	Err        error
	StackFrame []StackFrame
	Fields     map[string]any
}

func (x *ErrorX) WithStack(enable bool) *ErrorX {
	x.opts.stack = enable
	return x
}
func (x *ErrorX) WithStackDepth(n int) *ErrorX {
	x.opts.stackDepth = n
	return x
}
func (x *ErrorX) WithStackSkip(n int) *ErrorX {
	x.opts.stackDepth = n
	return x
}
func (x *ErrorX) WithStringify(f StringifyFunc) *ErrorX {
	x.opts.stringifyFunc = f
	return x
}
func (x *ErrorX) WithField(key string, value any) *ErrorX {
	x.lck.Lock()
	defer x.lck.Unlock()
	x.Fields[key] = value
	return x
}
func (x *ErrorX) Error() string {
	if x.Err == nil {
		return ""
	}
	var stringify = x.opts.stringifyFunc
	if stringify == nil {
		stringify = TextStringify()
	}
	return stringify(x)
}
func (x *ErrorX) UnWrap() error {
	if v, ok := x.Err.(interface {
		Unwrap() error
	}); ok {
		return v.Unwrap()
	}
	return x.Err
}

func stack(n, skip int) []StackFrame {
	var lines [][]byte
	var fName string
	var stacks []StackFrame
	var c int
	if skip > 0 {
		c = skip
	}
	for {
		if n > 0 && c > n {
			break
		}
		pc, file, line, ok := runtime.Caller(c)
		if !ok {
			break
		}

		var funcName = runtime.FuncForPC(pc).Name()
		if file != fName {
			f, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(f, []byte{'\n'})
			fName = file
		}
		stacks = append(stacks,
			StackFrame{
				Depth:    c - skip,
				LineN:    line - 1,
				Line:     strings.TrimSpace(string(lines[line-1])),
				File:     file,
				FuncName: funcName,
				PC:       pc,
			})
		c++
	}
	return stacks
}

func Stack(enable bool) {
	defaultOptions.WithStack(enable)
}
func StackDepth(n int) {
	defaultOptions.WithStackDepth(n)
}
func StackSkip(n int) {
	defaultOptions.WithStackSkip(n)
}
func Stringify(f StringifyFunc) {
	defaultOptions.WithStringify(f)
}

func New(err string) *ErrorX {
	return defaultOptions.New(err)
}
func Newf(f string, args ...any) *ErrorX {
	return defaultOptions.Newf(f, args...)
}
func Error(err error) *ErrorX {
	return defaultOptions.Error(err)
}
func ErrorF(f string, args ...any) *ErrorX {
	return defaultOptions.ErrorF(f, args...)
}

func Slient() *Options {
	return (&Options{}).WithStringify(TextStringify())
}
func Verbose() *Options {
	return (&Options{}).WithStack(true).WithStringify(TextStringify())
}
func JSlient() *Options {
	return (&Options{}).WithStringify(JsonStringify())
}
func JVerbose() *Options {
	return (&Options{}).WithStack(true).WithStringify(JsonStringify())
}

func init() {
	defaultOptions = Slient()
}
