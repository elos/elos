package command

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/elos/x/data"
	"github.com/elos/x/models"
	"github.com/elos/x/models/cal"
	"github.com/mitchellh/cli"
)

type Cal2Command struct {
	// UI is used to communicate (for IO) with the user.
	UI cli.Ui

	// UserID is the id of the user on whose behalf this
	// command acts
	UserID string

	// The client to the database
	data.DBClient
}

func (c *Cal2Command) Synopsis() string {
	return "Utilities for managing the [new] elos scheduling system"
}
func (c *Cal2Command) Help() string {
	return `
Usage:
	elos cal2 <subcommand>

Subcommands:
	week	list the events for this week
`
}

func (c *Cal2Command) Run(args []string) int {
	if len(args) == 0 {
		c.UI.Output(c.Help())
		return 0
	}

	switch args[0] {
	case "week":
		return c.runWeek(args[1:])
	default:
		c.UI.Output(c.Help())
		return 0
	}
}

func (c *Cal2Command) runWeek(args []string) int {
	results, err := c.DBClient.Query(context.Background(), &data.Query{
		Kind: models.Kind_FIXTURE,
		Filters: []*data.Filter{
			{
				Op:    data.Filter_EQ,
				Field: "owner_id",
				Reference: &data.Filter_String_{
					String_: c.UserID,
				},
			},
		},
	})
	if err != nil {
		c.UI.Error(fmt.Sprintf("w.db.Query error: %v", err))
		return 1
	}

	fixtures := make([]*models.Fixture, 0)
	for {
		rec, err := results.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.UI.Error(fmt.Sprintf("results.Recv error: %v", err))
			return 1
		}
		fixtures = append(fixtures, rec.Fixture)
	}

	firstDay := cal.DateFrom(time.Now())
	es := cal.EventsWithin(firstDay.Time(), firstDay.Time().AddDate(0, 1, 0), fixtures)
	for _, e := range es {
		c.UI.Output(fmt.Sprintf(" - %s [%s]", e.Name, e.Start.Time()))
	}
	return 0
}
