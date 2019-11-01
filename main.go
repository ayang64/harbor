package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"

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
		"run":  rc.run,  // args...
		"fork": rc.fork, // jid args...
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
		log.Fatal("%s run <cmd>", os.Args[0])
	}

	d, err := NewDispatcher(WithDebugWriter(os.Stderr))

	d.debug.Printf("debuging output is enabled.")

	if err != nil {
		log.Fatal(err)
	}

	if err := d.dispatch(os.Args...); err != nil {
		log.Fatal(err)
	}
}

func (d *Dispatcher) dispatch(args ...string) error {
	f, exists := d.funcmap[args[1]]
	if !exists {
		return fmt.Errorf("undefined subcommand %q", args[1])
	}
	if err := f(args[2:]...); err != nil {
		return err
	}
	return nil
}

func (d *Dispatcher) fork(args ...string) error {
	jid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	d.debug.Printf("about to jail.Attach(%d)", jid)
	if val, err := jail.Attach(jid); err != nil {
		d.debug.Printf("jail.Attach(%d) failed with val %d: %v", jid, val, err)
		return err
	}

	d.debug.Printf("Executing subcommand %#v %#v", args[1], args[2:])
	cmd := exec.Command(args[1], args[2:]...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (d *Dispatcher) run(args ...string) error {
	j, err := jail.New(jail.WithHostname("hell.ayan.net"), jail.WithPath("/"))
	if err != nil {
		return err
	}
	j.Create()

	d.log.Printf("running %v", args)

	cmd := exec.Command("/proc/curproc/file", append([]string{"fork", fmt.Sprintf("%d", j.ID)}, args[1:]...)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run(): cmd.Run() failed with %w", err)
	}

	return nil
}
