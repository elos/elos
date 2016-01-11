package main

import (
	"os"

	"github.com/mitchellh/cli"
)

func main() {
	// Construct a new CLI with our name and version
	c := cli.NewCLI("elos", "0.0.1")

	// Pass along all the arguments from the operating system
	c.Args = os.Args[1:]

	// Configure the commands (var 'Commands' is defined in init.go)
	c.Commands = Commands

	// Deploy to the correct command (let package cli take over)
	exitStatus, err := c.Run()

	// Acknowledge an error
	if err != nil {
		UI.Error(err.Error())
	}

	// Use the exit status of the CLI's run
	os.Exit(exitStatus)
}
