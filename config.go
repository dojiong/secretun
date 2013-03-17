package secretun

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type ConfigError struct {
	Errno int
	Field string
}

const (
	ErrNone = iota
	ErrMissing
	ErrInvalidType
	errMax
)

var errMsgs = []string{
	"no error",
	"config: missing %s",
	"config: %s invalid type",
	"config: %s invalid subconfig",
}

func (e *ConfigError) Error() string {
	if e.Errno >= errMax || e.Errno < 0 {
		return "invalid ConfigError"
	}
	return fmt.Sprintf(errMsgs[e.Errno], e.Field)
}

func NewConfigError(errno int, field string) *ConfigError {
	e := ConfigError{}
	e.Errno = errno
	e.Field = field
	return &e
}

type ConvertFunc func(interface{}, reflect.Value) *ConfigError
type ConvertFuncs []ConvertFunc

func (c ConvertFuncs) get(val reflect.Value) ConvertFunc {
	if _, ok := val.Interface().(Config); ok {
		return convertConfig
	}

	kind := val.Kind()
	if kind > reflect.Kind(len(c)) {
		return nil
	}
	return c[kind]
}

var convertFuncs ConvertFuncs

func init() {
	convertFuncs = []ConvertFunc{nil,
		convertBool,
		convertInt,
		nil, //convertInt8,
		nil, //convertInt16,
		nil, //convertInt32,
		nil, //convertInt64,
		convertUint,
		nil, //convertUint8,
		nil, //convertUint16,
		nil, //convertUint32,
		nil, //convertUint64,
		nil, //convertUintptr,
		convertFloat32,
		convertFloat64,
		nil, //convertComplex64,
		nil, //convertComplex128,
		nil, //convertArray,
		nil, //convertChan,
		nil, //convertFunc,
		nil, //convertInterface,
		nil, //convertMap,
		nil, //convertPtr,
		convertSlice,
		convertString,
		convertStruct}
}

type Config struct {
	Map  map[string]interface{}
	Name string
}

func ConfigFromJson(path string) (cfg Config, err error) {
	cfg.Name = ""
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return
	} else {
		defer f.Close()
		decoder := json.NewDecoder(f)
		if err = decoder.Decode(&cfg.Map); err != nil {
			return
		}
	}
	return
}

func (c *Config) GetConfig(name string) (cfg Config, err error) {
	cfg.Name = fmt.Sprintf("%s.%s", c.Name, name)
	if icfg, ok := c.Map[name]; !ok {
		err = NewConfigError(ErrMissing, cfg.Name)
		return
	} else if c, ok := icfg.(map[string]interface{}); !ok {
		err = NewConfigError(ErrInvalidType, cfg.Name)
		return
	} else {
		cfg.Map = c
	}
	return
}

func (c *Config) Has(name string) bool {
	_, ok := c.Map[name]
	return ok
}

func (c *Config) Get(name string, dest interface{}) error {
	if obj, ok := c.Map[name]; !ok {
		return NewConfigError(ErrMissing, c.Name+"."+name)
	} else {
		val := reflect.ValueOf(dest).Elem()
		if convertor := convertFuncs.get(val); convertor == nil {
			return NewConfigError(ErrInvalidType, c.Name+"."+name)
		} else if err := convertor(obj, val); err != nil {
			if len(err.Field) > 0 {
				err.Field = fmt.Sprintf("%s.%s.%s", c.Name, name, err.Field)
			} else {
				err.Field = fmt.Sprintf("%s.%s", c.Name, name)
			}
			return err
		}
	}

	return nil
}

func (c *Config) GetBool(name string) bool {
	var b bool
	if err := c.Get(name, &b); err != nil {
		return false
	}
	return b
}

func convertBool(in interface{}, val reflect.Value) *ConfigError {
	if b, ok := in.(bool); !ok {
		return &ConfigError{ErrInvalidType, ""}
	} else {
		val.SetBool(b)
	}
	return nil
}

