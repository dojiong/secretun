package secretun

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type RawTCP_ST struct {
	conn     net.Listener
	encoders Encoders
}

type RawTCP_CT struct {
	conn     net.Conn
	encoders Encoders
}

func packetTunnel(encoders Encoders, conn net.Conn, cli_ch ClientChan) {

	go func() {
		var err error
		var size uint16
		var header [3]byte

		for {
			if _, err := io.ReadFull(conn, header[:]); err != nil {
				cli_ch.End <- err
				return
			}
			size = binary.BigEndian.Uint16(header[1:])
			packet := new(Packet)
			packet.Type = header[0]
			buf := make([]byte, size)
			if _, err = io.ReadFull(conn, buf); err != nil {
				cli_ch.End <- err
				return
			}
			if packet.Data, err = encoders.Decode(buf); err != nil {
				log.Println(err)
				cli_ch.End <- err
				return
			}
			cli_ch.R <- packet
		}
	}()
	go func() {
		for {
			packet, ok := <-cli_ch.W
			if !ok {
				return
			}
			data, err := encoders.Encode(packet.Data)
			if err != nil {
				log.Println(err)
				cli_ch.End <- err
				return
			}

			size := len(data)
			buf := make([]byte, 0, 3+len(data))
			w := bytes.NewBuffer(buf)
			w.WriteByte(packet.Type)
			w.WriteByte(byte(size >> 8))
			w.WriteByte(byte(size & 0xFF))
			w.Write(data)
			if _, err = conn.Write(w.Bytes()); err != nil {
				cli_ch.End <- err
				return
			}
		}
	}()
}

func (t *RawTCP_ST) Init(cfg map[string]interface{}) (err error) {
	if iencoders, ok := cfg["encoders"]; !ok {
		return fmt.Errorf("tcptunnel missing encoders")
	} else if encoders, ok := iencoders.([]interface{}); !ok {
		return fmt.Errorf("tcptunnel.encoders invalid type ([]interface{} desired)")
	} else {
		if t.encoders, err = GetEncoders(encoders); err != nil {
			return err
		}
	}

	if iaddr, ok := cfg["addr"]; !ok {
		return fmt.Errorf("missing `tunnel.addr`")
	} else if addr, ok := iaddr.(string); !ok {
		return fmt.Errorf("tunnel.addr invalid type (string desired)")
	} else {
		t.conn, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *RawTCP_ST) Accept() (cli_ch ClientChan, err error) {
	var conn net.Conn
	conn, err = t.conn.Accept()
	if err != nil {
		return
	}
	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return
	}
	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return
	}

	cli_ch = NewClientChan()
	packetTunnel(t.encoders, conn, cli_ch)

	return cli_ch, nil
}

func (t *RawTCP_ST) Shutdown() error {
	return nil
}

func (t *RawTCP_CT) Init(cfg map[string]interface{}) (err error) {
	if iencoders, ok := cfg["encoders"]; !ok {
		return fmt.Errorf("tcptunnel missing encoders")
	} else if encoders, ok := iencoders.([]interface{}); !ok {
		return fmt.Errorf("tcptunnel.encoders invalid type ([]interface{} desired)")
	} else {
		if t.encoders, err = GetEncoders(encoders); err != nil {
			return err
		}
	}

	if iaddr, ok := cfg["addr"]; !ok {
		return fmt.Errorf("missing `tunnel.addr`")
	} else if addr, ok := iaddr.(string); !ok {
		return fmt.Errorf("tunnel.addr invalid type (string desired)")
	} else {
		t.conn, err = net.Dial("tcp", addr)
		if err != nil {
			return
		}
		err = t.conn.(*net.TCPConn).SetNoDelay(true)
		if err != nil {
			return
		}
		err = t.conn.(*net.TCPConn).SetKeepAlive(true)
		if err != nil {
			return
		}
	}

	return nil
}

func (t *RawTCP_CT) Start(cli_ch ClientChan) error {
	packetTunnel(t.encoders, t.conn, cli_ch)
	return nil
}

func (t *RawTCP_CT) Shutdown() error {
	return nil
}

func init() {
	RegisterClientTunnel("tcp", RawTCP_CT{})
	RegisterServerTunnel("tcp", RawTCP_ST{})
}
