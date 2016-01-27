package command

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// --- Testing Helpers (newTestUser, newTestTask, newMockTodoCommand) {{{

func newTestUser(t *testing.T, db data.DB) *models.User {
	u := models.NewUser()
	u.SetID(db.NewID())
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	if err := db.Save(u); err != nil {
		t.Fatalf("Error newTestUser: %s", err)
	}
	return u
}

func newTestTask(t *testing.T, db data.DB, u *models.User) *models.Task {
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = u.ID().String()
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatalf("Error newTestTask: %s", err)
	}
	return task
}

func newMockTodoCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *TodoCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUser(t, db)

	return ui, db, user, &TodoCommand{
		UI:     ui,
		UserID: user.ID().String(),
		DB:     db,
	}
}

// --- }}}

// --- Tests {{{

// --- Instantiation {{{

func TestTodoBasic(t *testing.T) {
	_, _, _, c := newMockTodoCommand(t)

	c.Help()
	c.Synopsis()

	if out := c.Run([]string{"garbage"}); out != 0 {
		t.Fatalf("TestTodoBasic should return success on unrecognized command")
	}
}

func TestTodoInadequateInitialization(t *testing.T) {
	// mock cli.Ui
	ui := new(cli.MockUi)

	// memory db
	db := mem.NewDB()

	// a new user, stored in db
	user := newTestUser(t, db)

	// note: this TodoCommand is missing a cli.Ui
	missingUI := &TodoCommand{
		UserID: user.ID().String(),
		DB:     db,
	}

	// note: this TodoCommand has no UserID
	missingUserID := &TodoCommand{
		UI: ui,
		DB: db,
	}

	// note: this TodoCommand lacks a database (DB field)
	missingDB := &TodoCommand{
		UI:     ui,
		UserID: user.ID().String(),
	}

	// expect missing a ui to fail
	if o := missingUI.Run([]string{"new"}); o != failure {
		t.Fatal("TodoCommand without ui should fail on run")
	}

	// expect missing a user id to fail
	if o := missingUserID.Run([]string{"new"}); o != failure {
		t.Fatal("TodoCommand without user id should fail on run")
	}

	// expect missing a db to fail
	if o := missingDB.Run([]string{"new"}); o != failure {
		t.Fatal("TodoCommand without db should fail on run")
	}
}

// --- }}}

// --- Integration {{{

// --- `elos todo complete` {{{

// TestTodoComplete tests the `complete` subcommand
func TestTodoComplete(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// setup that there is one task
	task := newTestTask(t, db, user)

	// load the input
	ui.InputReader = bytes.NewBuffer([]byte("0\n"))

	// run the effect of `elos todo complete`
	code := c.Run([]string{"complete"})

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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Checking that the task was completed")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)

	if task.Complete != true {
		t.Fatalf("Expected the task to be complete")
	}
}

// --- }}}

// --- `elos todo current` {{{

// TestTodoCurrent tests the `current` subcommand
func TestTodoCurrent(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// setup that there is one task
	task := newTestTask(t, db, user)
	taskName := "task name"
	task.Name = taskName
	task.Start()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	// run the effect of `elos todo complete`
	code := c.Run([]string{"current"})

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
	if !strings.Contains(output, taskName) {
		t.Fatalf("Output should have contained the running task's name: '%s'", taskName)
	}
}

// --- }}}

// --- `elos todo delete` {{{

// TestTodoDelete test the `delete` subcommand
func TestTodoDelete(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)

	// load the input
	ui.InputReader = bytes.NewBuffer([]byte("0\n"))

	// run `elos todo delete`
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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Trying to load the task from the database")
	err := db.PopulateByID(task)
	if err != data.ErrNotFound {
		t.Fatal("Expected the task to be not found, as it should have been deleted")
	}
}

// --- }}}

// --- `elos todo edit` {{{

