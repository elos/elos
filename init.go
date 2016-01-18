package main

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/gaia"
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

	if Configuration.DirectDB {
		if Configuration.DB != "" {
			db, databaseError = models.MongoDB(Configuration.DB)
		} else {
			databaseError = fmt.Errorf("No database listed")
		}
	} else {
		db = &gaia.DB{
			URL:      Configuration.Host,
			Username: Configuration.PublicCredential,
			Password: Configuration.PrivateCredential,
			Client:   new(http.Client),
		}
	}

	Commands = map[string]cli.CommandFactory{
		"conf": func() (cli.Command, error) {
			return &command.ConfCommand{
				Ui:     UI,
				Config: Configuration,
			}, nil
		},
		"habit": func() (cli.Command, error) {
			return &command.HabitCommand{
				UI:     UI,
				UserID: Configuration.UserID,
				DB:     db,
			}, databaseError
		},
		"people": func() (cli.Command, error) {
			return &command.PeopleCommand{
				UI:     UI,
				UserID: Configuration.UserID,
				DB:     db,
			}, databaseError
		},
		"setup": func() (cli.Command, error) {
			return &command.SetupCommand{
				UI:     UI,
				Config: Configuration,
			}, nil
		},
		"todo": func() (cli.Command, error) {
			return &command.TodoCommand{
				UI:     UI,
				UserID: Configuration.UserID,
				DB:     db,
			}, databaseError
		},
	}

}
