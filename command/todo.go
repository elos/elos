package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

type TodoCommand struct {
	UI cli.Ui
	*Config
	data.DB

	tasks []*models.Task
}

func (c *TodoCommand) Help() string {
	helpText := `
usage: elos todo <subcommand>

Available subcommands:
	complete	complete a task
	delete		delete a task
	edit		edit a task
	list		list all your tasks
	new			create a new task
	start		start a task
	stop		stop a task
	suggest		have elos suggest a task
`
	return strings.TrimSpace(helpText)
}

func (c *TodoCommand) Run(args []string) int {
	if i := c.init(); i != 0 {
		return i
	}

	switch len(args) {
	case 1:
		switch args[0] {
		case "new":
			_, err := c.promptNewTask()
			if err != nil {
				c.UI.Error(fmt.Sprintf("Error creating task: %s", err))
				return 1
			}
			break
		case "complete":
			c.printTaskList()

			var indexOfCurrent int
			var err error

			if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
				c.UI.Error(fmt.Sprintf("Input error: %s", err))
				return 1
			}

			if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
				c.UI.Warn("That isn't a valid index")
				break
			}

			task := c.tasks[indexOfCurrent]

			task.Complete = true
			task.UpdatedAt = time.Now()
			err = c.DB.Save(task)
			if err != nil {
				c.UI.Error(fmt.Sprintf("DB error: %s", err))
				return 1
			}
		case "start":
			c.printTaskList()

			var indexOfCurrent int
			var err error

			if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
				c.UI.Error(fmt.Sprintf("Input error: %s", err))
				return 1
			}

			if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
				c.UI.Warn("That isn't a valid index")
				break
			}

			task := c.tasks[indexOfCurrent]

			if len(task.Stages)%2 != 0 {
				c.UI.Error("Task is already in progress")
				return 1
			}

			task.Stages = append(task.Stages, time.Now())
			task.UpdatedAt = time.Now()

			err = c.DB.Save(task)
			if err != nil {
				c.UI.Error(fmt.Sprintf("DB error: %s", err))
				return 1
			}
		case "stop":
			c.printTaskList()

			var indexOfCurrent int
			var err error

			if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
				c.UI.Error(fmt.Sprintf("Input error: %s", err))
				return 1
			}

			if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
				c.UI.Warn("That isn't a valid index")
				break
			}

			task := c.tasks[indexOfCurrent]

			if len(task.Stages)%2 != 1 {
				c.UI.Error("Task is not in progress")
				return 1
			}

			task.Stages = append(task.Stages, time.Now())
			task.UpdatedAt = time.Now()

			err = c.DB.Save(task)
			if err != nil {
				c.UI.Error(fmt.Sprintf("DB error: %s", err))
				return 1
			}

			c.UI.Info(fmt.Sprintf("Worked for %s that time", task.Stages[len(task.Stages)-1].Sub(task.Stages[len(task.Stages)-2])))
		case "edit":
		case "delete":
			c.printTaskList()

			var indexOfCurrent int
			var err error

			if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
				c.UI.Error(fmt.Sprintf("Input error: %s", err))
				return 1
			}

			if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
				c.UI.Warn("That isn't a valid index")
				break
			}

			task := c.tasks[indexOfCurrent]

			err = c.DB.Delete(task)
			if err != nil {
				c.UI.Error(fmt.Sprintf("DB error: %s", err))
				return 1
			}
		case "list":
			c.UI.Output("Todos:")
			c.printTaskList()
		case "suggest":
			if len(c.tasks) == 0 {
				c.UI.Info("You have no tasks")
				break
			}

			c.UI.Output(models.NewTaskGraph(c.tasks).Suggest().Name)
		default:
			c.UI.Output(c.Help())
		}
	default:
		c.UI.Output(c.Help())
	}
	return 0
}

func (c *TodoCommand) promptNewTask() (task *models.Task, inputErr error) {
	var (
		hasDeadline bool
		hasPrereqs  bool
	)

	task = models.NewTask()
	task.SetID(c.DB.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = c.Config.UserID

	if task.Name, inputErr = stringInput(c.UI, "Name:"); inputErr != nil {
		return
	}

	if hasDeadline, inputErr = yesNo(c.UI, "Does it have a deadline?"); inputErr != nil {
		return
	} else if hasDeadline {
		if task.Deadline, inputErr = dateInput(c.UI, "Deadline:"); inputErr != nil {
			return
		}
	}

	if hasPrereqs, inputErr = yesNo(c.UI, "Does it have any prerequisites?"); inputErr != nil {
		return
	} else if hasPrereqs {
		var currentTaskPrereq, newTaskPrereq bool

		if len(c.tasks) > 0 {
			c.printTaskList()
			if currentTaskPrereq, inputErr = yesNo(c.UI, "Any dependencies that are current?"); inputErr != nil {
				return
			} else if currentTaskPrereq {
				for currentTaskPrereq {
					var indexOfCurrent int

					if indexOfCurrent, inputErr = intInput(c.UI, "Which number?"); inputErr != nil {
						return
					}

					if indexOfCurrent < 0 || indexOfCurrent > len(c.tasks)-1 {
						c.UI.Warn("That isn't a valid index")
						continue
					}

					task.IncludePrerequisite(c.tasks[indexOfCurrent])

					if currentTaskPrereq, inputErr = yesNo(c.UI, "Any more current prereqs?"); inputErr != nil {
						return
					}
				}
			}
		}

		if newTaskPrereq, inputErr = yesNo(c.UI, "Any dependencies that are new tasks?"); inputErr != nil {
			return
		} else if newTaskPrereq {
			var newTask *models.Task
			for newTaskPrereq {
				if newTask, inputErr = c.promptNewTask(); inputErr != nil {
					return
				}

				task.IncludePrerequisite(newTask)

				if newTaskPrereq, inputErr = yesNo(c.UI, "Any more new prereqs?"); inputErr != nil {
					return
				}

			}
		}
	}

	task.UpdatedAt = time.Now()

	if err := c.DB.Save(task); err != nil {
		inputErr = err
	} else {
		c.tasks = append(c.tasks, task)
		c.UI.Output("Task created")
	}

	return
}

func (c *TodoCommand) printTaskList() {
	for i, t := range c.tasks {
		c.UI.Output(fmt.Sprintf(" %d) %s [%s]\n\tSalience:%f", i, t.Name, t.Deadline.Format("Mon Jan 2 15:04"), t.Salience()))
	}
}

func (c *TodoCommand) init() int {
	// Guard on database
	if c.DB == nil {
		c.UI.Error("No database")
		return 1
	}

	// Guard on user id
	if c.Config.UserID == "" {
		c.UI.Error("No user id")
		return 1
	}

	q := c.DB.NewQuery(models.TaskKind)
	q.Select(data.AttrMap{
		"owner_id": c.Config.UserID,
		"complete": false,
	})
	iter, err := q.Execute()
	if err != nil {
		c.UI.Error("Error looking up tasks")
		return 1
	}

	t := models.NewTask()
	tasks := make([]*models.Task, 0)
	for iter.Next(t) {
		tasks = append(tasks, t)
		t = models.NewTask()
	}
	if err := iter.Close(); err != nil {
		c.UI.Error("Error iterating tasks")
		return 1
	}

	c.tasks = tasks
	return 0
}

func (c *TodoCommand) Synopsis() string {
	return "todo synppsis"
}
