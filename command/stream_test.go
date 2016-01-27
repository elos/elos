package command

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

func newMockStreamCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *StreamCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &StreamCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- `elos stream` {{{

// TestStream test the `stream" command
func TestStream(t *testing.T) {
	t.Skip()
	ui, db, user, c := newMockStreamCommand(t)

	// in another go routine start streaming
	go c.Run([]string{})

	// now give it an event

	changes := db.Changes()

	e := models.NewEvent()
	e.SetID(db.NewID())
	e.SetOwner(user)
	eventName := "event name"
	e.Name = eventName
	if err := db.Save(e); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	// wait for that change to go through the pipeline
	select {
	case <-*changes:
		t.Log("change recieved")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for change")
	}

	// now check outputs
	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	log.Print(errput)
	log.Print(output)
	t.Logf("Error output:\n %s", errput)
	t.Logf("Output:\n %s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify some of the output
	if !strings.Contains(output, eventName) {
		t.Fatalf("Output should have the event's name: '%s'", eventName)
	}
}

// --- }}}
