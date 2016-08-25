package command

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	oldmodels "github.com/elos/models"
	"github.com/elos/x/models"
	"github.com/elos/x/models/tag"
	"github.com/elos/x/models/task"
	"github.com/mitchellh/cli"
)

// --- Testing Helpers (newTestUser, newTestUserX, newTestTask, newMockTodoCommand) {{{

func newTestUser(t *testing.T, db data.DB) *oldmodels.User {
	u := oldmodels.NewUser()
	u.SetID(db.NewID())
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	if err := db.Save(u); err != nil {
		t.Fatalf("Error newTestUser: %s", err)
	}
	return u
}

func newTestUserX(t *testing.T, db data.DB) *models.User {
	u := new(models.User)
	u.SetID(db.NewID())
	if err := db.Save(u); err != nil {
		t.Fatalf("Error newTestUserX: %s", err)
	}
	return u
}

func newTestTask(t *testing.T, db data.DB, u *models.User) *models.Task {
	tsk := new(models.Task)
	tsk.SetID(db.NewID())
	tsk.CreatedAt = models.TimestampFrom(time.Now())
	tsk.OwnerId = u.ID().String()
	tsk.UpdatedAt = models.TimestampFrom(time.Now())
	if err := db.Save(tsk); err != nil {
		t.Fatalf("Error newTestTask: %s", err)
	}
	return tsk
}

func newMockTodoCommand(t *testing.T) (*cli.MockUi, data.DB, *models.User, *TodoCommand) {
	ui := new(cli.MockUi)
	db := mem.NewDB()
	user := newTestUserX(t, db)

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
	user := newTestUserX(t, db)

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
	tsk := newTestTask(t, db, user)

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

	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", tsk)

	if !task.IsComplete(tsk) {
		t.Fatalf("Expected the task to be complete")
	}
}

// --- }}}

// --- `elos todo current` {{{

