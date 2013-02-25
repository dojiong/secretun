package secretun

import (
	"fmt"
	"reflect"
)

type Encoder interface {
	Init(map[string]interface{}) error
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

var encoders = map[string]reflect.Type{}

func RegisterEncoder(name string, i interface{}) {
	t := reflect.TypeOf(i)
	if _, ok := reflect.New(t).Interface().(Encoder); !ok {
		panic(fmt.Errorf("invalid encoder: %s", name))
	}
	encoders[name] = t
}

func NewEncoder(name string) (en Encoder, err error) {
	if t, ok := encoders[name]; !ok {
		err = fmt.Errorf("can't find encoder: %s", name)
		return
	} else {
		return reflect.New(t).Interface().(Encoder), nil
	}
	return
}
