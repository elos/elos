package main

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

var (
	UI            cli.Ui
	Commands      map[string]cli.CommandFactory
	Configuration *command.Config
)

func init() {
	UI = &cli.BasicUi{Writer: os.Stdout, Reader: os.Stdin}

	user, err := user.Current()
	if err != nil {
		UI.Error(err.Error())
		os.Exit(1)
	}

	configPath := path.Join(user.HomeDir, command.ConfigFileName)

	c, err := command.ParseConfigFile(configPath)
	if err != nil {
		UI.Error(err.Error())
		os.Exit(1)
	}

	Configuration = c

	var db data.DB
	var databaseError error
	if Configuration.DB != "" {
		db, databaseError = models.MongoDB(Configuration.DB)
	} else {
		databaseError = fmt.Errorf("No databse listed")
	}

	Commands = map[string]cli.CommandFactory{
		"auth": func() (cli.Command, error) {
			return &command.AuthCommand{
				UI:     UI,
				Config: Configuration,
			}, nil
		},
		"cal": func() (cli.Command, error) {
			return &command.CalCommand{
				UI:     UI,
				Config: Configuration,
				DB:     db,
			}, databaseError
		},
		"conf": func() (cli.Command, error) {
			return &command.ConfCommand{
				Ui:     UI,
				Config: Configuration,
			}, nil
		},
		"data": func() (cli.Command, error) {
			return &command.DataCommand{
				Ui:     UI,
				Config: Configuration,
			}, nil
		},
		"init": func() (cli.Command, error) {
			return &command.InitCommand{
				Ui:     UI,
				Config: Configuration,
			}, nil
		},
		"note": func() (cli.Command, error) {
			return &command.NoteCommand{
				Ui:     UI,
				Config: Configuration,
				DB:     db,
			}, databaseError
		},
		"todo": func() (cli.Command, error) {
			return &command.TodoCommand{
				UI:     UI,
				Config: Configuration,
				DB:     db,
			}, databaseError
		},
	}

}
