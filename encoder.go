package secretun

import (
	"fmt"
	"reflect"
)

type Encoder interface {
	Init(Config) error
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type Encoders []Encoder

var registered_encoders = map[string]reflect.Type{}

func RegisterEncoder(name string, i interface{}) {
	t := reflect.TypeOf(i)
	if _, ok := reflect.New(t).Interface().(Encoder); !ok {
		panic(fmt.Errorf("invalid encoder: %s", name))
	}
	registered_encoders[name] = t
}

func NewEncoder(name string) (en Encoder, err error) {
	if t, ok := registered_encoders[name]; !ok {
		err = fmt.Errorf("can't find encoder: %s", name)
		return
	} else {
		return reflect.New(t).Interface().(Encoder), nil
	}
	return
}

func GetEncoders(cfgs []Config) (Encoders, error) {
	encoders := make([]Encoder, 0, len(cfgs))
	for _, cfg := range cfgs {
		var name string
		if err := cfg.Get("name", &name); err != nil {
			return nil, err
		}

		if encoder, err := NewEncoder(name); err != nil {
			return nil, err
		} else if err := encoder.Init(cfg); err != nil {
			return nil, err
		} else {
			encoders = append(encoders, encoder)
		}
	}
	return encoders, nil
}

func (es Encoders) Encode(data []byte) (d []byte, err error) {
	buf := data
	for _, encoder := range es {
		if buf, err = encoder.Encode(buf); err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func (es Encoders) Decode(data []byte) (d []byte, err error) {
	buf := data
	for i := len(es) - 1; i >= 0; i-- {
		encoder := es[i]
		if buf, err = encoder.Decode(buf); err != nil {
			return nil, err
		}
	}
	return buf, nil
}
