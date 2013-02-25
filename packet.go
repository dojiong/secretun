package secretun

import (
	"bytes"
	gob "encoding/gob"
)

const (
	// Packet Types
	PT_P2P = iota
	PT_AUTH
	PT_SHUTDOWN
	PT_UNKNOWN
)

type Packet struct {
	Type uint8
	Data []byte
}

func (p *Packet) Decode(e interface{}) error {
	buf := bytes.NewBuffer(p.Data)
	de := gob.NewDecoder(buf)
	return de.Decode(e)
}

func (p *Packet) Encode(e interface{}) error {
	buf := new(bytes.Buffer)
	en := gob.NewEncoder(buf)
	if err := en.Encode(e); err != nil {
		return err
	}
	p.Data = buf.Bytes()
	return nil
}

func NewPacket(t uint8, e interface{}) (pack *Packet) {
	pack = new(Packet)
	pack.Type = t
	switch v := e.(type) {
	case []byte:
		pack.Data = v
	default:
		if err := pack.Encode(e); err != nil {
			panic(err)
		}
	}

	return pack
}
