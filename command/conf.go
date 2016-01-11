package command

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
)

type ConfCommand struct {
	Ui cli.Ui
	*Config
}

func (c *ConfCommand) Help() string {
	helpText := `
Usage: elos conf {[field] | edit} [edit]

	Looks up the current elos configuration. If a field is provided,
	it looks up information specific to that field. If the edit suffix
	is provided, it opens the appropriate configuration editing prompt

	When no field provided elos conf prints the entire current configuration.

	Note: You may not edit or view your user credentials here, you must use
	'elos auth' to do so.

Examples:
	elos conf				Prints all configuration
	elos conf edit			Edits all configuration
	elos conf <field>		Prints field's configuration
	elos conf <field> edit	Edits fields configuration

`
	return strings.TrimSpace(helpText)
}

func (c *ConfCommand) Run(args []string) int {
	if len(args) == 0 {
		// Print the current output
		c.Ui.Output("Your current configuration:")
		c.Ui.Output(fmt.Sprintf("Host: %s", c.Config.Host))
		c.Ui.Output(fmt.Sprintf("DB: %s", c.Config.DB))
		return 0
	}

	switch args[0] {
	case "edit":
		return c.editConf(args)
	case "host":
		if len(args) == 2 && args[1] == "edit" {
			return c.editHost()
		}

		c.Ui.Output(fmt.Sprintf("Your current host is %s", c.Config.Host))
		break
	case "db":
		if len(args) == 2 && args[1] == "edit" {
			return c.editDB()
		}

		c.Ui.Output(fmt.Sprintf("Your current db is %s:", c.Config.DB))
		break
	case "help":
		fallthrough
	case "-help":
		fallthrough
	case "--help":
		fallthrough
	case "h":
		c.Ui.Output(c.Help())
		return 0
	default:
		c.Ui.Error(fmt.Sprintf("The %s configuration field is not recognized.", args[0]))
		return 1
	}

	return 0
}

func (c *ConfCommand) editConf(args []string) int {
	if o := c.editHost(); o != 0 {
		return o
	}

	if o := c.editDB(); o != 0 {
		return o
	}

	return 0
}

func (c *ConfCommand) editHost() int {
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
	}

	c.Ui.Output(fmt.Sprintf("Your new host is %s", c.Config.Host))

	return 0
}

func (c *ConfCommand) editDB() int {
	c.Ui.Output(fmt.Sprintf("Your current db is %s:", c.Config.DB))
	db, err := c.Ui.Ask("What would you like your new db to be?")

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if db != "" {
		c.Config.DB = db
	} else {
		c.Ui.Warn("You entered an empty db address")
	}

	if err := WriteConfigFile(c.Config); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to persist configuration change: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Your new db is %s", c.Config.DB))

	return 0
}

func (c *ConfCommand) Synopsis() string {
	return "Configuration information and utilities"
}