func convertInt(in interface{}, val reflect.Value) *ConfigError {
	switch v := in.(type) {
	case int:
		val.SetInt(int64(v))
	case int64:
		val.SetInt(int64(v))
	case float32:
		val.SetInt(int64(v))
	case float64:
		val.SetInt(int64(v))
	default:
		return &ConfigError{ErrInvalidType, ""}
	}
	return nil
}

func convertUint(in interface{}, val reflect.Value) *ConfigError {
	switch v := in.(type) {
	case int:
		val.SetUint(uint64(v))
	case int64:
		val.SetUint(uint64(v))
	case float32:
		val.SetUint(uint64(v))
	case float64:
		val.SetUint(uint64(v))
	default:
		return &ConfigError{ErrInvalidType, ""}
	}
	return nil
}

func convertFloat32(in interface{}, val reflect.Value) *ConfigError {
	switch v := in.(type) {
	case int:
		val.SetFloat(float64(v))
	case int64:
		val.SetFloat(float64(v))
	case float32:
		val.SetFloat(float64(v))
	case float64:
		val.SetFloat(float64(v))
	default:
		return &ConfigError{ErrInvalidType, ""}
	}
	return nil
}

func convertFloat64(in interface{}, val reflect.Value) *ConfigError {
	switch v := in.(type) {
	case int:
		val.SetFloat(float64(v))
	case int64:
		val.SetFloat(float64(v))
	case float32:
		val.SetFloat(float64(v))
	case float64:
		val.SetFloat(float64(v))
	default:
		return &ConfigError{ErrInvalidType, ""}
	}
	return nil
}

func convertSlice(in interface{}, val reflect.Value) *ConfigError {
	if iary, ok := in.([]interface{}); !ok {
		return &ConfigError{ErrInvalidType, ""}
	} else if len(iary) > 0 {
		var convertor ConvertFunc

		ele := reflect.MakeSlice(val.Type(), 1, 1).Index(0)
		convertor = convertFuncs.get(ele)
		if convertor == nil {
			return &ConfigError{ErrInvalidType, ""}
		}

		ary := reflect.MakeSlice(val.Type(), len(iary), len(iary))
		for i, ele := range iary {
			if err := convertor(ele, ary.Index(i)); err != nil {
				if len(err.Field) > 0 {
					err.Field = fmt.Sprintf("[%d].%s", i, err.Field)
				} else {
					err.Field = fmt.Sprintf("[%d]", i)
				}
				return err
			}
		}
		val.Set(ary)
	}
	return nil
}

func convertString(in interface{}, val reflect.Value) *ConfigError {
	if str, ok := in.(string); !ok {
		return &ConfigError{ErrInvalidType, ""}
	} else {
		val.SetString(str)
	}
	return nil
}

func convertStruct(in interface{}, val reflect.Value) *ConfigError {
	dict, ok := in.(map[string]interface{})
	if !ok || len(dict) != val.NumField() {
		return &ConfigError{ErrInvalidType, ""}
	}

	for k, from := range dict {
		to := val.FieldByName(strings.ToTitle(k[:1]) + k[1:])
		if !to.IsValid() {
			return &ConfigError{ErrInvalidType, ""}
		}

		convertor := convertFuncs.get(to)
		if convertor == nil {
			return &ConfigError{ErrInvalidType, ""}
		}

		if err := convertor(from, to); err != nil {
			if len(err.Field) > 0 {
				err.Field = fmt.Sprintf("%s.%s", k, err.Field)
			} else {
				err.Field = k
			}
			return err
		}
	}

	return nil
}

func convertConfig(in interface{}, val reflect.Value) *ConfigError {
	if m, ok := in.(map[string]interface{}); !ok {
		return &ConfigError{ErrInvalidType, ""}
	} else {
		val.Set(reflect.ValueOf(Config{m, ""}))
	}
	return nil
}
