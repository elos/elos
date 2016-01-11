package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

type AuthCommand struct {
	UI cli.Ui
	*Config
}

func (c *AuthCommand) Help() string {
	helpText := `
	auth help

	`
	return strings.TrimSpace(helpText)
}

func (c *AuthCommand) Run(args []string) int {
	switch len(args) {
	case 0:
		return c.askCredentials()
	case 1:
		switch args[0] {
		case "id":
			c.UI.Output("You seem to think you are in an admin mode")
			id, err := c.UI.Ask("What user id would you like to act on behalf?")
			if err != nil {
				c.UI.Error(err.Error())
				return 1
			}
			c.Config.UserID = id
			err = WriteConfigFile(c.Config)
			if err != nil {
				c.UI.Error("Failed to update user id info")
				return 1
			}
			return 0
		}

	}

	c.UI.Output(c.Help())
	return 0
}

func (c *AuthCommand) askCredentials() int {
	var public, private string
	var err error
	public, err = c.UI.Ask("Public Credential:")
	if err != nil {
		return 1
	}
	private, err = c.UI.AskSecret("Private Credential:")
	if err != nil {
		return 1
	}

	c.Config.PublicCredential = public
	c.Config.PrivateCredential = private

	err = WriteConfigFile(c.Config)
	if err != nil {
		c.UI.Error("Failed to update authorization information")
		return 1
	}
	return 0
}

func (c *AuthCommand) Synopsis() string {
	return "Authorization utilities"
}
