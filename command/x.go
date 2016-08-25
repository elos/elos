package command

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/tag"
	"github.com/elos/models/task"
	"github.com/mitchellh/cli"
)

// XCommand contains the state necessary to implement the
// 'elos x' command set.
//
// It implements the cli.Command interface
type XCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must be non-nil
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified.
	UserID string

	// DB is the elos database we interface with.
	// It must be non-nil
	data.DB
}

// Synopsis is a one-line, short summary of the 'x' command.
// It is guaranteed to be at most 50 characters.
func (c *XCommand) Synopsis() string {
	return "Experimental"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possibly a complete
// list of flags the 'x' command accepts.
func (c *XCommand) Help() string {
	helpText := `
Usage:
	elos x <subcommand>

	review		review things
	`
	return strings.TrimSpace(helpText)
}

func (c *XCommand) Run(args []string) int {
	if len(args) == 0 && c.UI != nil {
		c.UI.Output(c.Help())
		return success
	}

	switch args[0] {
	case "review":
		return c.runReview(args)
	case "tasktime":
		return c.runTaskTime(args)
	}

	return success
}

func (c *XCommand) runTaskTime(args []string) int {
	db := c.DB
	iter, _ := db.Query(models.TagKind).Select(data.AttrMap{"owner_id": c.UserID}).Execute()

	t := models.NewTag()
	for iter.Next(t) {
		tasks, _ := tag.TasksFor(db, t)

		var totalTime time.Duration
		for _, t := range tasks {
			totalTime += task.TimeSpent(t)
		}

		c.UI.Output(t.Name)
		c.UI.Output(fmt.Sprintf("\t%s", totalTime))
	}

	iter.Close()

	return success
}

func (c *XCommand) runReview(args []string) int {
	db := c.DB
	switch args[1] {
	case "taskweek":
		iter, err := db.Query(models.TaskKind).Select(data.AttrMap{"owner_id": c.UserID}).Execute()
		if err != nil {
			log.Fatal(err)
			return failure
		}

		oneWeekAgo := time.Now().Add(-7 * 24 * time.Hour)

		completedInLastWeek := make([]*models.Task, 0)

		t := models.NewTask()
		for iter.Next(t) {
			if t.CompletedAt.Local().After(oneWeekAgo.Local()) {
				completedInLastWeek = append(completedInLastWeek, t)
				t = models.NewTask()
			}
		}

		if len(completedInLastWeek) == 0 {
			c.UI.Output("No tasks completed in last week")
			return success
		}

		sort.Sort(task.ByCompletedAt(completedInLastWeek))

		for _, t := range completedInLastWeek {
			c.UI.Output(fmt.Sprintf("\t* %s [%s]", t.Name, t.CompletedAt.Local().Format("Mon Jan 2")))
		}

	}
	return success
}