// TestTodoCurrent tests the `current` subcommand
func TestTodoCurrent(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// setup that there is one task
	tsk := newTestTask(t, db, user)
	taskName := "task name"
	tsk.Name = taskName
	task.Start(tsk)
	if err := db.Save(tsk); err != nil {
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

// --- `elos todo fix` {{{

// TestTodoFix tests the `fix` subcommand
func TestTodoFix(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	task.DeadlineAt = models.TimestampFrom(time.Now().Add(-36 * time.Hour))
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	// load input
	input := strings.Join([]string{
		"2020", // year
		"1",    // month
		"1",    // day
		"12",   // hour
		"0",    // minute
	}, "\n")
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos todo fix`")
	code := c.Run([]string{"fix"})
	t.Log("command 'fix' terminated")

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
	if !strings.Contains(output, task.Name) {
		t.Fatalf("Output should have contained a the out of date task's name")
	}

	if !strings.Contains(output, "New Deadline") {
		t.Fatalf("Output should have asked for a new deadline")
	}

	t.Log("Checking that the task's deadline was changed")

	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", task)

	if !task.DeadlineAt.Time().After(time.Now()) {
		t.Fatalf("Expected the task's deadline to be after now")
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

	tg := "GOAL"

	t.Logf("GOALS tag:\n%+v", tg)

	tasks, err := tag.TasksFor(db, c.UserID, tg)
	if err != nil {
		t.Fatal(err)
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

	tg := "GOAL"

	for _, t := range task.Tags {
		if t == tg {
			goto dontadd
		}
	}
	task.Tags = append(task.Tags, tg)
dontadd:

	if err := db.Save(task); err != nil {
		t.Fatal(err)
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
	t.Log("command 'list' terminated")

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

// --- `elos todo list -t` {{{

// TestTodoListTag test the `list -t` subcommand
func TestTodoListTag(t *testing.T) {
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

	tagName := "TAGNAME"
	tag.Task(task1, tagName)
	ui.InputReader = bytes.NewBufferString("TAGNAME\n") // select first and only tag

	t.Log("running: `elos todo list -t`")
	code := c.Run([]string{"list", "-t"})
	t.Log("command 'list -t' terminated")

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

	if !strings.Contains(output, "task1") {
		t.Fatalf("Output should have contained 'task1' the name of the first task")
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
	top := new(models.Task)
	if err := db.PopulateByField("name", "top", top); err != nil {
		t.Fatal(err)
	}
	t.Logf("top: %+v", top)

	if len(top.PrerequisiteIds) != 3 {
		t.Fatal("Expected 'top' to have 3 prereqs")
	}

	if top.DeadlineAt.Time().Year() != 2020 {
		t.Fatal("Expected 'top' to have a deadline in 2020")
	}

	// then sub
	sub := new(models.Task)
	if err := db.PopulateByField("name", "sub", sub); err != nil {
		t.Fatal(err)
	}

	t.Log("'sub'")
	t.Logf("%+v", sub)

	if len(sub.PrerequisiteIds) != 1 {
		t.Fatal("Expected 'top' to have 3 prereqs")
	}

	prereqs := make([]*models.Task, len(sub.PrerequisiteIds))
	for i, id := range sub.PrerequisiteIds {
		tsk := &models.Task{Id: id}
		if err := db.PopulateByID(tsk); err != nil {
			t.Fatalf("db.PopulateByID error: %v", err)
		}
		prereqs[i] = tsk
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
	tsk := newTestTask(t, db, user)

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

	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", tsk)

	if !task.InProgress(tsk) {
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

	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", tsk)

	if task.InProgress(tsk) {
		t.Fatalf("Expected the task to _not_ in progress")
	}
}

// --- }}}

// --- `elos todo suggest` {{{

// TestTodoSuggest tests the `suggest` subcommand
func TestTodoSuggest(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	tsk := newTestTask(t, db, user)
	tsk.Name = "SUGGESTED"
	if err := db.Save(tsk); err != nil {
		t.Fatal(err)
	}

	tagName := "random tag"
	tag.Task(tsk, tagName)
	ui.InputReader = bytes.NewBufferString("y\n") // yes, start the task

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

	if !strings.Contains(output, tagName) {
		t.Fatal("Expected output to contain the task's tag's name")
	}

	if !strings.Contains(strings.ToLower(output), "start") {
		t.Fatal("Should ask if we want to start the task")
	}

	t.Log("Reloading the task")
	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}
	t.Logf("Task loaded:\n%+v", tsk)

	if !task.InProgress(tsk) {
		t.Fatal("The task should be in progress now, cause we indicated we wanted to start it")
	}
}

// --- }}}

// --- `elos todo tag` {{{

// TestTodoTag tests the `elos todo tag` subcommand
func TestTodoTag(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	tg := "tagname"

	// load input
	input := strings.Join([]string{
		"0", // selecting the task to tag
		tg,  // specifying the tag
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

	t.Log("Checking that the task now includes the tag")

	tsk := &models.Task{
		Id: task.Id,
	}
	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", tsk)
	t.Logf("Here's the tag:\n%+v", tg)

	if len(tsk.Tags) != 1 {
		t.Fatal("Expected the task to have one tag")
	}

	if tsk.Tags[0] != tg {
		t.Fatal("Expected the task to have the tag")
	}
}

// TestTodoTag tests the `elos todo tag -r` subcommand with the
// "r" flag
func TestTodoTagRemove(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	task := newTestTask(t, db, user)
	task.Name = "Take out the trash"
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	tg := "tag name"

	// now it's tagged
	tag.Task(task, tg)

	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	// load input
	input := strings.Join([]string{
		"0", // selecting the task to remove the tag from
		"0", // selecting the tag
	}, "\n")
	ui.InputReader = bytes.NewBufferString(input)

	t.Log("running: `elos todo tag -r`")
	code := c.Run([]string{"tag", "-r"})
	t.Log("command 'tag -r' terminated")

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

	if !strings.Contains(output, tg) {
		t.Fatalf("Output should have included the tag's name")
	}

	t.Log("Checking that the task is no longer tagged")

	tsk := &models.Task{
		Id: task.Id,
	}
	if err := db.PopulateByID(tsk); err != nil {
		t.Fatal(err)
	}

	t.Logf("Here's the task:\n%+v", tsk)
	t.Logf("Here's the tag:\n%+v", tg)

	if len(tsk.Tags) != 0 {
		t.Fatal("Expected the task to have no tag")
	}
}

// --- }}}

// --- `elos todo today` {{{
func TestTodoToday(t *testing.T) {
	ui, db, user, c := newMockTodoCommand(t)

	// load a task into the db
	tsk := newTestTask(t, db, user)
	taskName := "Take out the trash"
	tsk.Name = taskName
	task.StopAndComplete(tsk)
	if err := db.Save(tsk); err != nil {
		t.Fatal(err)
	}

	tsk2 := newTestTask(t, db, user)
	task2Name := "shouldn't show up"
	tsk2.Name = task2Name
	tsk2.CompletedAt = models.TimestampFrom(time.Now().Add(-48 * time.Hour))
	if err := db.Save(tsk2); err != nil {
		t.Fatal(err)
	}

	t.Log("running: `elos todo today`")
	code := c.Run([]string{"today"})
	t.Log("command 'today' terminated")

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
		t.Fatalf("Output should have contained a task we completed today: %s", taskName)
	}

	if strings.Contains(output, task2Name) {
		t.Fatalf("Output should not have contained: '%s', the name of a task completed 2 days ago", task2Name)
	}
}

// --- }}}

// --- }}}

// --- Internals {{{
// --- }}}

// --- }}}
