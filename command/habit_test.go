package command

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/models"
	"github.com/elos/models/habit"
	"github.com/mitchellh/cli"
)

// --- Testing Helpers {{{

func newTestHabit(t *testing.T, db data.DB, u *models.User, name string) *models.Habit {
	h, err := models.CreateHabit(db, u, name)
	if err != nil {
		t.Fatalf("newTestHabit Error: %s", err)
	}
	return h
}

func newMockHabitCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *HabitCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &HabitCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- }}}

// --- Tests {{{

// --- Instantiation {{{

func TestHabitBasic(t *testing.T) {
	_, _, _, c := newMockHabitCommand(t)

	c.Help()
	c.Synopsis()

	if out := c.Run([]string{"garbage"}); out != 0 {
		t.Fatal("'elos habit garbage' should return success, even though it is an unrecognized command")
	}
}

func TestPeopleInadequateInitialization(t *testing.T) {
	// mock cli.Ui
	ui := new(cli.MockUi)

	// memory db
	db := mem.NewDB()

	// a new user, stored in db
	user := newTestUser(t, db)

	// note: this HabitCommand is missing a cli.Ui
	missingUI := &HabitCommand{
		UserID: user.ID().String(),
		DB:     db,
	}

	// note: this HabitCommand has no UserID
	missingUserID := &HabitCommand{
		UI: ui,
		DB: db,
	}

	// note: this HabitCommand lacks a database (DB field)
	missingDB := &HabitCommand{
		UI:     ui,
		UserID: user.ID().String(),
	}

	// expect missing a ui to fail
	if o := missingUI.Run([]string{"new"}); o != failure {
		t.Fatal("HabitCommand without ui should fail on run")
	}

	// expect missing a user id to fail
	if o := missingUserID.Run([]string{"new"}); o != failure {
		t.Fatal("HabitCommand without user id should fail on run")
	}

	// expect missing a db to fail
	if o := missingDB.Run([]string{"new"}); o != failure {
		t.Fatal("HabitCommand without db should fail on run")
	}
}

// --- }}}

// --- Integration {{{

// --- `elos habit checkin` {{{
func TestHabitCheckin(t *testing.T) {
	ui, db, user, c := newMockHabitCommand(t)

	t.Log("Creating a new test habit")
	hbt := newTestHabit(t, db, user, "Test Habit")
	t.Log("Created")

	// checkin for the first habit
	// TODO: have this take notes
	ui.InputReader = bytes.NewBufferString("0\nCheckin notes\n")

	t.Log("running: `elos habit checkin`")
	code := c.Run([]string{"checkin"})
	t.Log("command `checkin` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "0)") {
		t.Fatalf("Output should have contained a 0) for listing habits")
	}

	// verify Test Habit was listed
	if !strings.Contains(output, "Test Habit") {
		t.Fatalf("Output should have contained the habit's name in some way")
	}

	t.Log("Reload the habit")
	// verify that the habit was checked off
	if err := db.PopulateByID(hbt); err != nil {
		t.Fatal(err)
	}
	t.Logf("Habit:\n%+v", hbt)

	if checkedIn, err := habit.DidCheckinOn(db, hbt, time.Now()); err != nil {
		t.Fatal("Error while checking if habit is checked off: %s", err)
	} else if !checkedIn {
		t.Fatalf("Habit should be checked off for today now")
	}
}

// --- }}}

// --- `elos habit delete` {{{
func TestHabitDelete(t *testing.T) {
	ui, db, user, c := newMockHabitCommand(t)

	t.Log("Creating a new test habit")
	habit := newTestHabit(t, db, user, "Test Habit")
	t.Log("Created")

	// delete the first habit, confirm
	ui.InputReader = bytes.NewBufferString("0\ny\n")

	t.Log("running: `elos habit delete`")
	code := c.Run([]string{"delete"})
	t.Log("command `delete` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "0)") {
		t.Fatalf("Output should have contained a 0) for listing habits")
	}

	// verify TestHabit was listed
	if !strings.Contains(output, "Test Habit") {
		t.Fatalf("Output should have contained the habit's name in some way")
	}

	// verify that the habit was deleted
	if err := db.PopulateByID(habit); err != data.ErrNotFound {
		t.Fatal("Should not have been able to retrieve habit")
	}
}

