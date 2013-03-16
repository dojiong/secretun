package secretun

import (
	"fmt"
	"log"
)

type authConfig struct {
	Username string
	Password string
}

type Client struct {
	cfg      Config
	tunnel   ClientTunnel
	cli_ch   ClientChan
	nat_info NatInfo

	auth_cfg   authConfig
	tunnel_cfg Config
}

func NewClient(cfg Config) (cli Client, err error) {
	cli.cfg = cfg
	if pkg_cfg, e := cfg.GetConfig("packet"); e != nil {
		err = e
		return
	} else if err = InitPacket(pkg_cfg); err != nil {
		return
	}

	if err = cfg.Get("auth", &cli.auth_cfg); err != nil {
		return
	}

	var tunnel_name string
	if cli.tunnel_cfg, err = cfg.GetConfig("tunnel"); err != nil {
		return
	} else if err = cli.tunnel_cfg.Get("name", &tunnel_name); err != nil {
		return
	}
	if cli.tunnel, err = NewClientTunnel(tunnel_name); err != nil {
		return
	}

	cli.cli_ch = NewClientChan()

	return
}

func (c *Client) Init() error {
	return c.tunnel.Init(c.tunnel_cfg)
}

func (c *Client) Run() error {
	defer c.cli_ch.Close()

	log.Println("client running")

	if err := c.tunnel.Start(c.cli_ch); err != nil {
		return err
	}
	if err := c.auth(); err != nil {
		return err
	}

	return c.nat()
}

func (c *Client) Shutdown() error {
	return nil
}

func (c *Client) auth() error {
	var rst AuthResult

	p := NewPacket(PT_AUTH, &AuthInfo{c.auth_cfg.Username, c.auth_cfg.Password})
	c.cli_ch.W <- p
	p = <-c.cli_ch.R
	if p.Decode(&rst) != nil {
		return fmt.Errorf("invalid auth result")
	}

	if !rst.Ok {
		return fmt.Errorf("auth fail: %s", rst.Message)
	}
	log.Println(rst)
	c.nat_info = rst.NatInfo

	return nil
}

func (c *Client) nat() error {
	tun, err := CreateTun("")

	if err != nil {
		return err
	}
	defer tun.Close()

	if err := tun.SetAddr(c.nat_info.IP, c.nat_info.Gateway); err != nil {
		return err
	}
	if err := tun.SetNetmask(c.nat_info.Netmask); err != nil {
		return err
	}
	if c.nat_info.MTU > 0 {
		if err := tun.SetMTU(c.nat_info.MTU); err != nil {
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
		case packet, ok := <-c.cli_ch.R:
			if !ok {
				log.Println("tunnel closed")
				return nil
			}
			if packet.Type == PT_P2P {
				if _, err := tun.Write(packet.Data); err != nil {
					log.Println("data:", packet.Data, err)
					//return err
				}
			} else {
				log.Println("end")
				return nil
			}
		case data, ok := <-tun_ch:
			if !ok {
				log.Println("chan closed")
				return nil
			}
			c.cli_ch.W <- NewPacket(PT_P2P, data)
		case err := <-c.cli_ch.End:
			return err
		}
	}

	return nil
}
