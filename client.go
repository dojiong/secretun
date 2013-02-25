package secretun

import (
	"fmt"
	"log"
)

type Client struct {
	cfg      map[string]map[string]interface{}
	tunnel   ClientTunnel
	cli_ch   ClientChan
	nat_info NatInfo
}

func NewClient(cfg map[string]map[string]interface{}) (cli Client, err error) {
	var name interface{}

	cli.cfg = cfg
	if _, ok := cfg["auth"]; !ok {
		err = fmt.Errorf("missing `auth`")
		return
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

	if cli.tunnel, err = NewClientTunnel(name.(string)); err != nil {
		return
	}
	cli.cli_ch = NewClientChan()
	return
}

func (c *Client) Init() error {
	return c.tunnel.Init(c.cfg["tunnel"])
}

func (c *Client) Run() error {
	defer c.cli_ch.Close()

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
	var user, passwd interface{}
	var ok bool
	var rst AuthResult

	if user, ok = c.cfg["auth"]["username"]; !ok {
		return fmt.Errorf("missing `username`")
	} else if _, ok = user.(string); !ok {
		return fmt.Errorf("invalid username type (string desired)")
	}
	if passwd, ok = c.cfg["auth"]["password"]; !ok {
		return fmt.Errorf("missing `password`")
	} else if _, ok = passwd.(string); !ok {
		return fmt.Errorf("invalid password type (string desired)")
	}

	p := NewPacket(PT_AUTH, &AuthInfo{user.(string), passwd.(string)})
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
					return err
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
