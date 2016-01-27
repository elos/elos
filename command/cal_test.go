package command

import (
	"bytes"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// --- Testing Helpers (newTestCalendar) {{{

func newTestCalendar(t *testing.T, db data.DB, u *models.User) *models.Calendar {
	c := models.NewCalendar()
	c.SetID(db.NewID())
	c.SetOwner(u)
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt

	return c
}

func newMockCalCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *CalCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &CalCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- }}}

// --- Tests {{{

// --- Instantiaion {{{

func TestCalBasic(t *testing.T) {
	ui, _, _, c := newMockCalCommand(t)
	// yes to init prompt to create a new calendar
	ui.InputReader = bytes.NewBufferString("y\n")

	c.Help()
	c.Synopsis()

	if out := c.Run([]string{"garbage"}); out != 0 {
		t.Fatalf("TestCalBasic should return success on unrecognized command")
	}
}

func TestCalInadequateInitialization(t *testing.T) {
	// mock cli.Ui
	ui := new(cli.MockUi)
	// yes to init prompt to create a new calendar, 3 times
	ui.InputReader = bytes.NewBufferString("y\ny\ny\n")

	// memory db
	db := mem.NewDB()

	// a new user, stored in db
	user := newTestUser(t, db)

	// note: this CalCommand is missing a cli.Ui
	missingUI := &CalCommand{
		UserID: user.ID().String(),
		DB:     db,
	}

	// note: this CalCommand has no UserID
	missingUserID := &CalCommand{
		UI: ui,
		DB: db,
	}

	// note: this CalCommand lacks a database (DB field)
	missingDB := &CalCommand{
		UI:     ui,
		UserID: user.ID().String(),
	}

	t.Log("Run command that doesn't have a UI")

	// expect missing a ui to fail
	if o := missingUI.Run([]string{"new"}); o != failure {
		t.Fatal("CalCommand without ui should fail on run")
	}

	t.Log("Completed")

	t.Log("Run command that doesn't have a user id")

	// expect missing a user id to fail
	if o := missingUserID.Run([]string{"new"}); o != failure {
		t.Fatal("CalCommand without user id should fail on run")
	}

	t.Log("Completed")

	t.Log("Run command that doesn't have a db")

	// expect missing a db to fail
	if o := missingDB.Run([]string{"new"}); o != failure {
		t.Fatal("CalCommand without db should fail on run")
	}

	t.Log("Completed")
}

// --- }}}

// --- }}}
