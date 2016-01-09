package command

import "github.com/mitchellh/cli"

type HostCommand struct {
	cli.Ui
}

func (c *HostCommand) Help() string {
	return ""
}

func (c *HostCommand) Run(args []string) int {
	// Print help if no args
	if len(args) == 0 {
		c.Output(c.Help())
		goto Success
	}

	switch args[0] {
	case "set":
		c.Output("I see you want to set something")
	}

Success:
	return 0
}

func (c *HostCommand) Synopsis() string {
	return "Configuration utilities for setting up the Elos CLI"
}
