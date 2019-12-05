package jail

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"unsafe"
)

// SetFlag encodes an int bit mask that is passed to jail.Set()
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
