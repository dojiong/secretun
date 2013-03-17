package secretun

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
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

func (t *RawTCP_ST) Init(cfg Config) (err error) {
	var certFile, keyFile, addr string
	var cert tls.Certificate

	if err = cfg.Get("addr", &addr); err != nil {
		return
	}

	if cfg.GetBool("tls") {
		if err := cfg.Get("cert", &certFile); err != nil {
			return err
		} else if err := cfg.Get("key", &keyFile); err != nil {
			return err
		} else if cert, err = tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			return err
		}
		tls_cfg := &tls.Config{}
		tls_cfg.NextProtos = []string{"http/1.1"}
		tls_cfg.Certificates = []tls.Certificate{cert}
		if l, err := net.Listen("tcp", addr); err != nil {
			return err
		} else {
			t.conn = tls.NewListener(l, tls_cfg)
		}
	} else {
		t.conn, err = net.Listen("tcp", addr)
	}

	log.Println("listen on ", addr)

	return
}

func (t *RawTCP_ST) Accept() (cli_ch ClientChan, err error) {
	var conn net.Conn
	conn, err = t.conn.Accept()
	if err != nil {
		return
	}

	if tcp_conn, ok := conn.(*net.TCPConn); ok {
		err = tcp_conn.SetNoDelay(true)
		if err != nil {
			return
		}
		err = tcp_conn.SetKeepAlive(true)
		if err != nil {
			return
		}
	}

	cli_ch = NewClientChan()
	packetTunnel(conn, cli_ch)

	return cli_ch, nil
}

func (t *RawTCP_ST) Shutdown() error {
	return nil
}

func (t *RawTCP_CT) Init(cfg Config) (err error) {
	var addr string
	if err = cfg.Get("addr", &addr); err != nil {
		return
	}

	log.Println("connect to ", addr)

	if cfg.GetBool("tls") {
		tls_cfg := &tls.Config{}
		tls_cfg.InsecureSkipVerify = true
		if t.conn, err = tls.Dial("tcp", addr, tls_cfg); err != nil {
			return
		}
	} else {
		if t.conn, err = net.Dial("tcp", addr); err != nil {
			return
		}
	}

	if tcp_conn, ok := t.conn.(*net.TCPConn); ok {
		if err = tcp_conn.SetNoDelay(true); err != nil {
			return
		}

		err = tcp_conn.SetKeepAlive(true)
	}

	return
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
