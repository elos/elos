package command

import (
	"fmt"
	"log"

	"go/build"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// PackagePath finds the full path for the specified
// golang import path
//
// i.e., PackPath("github.com/elos/ehttp")
// => "~/Nick/workspace/go/src/github.com/elos/ehttp" (on my computer)
func PackagePath(importPath string) string {
	p, err := build.Default.Import(importPath, "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

var kinds = map[data.Kind]map[string]metis.Primitive{
	"note": models.NoteStructure,
}

type DataCommand struct {
	Ui cli.Ui
	*Config
}

func (c *DataCommand) Help() string {
	return "elos data --help"
}

func (c *DataCommand) Run(args []string) int {
	// If we were given no information, simply print the help
	if len(args) == 0 {
		c.Ui.Output(c.Help())
		return 0
	}

	switch args[0] {
	case "kinds":
		c.Ui.Output("The recognized data kinds are:")
		for k, _ := range kinds {
			c.Ui.Output(fmt.Sprintf(" * %s", k))
		}
		break
	case "new":
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

		kind := ""
		for ok := false; !ok && err == nil; _, ok = kinds[data.Kind(kind)] {
			if kind != "" {
				c.Ui.Warn("That is not a recognized kind")
			}
			kind, err = c.Ui.Ask("What kind of data?")
		}

		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		structure := kinds[data.Kind(kind)]

		c.Ui.Info(fmt.Sprintf("Ok, here are the traits of a %s", kind))
		for k, _ := range structure {
			c.Ui.Info(fmt.Sprintf(" * %s", k))
		}

		attrs := make(data.AttrMap)

		for key, _ := range kinds[data.Kind(kind)] {
			input, err := c.Ui.Ask(fmt.Sprintf("%s:", key))

			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}

			attrs[key] = input
		}

		m := models.ModelFor(data.Kind(kind))
		log.Printf("%+v", attrs)
		transfer.Unmarshal(attrs, m)

		c.Ui.Info("Saving record...")
		if m.ID().String() == "" {
			m.SetID(db.NewID())
		}

		log.Printf("%+v", m)

		err = db.Save(m)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		c.Ui.Info("Saved")
		break
	}

	return 0
}

func (c *DataCommand) Synopsis() string {
	return "Utility for managing elos data"
}
