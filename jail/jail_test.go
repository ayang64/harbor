package jail

import (
	"net"
	"testing"
	"time"
)

func TestJailCreate(t *testing.T) {
	j := Jail{
		Version:  2,
		Path:     "/tmp",
		Hostname: "ayan.net",
		IP:       []net.IP{net.ParseIP("127.0.0.1")},
	}
	if err := j.Create(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Second)
}
