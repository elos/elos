package command

import (
	"fmt"
	"strings"
	"time"

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
	return "Stream your events"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possible a complete
// list of flags the 'stream' command accepts.
func (c *StreamCommand) Help() string {
	helpText := `
Usage:
	elos stream		start streaming the events
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

	changes := *c.DB.Changes()

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				c.UI.Output("Connection closed by server")
				return success
			}

			if change.ChangeKind != data.Update {
				continue
			}

			if change.Record.Kind() != models.EventKind {
				continue
			}

			e := change.Record.(*models.Event)

			tags, err := e.Tags(c.DB)
			if err != nil {
				// TODO errorf
				return failure
			}

			tagString := ""
			for _, t := range tags {
				tagString += fmt.Sprintf(" [%s]", t.Name)
			}
			if tagString == "" {
				tagString = " "
			} else {
				tagString += ": "
			}

			loc, err := e.Location(c.DB)
			if err != nil && err != models.ErrEmptyLink {
				// TODO errorf
				return failure
			}

			locString := ""
			if loc != nil {
				locString = fmt.Sprintf("(lat: %f, lon: %f, alt: %f)", loc.Latitude, loc.Longitude, loc.Altitude)
			}
			c.UI.Output(fmt.Sprintf("%s%s %s", tagString, e.Name, locString))

			n, err := e.Note(c.DB)
			if err != nil && err != models.ErrEmptyLink {
				// TODO errorf
				return failure
			}
			if n != nil {
				c.UI.Output(fmt.Sprintf("\tNote: %s", n.Text))
			}
		case <-time.After(5 * time.Second):
			c.UI.Output("5 second heartbeat")
		}
	}

	return success
}

// errorf is a IO function which performs the equivalent of log.Errorf
// in the standard lib, except using the cli.Ui interface with which
// the StreamCommand was provided.
func (c *StreamCommand) errorf(s string, values ...interface{}) {
	c.UI.Error("[elos stream] Error: " + fmt.Sprintf(s, values...))
}
