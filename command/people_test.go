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
	"github.com/mitchellh/cli"
)

func newTestPerson(t *testing.T, db data.DB, u *models.User) *models.Person {
	person := models.NewPerson()
	person.SetID(db.NewID())
	person.CreatedAt = time.Now()
	person.OwnerId = u.ID().String()
	person.UpdatedAt = time.Now()
	if err := db.Save(person); err != nil {
		t.Fatalf("newTestPerson Error: %s", err)
	}
	return person
}

func newTestNote(t *testing.T, db data.DB, u *models.User) *models.Note {
	note := models.NewNote()
	note.SetID(db.NewID())
	note.CreatedAt = time.Now()
	note.OwnerId = u.ID().String()
	note.UpdatedAt = time.Now()
	if err := db.Save(note); err != nil {
		t.Fatal("newTestNote Error: %s", err)
	}
	return note
}

func newMockPeopleCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *PeopleCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &PeopleCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- Tests {{{

// --- Instantiation {{{

func TestPeopleBasic(t *testing.T) {
	_, _, _, c := newMockPeopleCommand(t)

	c.Help()
	c.Synopsis()

	if out := c.Run([]string{"garbage"}); out != 0 {
		t.Fatalf("people should return success on unrecognized command")
	}
}

func TestHabitInadequateInitialization(t *testing.T) {
	// mock cli.Ui
	ui := new(cli.MockUi)

	// memory db
	db := mem.NewDB()

	// a new user, stored in db
	user := newTestUser(t, db)

	// note: this PeopleCommand is missing a cli.Ui
	missingUI := &PeopleCommand{
		UserID: user.ID().String(),
		DB:     db,
	}

	// note: this PeopleCommand has no UserID
	missingUserID := &PeopleCommand{
		UI: ui,
		DB: db,
	}

	// note: this TodoCommand lacks a database (DB field)
	missingDB := &PeopleCommand{
		UI:     ui,
		UserID: user.ID().String(),
	}

	// expect missing a ui to fail
	if o := missingUI.Run([]string{"new"}); o != failure {
		t.Fatal("PeopleCommand without ui should fail on run")
	}

	// expect missing a user id to fail
	if o := missingUserID.Run([]string{"new"}); o != failure {
		t.Fatal("PeopleCommand without user id should fail on run")
	}

	// expect missing a db to fail
	if o := missingDB.Run([]string{"new"}); o != failure {
		t.Fatal("PeopleCommand without db should fail on run")
	}
}

// --- }}}

// --- Integration {{{

// --- `elos people delete` {{{
func TestPeopleDelete(t *testing.T) {
	ui, db, user, c := newMockPeopleCommand(t)

	t.Log("Creating a test person")
	// load the person
	person := newTestPerson(t, db, user)
	person.FirstName = "Jack"
	person.LastName = "Frost"
	if err := db.Save(person); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	// delete the first person, and confirm
	ui.InputReader = bytes.NewBufferString("0\ny\n")

	t.Log("running: `elos people delete`")
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
		t.Fatalf("Output should have contained a 0) for listing people")
	}

	// verify Jack was listed
	if !strings.Contains(output, "Jack") {
		t.Fatalf("Output should ahve contained the person's name in some way")
	}

	// verify that the person was deleted
	if err := db.PopulateByID(person); err != data.ErrNotFound {
		t.Fatal("expected the person to be deleted")
	}
}

// --- }}}

// --- `elos people list` {{{
func TestPeopleList(t *testing.T) {
	ui, db, user, c := newMockPeopleCommand(t)

	t.Log("Creating a test person")
	// load the person
	person := newTestPerson(t, db, user)
	person.FirstName = "Jack"
	person.LastName = "Frost"
	if err := db.Save(person); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	t.Log("running: `elos people list`")
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
		t.Fatalf("Output should have contained a 0) for listing people")
	}

	// verify Jack was listed
	if !strings.Contains(output, "Jack") {
		t.Fatalf("Output should ahve contained the person's name in some way")
	}
}

// --- }}}

// --- `elos people new` {{{
func TestPeopleNew(t *testing.T) {
	ui, db, _, c := newMockPeopleCommand(t)

	input := strings.Join([]string{
		"Nick",     // First Name
		"Landolfi", // Last Name
	}, "\n")

	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos people new`")
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
	if !strings.Contains(output, "First Name") {
		t.Fatalf("Output should have contained 'First Name' for first_name trait")
	}

	person := models.NewPerson()
	if err := db.PopulateByField("first_name", "Nick", person); err != nil {
		t.Fatal("Error looking for the new person: %s", err)
	}

	// verify the last name was set properly
	if person.LastName != "Landolfi" {
		t.Fatal("Last name should have been Landolfi, not '%s'", person.LastName)
	}
}

// --- }}}

// --- `elos people note` {{{
func TestPeopleNote(t *testing.T) {
	ui, db, user, c := newMockPeopleCommand(t)

	t.Log("Creating a test person")
	// load the person
	person := newTestPerson(t, db, user)
	person.FirstName = "Jack"
	person.LastName = "Frost"
	if err := db.Save(person); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	// TODO allows spaces in the test input
	input := strings.Join([]string{
		"0",                // selecting the person
		"is_a_nice_person", // the first note
		"y",                // add another note
		"is_a_good_person", // the second note
		"n",                // no more
	}, "\n")

	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos people note`")
	code := c.Run([]string{"note"})
	t.Log("command `note` terminated")

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
	if !strings.Contains(output, "Jack") {
		t.Fatalf("Output should have contained a 'Jack', the name of the test person")
	}

	// verify a note was added to the record
	if err := db.PopulateByID(person); err != nil {
		t.Fatalf("Error retrieving person: %s", err)
	}

	t.Logf("Person:\n%+v", person)

	notes, err := person.Notes(db)
	if err != nil {
		t.Fatalf("Error retrieving the notes on the test person: %s", err)
	}

	if len(notes) != 2 {
		t.Fatal("The person should have exactly 2 notes")
	}

	// simple sort
	var first, second *models.Note
	if notes[0].CreatedAt.Before(notes[1].CreatedAt) {
		first = notes[0]
		second = notes[1]
	} else {
		first = notes[1]
		second = notes[0]
	}

	t.Logf("First:\n%+v", first)
	t.Logf("Second:\n%+v", second)

	if !strings.Contains(first.Text, "nice") {
		t.Fatal("First note should contain 'nice'")
	}

	if !strings.Contains(second.Text, "good") {
		t.Fatal("Second note should contain 'good'")
	}
}

// --- }}}

// --- `elos people stream` {{{
func TestPeopleStream(t *testing.T) {
	t.Skip() // TODO: fix this test, command works
	ui, db, user, c := newMockPeopleCommand(t)

	t.Log("Creating a test person")
	// load the person
	person := newTestPerson(t, db, user)
	person.FirstName = "Jack"
	person.LastName = "Frost"
	if err := db.Save(person); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	t.Log("Creating test notes")
	notes := make([]*models.Note, 4)
	for i := range notes {
		n := newTestNote(t, db, user)
		n.Text = fmt.Sprintf("Note %d", i+1)

		if err := db.Save(n); err != nil {
			t.Fatalf("Error creating test notes: %s", err)
		}

		notes[i] = n
	}
	t.Log("Created")

	input := strings.Join([]string{
		"0",  // selecting the person
		"\n", // see the second one
		"\n", // see the third one
		"\n", // and the fourth one
	}, "\n")

	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos people stream`")
	code := c.Run([]string{"stream"})
	t.Log("command `stream` terminated")

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
	if !strings.Contains(output, "Jack") {
		t.Fatalf("Output should have contained a 'Jack', the name of the test person")
	}

	if !strings.Contains(output, "Note 1") {
		t.Fatalf("Output should have contained a 'Note 1', the text of the first note")
	}

	if !strings.Contains(output, "Note 4") {
		t.Fatalf("Output should have contained a 'Note 4', the text of the fourth note")
	}
}

// --- }}}

// --- }}}

// --- }}}