// TestTodoEdit tests the `edit` subcommand
func TestTodoEdit(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	// load input
	input := strings.Join([]string{
		"0",
		"name",
		"newname",
	}, "\n")
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos todo edit`")
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
	if !strings.Contains(output, "0)") {
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Checking that the task's name was changed")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)

	if task.Name != "newname" {
		t.Fatalf("Expected the task's name to have changed to 'newname'")
	}
}

// --- }}}

// --- `elos todo goal` {{{

// TestTodoGoal tests the `goal` subcommand
func TestTodoGoal(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	// load input
	input := strings.Join([]string{
		"0",
	}, "\n")
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos todo goal`")
	code := c.Run([]string{"goal"})
	t.Log("command 'goal' terminated")

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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Checking that the task is now a member of goals")

	// reload task:
	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Task:\n%+v", task)

	// load tag
	tag, err := models.TagByName(db, models.GoalTagName, user)
	if err != nil {
		log.Fatal(err)
	}

	t.Logf("GOALS tag:\n%+v", tag)

	tasks, err := tag.Tasks(db)
	if err != nil {
		log.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected goals tag to contain just one task, contained: %d", len(tasks))
	}

	if tasks[0].Id != task.Id {
		t.Fatal("Expected task to now be a part of goals")
	}
}

// --- }}}

// --- `elos todo goals` {{{

// TestTodoGoals tests the `goals` subcommand
func TestTodoGoals(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	tag, err := models.TagByName(db, models.GoalTagName, user)
	if err != nil {
		t.Fatal(err)
	}

	task.IncludeTag(tag)

	if err := db.Save(task); err != nil {
		log.Fatal(err)
	}

	t.Log("running: `elos todo goals`")
	code := c.Run([]string{"goals"})
	t.Log("command 'goals' terminated")

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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, task.Name) {
		t.Fatalf("Output should have contained task name")
	}

}

// --- }}}

// --- `elos todo list` {{{

// TestTodoList test the `list` subcommand
func TestTodoList(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	task1 := newTestTask(t, db, user)
	task2 := newTestTask(t, db, user)
	task1.Name = "task1"
	if err := db.Save(task1); err != nil {
		t.Fatal(err)
	}
	task2.Name = "task2"
	if err := db.Save(task2); err != nil {
		t.Fatal(err)
	}

	t.Log("running: `elos todo list`")
	code := c.Run([]string{"list"})
	t.Log("command 'start' terminated")

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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "1)") {
		t.Fatalf("Output should have contained a 1) for listing tasks")
	}

	if !strings.Contains(output, "task1") {
		t.Fatalf("Output should have contained 'task1' the name of the first task")
	}

	if !strings.Contains(output, "task2") {
		t.Fatalf("Output should have contained 'task2' the name of the second task")
	}
}

// --- }}}

// --- `elos todo new` {{{

