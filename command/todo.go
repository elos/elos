package command

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// exit statuses
const (
	success = 0
	failure = 1
)

// TodoCommand contains the state necessary to implement the
// 'elos todo' command set.
//
// It implements the cli.Command interface
type TodoCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must be non-null
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified.
	UserID string

	// DB is the elos database we interface with.
	data.DB

	// The tasks of the user given by c.UserID
	//
	// During the lifecycle of the command, and assuming
	// the user is only accessing the elos system through
	// the command prompt, the task list is complete and
	// definitive (reflects exactly what is in the database).
	tasks []*models.Task
}

// Synopsis is a one-line, short summary of the 'todo' command.
// It is guaranteed to be at most 50 characters.
func (c *TodoCommand) Synopsis() string {
	return "Utilities for managing elos tasks"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possibly a complete
// list of flags the 'todo' command accepts.
func (c *TodoCommand) Help() string {
	helpText := `
Usage:
	elos todo <subcommand>

Subcommands:
	complete	complete a task
	delete		delete a task
	edit		edit a task
	list		list all your tasks
	new		create a new task
	start		start a task
	stop		stop a task
	suggest		have elos suggest a task
`
	return strings.TrimSpace(helpText)
}

// Run runs the 'todo' command with the given command-line arguments.
// It returns an exit status when it finishes. 0 indicates a sucess,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface.
func (c *TodoCommand) Run(args []string) int {
	// short circuit to avoid loading tasks
	if len(args) == 0 && c.UI != nil {
		c.UI.Output(c.Help())
		return success
	}

	// fully initialize the command, and bail if not a success
	if i := c.init(); i != success {
		return i
	}

	switch len(args) {
	case 1:
		switch args[0] {
		case "c":
		case "complete":
			return c.runComplete()
		case "d":
		case "delete":
			c.runDelete()
		case "e":
		case "edit":
			c.runEdit()
		case "l":
		case "list":
			c.runList()
		case "n":
		case "new":
			return c.runNew()
		case "sta":
		case "start":
			c.runStart()
		case "sto":
		case "stop":
			c.runStop()
		case "su":
		case "suggest":
			c.runSuggest()
		default:
			c.UI.Output(c.Help())
		}
	default:
		c.UI.Output(c.Help())
	}

	return success
}

// init performs some verification that the TodoCommand object
// is valid (has a non-null database & UI and a user id).
//
// It loads all of the UserID's tasks into the tasks field of the
// TodoCommand object.
//
// A 0 return value indicates success, a 1 indiciates failure. The
// init command handles appropriate error printing the the UI.
func (c *TodoCommand) init() int {
	// ensure that we have a interface
	if c.UI == nil {
		return failure // we c.errorf because the user interface isn't defined
	}

	// ensure that we have a database
	if c.DB == nil {
		c.errorf("initialization: no database")
		return failure
	}

	// ensure that we have a user id
	if c.UserID == "" {
		c.errorf("initialization: no user id")
		return failure
	}

	taskQuery := c.DB.Query(models.TaskKind)

	// only retrieve _incomplete_ tasks
	taskQuery.Select(data.AttrMap{
		"owner_id": c.UserID,
		"complete": false,
	})

	iter, err := taskQuery.Execute()
	if err != nil {
		c.errorf("data retrieval: querying tasks")
		return failure
	}

	t := models.NewTask()
	tasks := make([]*models.Task, 0)
	for iter.Next(t) {
		tasks = append(tasks, t)
		t = models.NewTask()
	}

	if err := iter.Close(); err != nil {
		c.errorf("data retrieval: querying tasks")
		return 1
	}

	c.tasks = tasks

	return success
}

// errorf is a IO function which performs the equivalent of log.Errorf
// in the standard lib, except using the cli.Ui interface with which
// the TodoCommand was provided.
func (c *TodoCommand) errorf(s string, values ...interface{}) {
	c.UI.Error("[elos todo] Error: " + fmt.Sprintf(s, values...))
}

// removeTask removes the task at the given index.
// You may use this for removing a task from memory after
// it has been completed, or deleted.
func (c *TodoCommand) removeTask(index int) {
	c.tasks = append(c.tasks[index:], c.tasks[index+1:]...)
}

// runComplete executes the "elos todo complete" command.
//
// Complete first prints a numbered list of the user's tasks.
// Then it prompts the user for which number task to complete.
// The user's tasks list (c.tasks) only contains incomplete tasks.
// If the task is in progress, it is also stopped. Finally, the task is
// removed from the c.tasks.
func (c *TodoCommand) runComplete() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	task.StopAndComplete()

	err := c.DB.Save(task)
	if err != nil {
		c.errorf("(subcommand complete) Error: %s", err)
		return failure
	}

	// remove the tasks from the list becuase it is now complete
	c.removeTask(index)

	c.UI.Info(fmt.Sprintf("Completed '%s'", task.Name))

	return success
}