// --- }}}

// --- `elos habit history` {{{
func TestHabitHistory(t *testing.T) {
	ui, db, user, c := newMockHabitCommand(t)

	t.Log("Creating a new test habit")
	hbt := newTestHabit(t, db, user, "hello")
	habit.CheckinFor(db, hbt, "first checkin", time.Now().Add(-24*time.Hour))
	habit.CheckinFor(db, hbt, "second checkin", time.Now())
	t.Log("Created")

	// select the first habit
	ui.InputReader = bytes.NewBufferString("0\n")

	t.Log("running: `elos habit history`")
	code := c.Run([]string{"history"})
	t.Log("command `history` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "0)") {
		t.Fatalf("Output should have contained a 0) for listing checkins")
	}

	// verify checkins appeared
	if !strings.Contains(output, "first checkin") {
		t.Fatalf("Output should have contained the text of the first checkin")
	}

	if !strings.Contains(output, "second checkin") {
		t.Fatalf("Output should have contained the text of the second checkin")
	}
}

// --- }}}

// --- `elos habit list` {{{
func TestHabitList(t *testing.T) {
	ui, db, user, c := newMockHabitCommand(t)

	t.Log("Creating test habits")
	habits := make([]*models.Habit, 4)
	for i := range habits {
		habits[i] = newTestHabit(t, db, user, fmt.Sprintf("Habit %d", i+1))
	}
	t.Log("Created")

	t.Log("running: `elos habit list`")
	code := c.Run([]string{"list"})
	t.Log("command `list` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "0)") {
		t.Fatalf("Output should have contained a 0) for listing habits")
	}

	// verify first habit appeared
	if !strings.Contains(output, "Habit 1") {
		t.Fatalf("Output should have 'Habit 1', the name of the first habit")
	}

	if !strings.Contains(output, "Habit 4") {
		t.Fatalf("Output should have 'Habit 4', the name of the fourth habit")
	}
}

// --- }}}

// --- `elos habit new` {{{
func TestHabitNew(t *testing.T) {
	ui, db, _, c := newMockHabitCommand(t)

	habitName := "MyHabit"

	input := strings.Join([]string{
		habitName,
	}, "\n")

	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos habit new`")
	code := c.Run([]string{"new"})
	t.Log("command `new` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "Name") {
		t.Fatalf("Output should have contained 'Name' for inputting the name of the new habit")
	}

	h := models.NewHabit()
	if err := db.PopulateByField("name", habitName, h); err != nil {
		t.Fatalf("Error looking for habit with name: '%s': %s", habitName, err)
	}

	if tag, err := h.Tag(db); err != nil {
		t.Fatalf("Error while looking up tag for a habit: %s", err)
	} else {
		if tag.Name != habitName {
			t.Fatalf("Should also create a tag name with the proper name")
		}
	}
}

// --- }}}

// --- `elos habit today` {{{
func TestHabitToday(t *testing.T) {
	ui, db, user, c := newMockHabitCommand(t)

	t.Log("creating two habits")
	h1 := newTestHabit(t, db, user, "first")
	newTestHabit(t, db, user, "second")
	t.Log("created")
	t.Log("checking one off")
	if _, err := habit.CheckinFor(db, h1, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	t.Log("checked off")

	t.Log("running: `elos habit today`")
	code := c.Run([]string{"today"})
	t.Log("command `today` terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != success {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	if !strings.Contains(output, "first") {
		t.Fatalf("output should contain name of first habit")
	}

	if !strings.Contains(output, "second") {
		t.Fatalf("output should contain name of second habit")
	}

	if !strings.Contains(output, "✓") {
		t.Fatal("Should have found a '✓' in the output")
	}
}

// --- }}}

// --- }}}

// --- }}}