// TestTodoNew tests the `new` subcommand
func TestTodoNew(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// set up tasks
	task1 := newTestTask(t, db, user)
	task1.Name = "task1"
	task2 := newTestTask(t, db, user)
	task2.Name = "task2"
	if err := db.Save(task1); err != nil {
		t.Fatal(err)
	}
	if err := db.Save(task2); err != nil {
		t.Fatal(err)
	}

	// Here we create 3 tasks, the first has a deadline and
	// is called "top". "top" has "task1" and "task2" as
	// prereqs. Then we create another prereq: 'sub'. 'sub'
	// has no deadline, but has one prereq, another task
	// we create, called 'bottom'
	input := strings.Join([]string{
		"top",    // Name
		"y",      // Does it have a deadline
		"n",      // use current time?
		"2020",   // year
		"1",      // month
		"1",      // date
		"12",     // hour
		"0",      // minute
		"y",      // prereqs?
		"y",      // current tasks?
		"0",      // index of prereq
		"y",      // any more current prereqs?
		"0",      // shouldn't error, even though already added
		"y",      // any more current prereqs?
		"1",      // now add `task2`
		"n",      // any more current prereqs?
		"y",      // any dependencies that are new?
		"sub",    // name
		"n",      // deadline
		"y",      // prereqs?
		"n",      // current?
		"y",      // new?
		"bottom", // name
		"n",      //deadline?
		"n",      //prereqs? => Task Created
		"n",      // any more new prereqs? => TaskCreated
		"n",      // any more new prereqs? => TaskCreated
	}, "\n")

	// load input
	ui.InputReader = bytes.NewBufferString(input)

	// run command
	t.Log("running: `elos todo new`")
	code := c.Run([]string{"new"})
	t.Log("command 'new' terminated")

	// basic checks
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

	// a small sample of keywords in output
	if !strings.Contains(output, "Task created") {
		t.Fatalf("Output should contain 'Task Created'")
	}
	if !strings.Contains(output, "deadline") {
		t.Fatalf("Output should have contained 'deadline'")
	}

	// now to verify that the tasks were created
	// first top
	top := models.NewTask()
	if err := db.PopulateByField("name", "top", top); err != nil {
		t.Fatal(err)
	}
	t.Log("'top'")
	t.Logf("%+v", top)

	if len(top.PrerequisitesIds) != 3 {
		t.Fatal("Expected 'top' to have 3 prereqs")
	}

	if top.Deadline.Year() != 2020 {
		t.Fatal("Expected 'top' to have a deadline in 2020")
	}

	// then sub
	sub := models.NewTask()
	if err := db.PopulateByField("name", "sub", sub); err != nil {
		t.Fatal(err)
	}

	t.Log("'sub'")
	t.Logf("%+v", sub)

	if len(sub.PrerequisitesIds) != 1 {
		t.Fatal("Expected 'top' to have 3 prereqs")
	}

	prereqs, err := sub.Prerequisites(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(prereqs) != 1 {
		t.Fatal("The length of the prereqs should be 1")
	}

	if prereqs[0].Name != "bottom" {
		t.Fatal("the prerequiste of 'sub' should have been 'bottom'")
	}
}

// --- }}}

// ---	`elos todo start' & `elos todo stop` {{{

// TestTodoStartStop tests the `start` and `stop` subcommands
func TestTodoStartStop(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)

	// load the input
	ui.InputReader = bytes.NewBuffer([]byte("0\n"))

	t.Log("running: `elos todo start`")
	code := c.Run([]string{"start"})
	t.Log("command 'start' terminated")

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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Checking that the task was started")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)

	if !task.InProgress() {
		t.Fatalf("Expected the task to in progress")
	}

	// get a fresh ui
	ui = new(cli.MockUi)
	c.UI = ui

	// load the input
	ui.InputReader = bytes.NewBuffer([]byte("0\n"))

	t.Log("running: `elos todo stop`")
	code = c.Run([]string{"stop"})
	t.Log("command run terminated")

	errput = ui.ErrorWriter.String()
	output = ui.OutputWriter.String()
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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	t.Log("Checking that the task was stopped")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)

	if task.InProgress() {
		t.Fatalf("Expected the task to _not_ in progress")
	}
}

// --- }}}

// --- `elos todo suggest` {{{

// TestTodoSuggest tests the `suggest` subcommand
func TestTodoSuggest(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "SUGGESTED"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	t.Log("running: `elos todo suggest`")
	code := c.Run([]string{"suggest"})
	t.Log("command 'start' terminated")

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

	// verify output
	if !strings.Contains(output, "SUGGESTED") {
		t.Fatal("Expected output to containe 'SUGGESTED', the name of the only task")
	}
}

// --- }}}

// --- `elos todo tag` {{{
func TestTodoTag(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	tagName := "tag name"
	tag, err := models.TagByName(db, tagName, user)
	if err != nil {
		t.Fatal(err)
	}

	// load input
	input := strings.Join([]string{
		"0", // selecting the task to tag
		"0", // selecting the tag
	}, "\n")
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos todo tag`")
	code := c.Run([]string{"tag"})
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
		t.Fatalf("Output should have contained a 0) for listing tasks")
	}

	if !strings.Contains(output, "Which number?") {
		t.Fatalf("Output should have asked for a task number")
	}

	if !strings.Contains(output, tagName) {
		t.Fatalf("Output should have included the tag's name")
	}

	t.Log("Checking that the task now includes the tag")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)
	t.Logf("Here's the tag:\n%+v", tag)

	if len(task.TagsIds) != 1 {
		t.Fatal("Expected the task to have one tag")
	}

	if task.TagsIds[0] != tag.Id {
		t.Fatal("Expected the task to have the tag")
	}
}

// --- }}}

// --- }}}

// --- Internals {{{
// --- }}}

// --- }}}
