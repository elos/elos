package command

import (
	"fmt"
	"log"
	"strings"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// StreamCommand contains the state necessary to implmenet the
// 'elos stream' command.
//
// It implements the cli.Command interface
type StreamCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must be non-nil
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified
	UserID string

	// DB is the elos database we interface with.
	data.DB
}

// Synopsis is a one-line, short summary of the 'stream' command.
// It is guaranteed to be at most 50 characters.
func (c *StreamCommand) Synopsis() string {
	return "Stream you events"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possible a complete
// list of flags the 'stream' command accepts.
func (c *StreamCommand) Help() string {
	helpText := `
Usage:
	elos stream
	`
	return strings.TrimSpace(helpText)
}

func (c *StreamCommand) Run(args []string) int {
	if c.UI == nil {
		return failure
	}

	if c.UserID == "" {
		c.errorf("no user id")
		return failure
	}

	if c.DB == nil {
		c.errorf("no db")
		return failure
	}

	// TODO fix this assumption:
	// asumption that this is a gaia db, which means that
	// the only changes are event changes de facto, need to
	// figure out how to transfer kind information over the wire
	changes := c.DB.Changes()

	log.Print("waiting for changes")
	for change := range *changes {
		log.Print("CHANGE")

		if change.ChangeKind != data.Update {
			continue
		}

		event := change.Record.(*models.Event)

		c.UI.Output(event.Name)
	}

	return success
}

// errorf is a IO function which performs the equivalent of log.Errorf
// in the standard lib, except using the cli.Ui interface with which
// the StreamCommand was provided.
func (c *StreamCommand) errorf(s string, values ...interface{}) {
	c.UI.Error("[elos stream] Error: " + fmt.Sprintf(s, values...))
}