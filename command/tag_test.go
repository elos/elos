package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// --- Testing Helper (newTestTag, newMockTagCommand) {{{

func newTestTag(t *testing.T, db data.DB, u *models.User) *models.Tag {
	tg := models.NewTag()
	tg.SetID(db.NewID())
	tg.SetOwner(u)

	if err := db.Save(tg); err != nil {
		t.Fatal(err)
	}

	return tg
}

func newMockTagCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *TagCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &TagCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- }}}

// --- Tests {{{

// --- Instantiation {{{

func TestTagBasic(t *testing.T) {
	_, _, _, c := newMockTagCommand(t)

	c.Help()
	c.Synopsis()

	if out := c.Run([]string{"garbage"}); out != 0 {
		t.Fatalf("TestTagBasic should return success on unrecognized command")
	}
}

func TestTagInadequateInitialization(t *testing.T) {
	// mock cli.Ui
	ui := new(cli.MockUi)

	// memory db
	db := mem.NewDB()

	// a new user, stored in db
	user := newTestUser(t, db)

	// note: this TagCommand is missing a cli.Ui
	missingUI := &TagCommand{
		UserID: user.ID().String(),
		DB:     db,
	}

	// note: this TagCommand has no UserID
	missingUserID := &TagCommand{
		UI: ui,
		DB: db,
	}

	// note: this TagCommand lacks a database (DB field)
	missingDB := &TagCommand{
		UI:     ui,
		UserID: user.ID().String(),
	}

	// expect missing a ui to fail
	if o := missingUI.Run([]string{"new"}); o != failure {
		t.Fatal("TagCommand without ui should fail on run")
	}

	// expect missing a user id to fail
	if o := missingUserID.Run([]string{"new"}); o != failure {
		t.Fatal("TagCommand without user id should fail on run")
	}

	// expect missing a db to fail
	if o := missingDB.Run([]string{"new"}); o != failure {
		t.Fatal("TagCommand without db should fail on run")
	}
}

// --- }}}

// --- Integration {{{

// --- `elos tag delete` {{{

// TestTagDelete test the `delete` subcommand
func TestTagDelete(t *testing.T) {
	ui, db, user, c := newMockTagCommand(t)

	// load a tag into the db
	tag := newTestTag(t, db, user)

	// load the input
	// first tag, and then confirm
	ui.InputReader = bytes.NewBuffer([]byte("0\ny\n"))

	// run `elos tag delete`
	code := c.Run([]string{"delete"})

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n %s", errput)
	t.Logf("Output:\n %s", output)

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
		t.Fatalf("Output should have contained a 0) for listing tags")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a tag number")
	}

	t.Log("Trying to load the tag from the database")
	err := db.PopulateByID(tag)
	if err != data.ErrNotFound {
		t.Fatal("Expected the tag to be not found, as it should have been deleted")
	}
}

// --- }}}

// --- `elos tag list` {{{

// TestTagList test the `list` subcommand
func TestTagList(t *testing.T) {
	ui, db, user, c := newMockTagCommand(t)

	tag1 := newTestTag(t, db, user)
	tag2 := newTestTag(t, db, user)
	tag1.Name = "tag1"
	if err := db.Save(tag1); err != nil {
		t.Fatal(err)
	}
	tag2.Name = "tag2"
	if err := db.Save(tag2); err != nil {
		t.Fatal(err)
	}

	t.Log("running: `elos tag list`")
	code := c.Run([]string{"list"})
	t.Log("command 'tag' terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n %s", errput)
	t.Logf("Output:\n %s", output)

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
		t.Fatalf("Output should have contained a 0) for listing tag1")
	}

	if !strings.Contains(output, "1)") {
		t.Fatalf("Output should have contained a 1) for listing tag2")
	}

	if !strings.Contains(output, "tag1") {
		t.Fatalf("Output should have contained 'tag1' the name of the first tag")
	}

	if !strings.Contains(output, "tag2") {
		t.Fatalf("Output should have contained 'tag2' the name of the second tag")
	}
}

// --- }}}

// --- `elos tag new` {{{

// TestTagNew test the `new` subcommand
func TestTagNew(t *testing.T) {
	ui, db, _, c := newMockTagCommand(t)

	tagName := "asdkfjlasdjfasfdA"

	// just input the name
	ui.InputReader = bytes.NewBufferString(tagName + "\n")

	t.Log("running: `elos tag new`")
	code := c.Run([]string{"new"})
	t.Log("command 'new' terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n %s", errput)
	t.Logf("Output:\n %s", output)

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
		t.Fatalf("Output should have contained a 'Name' for asking for the tag's name")
	}

	// check that it was created
	tg := models.NewTag()
	if err := db.PopulateByField("name", tagName, tg); err != nil {
		t.Fatal(err)
	}
}

// --- }}}

// --- `elos tag edit` {{{

// TestTagEdit test the `edit` subcommand
func TestTagEdit(t *testing.T) {
	ui, db, u, c := newMockTagCommand(t)

	tg := newTestTag(t, db, u)
	tagName := "asdkfjlasdjfasfdA"
	tg.Name = tagName
	if err := db.Save(tg); err != nil {
		t.Fatal(err)
	}

	newTagName := "not_the_other_tag_name"

	input := strings.Join([]string{
		"0",
		"name",
		newTagName,
	}, "\n")

	// just input the name
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos tag edit`")
	code := c.Run([]string{"edit"})
	t.Log("command 'edit' terminated")

	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n %s", errput)
	t.Logf("Output:\n %s", output)

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
		t.Fatalf("Output should have contained a 'Name' for asking for the tag's name")
	}

	if err := db.PopulateByID(tg); err != nil {
		t.Fatal(err)
	}

	t.Logf("Tag:\n%+v", tg)

	if tg.Name != newTagName {
		t.Fatalf("Expected tag's name to become '%s', but was '%s'", newTagName, tg.Name)
	}
}

// --- }}}

// --- }}}

// --- }}}
