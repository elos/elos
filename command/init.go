package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

type InitCommand struct {
	Ui cli.Ui
	*Config
}

func (c *InitCommand) Help() string {
	helpText := `
	`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Run(args []string) int {
	if c.Config.DB == "" {
		c.Ui.Error("No database listed")
		return 1
	}
	c.Ui.Info("Connecting to db...")
	db, err := models.MongoDB(c.Config.DB)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	c.Ui.Info("Connected")
	c.Ui.Output("Welcome to Elos!")
	c.Ui.Output("We first need to create you an elos user account")

	pass, err := c.Ui.Ask("What would you like your master password to be?")
	if err != nil {
		return 1
	}

	newUser := models.NewUser()
	newUser.SetID(db.NewID())
	newUser.CreatedAt = time.Now()
	newUser.Password = pass
	newUser.UpdatedAt = time.Now()
	if err = db.Save(newUser); err != nil {
		c.Ui.Error(fmt.Sprintf("Failure to save user: %s", err))
		return 1
	}

	c.Config.UserID = newUser.ID().String()
	err = WriteConfigFile(c.Config)
	if err != nil {
		c.Ui.Error("Failed to update user id info")
		return 1
	}

	c.Ui.Info(fmt.Sprintf("User account created, your id is: %s", newUser.ID()))

	return 0
}

func (c *InitCommand) Synopsis() string {
	return "init synopsis"
}
