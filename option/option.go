package option

import (
	"encoding/json"
	"maps"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/mohae/deepcopy"
)

func AsInt(v any) (int64, bool) {
	var vv = reflect.ValueOf(v)
	if vv.IsValid() {
		switch vv.Kind() {
		case reflect.String:
			if v, ok := strconv.ParseInt(vv.String(), 10, 64); ok == nil {
				return v, true
			}
		case reflect.Bool:
			if vv.Bool() {
				return 1, true
			}
			return 0, true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return vv.Int(), true
		default:
		}
	}
	return 0, false
}
func AsUint(v any) (uint64, bool) {
	var vv = reflect.ValueOf(v)
	if vv.IsValid() {
		switch vv.Kind() {
		case reflect.String:
			if v, ok := strconv.ParseUint(vv.String(), 10, 64); ok == nil {
				return v, true
			}
		case reflect.Bool:
			if vv.Bool() {
				return 1, true
			}
			return 0, true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return vv.Uint(), true
		default:
		}
	}
	return 0, false
}
func AsString(v any) (string, bool) {
	var vv = reflect.ValueOf(v)
	if vv.IsValid() {
		switch vv.Kind() {
		case reflect.String:
			return vv.String(), true
		case reflect.Bool:
			return strconv.FormatBool(vv.Bool()), true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(vv.Int(), 10), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return strconv.FormatUint(vv.Uint(), 10), true
		default:
		}
	}
	return "", false
}
func AsBool(v any) (bool, bool) {
	var vv = reflect.ValueOf(v)
	if vv.IsValid() {
		switch vv.Kind() {
		case reflect.String:
			if ok, err := strconv.ParseBool(vv.String()); err == nil {
				return ok, true
			}
		case reflect.Bool:
			return vv.Bool(), true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return vv.Int() > 0, true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return vv.Int() > 0, true
		default:
		}
	}
	return false, false
}
func AsFloat(v any) (float64, bool) {
	var vv = reflect.ValueOf(v)
	if vv.IsValid() {
		switch vv.Kind() {
		case reflect.String:
			if v, ok := strconv.ParseFloat(vv.String(), 64); ok == nil {
				return v, true
			}
		case reflect.Bool:
			if vv.Bool() {
				return 1, true
			}
			return 0, true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(vv.Uint()), true
		default:
		}
	}
	return 0, false
}
func AsTime(v any) (time.Time, bool) {
	switch vv := v.(type) {
	case time.Time:
		return vv, true
	case string:
		t, err := time.ParseInLocation(time.DateTime, vv, time.Local)
		if err != nil {
			t, err = time.ParseInLocation(time.DateOnly, vv, time.Local)
			if err != nil {
				t, err = time.ParseInLocation(time.TimeOnly, vv, time.Local)
				if err != nil {
					timestamp, err := strconv.ParseInt(vv, 10, 64)
					if err == nil {
						if len(vv) == 10 {
							t = time.Unix(timestamp, 0).Local()
						}
						if len(vv) == 13 {
							t = time.UnixMilli(timestamp).Local()
						}
					}
				}
			}
		}
		return t, true
	case int64:
		var timestamp = strconv.FormatInt(vv, 10)
		if len(timestamp) == 10 {
			return time.Unix(vv, 0).Local(), true
		}
		if len(timestamp) == 13 {
			return time.UnixMilli(vv).Local(), true
		}
	}
	return time.Time{}, false
}
func AsDuration(v any) (time.Duration, bool) {
	switch vv := v.(type) {
	case time.Duration:
		return vv, true
	case string:
		var dur, err = time.ParseDuration(vv)
		if err == nil {
			return dur, true
		}
	case int64:
		return time.Duration(vv), true
	}
	return 0, false
}

type Option struct {
	lck sync.Mutex
	obj map[string]any
}

func (opt *Option) AsInt(key string) (int64, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsInt(v)
	}
	return 0, false
}
func (opt *Option) AsUint(key string) (uint64, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsUint(v)
	}
	return 0, false
}
func (opt *Option) AsString(key string) (string, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsString(v)
	}
	return "", false
}
func (opt *Option) AsBool(key string) (bool, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsBool(v)
	}
	return false, false
}
func (opt *Option) AsFloat(key string) (float64, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsFloat(v)
	}
	return 0, false
}
func (opt *Option) AsTime(key string) (time.Time, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsTime(v)
	}
	return time.Time{}, false
}
func (opt *Option) AsDuration(key string) (time.Duration, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if v, ok := opt.obj[key]; ok {
		return AsDuration(v)
	}
	return 0, false
}
func (opt *Option) AsMap(key string) (map[string]any, bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	v, ok := opt.obj[key].(map[string]any)
	return v, ok
}
func (opt *Option) AsObject(key string, out any) bool {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if val, ok := opt.obj[key]; ok {
		dest := reflect.ValueOf(out)
		src := reflect.ValueOf(val)
		if !dest.IsValid() || !src.IsValid() {
			return false
		}
		if dest.Kind() != reflect.Pointer {
			return false
		}
		destValue := dest.Elem()
		if !src.Type().AssignableTo(destValue.Type()) {
			if src.CanConvert(destValue.Type()) {
				destValue.Set(src.Convert(destValue.Type()))
				return true
			}
			return false
		}
		destValue.Set(src)
		return true
	}
	return false
}

func (opt *Option) Set(key string, val any) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if _, ok := val.(*Option); ok && opt == val {
		return
	}
	opt.obj[key] = val
}
func (opt *Option) Del(key string) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	delete(opt.obj, key)
}
func (opt *Option) Keys() []string {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	var keys []string
	for k := range opt.obj {
		keys = append(keys, k)
	}
	return keys
}
func (opt *Option) Count() int {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	return len(opt.obj)
}
func (opt *Option) Len() int {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	return len(opt.obj)
}
func (opt *Option) Raw() map[string]any {
	return opt.obj
}
func (opt *Option) Exists(key string) (ok bool) {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	_, ok = opt.obj[key]
	return
}

func (opt *Option) Clear() {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	for k := range opt.obj {
		delete(opt.obj, k)
	}
}
func (opt *Option) Clone() *Option {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	var cpy = New(nil)
	for k, v := range opt.obj {
		if o, ok := v.(*Option); ok {
			cpy.Set(k, o.Clone())
		} else {
			cpy.Set(k, deepcopy.Copy(v))
		}
	}
	return cpy
}

func (opt *Option) String() string {
	opt.lck.Lock()
	defer opt.lck.Unlock()
	if raw, err := json.Marshal(opt.obj); err != nil {
		return "{}"
	} else {
		return string(raw)
	}
}
func (opt *Option) GJson() gjson.Result {
	return gjson.Parse(opt.String())
}
func (opt *Option) UnmarshalJSON(b []byte) error {
	if b == nil {
		return nil
	}
	return json.Unmarshal(b, &opt.obj)
}
func (opt *Option) MarshalJSON() ([]byte, error) {
	return json.Marshal(opt.obj)
}
func New(obj map[string]any) *Option {
	if obj == nil {
		obj = make(map[string]any)
	} else {
		obj = maps.Clone(obj)
	}
	return &Option{obj: obj}
}

func FromJson(raw string) (*Option, error) {
	var jsonMap = make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &jsonMap); err != nil {
		return nil, err
	} else {
		return New(jsonMap), nil
	}
}
func FromS(s any) (*Option, error) {
	if raw, err := json.Marshal(s); err != nil {
		return nil, err
	} else {
		return FromJson(string(raw))
	}
}
