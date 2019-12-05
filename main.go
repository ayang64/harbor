package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ayang64/harbor/jail"
)

type FuncMap map[string]func(...string) error
type Dispatcher struct {
	root    string
	log     *log.Logger
	debug   *log.Logger
	funcmap FuncMap
}

func WithRoot(r string) func(*Dispatcher) error {
	return func(d *Dispatcher) error {
		d.root = r
		return nil
	}
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
	log.SetFlags(log.Llongfile | log.LstdFlags)
	if len(os.Args) < 2 {
		log.Fatalf("%s run <cmd>", os.Args[0])
	}

	fs := flag.NewFlagSet("run", flag.ExitOnError)

	switch os.Args[1] {
	case "run":
		root := fs.String("root", "./freebsd", "path of jail root")
		fs.Parse(os.Args[2:])
		log.Printf("args = %#v", fs.Args())
		abspath, err := filepath.Abs(*root)
		if err != nil {
			log.Fatal(err)
		}
		if err := run(abspath, fs.Args()); err != nil {
			log.Fatal(err)
		}
	}
}

func run(root string, cmd []string) error {

	d, err := NewDispatcher(WithDebugWriter(os.Stderr), WithRoot(root))

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
	// set current working directory to our new root
	if err := os.Chdir(d.root); err != nil {
		return err
	}

	rndhost := func() string {
		// store 128 random bits
		buf := make([]byte, 16, 16)
		rand.Read(buf)
		return fmt.Sprintf("%x", buf)
	}

	// build sub-command execution context.
	id, err := jail.Set(jail.CREATE|jail.ATTACH,
		"path", d.root,
		"name", "foobar",
		"host.domainname", "ayan.net",
		"ip4.addr", net.ParseIP("10.133.88.6").To4(),
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

	log.Printf("root = %s", d.root)
	log.Printf("created jail #%d", id)

	cmd := exec.Command(args[0], args[1:]...)

	// setup stdout, stderr, and stdin for our sub-process.
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	// execute our sub-command within our jail
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run(): cmd.Run() failed with %w", err)
	}

	return nil
}
