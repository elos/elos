package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	"google.golang.org/grpc"

	olddata "github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/gaia"
	"github.com/elos/models"
	"github.com/elos/x/auth"
	"github.com/elos/x/data"
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

	var db olddata.DB
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
	conn, err := grpc.Dial(
		"elos.pw:4444",
		grpc.WithPerRPCCredentials(
			auth.RawCredentials(
				Configuration.Credential.Public,
				Configuration.Credential.Private,
			),
		),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal(err)
	}
	// don't close connection because we'd lose connection should
	// move declaration of dbc higher in scope TODO(nclandolfi)
	dbc := data.NewDBClient(conn)

	Commands = map[string]cli.CommandFactory{
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
		"tag": func() (cli.Command, error) {
			return &command.TagCommand{
				UI:     UI,
				UserID: Configuration.UserID,
				DB:     db,
			}, databaseError
		},
		"stream": func() (cli.Command, error) {
			return &command.StreamCommand{
				UI:     UI,
				UserID: Configuration.UserID,
				DB:     db,
			}, databaseError
		},
		"todo": func() (cli.Command, error) {
			return &command.TodoCommand{
				UI:     UI,
				UserID: Configuration.Credential.OwnerID,
				DB:     data.DB(dbc),
			}, databaseError
		},
		"cal2": func() (cli.Command, error) {
			return &command.Cal2Command{
				UI:       UI,
				UserID:   Configuration.Credential.OwnerID,
				DBClient: dbc,
			}, nil
		},
		"records": func() (cli.Command, error) {
			return &command.RecordsCommand{
				UI:       UI,
				UserID:   Configuration.Credential.OwnerID,
				DBClient: dbc,
			}, nil
		},
	}
}
