package secretun

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type userConfig struct {
	Users string
}

type natConfig struct {
	Net     string
	Gateway string
	Mtu     int
}

type Server struct {
	cfg        Config
	user_cfg   userConfig
	nat_cfg    natConfig
	tunnel_cfg Config

	tunnel ServerTunnel
	ippool IPPool
}

func NewServer(cfg Config) (ser Server, err error) {
	ser.cfg = cfg
	if pkg_cfg, e := cfg.GetConfig("packet"); e != nil {
		err = e
		return
	} else if err = InitPacket(pkg_cfg); err != nil {
		return
	}

	if err = cfg.Get("auth", &ser.user_cfg); err != nil {
		return
	}

	if err = cfg.Get("nat", &ser.nat_cfg); err != nil {
		return
	}
	if ser.ippool, err = NewIPPool(ser.nat_cfg.Net, ser.nat_cfg.Gateway); err != nil {
		return
	}

	var tunnel_name string
	if ser.tunnel_cfg, err = cfg.GetConfig("tunnel"); err != nil {
		return
	} else if err = ser.tunnel_cfg.Get("name", &tunnel_name); err != nil {
		return
	}

	ser.tunnel, err = NewServerTunnel(tunnel_name)

	return
}

func (s *Server) Init() error {
	return s.tunnel.Init(s.tunnel_cfg)
}

func (s *Server) Run() error {
	for {
		if cli_ch, err := s.tunnel.Accept(); err != nil {
			return err
		} else {
			go s.handle_client(cli_ch)
		}
	}
	return nil
}

func (s *Server) Shutdown() error {
	return nil
}

func (s *Server) handle_client(cli_ch ClientChan) {
	defer cli_ch.Close()

	nat_info, err := s.auth(&cli_ch)
	if err != nil {
		log.Println(err)
		return
	}
	if err = s.nat(&cli_ch, nat_info); err != nil {
		log.Println(err)
	}
}

func (s *Server) auth(cli_ch *ClientChan) (nf NatInfo, err error) {
	var auth_info AuthInfo
	var rst AuthResult

	p := <-cli_ch.R
	if p.Decode(&auth_info) != nil {
		err = fmt.Errorf("invalid auth info")
		return
	}

	if !s.check_user(&auth_info) {
		rst.Ok = false
		err = fmt.Errorf("invalid user")
	} else if s.ippool.IsEmpty() {
		rst.Ok = false
		rst.Message = "ip used up"
		err = fmt.Errorf("ip used up")
	} else {
		rst.Ok = true
		rst.NatInfo.Gateway = s.ippool.Gateway
		rst.NatInfo.Netmask = s.ippool.IPNet.Mask
		rst.NatInfo.IP = s.ippool.Next()
		rst.NatInfo.MTU = s.nat_cfg.Mtu
		nf = rst.NatInfo
	}

	if p.Encode(&rst) != nil {
		err = fmt.Errorf("encode AuthResult fail")
		return
	}
	cli_ch.W <- p

	return
}

func (s *Server) nat(cli_ch *ClientChan, nat_info NatInfo) error {
	tun, err := CreateTun("")

	if err != nil {
		return err
	}
	defer tun.Close()

	if err := tun.SetAddr(nat_info.Gateway, nat_info.IP); err != nil {
		return err
	}
	if err := tun.SetNetmask(nat_info.Netmask); err != nil {
		return err
	}
	if s.nat_cfg.Mtu > 0 {
		if err := tun.SetMTU(s.nat_cfg.Mtu); err != nil {
			return err
		}
	}

	tun_ch, err := tun.ReadChan()
	if err != nil {
		return err
	}
	if err := tun.Up(); err != nil {
		return err
	}

	for {
		select {
		case packet, ok := <-cli_ch.R:
			if !ok {
				log.Println("tunnel closed")
				return nil
			}
			if packet.Type == PT_P2P {
				if _, err := tun.Write(packet.Data); err != nil {
					return nil
				}
			} else {
				return nil
			}
		case data, ok := <-tun_ch:
			if !ok {
				log.Println("chan closed")
				return nil
			}
			p := NewPacket(PT_P2P, data)
			cli_ch.W <- p
		case err := <-cli_ch.End:
			return err
		}
	}

	return nil
}

func (s *Server) check_user(info *AuthInfo) bool {
	f, err := os.Open(s.user_cfg.Users)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		segs := strings.Split(line, " ")
		if len(segs) != 2 {
			continue
		}
		if segs[1][len(segs[1])-1] == '\n' {
			segs[1] = segs[1][:len(segs[1])-1]
		}
		if string(segs[0]) == info.Username {
			return string(segs[1]) == info.Password
		}
	}
	return false
}
