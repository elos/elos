package command

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/tag"
	"github.com/elos/models/task"
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
	// It must be non-nil
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified.
	UserID string

	// DB is the elos database we interface with.
	// It must be non-nil
	data.DB

	// The tasks of the user given by c.UserID
	//
	// During the lifecycle of the command, and assuming
	// the user is only accessing the elos system through
	// the command prompt, the task list is complete and
	// definitive (reflects exactly what is in the database).
	tasks []*models.Task

	// The tags of the user given by c.UserID
	tags map[data.ID]*models.Tag
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
	current		list current tasks
	delete		delete a task
	edit		edit a task
	fix		set new deadlines for passed tasks
	goal		set a task as a goal
	goals		list task goals
	list		list all your tasks
	new		create a new task
	start		start a task
	stop		stop a task
	suggest		have elos suggest a task
	tag		tag a task
	today		list the tasks you completed today
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

	switch args[0] {
	case "co":
	case "complete":
		return c.runComplete()
	case "cu":
	case "current":
		return c.runCurrent()
	case "d":
	case "delete":
		return c.runDelete()
	case "e":
	case "edit":
		return c.runEdit()
	case "f":
	case "fix":
		return c.runFix()
	case "g":
	case "goal":
		return c.runGoal()
	case "gs":
	case "goals":
		return c.runGoals()
	case "l":
	case "list":
		return c.runList()
	case "n":
	case "new":
		return c.runNew()
	case "sta":
	case "start":
		return c.runStart()
	case "sto":
	case "stop":
		return c.runStop()
	case "su":
	case "suggest":
		return c.runSuggest()
	case "ta":
	case "tag":
		if len(args) == 2 && args[1] == "-r" {
			return c.runRemoveTag()
		}

		return c.runTag()
	case "to":
	case "today":
		return c.runToday()
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
// It loads all of the UserID's tags into the tags field of the
// TodoCommand object.
//
// A 0 return value indicates success, a 1 indiciates failure. The
// init command handles appropriate error printing the the UI.
func (c *TodoCommand) init() int {
	// ensure that we have a interface
	if c.UI == nil {
		return failure // we can't c.errorf because the user interface isn't defined
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

	// Load the tasks

	taskQuery := c.DB.Query(models.TaskKind)

	taskQuery.Select(data.AttrMap{
		"owner_id": c.UserID,
	})

	iter, err := taskQuery.Execute()
	if err != nil {
		c.errorf("data retrieval: querying tasks")
		return failure
	}

	t := models.NewTask()
	tasks := make([]*models.Task, 0)
	for iter.Next(t) {
		if !task.IsComplete(t) {
			tasks = append(tasks, t)
			t = models.NewTask()
		}
	}

	if err := iter.Close(); err != nil {
		c.errorf("data retrieval: querying tasks")
		return failure
	}

	c.tasks = tasks

	// Load the tags
	c.tags = make(map[data.ID]*models.Tag)

	iter, err = c.DB.Query(models.TagKind).Select(data.AttrMap{
		"owner_id": c.UserID,
	}).Execute()
	if err != nil {
		c.errorf("data retrieval: querying tags")
		return failure
	}

	tag := models.NewTag()
	for iter.Next(tag) {
		c.tags[tag.ID()] = tag
		tag = models.NewTag()
	}

	if err := iter.Close(); err != nil {
		c.errorf("data retrieval: querying tags")
		return failure
	}

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
	tsk, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	task.StopAndComplete(tsk)

	err := c.DB.Save(tsk)
	if err != nil {
		c.errorf("(subcommand complete) Error: %s", err)
		return failure
	}

	// remove the tasks from the list becuase it is now complete
	c.removeTask(index)

	c.UI.Info(fmt.Sprintf("Completed '%s'", tsk.Name))
	c.UI.Info(fmt.Sprintf("Worked for %s total", task.TimeSpent(tsk)))

	return success
}

// runCurrent executes the "elos todo current" command.
//
// Current prints the tasks that are currently in progress
func (c *TodoCommand) runCurrent() int {
	printedTask := false
	c.printTaskList(func(t *models.Task) bool {
		ok := task.InProgress(t)

		if ok {
			printedTask = true
		}

		return ok
	})

	if !printedTask {
		c.UI.Output("You have no tasks in progress")
	}

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

// runEdit runs the 'edit' subcommand. It returns a status code, 0 indicates
// success, and 1 failure.
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
	if err != nil {
		return failure
	}

	switch attributeToEdit {
	case "completed_at":
		task.CompletedAt, err = dateInput(c.UI, "CompletedAt?")
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

	c.UI.Info("Task updated")

	return success
}

// runFix executes the "elos todo fix" command.
//
// Fix goes through the tasks whose deadline has passed and
// prompts the user to set a new deadline
func (c *TodoCommand) runFix() int {
	var inputError error

	neededFix := false

	// Only need the incomplete tasks, which are in c.tasks
	for i, t := range c.tasks {
		// If the deadline is in the future
		if t.Deadline.IsZero() || t.Deadline.Local().After(time.Now()) {
			continue
		}

		neededFix = true

		c.UI.Output(fmt.Sprintf("%d) %s %s", i, t.Name, t.Deadline.Format("Mon Jan 2 15:04")))

	fix:
		if t.Deadline, inputError = dateInput(c.UI, "New Deadline"); inputError != nil {
			c.errorf("(subcommand fix) Input Error: %s", inputError)
			return failure
		}

		if t.Deadline.Local().Before(time.Now()) {
			c.UI.Output(fmt.Sprintf("Shoot, %s is still in the past, try again?", t.Deadline.Format("Mon Jan 2 15:04")))
			goto fix
		}

		if err := c.DB.Save(t); err != nil {
			c.errorf("(subcommand fix) Error: saving task: %s", err)
			return failure
		} else {
			c.UI.Output(fmt.Sprintf("Deadline changed to %s", t.Deadline.Local().Format("Mon Jan 2 15:04")))
		}
	}

	if !neededFix {
		c.UI.Output("No tasks out of date")
	}

	return success
}

// runGoal runs the 'goal' subcommand, which adds this task to this
// user's goals
func (c *TodoCommand) runGoal() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	u := &models.User{Id: c.UserID}
	if err := c.DB.PopulateByID(u); err != nil {
		c.errorf("retrieving user: %s", err)
		return failure
	}

	goalTag, err := tag.ForName(c.DB, u, tag.Goal)
	if err != nil {
		c.errorf("retrieving GOAL tag: %s", err)
		return failure
	}

	task.IncludeTag(goalTag)

	if err := c.DB.Save(task); err != nil {
		c.errorf("saving task: %s", err)
		return failure
	}

	return success
}

// runGoals runs the 'goals' subcommand, which prints the user's goals
func (c *TodoCommand) runGoals() int {
	goalTag, err := tag.ForName(c.DB, &models.User{Id: c.UserID}, tag.Goal)
	if err != nil {
		c.errorf("retrieving GOAL tag: %s", err)
		return failure
	}

	tasks, err := tag.TasksFor(c.DB, goalTag)
	if err != nil {
		c.errorf("retrieving GOAL tasks: %s", err)
		return failure
	}

	taskIds := make(map[data.ID]bool)
	for i := range tasks {
		if !task.IsComplete(tasks[i]) {
			taskIds[tasks[i].ID()] = true
		}
	}

	if len(taskIds) == 0 {
		c.UI.Output("No goals set. Use `elos todo goal` to add a goal.")
		return success
	}

	c.UI.Output("Current Goals:")
	c.printTaskList(func(t *models.Task) bool {
		_, ok := taskIds[t.ID()]
		return ok
	})

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
	tsk, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	// task.Start(tsk) is idempotent, and simply won't
	// do anything if the task is in progress, but we
	// want to indicate to the user if they are not
	// actually starting the task
	if task.InProgress(tsk) {
		c.UI.Warn("Task is already in progress")
		return success
	}

	task.Start(tsk)

	if err := c.DB.Save(tsk); err != nil {
		c.errorf("(subcommand start) Error: %s", err)
		return failure
	}

	c.UI.Info(fmt.Sprintf("Started '%s'", tsk.Name))

	return success
}

// runStop runs the 'stop' command, which stops a task specified
// by the user.
func (c *TodoCommand) runStop() int {
	anyInProgress := false
	for _, t := range c.tasks {
		if task.InProgress(t) {
			anyInProgress = true
			break
		}
	}

	if !anyInProgress {
		c.UI.Output("No tasks in progress")
		return success
	}

	tsk, index := c.promptSelectTask(func(t *models.Task) bool {
		return task.InProgress(t)
	})

	if index < 0 {
		return failure
	}

	// task.Stop(tsk) is idempotent, meaning it won't stop the task
	// if it is not in progress, but we want to indicate this condition
	// to the user.
	if !task.InProgress(tsk) {
		c.UI.Warn("Task is not in progress")
		return success
	}

	task.Stop(tsk)

	if err := c.DB.Save(tsk); err != nil {
		c.errorf("(subcommand stop) Error: %s", err)
		return failure
	}

	// Info, i.e., "You worked for 20m that time"
	c.UI.Info(fmt.Sprintf("You worked for %s that time", tsk.Stages[len(tsk.Stages)-1].Sub(tsk.Stages[len(tsk.Stages)-2])))
	return success
}

// runSuggest runs the 'suggest' subcommand, which uses elos'
// most important task algorithm to suggest the one to work on
func (c *TodoCommand) runSuggest() int {
	if len(c.tasks) == 0 {
		c.UI.Info("You have no tasks")
		return success
	}

	c.UI.Output(task.NewGraph(c.tasks).Suggest().Name)
	return success
}

// runTag runs the 'tag' subcommand, which uses elos'
// tagging system to tag a particular task
func (c *TodoCommand) runTag() int {
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	c.UI.Output("Which tag to add?")
	tag := c.promptSelectTag()
	if tag == nil {
		return failure
	}

	task.IncludeTag(tag)

	if err := c.DB.Save(task); err != nil {
		c.errorf("saving task")
		return failure
	}

	c.UI.Output(fmt.Sprintf("Added '%s' to task", tag.Name))

	return success
}

// runRemoveTag runs the 'tag' subcommand with the -r flag,
// which removes a tag from a task
func (c *TodoCommand) runRemoveTag() int {
	c.UI.Output("Select which task to remove a tag from")
	task, index := c.promptSelectTask()
	if index < 0 {
		return failure
	}

	c.UI.Output("Which tag to remove?")
	tag := c.promptSelectTagFromTask(task)
	if tag == nil {
		return failure
	}

	task.ExcludeTag(tag)

	if err := c.DB.Save(task); err != nil {
		c.errorf("saving task")
		return failure
	}

	c.UI.Output(fmt.Sprintf("Removed '%s' from task", tag.Name))

	return success
}

// runToday executes the "elos todo today" command.
//
// Today prints the tasks that are were completed today
func (c *TodoCommand) runToday() int {
	iter, err := c.DB.Query(models.TaskKind).Select(data.AttrMap{
		"owner_id": c.UserID,
	}).Execute()

	if err != nil {
		c.errorf("querying tasks: %s", err)
	}

	t := models.NewTask()
	i := 0
	for iter.Next(t) {
		if task.IsComplete(t) && models.DayEquivalent(t.CompletedAt.Local(), time.Now()) {
			c.UI.Output(fmt.Sprintf("%d) %s", i, t.Name))
			i++
		}
	}

	if i == 0 {
		c.UI.Output("You have completed no tasks today")
	}

	return success
}

// printTaskList prints the list of tasks, with deadline and salience info
// the list is numbered, and can be useful for tasks that involve the user
// looking at / selecting a particular task (however use promptSelectTask
// for the case of selecting a single task from the c.tasks)
func (c *TodoCommand) printTaskList(selectors ...func(*models.Task) bool) {
PrintLoop:
	for i, t := range c.tasks {
		for i := range selectors {
			if !selectors[i](t) {
				continue PrintLoop
			}
		}

		// Tags
		tagList := ""
		for _, id := range t.TagsIds {
			tagList += fmt.Sprintf(" [%s]", c.tags[data.ID(id)].Name)
		}
		if tagList != "" {
			tagList += ": "
		} else {
			tagList = " " + tagList
		}

		// Deadline
		deadline := ""
		if !t.Deadline.Equal(*new(time.Time)) {
			deadline = fmt.Sprintf("(%s)", t.Deadline.Local().Format("Mon Jan 2 15:04"))
		}

		c.UI.Output(fmt.Sprintf("%d)%s%s %s\n\tSalience:%f; Time Spent:%s", i, tagList, t.Name, deadline, task.Salience(t), task.TimeSpent(t)))
	}
}

// promptSelectTask prompts the user to select one of their tasks. The
// first return argument is the task the user selected, and the second is
// the index of that task. If the index is negative, then there was either an
// error retrieving a task selection from the user, or the user has no tasks,
// in either case the value of the first return argument is undefined.
//
// Use promptSelectTask for todo subcommands which operate on a task.
func (c *TodoCommand) promptSelectTask(selectors ...func(*models.Task) bool) (*models.Task, int) {
	if len(c.tasks) == 0 {
		c.UI.Warn("You do not have any tasks")
		return nil, -1
	}

	c.printTaskList(selectors...)

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

func (c *TodoCommand) printTagList(tags []*models.Tag) {
	for i, t := range tags {
		c.UI.Output(fmt.Sprintf("%d) %s", i, t.Name))
	}
}

func (c *TodoCommand) promptSelectTag() *models.Tag {
	if len(c.tags) == 0 {
		c.UI.Warn("You do not have any tags")
		return nil
	}

	tags := make([]*models.Tag, len(c.tags))
	count := 0
	for _, t := range c.tags {
		tags[count] = t
		count++
	}

	sort.Sort(tag.ByName(tags))

	c.printTagList(tags)

	var (
		indexOfCurrent int
		err            error
	)

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.tags)-1))
		return nil
	}

	return tags[indexOfCurrent]
}

func (c *TodoCommand) promptSelectTagFromTask(t *models.Task) *models.Tag {
	tags, err := t.Tags(c.DB)
	if err != nil {
		c.errorf("data Error: %s", err)
		return nil
	}

	if len(tags) == 0 {
		c.UI.Warn("That task has no tags")
		return nil
	}

	sort.Sort(tag.ByName(tags))

	c.printTagList(tags)

	var indexOfCurrent int

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.tags)-1))
		return nil
	}

	return tags[indexOfCurrent]
}
