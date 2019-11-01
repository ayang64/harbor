package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/ayang64/harbor/jail"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("%s run <cmd>", os.Args[0])
	}
	switch os.Args[1] {
	case "run":
		if err := run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	}
}

func run(args []string) error {
	j, err := jail.New(jail.WithHostname("hell.ayan.net"), jail.WithPath("/tmp"))

	if err != nil {
		return err
	}

	j.Create()

	rc, err := j.Attach()

	if err != nil {
		log.Fatal(fmt.Errorf("j.Attach(): %d, %v", rc, err))
	}

	log.Printf("running %v", args)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
