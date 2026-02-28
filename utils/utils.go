package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/xid"
	"github.com/shirou/gopsutil/v4/process"
)

type MSOptions struct {
	TimeLayout  string
	TimeKey     string
	Tag         string
	DecodeHooks []any
}

const defaultTimeKey = "__timeKey__"
const AppName = "go-cas"

var DefaultFixBackoffDuration = time.Millisecond * 100

func defaultMSOptions() *MSOptions {
	return &MSOptions{TimeKey: defaultTimeKey, TimeLayout: time.DateTime, Tag: "json"}
}

func resetTimeFields(input reflect.Value, names []string, timeKey string) {
	if !input.IsValid() {
		return
	}
	if input.Kind() != reflect.Ptr {
		return
	}
	input = input.Elem()
	if input.Kind() != reflect.Map {
		return
	}
	var mapIter = input.MapRange()
	for mapIter.Next() {
		var field = mapIter.Value()
		var key = mapIter.Key()
		if key.Kind() == reflect.String &&
			slices.Contains(names, key.Interface().(string)) {
			var timeField = reflect.ValueOf(field.Interface())
			if timeField.Type().Kind() == reflect.Map {
				var keys = timeField.MapKeys()
				if len(keys) == 1 {
					if keys[0].Kind() == reflect.String && keys[0].Interface().(string) == timeKey {
						input.SetMapIndex(key, timeField.MapIndex(keys[0]))
					}
				}
			}
		}
		if field.Kind() == reflect.Map {
			resetTimeFields(field, names, timeKey)
		}
	}

}
func timeFields(input reflect.Value) []string {
	if !input.IsValid() {
		return nil
	}
	if input.Kind() == reflect.Ptr {
		input = input.Elem()
	}
	if input.Kind() != reflect.Struct {
		return nil
	}
	var typeof = input.Type()
	var names []string
	for i := 0; i < typeof.NumField(); i++ {
		field := typeof.Field(i)
		if field.Type == reflect.TypeOf(time.Time{}) {
			names = append(names, field.Name)
		}
		if field.Type.Kind() == reflect.Struct {
			var _names = timeFields(input.Field(i))
			if len(_names) > 0 {
				names = append(names, _names...)
			}
		}
	}
	return names
}
func patchStringToTimeHookFunc(layout string) any {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}
		return time.ParseInLocation(layout, data.(string), time.Local)
	}
}

func patchTimeToStringHookFunc(layout, timeKey string) any {
	if layout == "" {
		layout = time.DateTime
	}
	if timeKey == "" {
		timeKey = defaultTimeKey
	}
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t.Kind() != reflect.Map {
			return data, nil
		}
		if f != reflect.TypeOf(&time.Time{}) {
			return data, nil
		}
		return map[string]any{timeKey: data.(*time.Time).Format(layout)}, nil
	}
}

type BackoffFunc func(int64) time.Duration

func FixedDuration(dur time.Duration) BackoffFunc {
	return func(i int64) time.Duration {
		return dur
	}
}
func RandomDuration(max time.Duration) BackoffFunc {
	return func(i int64) time.Duration {
		return time.Duration(rand.Int63n(int64(max)))
	}
}
func DefaultBackoffDuration(interval time.Duration, maxDur time.Duration) BackoffFunc {
	const _max int64 = 62
	if interval <= 0 {
		interval = 1
	}
	var maxBackoffN = _max - int64(math.Floor(math.Log2(float64(interval))))
	return func(i int64) time.Duration {
		if i > maxBackoffN {
			i = maxBackoffN
		}
		var dur = interval << i
		if maxDur > 0 && dur > maxDur {
			dur = maxDur
		}
		return dur
	}
}

func UUID() (string, error) {
	u, err := DefaultUUID()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(u, "-", ""), nil
}
func DefaultUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
func MustUUID() string {
	u, err := UUID()
	if err != nil {
		panic(err)
	}
	return u
}
func MustDefaultUUID() string {
	u, err := DefaultUUID()
	if err != nil {
		panic(err)
	}
	return u
}

func KillProcess(pid int) error {
	if pid > 0 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return p.Kill()
	}
	return nil
}
func AliveProcess(pid int) bool {
	if pid > 0 {
		_, err := os.FindProcess(pid)
		return err == nil
	}
	return false
}

func MachineId() string {
	return hex.EncodeToString(xid.New().Machine())
}

