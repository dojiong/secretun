package secretun

// +build linux

/*
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netinet/in.h>
#include <linux/if.h>
#include <linux/if_tun.h>

int create_tun(int fd, char *name) {
    int err;
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    if (name && name[0]) {
        strncpy(ifr.ifr_name, name, IFNAMSIZ - 1);
    }
    ifr.ifr_flags = IFF_NO_PI | IFF_TUN;
    if ((err = ioctl(fd, TUNSETIFF, (void*)&ifr)) < 0) {
        return err;
    }
    if (name) {
        strcpy(name, ifr.ifr_name);
    }

    return 0;
}

int if_ioctl(int cmd, struct ifreq *ifr) {
    int ret;
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        return -1;
    }
    ret = ioctl(sock, cmd, (void*)ifr);
    close(sock);

    return ret;
}

int if_setaddr(int cmd, const char *name, const char *ip) {
    int err;
    struct ifreq ifr;
    struct sockaddr_in *addr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    addr = (struct sockaddr_in*)&ifr.ifr_ifru.ifru_dstaddr;
    addr->sin_family = AF_INET;
    if ((err = inet_aton(ip, &addr->sin_addr)) < 0) {
        return err;
    }
    return if_ioctl(cmd, &ifr);
}

int if_getaddr(int cmd, const char *name, char **addr) {
    int err;
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    if ((err = if_ioctl(cmd, &ifr)) < 0) {
        return err;
    }

    *addr = inet_ntoa(((struct sockaddr_in*)&ifr.ifr_ifru.ifru_addr)->sin_addr);
    return 0;
}

int if_setmtu(const char *name, int mtu) {
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    ifr.ifr_ifru.ifru_mtu = mtu;
    return if_ioctl(SIOCSIFMTU, &ifr);
}

int if_getmtu(const char *name) {
    int err;
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    if ((err = if_ioctl(SIOCGIFMTU, &ifr)) < 0) {
        return err;
    }

    return ifr.ifr_ifru.ifru_mtu;
}

int if_up(const char *name) {
    int err;
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    if ((err = if_ioctl(SIOCGIFFLAGS, &ifr)) < 0) {
        return err;
    }
    if (!(ifr.ifr_flags & IFF_UP)) {
        ifr.ifr_flags |= IFF_UP;
        return if_ioctl(SIOCSIFFLAGS, &ifr) < 0;
    }
    return 0;
}

int if_down(const char *name) {
    int err;
    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));

    strcpy(ifr.ifr_name, name);
    if ((err = if_ioctl(SIOCGIFFLAGS, &ifr)) < 0) {
        return err;
    }
    if (ifr.ifr_flags & IFF_UP) {
        ifr.ifr_flags &= ~IFF_UP;
        return if_ioctl(SIOCSIFFLAGS, &ifr) < 0;
    }
    return 0;
}

*/
import "C"

import (
	"fmt"
	"log"
	"net"
	"os"
	"unsafe"
)

const TUN_DEV = "/dev/net/tun"

type Tun struct {
	Name string
	file *os.File
}

func CreateTun(name string) (*Tun, error) {
	f, err := os.OpenFile(TUN_DEV, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	cbuf := (*C.char)(C.malloc(C.IFNAMSIZ))
	defer C.free(unsafe.Pointer(cbuf))
	C.memset(unsafe.Pointer(cbuf), 0, C.IFNAMSIZ)
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	C.strncpy(cbuf, c_name, C.IFNAMSIZ-1)

	if C.create_tun(C.int(f.Fd()), cbuf) < 0 {
		return nil, fmt.Errorf("create tun fail")
	}

	return &Tun{C.GoString(cbuf), f}, nil
}

func (t *Tun) Close() error {
	t.Down()
	return t.file.Close()
}

func (t *Tun) SetSelfAddr(ip net.IP) error {
	c_name, c_addr := C.CString(t.Name), C.CString(ip.String())
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_addr))

	if C.if_setaddr(C.SIOCSIFADDR, c_name, c_addr) < 0 {
		return fmt.Errorf("set addr fail")
	}
	return nil
}

func (t *Tun) GetSelfAddr() (ip net.IP, err error) {
	var c_addr *C.char
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))
	if C.if_getaddr(C.SIOCGIFADDR, c_name, &c_addr) < 0 {
		err = fmt.Errorf("get self addr fail")
		return
	}
	return net.ParseIP(C.GoString(c_addr)), nil
}

func (t *Tun) SetDestAddr(ip net.IP) error {
	c_name, c_addr := C.CString(t.Name), C.CString(ip.String())
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_addr))

	if C.if_setaddr(C.SIOCSIFDSTADDR, c_name, c_addr) < 0 {
		return fmt.Errorf("set dest addr fail")
	}
	return nil
}

func (t *Tun) GetDestAddr() (ip net.IP, err error) {
	var c_addr *C.char
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))
	if C.if_getaddr(C.SIOCGIFDSTADDR, c_name, &c_addr) < 0 {
		err = fmt.Errorf("get dest addr fail")
		return
	}
	return net.ParseIP(C.GoString(c_addr)), nil
}

func (t *Tun) SetAddr(self net.IP, dest net.IP) error {
	if err := t.SetSelfAddr(self); err != nil {
		return err
	}
	return t.SetDestAddr(dest)
}

func (t *Tun) SetNetmask(mask net.IPMask) error {
	c_name, c_mask := C.CString(t.Name), C.CString(net.IP(mask).String())
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_mask))

	if C.if_setaddr(C.SIOCSIFNETMASK, c_name, c_mask) < 0 {
		return fmt.Errorf("set netmask fail")
	}
	return nil
}

func (t *Tun) GetNetmask() (mask net.IPMask, err error) {
	var c_addr *C.char
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))
	if C.if_getaddr(C.SIOCGIFNETMASK, c_name, &c_addr) < 0 {
		err = fmt.Errorf("get netmask fail")
		return
	}
	return net.IPMask(net.ParseIP(C.GoString(c_addr))), nil
}

func (t *Tun) SetMTU(mtu int) error {
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))

	if C.if_setmtu(c_name, C.int(mtu)) < 0 {
		return fmt.Errorf("set mtu fail")
	}
	return nil
}

func (t *Tun) GetMTU() (int, error) {
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))

	if mtu := int(C.if_getmtu(c_name)); mtu < 0 {
		return 0, fmt.Errorf("get mtu fail")
	} else {
		return mtu, nil
	}
	return 0, nil
}

func (t *Tun) Up() error {
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))

	if C.if_up(c_name) < 0 {
		return fmt.Errorf("if up fail")
	}
	return nil
}

func (t *Tun) Down() error {
	c_name := C.CString(t.Name)
	defer C.free(unsafe.Pointer(c_name))

	if C.if_down(c_name) < 0 {
		return fmt.Errorf("if down fail")
	}
	return nil
}

func (t *Tun) Read(p []byte) (int, error) {
	return t.file.Read(p)
}

func (t *Tun) Write(p []byte) (n int, err error) {
	return t.file.Write(p)
}

func (t *Tun) ReadChan() (chan []byte, error) {
	ch := make(chan []byte)
	mtu, err := t.GetMTU()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, mtu*2)
	go func() {
		defer close(ch)
		for {
			if n, err := t.Read(buf); err != nil {
				log.Println("read tun fail:", err)
				return
			} else {
				snd := make([]byte, n)
				copy(snd, buf[:n])
				ch <- snd
			}
		}
	}()

	return ch, err
}
