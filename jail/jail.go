package jail

import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"
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

func (j *Jail) Get() {
}

func (j *Jail) Remove() {
}

type SetFlag int

const (
	CREATE = SetFlag(0x1 << iota) // Create jail if it doesn't exist
	UPDATE                        // Update parameters of existing jail
	ATTACH                        // Attach to jail upon creation
)

func (s SetFlag) String() string {
	f := []string(nil)
	if s&CREATE != 0 {
		f = append(f, "jail.CREATE")
	}
	if s&UPDATE != 0 {
		f = append(f, "jail.UPDATE")
	}
	if s&ATTACH != 0 {
		f = append(f, "jail.ATTACH")
	}

	switch len(f) {
	case 0:
		return "*no-flags-set*"
	case 1:
		return f[0]
	default:
		return `(` + strings.Join(f, "|") + `)`
	}
}

/*
	reasons for EINVAL

		A supplied parameter is the wrong size.
		A supplied parameter is out of range.
		A supplied string parameter is not null-terminated.
		A supplied parameter name does not match any known parameters.
		One of the JAIL_CREATE or JAIL_UPDATE flags is not set.
*/

func toIovec(opts ...interface{}) ([]syscall.Iovec, error) {
	// yes, i know i could have used sysctl.BytePtrFromString()
	cstr := func(s string) (*byte, uint64) {
		str := append([]byte(s), byte(0))
		return (*byte)(unsafe.Pointer(&str[0])), uint64(len(str))
	}
	iov := make([]syscall.Iovec, len(opts), len(opts))
	for i, opt := range opts {
		cur := syscall.Iovec{}
		switch i % 2 {
		case 0:
			// this must be a string
			switch v := opt.(type) {
			case string:
				cur.Base, cur.Len = cstr(v)
			default:
				return nil, fmt.Errorf("parameter must be a string")
			}
		case 1:
			switch v := opt.(type) {
			case string:
				cur.Base, cur.Len = cstr(v)
			case int:
				// this must be 32 bit int; jail() is old.
				i := uint32(v)
				cur.Base, cur.Len = (*byte)(unsafe.Pointer(&i)), uint64(4)
			case net.IP:
				cur.Base, cur.Len = (*byte)(unsafe.Pointer(&v[0])), uint64(len(v))
			case []byte:
				cur.Base, cur.Len = (*byte)(unsafe.Pointer(&v[0])), uint64(len(v))
			case bool:
				cur.Base, cur.Len = (*byte)(unsafe.Pointer(uintptr(0))), 0
			default:
				return nil, fmt.Errorf("unexpected type")
			}
		}
		iov[i] = cur
	}
	return iov, nil
}

func Set(flags SetFlag, opts ...interface{}) (int, error) {
	iov, err := toIovec(opts...)
	if err != nil {
		return -1, err
	}
	niov := len(iov)
	rc, _, errno := syscall.Syscall(syscall.SYS_JAIL_SET, uintptr(unsafe.Pointer(&iov[0])), uintptr(niov), uintptr(flags))
	if errno != 0 {
		return -1, error(errno)
	}
	return int(rc), nil
}