// runDelete runs the 'delete' subcommand.
//
// It returns an exit status:
// 0 := success
// 1 := failure
func (c *TodoCommand) runDelete() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	err := c.DB.Delete(task)
	if err != nil {
		c.errorf("(subcommand delete) Error: %s", err)
		return failure
	}

	c.removeTask(index)

	c.UI.Info(fmt.Sprintf("Deleted '%s'", task.Name))

	return success
}

// runEdit runs the 'edit' subcommand, which is currently not implemented
func (c *TodoCommand) runEdit() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	bytes, err := json.MarshalIndent(task, "", "	")
	if err != nil {
		return failure
	}
	c.UI.Output(string(bytes))

	var attributeToEdit string
	attributeToEdit, err = stringInput(c.UI, "Which attribute?")

	switch attributeToEdit {
	case "complete":
		task.Complete, err = boolInput(c.UI, "Completed?")
	case "created_at":
		task.CreatedAt, err = dateInput(c.UI, "CreatedAt?")
	case "deadline":
		task.Deadline, err = dateInput(c.UI, "New deadline?")
	case "deleted_at":
		task.DeletedAt, err = dateInput(c.UI, "DeletedAt?")
	case "name":
		task.Name, err = stringInput(c.UI, "New name?")
	default:
		c.UI.Warn("That attribute is not recognized/supported")
		return success
	}

	if err != nil {
		c.errorf("(subcommand edit) Error %s", err)
		return failure
	}

	if err = c.DB.Save(task); err != nil {
		c.errorf("(subcommand edit) Error: %s", err)
		return failure
	}

	return success
}

// runList runs the 'list' subcommand. It prints a list of the
// tasks cached in c.tasks.
func (c *TodoCommand) runList() int {
	c.UI.Output("Todos:")
	c.printTaskList()
	return success
}

// runNew runs the 'new' subcommand, which prompts the user to
// create a new task.
func (c *TodoCommand) runNew() int {
	_, err := c.promptNewTask()
	if err != nil {
		c.errorf("(subcommand  new): Error: %s", err)
		return failure
	}
	return success
}

func (c *TodoCommand) runStart() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	// task.Start() is idempotent, and simply won't
	// do anything if the task is in progress, but we
	// want to indicate to the user if they are not
	// actually starting the task
	if task.InProgress() {
		c.UI.Warn("Task is already in progress")
		return success
	}

	task.Start()

	if err := c.DB.Save(task); err != nil {
		c.errorf("(subcommand start) Error: %s", err)
		return failure
	}

	c.UI.Info(fmt.Sprintf("Started '%s'", task.Name))

	return success
}

// runStop runs the 'stop' command, which stops a task specified
// by the user.
func (c *TodoCommand) runStop() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	// task.Stop() is idempotent, meaning it won't stop the task
	// if it is not in progress, but we want to indicate this condition
	// to the user.
	if !task.InProgress() {
		c.UI.Warn("Task is not in progress")
		return success
	}

	task.Stop()

	if err := c.DB.Save(task); err != nil {
		c.errorf("(subcommand stop) Error: %s", err)
		return failure
	}

	// Info, i.e., "You worked for 20m that time"
	c.UI.Info(fmt.Sprintf("You worked for %s that time", task.Stages[len(task.Stages)-1].Sub(task.Stages[len(task.Stages)-2])))
	return success
}

