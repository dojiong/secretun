package secretun

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Server struct {
	cfg    map[string]map[string]interface{}
	tunnel ServerTunnel
	ippool IPPool
}

func NewServer(cfg map[string]map[string]interface{}) (ser Server, err error) {
	var name interface{}

	ser.cfg = cfg
	if auth_cfg, ok := cfg["auth"]; !ok {
		err = fmt.Errorf("missing `auth`")
		return
	} else if users, ok := auth_cfg["users"]; !ok {
		err = fmt.Errorf("missing `ath.users`")
		return
	} else if _, ok := users.(string); !ok {
		err = fmt.Errorf("auth.users type invalid (string desired)")
		return
	}

	if nat_cfg, ok := cfg["nat"]; !ok {
		err = fmt.Errorf("missing `nat`")
		return
	} else {
		var net, gateway string
		if inet, ok := nat_cfg["net"]; !ok {
			err = fmt.Errorf("missing `nat.net`")
			return
		} else if net, ok = inet.(string); !ok {
			err = fmt.Errorf("nat.net type invalid (string desired)")
			return
		}
		if igw, ok := nat_cfg["gateway"]; !ok {
			err = fmt.Errorf("missing `nat.gateway`")
			return
		} else if gateway, ok = igw.(string); !ok {
			err = fmt.Errorf("nat.gateway type invalid (string desired)")
			return
		}
		ser.ippool, err = NewIPPool(net, gateway)
		if err != nil {
			return
		}
	}

	tunnel_cfg, ok := cfg["tunnel"]
	if !ok {
		err = fmt.Errorf("missing `tunnel`")
		return
	}
	name, ok = tunnel_cfg["name"]
	if !ok {
		err = fmt.Errorf("missing `tunnel.name`")
		return
	} else {
		if _, ok = name.(string); !ok {
			err = fmt.Errorf("tunnel.name is not a string")
			return
		}
	}

	if ser.tunnel, err = NewServerTunnel(name.(string)); err != nil {
		return
	}

	return
}

func (s *Server) Init() error {
	return s.tunnel.Init(s.cfg["tunnel"])
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
	if err := tun.SetMTU(1400); err != nil {
		return err
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
	f, err := os.Open(s.cfg["auth"]["users"].(string))
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
