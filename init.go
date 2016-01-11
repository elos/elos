package main

import (
	"os"
	"os/user"
	"path"

	"github.com/elos/elos/command"
	"github.com/mitchellh/cli"
)

var (
	UI            cli.Ui
	Commands      map[string]cli.CommandFactory
	Configuration *command.Config
)

func init() {
	UI = &cli.BasicUi{Writer: os.Stdout, Reader: os.Stdin}

	Commands = map[string]cli.CommandFactory{
		"auth": func() (cli.Command, error) {
			return &command.AuthCommand{
				UI:     UI,
				Config: Configuration,
			}, nil
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
			}, nil
		},
	}

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
}
