package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/habit"
	"github.com/mitchellh/cli"
)

// HabitCommand contains the state necessary to implement the
// 'elos habit' command set.
//
// It implements the cli.Command interface
type HabitCommand struct {
	// UI is used the communicate (for IO) with the user
	// It must not be nil
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified.
	UserID string

	// DB is the elos database we interfae with.
	// It must not be nil.
	data.DB

	// habits is the list of this user's habits
	habits []*models.Habit
}

// Synopsis is a one-line, short summary of the 'habit' command.
// It is guaranteed to be at most 50 characters.
func (c *HabitCommand) Synopsis() string {
	return "Utilities for managing & building habits"
}

func (c *HabitCommand) Help() string {
	helpText := `
Usage:
	elos habit <subcommand>

Subcommands:
	checkin		mark a habit as complete for today
	delete		delete a habit
	history		see all checkins for a habit
	list		list all habits
	new		create a new habit
	today		see today's habits and which have been checked off
`
	return strings.TrimSpace(helpText)
}

// Run runs the 'habit' command with the given command-line arguments.
// It returns an exit status when it finishs. 0 indicates a success,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface
func (c *HabitCommand) Run(args []string) int {
	// short circuit to avoid loading habits
	if len(args) == 0 && c.UI != nil {
		c.UI.Output(c.Help())
		return success
	}

	// fully initialize the command, and bail if not a success
	if i := c.init(); i != success {
		return i
	}

	switch args[0] {
	case "checkin":
		return c.runCheckin(args)
	case "delete":
		return c.runDelete(args)
	case "history":
		return c.runHistory(args)
	case "list":
		return c.runList(args)
	case "new":
		return c.runNew(args)
	case "today":
		return c.runToday(args)
	default:
		c.UI.Output(c.Help())
	}

	return success
}

// removeHabit removes the person at the given index.
// You may use this for removing a habit after it have
// been deleted
func (c *HabitCommand) removeHabit(index int) {
	c.habits = append(c.habits[index:], c.habits[index+1:]...)
}

// errorf calls UI.Error with a formatted, prefixed error string
// always use it to print an error, avoid using UI.Error directly
func (c *HabitCommand) errorf(format string, values ...interface{}) {
	c.UI.Error(fmt.Sprintf("(elos habit) Error: "+format, values...))
}

// printf calls UI.Output with the formmated string
// always prefer printf over c.UI.Output
func (c *HabitCommand) printf(format string, values ...interface{}) {
	c.UI.Output(fmt.Sprintf(format, values...))
}

// init performs the necessary initliazation for the *HabitCommand
// It ensures we have a UI, DB and UserID, so those can be treated
// as invariants throughought the rest of the code.
//
// Additionally it loads this user's habit list.
func (c *HabitCommand) init() int {
	if c.UI == nil {
		// can't use errorf, because the UI is not defined
		return failure
	}

	if c.UserID == "" {
		c.errorf("no UserID provided")
		return failure
	}

	if c.DB == nil {
		c.errorf("no database")
		return failure
	}

	q := c.DB.Query(models.HabitKind)
	q.Select(data.AttrMap{
		"owner_id": c.UserID,
	})
	iter, err := q.Execute()
	if err != nil {
		c.errorf("while querying for habits: %s", err)
		return failure
	}

	c.habits = make([]*models.Habit, 0)
	habit := models.NewHabit()

	for iter.Next(habit) {
		c.habits = append(c.habits, habit)
		habit = models.NewHabit()
	}

	if err := iter.Close(); err != nil {
		c.errorf("while querying for habits: %s", err)
		return failure
	}

	return success
}

// printHabitList prints a numbered list of the habits in the habits slice
func (c *HabitCommand) printHabitList() {
	for i, h := range c.habits {
		c.printf("%d) %s", i, h.Name)
	}
}

