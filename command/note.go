package command

import (
	"fmt"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

type NoteCommand struct {
	Ui cli.Ui
	*Config
	data.DB
}

func (c *NoteCommand) Help() string {
	return "note help"
}

func (c *NoteCommand) Run(args []string) int {
	switch len(args) {
	case 0:
		c.Ui.Output(c.Help())
	case 1:
		if c.DB == nil {
			c.Ui.Error("No database listed")
			return 1
		}

		if c.Config.UserID == "" {
			c.Ui.Error("No user id listed")
			return 1
		}

		switch args[0] {
		case "new":
			text, err := c.Ui.Ask("What would you like to make note of?:")
			if err != nil {
				return 1
			}

			note := models.NewNote()
			note.SetID(c.DB.NewID())
			note.OwnerId = c.Config.UserID
			note.CreatedAt = time.Now()
			note.Text = text
			note.UpdatedAt = time.Now()

			err = c.DB.Save(note)
			if err != nil {
				c.Ui.Error("Failed to save note")
				return 1
			}

			c.Ui.Output("Noted")
		case "list":
			q := c.DB.NewQuery(models.NoteKind)
			q.Select(data.AttrMap{
				"owner_id": c.Config.UserID,
			})
			iter, err := q.Execute()
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error executing query: %s", err))
				return 1
			}

			n := models.NewNote()
			notes := make([]*models.Note, 0)
			for iter.Next(n) {
				notes = append(notes, n)
				n = models.NewNote()
			}

			if err := iter.Close(); err != nil {
				c.Ui.Error(fmt.Sprintf("Error executing query: %s", err))
				return 1
			}

			c.Ui.Output("Here are your notes")
			for i := range notes {
				c.Ui.Output(fmt.Sprintf("-----------%d-------------", i))
				c.Ui.Output(notes[i].Text)
			}

			t, err := c.Ui.Ask("Would you like to [D]elete or [E]dit any? (enter to continue)")
			if err != nil {
				return 1
			}

			var i int
			if t != "" {
				i, err = intInput(c.Ui, "Which one?")
				if err != nil {
					return -1
				}
			}

			switch t {
			case "D":

				err = c.DB.Delete(notes[i])
				if err != nil {
					c.Ui.Error("Error deleting the note")
					return -1
				}
			case "E":
				c.Ui.Output(fmt.Sprintf("Current text is: %s", notes[i].Text))
				text, err := c.Ui.Ask("What would you like instead?:")
				if err != nil {
					return -1
				}

				notes[i].Text = text
				notes[i].UpdatedAt = time.Now()
				err = c.DB.Save(notes[i])
				if err != nil {
					c.Ui.Error(fmt.Sprintf("Error saving record: %s", err))
					return -1
				}
			}
		}
	}

	return 0
}

func (c *NoteCommand) Synopsis() string {
	return "note synopsis"
}
