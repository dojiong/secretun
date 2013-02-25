package secretun

import (
	"fmt"
	"net"
)

type IPPool struct {
	Gateway net.IP
	IPNet   *net.IPNet
	last    uint
	gw_idx  uint
	max     uint
}

func get_gw_idx(gw net.IP, mask net.IPMask) uint {
	var bs net.IP
	var idx uint = 0

	bs = gw.To4()
	if bs == nil {
		bs = gw
	}

	for i, b := range mask {
		idx = (idx << 8) + (uint(bs[i]) & (^uint(b)))
	}
	return idx
}

func NewIPPool(cidr string, gw string) (p IPPool, err error) {
	p.Gateway = net.ParseIP(gw)
	if _, p.IPNet, err = net.ParseCIDR(cidr); err != nil {
		return
	}
	if !p.IPNet.Contains(p.Gateway) {
		err = fmt.Errorf("invalid gateway or net")
	}
	ones, _ := p.IPNet.Mask.Size()
	p.max = (1 << uint(32-ones)) - 1
	p.last = 0
	p.gw_idx = get_gw_idx(p.Gateway, p.IPNet.Mask)

	return
}

func (p *IPPool) Next() (ip net.IP) {
	for p.last < p.max {
		p.last += 1
		idx := p.last
		if idx&0xFF == 0xFF || idx&0xFF == 0 || idx == p.gw_idx {
			continue
		}

		ip = make([]byte, len(p.IPNet.IP))
		copy(ip, p.IPNet.IP)
		pos := len(ip) - 1
		for idx > 0 {
			ip[pos] |= byte(idx & 0xFF)
			idx >>= 8
			pos -= 1
		}
		break
	}

	return
}

func (p *IPPool) IsEmpty() bool {
	return p.last == p.max
}
