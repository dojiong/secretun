package secretun

import (
	"fmt"
	"reflect"
)

type ClientChan struct {
	R   chan *Packet
	W   chan *Packet
	End chan error
}

func NewClientChan() (c ClientChan) {
	c.R = make(chan *Packet)
	c.W = make(chan *Packet)
	c.End = make(chan error)
	return c
}

func (c *ClientChan) Close() {
	close(c.R)
	close(c.W)
	close(c.End)
}

type ClientTunnel interface {
	Init(Config) error
	Start(ClientChan) error
	Shutdown() error
}

type ServerTunnel interface {
	Init(Config) error
	Accept() (ClientChan, error)
	Shutdown() error
}

var clientTunnels = map[string]reflect.Type{}
var serverTunnels = map[string]reflect.Type{}

func RegisterClientTunnel(name string, i interface{}) {
	t := reflect.TypeOf(i)
	if _, ok := reflect.New(t).Interface().(ClientTunnel); !ok {
		panic(fmt.Errorf("invalid ClientTunnel: %s", name))
	}
	clientTunnels[name] = t
}

func NewClientTunnel(name string) (c ClientTunnel, err error) {
	t, ok := clientTunnels[name]
	if !ok {
		err = fmt.Errorf("invalid ClientTunnel: %s", name)
		return
	}
	return reflect.New(t).Interface().(ClientTunnel), nil
}

func RegisterServerTunnel(name string, i interface{}) {
	t := reflect.TypeOf(i)
	if _, ok := reflect.New(t).Interface().(ServerTunnel); !ok {
		panic(fmt.Errorf("invalid ServerTunnel: %s", name))
	}
	serverTunnels[name] = t
}

func NewServerTunnel(name string) (c ServerTunnel, err error) {
	t, ok := serverTunnels[name]
	if !ok {
		err = fmt.Errorf("invalid ServerTunnel: %s", name)
		return
	}
	return reflect.New(t).Interface().(ServerTunnel), nil
}
