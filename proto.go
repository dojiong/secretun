package secretun

import (
	"net"
)

type AuthInfo struct {
	Username string
	Password string
}

type NatInfo struct {
	IP      net.IP
	Gateway net.IP
	Netmask net.IPMask
	MTU     int
}

type AuthResult struct {
	Ok      bool
	Message string
	NatInfo NatInfo
}
