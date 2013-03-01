package secretun

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type RawTCP_ST struct {
	conn net.Listener
}

type RawTCP_CT struct {
	conn net.Conn
}

func packetTunnel(conn net.Conn, cli_ch ClientChan) {

	go func() {
		var size uint16
		var header [2]byte

		for {
			if _, err := io.ReadFull(conn, header[:]); err != nil {
				cli_ch.End <- err
				return
			}
			size = binary.BigEndian.Uint16(header[:])
			data := make([]byte, size)
			if _, err := io.ReadFull(conn, data); err != nil {
				cli_ch.End <- err
				return
			}

			if packet, err := DeserializePacket(data); err != nil {
				cli_ch.End <- err
				return
			} else {
				cli_ch.R <- packet
			}
		}
	}()
	go func() {
		for {
			packet, ok := <-cli_ch.W
			if !ok {
				return
			}
			data, err := packet.Serialize()
			if err != nil {
				cli_ch.End <- err
				return
			}

			size := len(data)
			buf := make([]byte, 0, 2+len(data))
			w := bytes.NewBuffer(buf)
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
	packetTunnel(conn, cli_ch)

	return cli_ch, nil
}

func (t *RawTCP_ST) Shutdown() error {
	return nil
}

func (t *RawTCP_CT) Init(cfg map[string]interface{}) (err error) {
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
	packetTunnel(t.conn, cli_ch)
	return nil
}

func (t *RawTCP_CT) Shutdown() error {
	return nil
}

func init() {
	RegisterClientTunnel("tcp", RawTCP_CT{})
	RegisterServerTunnel("tcp", RawTCP_ST{})
}
