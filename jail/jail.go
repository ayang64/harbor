package jail

import (
	"fmt"
	"log"
	"net"
	"syscall"
	"unsafe"
)

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

/*
	SYS_JAIL                     = 338 // { int jail(struct jail *jail); }
	SYS_JAIL_ATTACH              = 436 // { int jail_attach(int jid); }
	SYS_JAIL_GET                 = 506 // { int jail_get(struct iovec *iovp, \
	SYS_JAIL_SET                 = 507 // { int jail_set(struct iovec *iovp, \
	SYS_JAIL_REMOVE              = 508 // { int jail_remove(int jid); }
*/

// jail_attach		- Jail.Attach()
// jail create		- Jail.Create()
// jail_get				- Jail.Get()
// jail_remove		- Jail.Remove()
// jail_set				- Jail.Set()

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
	rc := Jail{
		Version:  2,
		Path:     "/tmp",
		Hostname: "random-hostname",
		IP:       []net.IP{net.ParseIP("127.0.0.1")},
	}

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}

func (j *Jail) Attach() (int, error) {
	attach := func(jid int) (int, error) {
		log.Printf("attaching to %d", jid)
		rc, _, errno := syscall.Syscall(syscall.SYS_JAIL_ATTACH, uintptr(jid), 0, 0)
		if errno != 0 {
			return int(rc), error(errno)
		}
		return int(rc), nil
	}

	rc, err := attach(j.ID)

	if err != nil {
		return rc, err
	}

	return rc, nil
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

	// jid := (*int)(unsafe.Pointer(rc1))
	log.Printf("jid = %d", jid)

	j.ID = jid

	return nil
}

func (j *Jail) Get() {
}

func (j *Jail) Remove() {
}

func (j *Jail) Set() {
}
