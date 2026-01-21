package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type SetHook func(key string, value any) error
type OptionFunc func(*Config)

func WithSetHook(f SetHook) OptionFunc {
	return func(c *Config) {
		c.setHook = f
	}
}
func StructHook(opts ...ValidateOptionFunc[any]) OptionFunc {
	return WithSetHook(func(key string, value any) error {
		var val = reflect.ValueOf(value)
		if val.IsValid() {
			if val.Kind() == reflect.Pointer {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				if _, err := Validate(value, opts...); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

type DecoderConfigOption viper.DecoderConfigOption

type Config struct {
	viper   *viper.Viper
	setHook SetHook
}

func (config *Config) Set(key string, value any) error {
	if config.setHook != nil {
		if err := config.setHook(key, value); err != nil {
			return err
		}
	}
	config.viper.Set(key, value)
	return nil
}
func (config *Config) SetDefault(key string, value any) error {
	if config.setHook != nil {
		if err := config.setHook(key, value); err != nil {
			return err
		}
	}
	config.viper.SetDefault(key, value)
	return nil
}
func (config *Config) Get(key string) any {
	return config.viper.Get(key)
}
func (config *Config) GetBool(key string) bool {
	return config.viper.GetBool(key)
}
func (config *Config) GetDuration(key string) time.Duration {
	return config.viper.GetDuration(key)
}
func (config *Config) GetFloat64(key string) float64 {
	return config.viper.GetFloat64(key)
}
func (config *Config) GetInt(key string) int {
	return config.viper.GetInt(key)
}
func (config *Config) GetInt32(key string) int32 {
	return config.viper.GetInt32(key)
}
func (config *Config) GetInt64(key string) int64 {
	return config.viper.GetInt64(key)
}
func (config *Config) GetUint(key string) uint {
	return config.viper.GetUint(key)
}
func (config *Config) GetUint16(key string) uint16 {
	return config.viper.GetUint16(key)
}
func (config *Config) GetUint32(key string) uint32 {
	return config.viper.GetUint32(key)
}
func (config *Config) GetUInt64(key string) uint64 {
	return config.viper.GetUint64(key)
}
func (config *Config) GetString(key string) string {
	return config.viper.GetString(key)
}
func (config *Config) GetTime(key string) time.Time {
	return config.viper.GetTime(key)
}
func (config *Config) GetIntSlice(key string) []int {
	return config.viper.GetIntSlice(key)
}
func (config *Config) GetStringMap(key string) map[string]any {
	return config.viper.GetStringMap(key)
}
func (config *Config) GetStringSlice(key string) []string {
	return config.viper.GetStringSlice(key)
}
func (config *Config) GetStringMapString(key string) map[string]string {
	return config.viper.GetStringMapString(key)
}
func (config *Config) Write(writer io.Writer, configType string) error {
	var fs = NewBufferedFs()
	var id = fs.NewID()
	config.viper.SetFs(fs)
	defer config.viper.SetFs(afero.NewOsFs())
	var filename = id + "." + configType
	if err := config.viper.WriteConfigAs(filename); err != nil {
		return err
	} else {
		defer fs.Remove(filename)
		if v, err := fs.Open(filename); err != nil {
			return err
		} else {
			_, err = io.Copy(writer, v)
			return err
		}
	}
}
func (config *Config) WriteBytes(configType string) ([]byte, error) {
	var buf = new(bytes.Buffer)
	if err := config.Write(buf, configType); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (config *Config) WriteString(configType string) (string, error) {
	var raw, err = config.WriteBytes(configType)
	return string(raw), err
}
func (config *Config) WriteFile(name string) error {
	return config.viper.WriteConfigAs(name)
}
func (config *Config) WriteFileSafe(name string) error {
	config.viper.SetFs(afero.NewOsFs())
	return config.viper.SafeWriteConfigAs(name)
}
func (config *Config) SetConfigPermissions(perm os.FileMode) {
	config.viper.SetConfigPermissions(perm)
}
func (config *Config) SetConfigFile(in string) {
	config.viper.SetConfigFile(in)
}
func (config *Config) ConfigFileUsed() string {
	return config.viper.ConfigFileUsed()
}
func (config *Config) SetConfigName(in string) {
	config.viper.SetConfigName(in)
}
func (config *Config) SetConfigType(in string) {
	config.viper.SetConfigType(in)
}
func (config *Config) ReadConfig(reader io.Reader) error {
	return config.viper.ReadConfig(reader)
}
func (config *Config) ReadInConfig() error {
	return config.viper.ReadInConfig()
}
func (config *Config) MergeInConfig() error {
	return config.viper.MergeInConfig()
}
func (config *Config) MergeConfig(reader io.Reader) error {
	return config.viper.MergeConfig(reader)
}
func (config *Config) MergeConfigMap(cfg map[string]any) error {
	return config.viper.MergeConfigMap(cfg)
}
func (config *Config) AllKeys() []string {
	return config.viper.AllKeys()
}
func (config *Config) AllSettings() map[string]any {
	return config.viper.AllSettings()
}
func (config *Config) InConfig(key string) bool {
	return config.viper.InConfig(key)
}
func (config *Config) IsSet(key string) bool {
	return config.viper.IsSet(key)
}
func (config *Config) Unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error {
	if err := config.viper.Unmarshal(rawVal, opts...); err != nil {
		return err
	} else {
		if config.setHook != nil {
			return config.setHook("", rawVal)
		}
	}
	return nil
}
func (config *Config) UnmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	if err := config.viper.UnmarshalKey(key, rawVal, opts...); err != nil {
		return err
	} else {
		if config.setHook != nil {
			return config.setHook(key, rawVal)
		}
	}
	return nil
}
func (config *Config) UnmarshalExact(rawVal any, opts ...viper.DecoderConfigOption) error {
	if err := config.viper.UnmarshalExact(rawVal, opts...); err != nil {
		return err
	} else {
		if config.setHook != nil {
			return config.setHook("", rawVal)
		}
	}
	return nil
}
func (config *Config) Sub(key string) *Config {
	var c = *config
	c.viper = config.viper.Sub(key)
	return &c
}

type validateOption[T any] struct {
	ctx            context.Context
	onValidFail    OnValidFail[T]
	resetNoDefault bool
	onValidations  []func(*validator.Validate) error
}

type OnValidFail[T any] func(T, validator.FieldError)
type ValidateOptionFunc[T any] func(*validateOption[T])

func WithCtx[T any](ctx context.Context) ValidateOptionFunc[T] {
	return func(vo *validateOption[T]) {
		vo.ctx = ctx
	}
}
func WithOnValidFail[T any](f OnValidFail[T]) ValidateOptionFunc[T] {
	return func(vo *validateOption[T]) {
		vo.onValidFail = f
	}
}
func WithResetNoDefault[T any]() ValidateOptionFunc[T] {
	return func(vo *validateOption[T]) {
		vo.resetNoDefault = true
	}
}
func WithRegisterValidation[T any](tag string, fn validator.Func, callValidationEvenIfNull ...bool) ValidateOptionFunc[T] {
	return func(vo *validateOption[T]) {
		vo.onValidations = append(vo.onValidations, func(v *validator.Validate) error {
			return v.RegisterValidation(tag, fn, callValidationEvenIfNull...)
		})
	}
}
func WithOnValidator[T any](f func(v *validator.Validate) error) ValidateOptionFunc[T] {
	return func(vo *validateOption[T]) {
		vo.onValidations = append(vo.onValidations, f)
	}
}

func defaultValidateOption[T any]() *validateOption[T] {
	return &validateOption[T]{ctx: context.Background()}
}
func hasDefaultTag(value reflect.Type, fieldNames []string, index int) bool {
	for i := 0; i < value.NumField(); i++ {
		var field = value.Field(i)
		if field.Name == fieldNames[index] {
			index++
			if index > len(fieldNames)-1 {
				if _, ok := field.Tag.Lookup("default"); ok {
					return true
				}
				return false
			} else {
				var _type = field.Type
				if _type.Kind() == reflect.Pointer {
					_type = _type.Elem()
				}
				return hasDefaultTag(_type, fieldNames, index)
			}
		}
	}
	return false
}
func defaultValue(value reflect.Value) {
	var _type = value.Type()
	for i := 0; i < _type.NumField(); i++ {
		var field = _type.Field(i)
		var _value = value.Field(i)
		if field.IsExported() && _value.Kind() == reflect.Pointer {
			if _, ok := field.Tag.Lookup("default"); ok && _value.IsNil() {
				_value.Set(reflect.New(_value.Type().Elem()))
			}
			defaultValue(_value.Elem())
		}

	}
}

func resetValue(value reflect.Value, fieldNames []string, index int) {
	if index > len(fieldNames)-1 {
		if value.Kind() == reflect.Pointer {
			value.Set(reflect.New(value.Type().Elem()))
			//defaultValue(value.Elem())
		} else {
			if !value.IsZero() {
				value.SetZero()
			}
		}

		return
	}
	var _type = value.Type()
	if _type.Kind() == reflect.Pointer {
		_type = _type.Elem()
		value = value.Elem()
	}
	for i := 0; i < _type.NumField(); i++ {
		var field = _type.Field(i)
		if field.Name == fieldNames[index] {
			index++
			resetValue(value.Field(i), fieldNames, index)
			return
		}

	}
}

func Validate[T any](value T, opts ...ValidateOptionFunc[T]) (T, error) {
	var val = reflect.ValueOf(value)
	if !val.IsValid() {
		return value, fmt.Errorf("bad value %v", value)
	}
	if val.Kind() != reflect.Pointer {
		return value, fmt.Errorf("bad value type %T, require ptr", value)
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return value, fmt.Errorf("bad value type %T, require struct", value)
	}
	var _opt = defaultValidateOption[T]()
	for _, opt := range opts {
		opt(_opt)
	}
	if _opt.ctx == nil {
		_opt.ctx = context.Background()
	}
	var _validator = validator.New()
	for _, v := range _opt.onValidations {
		if err := v(_validator); err != nil {
			return value, err
		}
	}
	//defaultValue(val)
	var err = _validator.StructCtx(_opt.ctx, value)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			for _, e := range validateErrs {
				//reset field value if has default tag
				if !_opt.resetNoDefault {
					var names = strings.Split(e.StructNamespace(), ".")
					if hasDefaultTag(val.Type(), names, 1) {
						resetValue(val, names, 1)
					}
				}
				if _opt.onValidFail != nil {
					_opt.onValidFail(value, e)
				}
			}
		}
	}
	err = defaults.Set(value)
	if err != nil {
		return value, err
	}
	err = _validator.StructCtx(_opt.ctx, value)
	if err != nil {
		return value, err
	}
	return value, nil
}

func TryValidate[T any](value T, opts ...ValidateOptionFunc[T]) T {
	Validate(value, opts...)
	return value
}

func MustValidate[T any](value T, opts ...ValidateOptionFunc[T]) T {
	_, err := Validate(value, opts...)
	if err != nil {
		panic(err)
	}
	return value
}

func New(opts ...OptionFunc) *Config {
	var c = &Config{
		viper: viper.New(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
func WithStructHook(opts ...ValidateOptionFunc[any]) *Config {
	return New(StructHook(opts...))
}