func RandString(length int) string {
	sb := strings.Builder{}
	baseStr := "ABCDEFGHIGKLMNOPQRSTUVWXYZabcdefghigklmnopqrstuvwxyz0123456789"
	for i := 0; i < length; i++ {
		sb.WriteByte(baseStr[rand.Intn(len(baseStr))])
	}
	return sb.String()
}
func RandHexString(length int) string {
	sb := strings.Builder{}
	baseStr := "abcdef0123456789"
	for i := 0; i < length; i++ {
		sb.WriteByte(baseStr[rand.Intn(len(baseStr))])
		sb.WriteByte(baseStr[rand.Intn(len(baseStr))])
	}
	return sb.String()
}
func RandDigestString(length int) string {
	sb := strings.Builder{}
	baseStr := "0123456789"
	sb.WriteByte(baseStr[1+rand.Intn(len(baseStr)-1)])
	for i := 0; i < length-1; i++ {
		sb.WriteByte(baseStr[rand.Intn(len(baseStr))])
	}
	return sb.String()
}
func GetCookiesItem(cookies, key string) []string {
	if key == "" {
		return nil
	}
	var values []string
	for _, item := range strings.Split(cookies, ";") {
		var cookie = strings.TrimSpace(item)
		var items = strings.Split(cookie, "=")
		if len(items) == 2 && items[0] == key {
			values = append(values, items[1])
		}
	}
	return values
}
func TextBetween(source, left, right string) string {
	var leftIndex, rightIndex int
	if source == "" || left == "" || right == "" {
		return ""
	}
	leftIndex = strings.Index(source, left)
	if leftIndex == -1 {
		return ""
	}
	rightIndex = strings.Index(source[leftIndex+len(left):], right)
	if rightIndex == -1 {
		return ""
	}
	return source[leftIndex+len(left) : leftIndex+len(left)+rightIndex]
}
func IntBetween(min, max int64) int64 {
	if min < 0 || max < 0 || max < min {
		return 0
	}
	return min + rand.Int63n(max-min+1)
}
func ValidIp(ip string) bool {
	var pattern = `\b((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\b:([0-9]{1,4}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])`
	ok, _ := regexp.MatchString(pattern, ip)
	return ok
}
func KillRemoteDebugChromeProcess() error {
	ps, err := process.Processes()
	if err != nil {
		return err
	}
	for _, p := range ps {
		name, err := p.Name()
		if err == nil {
			cmdline, err := p.Cmdline()
			if err == nil {
				if name == "chrome.exe" && strings.Contains(cmdline, "--remote-debugging-port") {
					return p.Kill()
				}
			}
		}
	}
	return nil
}
func KillProcessByName(n string) error {
	ps, err := process.Processes()
	if err != nil {
		return err
	}
	for _, p := range ps {
		name, err := p.Name()
		if err == nil {
			if name == n {
				p.Kill()
			}
		}
	}
	return nil
}
func StringHash(s string) uint64 {
	return xxhash.Sum64String(s)
}
func UnEscapeFilePath(p string) string {
	var buf []string
	for _, item := range strings.Split(p, "\\\\") {
		buf = append(buf, strings.ReplaceAll(item, "\\", "\\\\"))
	}
	return strings.Join(buf, "\\\\")
}
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}
func MS(input, out any, opts *MSOptions) error {
	if opts == nil {
		opts = defaultMSOptions()
	}
	var hooks = []mapstructure.DecodeHookFunc{
		mapstructure.StringToTimeDurationHookFunc(),
		patchStringToTimeHookFunc(opts.TimeLayout),
		patchTimeToStringHookFunc(opts.TimeLayout, opts.TimeKey),
	}
	for _, hook := range opts.DecodeHooks {
		hooks = append(hooks, hook)
	}
	config := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     out,
		TagName:    opts.Tag,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(hooks...),
	}
	var decoder, err = mapstructure.NewDecoder(config)
	if err != nil {
		return nil
	}
	if err := decoder.Decode(input); err != nil {
		return err
	}
	var outKind = reflect.TypeOf(out).Elem().Kind()
	if outKind == reflect.Map {
		var names = timeFields(reflect.ValueOf(input))
		if len(names) > 0 {
			resetTimeFields(reflect.ValueOf(out), names, opts.TimeKey)
		}
	}
	return nil
}
func GetDefaultHomeDir(modules ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	} else {
		var prefix = []string{home, "." + AppName}
		prefix = append(prefix, modules...)
		home = filepath.Clean(filepath.Join(prefix...))
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(home); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(home, 0644)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return home, nil
}
func MustGetDefaultHomeDir(modules ...string) string {
	dir, err := GetDefaultHomeDir(modules...)
	if err != nil {
		panic(fmt.Errorf("get default home dir failed,%w", err))
	}
	return dir
}
func GetDefaultLogDir() string {
	var logDir string
	switch runtime.GOOS {
	default:
		logDir = "."
	case "windows":
		home, err := os.UserHomeDir()
		if err != nil {
			logDir = "."
		} else {
			logDir = filepath.Join(home, "."+AppName, "logs")
		}
	case "linux", "darwin":
		logDir = filepath.Join("/var/logx", AppName)
	}
	if logDir == "." {
		logDir = filepath.Join(".", "logs", AppName)
	}
	return logDir
}
func ValidJs(js string) bool {
	result := api.Transform(js, api.TransformOptions{
		Loader: api.LoaderJS,
	})
	return len(result.Errors) == 0
}