// runSuggest runs the 'suggest' subcommand, which uses elos'
// most important task algorithm to suggest the one to work on
func (c *TodoCommand) runSuggest() int {
	if len(c.tasks) == 0 {
		c.UI.Info("You have no tasks")
		return success
	}

	c.UI.Output(models.NewTaskGraph(c.tasks).Suggest().Name)
	return success
}

// printTaskList prints the list of tasks, with deadline and salience info
// the list is numbered, and can be useful for tasks that involve the user
// looking at / selecting a particular task (however use promptSelectTask
// for the case of selecting a single task from the c.tasks)
func (c *TodoCommand) printTaskList() {
	for i, t := range c.tasks {
		c.UI.Output(fmt.Sprintf(" %d) %s [%s]\n\tSalience:%f", i, t.Name, t.Deadline.Format("Mon Jan 2 15:04"), t.Salience()))
	}
}

// promptSelectTask prompts the user to select one of their tasks. The
// first return argument is the task the user selected, and the second is
// the index of that task. If the index is negative, then there was either an
// error retrieving a task selection from the user, or the user has no tasks,
// in either case the value of the first return argument is undefined.
//
// Use promptSelectTask for todo subcommands which operate on a task.
func (c *TodoCommand) promptSelectTask() (*models.Task, int) {
	if len(c.tasks) == 0 {
		c.UI.Warn("You do not have any tasks")
		return nil, -1
	}

	c.printTaskList()

	var (
		indexOfCurrent int
		err            error
	)

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil, -1
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.tasks)-1))
		return nil, -1 // to indicate the parent command to exit
	}

	return c.tasks[indexOfCurrent], indexOfCurrent
}

// promptNewTask implements the process of creating a task using text
// input and output
//
// Use for creating a new task, which promptNewTask returns a handle to.
//
// promptNewTask adds the task to c.tasks.
//
func (c *TodoCommand) promptNewTask() (task *models.Task, err error) {
	var (
		hasDeadline bool
		hasPrereqs  bool
	)

	task = models.NewTask()
	task.SetID(c.DB.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = c.UserID

	if task.Name, err = stringInput(c.UI, "Name:"); err != nil {
		return
	}

	if hasDeadline, err = yesNo(c.UI, "Does it have a deadline?"); err != nil {
		return
	} else if hasDeadline {
		if task.Deadline, err = dateInput(c.UI, "Deadline:"); err != nil {
			return
		}
	}

	if hasPrereqs, err = yesNo(c.UI, "Does it have any prerequisites?"); err != nil {
		return
	} else if hasPrereqs {
		var currentTaskPrereq, newTaskPrereq bool

		if len(c.tasks) > 0 {
			c.printTaskList()
			if currentTaskPrereq, err = yesNo(c.UI, "Any dependencies that are current?"); err != nil {
				return
			} else if currentTaskPrereq {
				for currentTaskPrereq {
					var indexOfCurrent int

					if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
						return
					}

					if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
						c.UI.Warn("That isn't a valid index")
						continue
					}

					task.IncludePrerequisite(c.tasks[indexOfCurrent])

					if currentTaskPrereq, err = yesNo(c.UI, "Any more current prereqs?"); err != nil {
						return
					}
				}
			}
		}

		if newTaskPrereq, err = yesNo(c.UI, "Any dependencies that are new tasks?"); err != nil {
			return
		} else if newTaskPrereq {
			var newTask *models.Task
			for newTaskPrereq {
				if newTask, err = c.promptNewTask(); err != nil {
					return
				}

				task.IncludePrerequisite(newTask)

				if newTaskPrereq, err = yesNo(c.UI, "Any more new prereqs?"); err != nil {
					return
				}

			}
		}
	}

	task.UpdatedAt = time.Now()

	// if successful save
	if err = c.DB.Save(task); err == nil {
		c.tasks = append(c.tasks, task)
		c.UI.Output("Task created")
	}

	return
}
