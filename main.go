package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/ayang64/harbor/jail"
)

type FuncMap map[string]func(...string) error
type Dispatcher struct {
	log     *log.Logger
	debug   *log.Logger
	funcmap FuncMap
}

func WithLogWriter(w io.Writer) func(*Dispatcher) error {
	return func(d *Dispatcher) error {
		d.log = log.New(w, "", log.LstdFlags|log.Lshortfile)
		return nil
	}
}

func WithDebugWriter(w io.Writer) func(*Dispatcher) error {
	return func(d *Dispatcher) error {
		d.debug = log.New(w, "DEBUG ", log.LstdFlags|log.Lshortfile)
		return nil
	}
}

func NewDispatcher(opts ...func(*Dispatcher) error) (*Dispatcher, error) {
	rc := Dispatcher{
		log:   log.New(ioutil.Discard, "", 0),
		debug: log.New(ioutil.Discard, "", 0),
	}

	rc.funcmap = FuncMap{
		"run": rc.run, // args...
	}

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("%s run <cmd>", os.Args[0])
	}
	switch os.Args[1] {
	case "run":
		if err := run(os.Args[2], os.Args[3:]); err != nil {
			log.Fatal(err)
		}
	}
}

func run(root string, args []string) error {
	// set current working directory to our new root
	if err := os.Chdir(root); err != nil {
		return err
	}

	rndhost := func() string {
		// store 128 random bits
		buf := make([]byte, 16, 16)
		rand.Read(buf)
		return fmt.Sprintf("%x", buf)
	}

	id, err := jail.Set(jail.CREATE|jail.ATTACH,
		"path", "/home/ayan/jailroot",
		"name", "foobar",
		"host.domainname", "ayan.net",
		"ip4.addr", net.ParseIP("192.168.0.18").To4(),
		"allow.raw_sockets", true,
		"children.max", 10,
		"enforce_statfs", 1,
		"allow.socket_af", true,
		"allow.mount", true,
		"allow.mount.devfs", true,
		"allow.mount.procfs", true,
		"host.hostname", rndhost())

	if err != nil {
		return err
	}

	log.Printf("created jail #%d", id)

	d, err := NewDispatcher(WithDebugWriter(os.Stderr))

	d.debug.Printf("debuging output is enabled.")

	if err != nil {
		log.Fatal(err)
	}

	if err := d.dispatch(os.Args...); err != nil {
		return err
	}

	return nil
}

func (d *Dispatcher) dispatch(args ...string) error {
	f, exists := d.funcmap[args[1]]
	if !exists {
		return fmt.Errorf("undefined subcommand %q", args[1])
	}
	if err := f(args[3:]...); err != nil {
		return err
	}
	return nil
}

func (d *Dispatcher) run(args ...string) error {
	// build sub-command execution context.
	cmd := exec.Command(args[0], args[1:]...)

	// setup stdout, stderr, and stdin for our sub-process.
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	// execute our sub-command within our jail
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run(): cmd.Run() failed with %w", err)
	}

	return nil
}
