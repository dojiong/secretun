package secretun

import (
	"bytes"
	gob "encoding/gob"
	"fmt"
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

var encoders Encoders

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

func (p *Packet) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := buf.WriteByte(p.Type); err != nil {
		return nil, err
	}
	if _, err := buf.Write(p.Data); err != nil {
		return nil, err
	}
	return encoders.Encode(buf.Bytes())
}

func DeserializePacket(data []byte) (*Packet, error) {
	if decoded_data, err := encoders.Decode(data); err != nil {
		return nil, err
	} else {
		p := new(Packet)
		p.Type = decoded_data[0]
		p.Data = decoded_data[1:]
		return p, nil
	}
	return nil, nil
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

func InitPacket(cfg map[string]map[string]interface{}) error {
	if pkg_cfg, ok := cfg["packet"]; !ok {
		return fmt.Errorf("missing `packet`")
	} else if iencoders, ok := pkg_cfg["encoders"]; !ok {
		return fmt.Errorf("missing `packet.encoders`")
	} else if encoders_cfg, ok := iencoders.([]interface{}); !ok {
		return fmt.Errorf("encoders invalid type ([]interface{} desired)")
	} else {
		var err error
		if encoders, err = GetEncoders(encoders_cfg); err != nil {
			return err
		}
	}
	return nil
}
