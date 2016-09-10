package command

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	olddata "github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	oldmodels "github.com/elos/models"
	"github.com/elos/x/data"
	"github.com/elos/x/models"
	"github.com/mitchellh/cli"
)

// --- Testing Helpers (newTestCalendar) {{{

func newTestCalendar(t *testing.T, db olddata.DB, u *oldmodels.User) *oldmodels.Calendar {
	c := oldmodels.NewCalendar()
	c.SetID(db.NewID())
	c.SetOwner(u)
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt

	return c
}

func newMockCalCommand(t *testing.T) (*cli.MockUi, olddata.DB, *oldmodels.User, *CalCommand) {
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

func TestCal(t *testing.T) {
	//TODO(nclandolfi): fix test
	t.Skip()
	NOW := time.Now()
	cases := map[string]struct {
		prior  data.State
		userID string
		args   []string
		input  io.Reader
		code   int
		errput string
		output string
		// IF posterior is nil, prior will be used
		posterior data.State
	}{
		"simple cal week": {
			prior: data.State{
				models.Kind_USER: []*data.Record{
					&data.Record{
						Kind: models.Kind_USER,
						User: &models.User{
							Id: "1",
						},
					},
				},
				models.Kind_CREDENTIAL: []*data.Record{
					&data.Record{
						Kind: models.Kind_CREDENTIAL,
						Credential: &models.Credential{
							Id:      "2",
							OwnerId: "1",
							Type:    models.Credential_PASSWORD,
							Public:  "pu",
							Private: "pr",
						},
					},
				},
				models.Kind_FIXTURE: []*data.Record{
					&data.Record{
						Kind: models.Kind_FIXTURE,
						Fixture: &models.Fixture{
							Id:        "3",
							OwnerId:   "1",
							StartTime: models.TimestampFrom(NOW.Add(1 * time.Hour)),
							EndTime:   models.TimestampFrom(NOW.Add(2 * time.Hour)),
						},
					},
				},
			},
			input:  new(bytes.Buffer),
			userID: "1",
			args:   []string{"week"},
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			db := mem.NewDB()
			dbc, conn, err := data.DBBothLocal(db)
			if err != nil {
				t.Fatalf("data.DBBothLocal error: %v", err)
			}
			defer conn.Close()
			if err := data.Seed(context.Background(), dbc, c.prior); err != nil {
				t.Fatalf("data.Seed error: %v", err)
			}

			if c.input == nil {
				t.Fatal("c.input must be non-nil")
			}
			ui := &cli.MockUi{
				InputReader: c.input,
			}
			cmd := &CalCommand{
				UI:     ui,
				UserID: c.userID,
				DB:     data.DB(dbc),
			}

			if got, want := cmd.Run(c.args), c.code; got != want {
				t.Log(ui.ErrorWriter.String())
				t.Fatalf("cmd.Run(%v): got %d, want %d", c.args, got, want)
			}

			if got, want := ui.ErrorWriter.String(), c.errput; got != want {
				t.Fatalf("ui.ErrorWriter.String(): got %q, want %q", got, want)
			}

			if got, want := ui.OutputWriter.String(), c.output; got != want {
				t.Fatalf("ui.OutputWriter.String(): got %q, want %q", got, want)
			}

			finalState := c.prior
			if c.posterior != nil {
				finalState = c.posterior
			}

			if got, want := data.CompareState(context.Background(), dbc, finalState), error(nil); got != want {
				t.Fatalf("data.CompareState: got %v, want %v", got, want)
			}
		})
	}
}
