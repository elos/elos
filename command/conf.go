package command

import (
	"fmt"

	"github.com/mitchellh/cli"
)

type ConfCommand struct {
	Ui cli.Ui
	*Config
}

func (c *ConfCommand) Help() string {
	return ""
}

func (c *ConfCommand) Run(args []string) int {
	if len(args) == 0 {
		c.Ui.Output("Your current configuration:")
		c.Ui.Output(fmt.Sprintf("Host: %s", c.Config.Host))
		return 0
	}

	switch args[0] {
	case "host":
		c.Ui.Output(fmt.Sprintf("Your current host is %s", c.Config.Host))
		host, err := c.Ui.Ask("What would you like your new host to be?")

		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		// if host valid, currently only checks non empty
		if host != "" {
			c.Config.Host = host
		} else {
			c.Ui.Warn("You entered an empty host name")
		}

		if err := WriteConfigFile(c.Config); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to persist configuration change: %s", err))
			return 1
		} else {

			c.Ui.Output(fmt.Sprintf("Your new host is %s", c.Config.Host))
			break
		}
	}

	return 0
}

func (c *ConfCommand) Synopsis() string {
	return "Configuration information and utilities"
}