// promptSelectHabit prompts the user to select a habits from their list
// of habits.
//
// Use promptNewHabit for any subcommand which acts on a habit.
// Retrieve the habit:
//		habit, index := promptSelectHabit()
//		if index < 0 {
//			return failure
//		}
//
// As you can see in the example above, a negative index indicates failure.
//
// NOTE: the integer here is not a status code, but rather the index of the habit
// in the command's habits slice
func (c *HabitCommand) promptSelectHabit() (*models.Habit, int) {
	if len(c.habits) == 0 {
		c.UI.Warn("You do not have any habits")
		return nil, -1
	}

	c.printHabitList()

	var (
		indexOfCurrent int
		err            error
	)

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil, -1
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.habits)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.habits)-1))
		return nil, -1 // to indicate the parent command to exit
	}

	return c.habits[indexOfCurrent], indexOfCurrent
}

// promptNewHabit provides the input prompts necessary to construct a new habit.
//
// Use this to implement the 'new' subcommand, and for any subcommand which requires
// creating a new habit as part of it's functionality.
//
// It returns a habit and status code. A 0 status code indicates success, any other
// status indicates failure, and the caller should exit immediately. promptNewHabit
// will have taken care of printing the error output.
func (c *HabitCommand) promptNewHabit() (*models.Habit, int) {
	var name string
	var inputErr error

	if name, inputErr = stringInput(c.UI, "Name"); inputErr != nil {
		c.errorf("input error: %s", inputErr)
		return nil, failure
	}

	if name == "" {
		c.printf("Name can't be empty")
		return nil, failure
	}

	id, err := c.DB.ParseID(c.UserID)
	if err != nil {
		c.errorf("error parsing user id: %s", err)
		return nil, failure
	}

	u, err := models.FindUser(c.DB, id)
	if err != nil {
		c.errorf("error finding user: %s", err)
		return nil, failure
	}

	h, err := habit.Create(c.DB, u, name)
	if err != nil {
		c.errorf("error creating habit: %s", err)
		return nil, failure
	}

	return h, success
}

func (c *HabitCommand) runCheckin(args []string) int {
	hbt, index := c.promptSelectHabit()
	if index < 0 {
		return failure
	}

	if _, err := habit.CheckinFor(c.DB, hbt, "", time.Now()); err != nil {
		c.errorf("while checking in: %s", err)
		return failure
	}

	return success
}

func (c *HabitCommand) runDelete(args []string) int {
	habit, index := c.promptSelectHabit()
	if index < 0 {
		return failure
	}

	if confirm, err := yesNo(c.UI, fmt.Sprintf("Are you sure you want to delete %s", habit.Name)); err != nil {
		c.errorf(err.Error())
	} else if !confirm {
		c.printf("Cancelled")
	}

	if err := c.DB.Delete(habit); err != nil {
		c.errorf("%s", err)
		return failure
	}

	c.removeHabit(index)
	c.printf("Deleted %s", habit.Name)

	return success
}

func (c *HabitCommand) runHistory(args []string) int {
	habit, index := c.promptSelectHabit()
	if index < 0 {
		return failure
	}

	checkins, err := habit.Checkins(c.DB)
	if err != nil {
		c.errorf("while retrieving checkins")
		return failure
	}

	if len(checkins) == 0 {
		c.printf("You have no history for this habit")
		return success
	}

	for _, event := range checkins {
		c.printf("Checkin on %s", event.Time.Format("Mon Jan 2 15:04"))

		if n, err := event.Note(c.DB); err != nil {
			c.errorf("error retrieving event's note: %s", err)
		} else if n.Text != "" {
			c.printf("\tNotes: %s", n.Text)
		}
	}

	return success
}

func (c *HabitCommand) runList(args []string) int {
	if len(c.habits) == 0 {
		c.printf("You have no habits")
		return success
	}

	c.printf("Here are your habits:")
	c.printHabitList()
	return success
}

func (c *HabitCommand) runNew(args []string) int {
	habit, out := c.promptNewHabit()
	if out != success {
		return out
	}

	c.printf("Created %s", habit.Name)
	return success
}

func (c *HabitCommand) runToday(args []string) int {
	c.printf("Here is today's lineup:")
	var complete string
	for _, h := range c.habits {
		if checkedIn, err := habit.DidCheckinOn(c.DB, h, time.Now()); err != nil {
			c.errorf("error checking if habit is complete: %s", err)
			return failure
		} else if checkedIn {
			complete = "✓"
		} else {
			complete = ""
		}

		c.printf("%s: %s", h.Name, complete)
	}
	return success
}
