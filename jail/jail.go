package jail

import (
	"crypto/rand"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// Description of the prision that we're building.
type jail struct {
	version  uint32
	path     unsafe.Pointer
	hostname unsafe.Pointer
	jailname unsafe.Pointer
	ip4s     int
	ip6s     int
	ip4      unsafe.Pointer
	ip6      unsafe.Pointer
}

type Jail struct {
	ID       int
	Version  uint32
	Path     string
	Hostname string
	Name     string
	IP       []net.IP
}

func WithIPv4Address(a string) func(*Jail) error {
	return func(j *Jail) error {
		j.IP = append(j.IP, net.ParseIP(a))
		return nil
	}
}

func WithPath(p string) func(*Jail) error {
	return func(j *Jail) error {
		j.Path = p
		return nil
	}
}

func WithHostname(h string) func(*Jail) error {
	return func(j *Jail) error {
		j.Hostname = h
		return nil
	}
}

func New(opts ...func(*Jail) error) (*Jail, error) {
	rndhost := func() string {
		// create 128 bytes of random bits
		buf := make([]byte, 128, 128)
		rand.Read(buf)
		return fmt.Sprintf("%x", buf)
	}

	rc := Jail{
		Version:  2,
		Path:     "/tmp",
		Hostname: rndhost(),
		IP:       []net.IP{net.ParseIP("127.0.0.1")},
	}

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}

func (j *Jail) String() string {
	return fmt.Sprintf("{id: %d, path: %s}", j.ID, j.Path)
}

func (j *Jail) book() (*jail, error) {
	cstr := func(s string) unsafe.Pointer {
		str := append([]byte(s), 0)
		return unsafe.Pointer(&str[0])
	}

	rc := jail{
		version:  j.Version,
		path:     cstr(j.Path),
		hostname: cstr(j.Hostname),
		jailname: cstr(j.Name),
	}

	v4addr, v6addr := [][]byte{}, [][]byte{}

	for i := range j.IP {
		switch len(j.IP[i]) {
		case 4:
			v4addr = append(v4addr, []byte(j.IP[i]))
		case 16:
			v6addr = append(v6addr, []byte(j.IP[i]))
		default:
			return nil, fmt.Errorf("unexpected address type found in jail template")
		}
	}

	if rc.ip4s = len(v4addr); rc.ip4s > 0 {
		rc.ip4 = unsafe.Pointer(&v4addr[0])
	}

	if rc.ip6s = len(v6addr); rc.ip6s > 0 {
		rc.ip6 = unsafe.Pointer(&v6addr[0])
	}
	return &rc, nil
}

func (j *Jail) Create() error {
	_j, err := j.book()
	if err != nil {
		return err
	}

	jail := func() (int, error) {
		rc, _, errno := syscall.Syscall(syscall.SYS_JAIL, uintptr(unsafe.Pointer(_j)), 0, 0)
		if errno != 0 {
			return -1, error(errno)
		}
		return int(rc), nil
	}

	jid, err := jail()
	if err != nil {
		return err
	}
	j.ID = jid
	return nil
}
