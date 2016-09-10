package command

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/elos/x/data"
	"github.com/elos/x/models"
	"github.com/mitchellh/cli"
)

type RecordsCommand struct {
	UI cli.Ui

	UserID string

	data.DBClient
}

func (c *RecordsCommand) Synopsis() string {
	return "Utilities for managing elos records"
}

func (c *RecordsCommand) Help() string {
	helpText := `
Usage:
	elos records <subcommand>

Subcommands:
	kinds	    list known kinds
	count       count records
	query		create a query
	changes		listen for changes
`
	return strings.TrimSpace(helpText)
}

func (c *RecordsCommand) Run(args []string) int {
	if len(args) == 0 && c.UI != nil {
		c.UI.Output(c.Help())
		return success
	}

	switch args[0] {
	case "kinds":
		return c.runKinds()
	case "count":
		return c.runCount()
	case "query":
		return c.runQuery()
	case "changes":
		return c.runChanges()
	}

	c.UI.Output(c.Help())
	return success
}

var kinds string

func init() {
	s := make([]string, len(models.Kinds))
	for i, k := range models.Kinds {
		s[i] = "* " + k.String()
	}
	kinds = strings.Join(s, "\n")
}

func (c *RecordsCommand) runKinds() int {
	c.UI.Output(kinds)
	return success
}

func (c *RecordsCommand) runCount() int {
	k, err := stringInput(c.UI, "Which kind?")
	if err != nil {
		return failure
	}

	results, err := c.DBClient.Query(context.Background(),
		&data.Query{
			Kind: models.Kind(models.Kind_value[strings.ToUpper(k)]),
		})
	if err != nil {
		return failure
	}

	n := 0

	for {
		_, err := results.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return failure
		}

		n++
	}

	c.UI.Output(fmt.Sprintf("%d", n))

	return success
}

func (c *RecordsCommand) runQuery() int {
	k, err := stringInput(c.UI, "Which kind?")
	if err != nil {
		return failure
	}

	results, err := c.DBClient.Query(context.Background(),
		&data.Query{
			Kind: models.Kind(models.Kind_value[strings.ToUpper(k)]),
		})
	if err != nil {
		return failure
	}

	n := 0

	for {
		r, err := results.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return failure
		}
		c.UI.Output(fmt.Sprintf("%v", r))

		n++
	}

	c.UI.Output(fmt.Sprintf("%d results", n))

	return success
}

func (c *RecordsCommand) runChanges() int {
	k, err := stringInput(c.UI, "Which kind?")
	if err != nil {
		return failure
	}

	results, err := c.DBClient.Changes(context.Background(),
		&data.Query{
			Kind: models.Kind(models.Kind_value[strings.ToUpper(k)]),
		})
	if err != nil {
		return failure
	}

	for {
		r, err := results.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return failure
		}
		c.UI.Output(fmt.Sprintf("%v", r))
	}

	c.UI.Output("stream closed by server")

	return success
}
