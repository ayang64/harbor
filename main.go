package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/ayang64/harbor/jail"
)

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
		log.Fatal(err)
	}

	log.Printf("started jail #%d", id)

	// build sub-command execution context.
	cmd := exec.Command(args[0], args[1:]...)

	// setup stdout, stderr, and stdin for our sub-process.
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	// execute our sub-command within our jail
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
