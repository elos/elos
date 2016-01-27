package command

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

// CalCommand contains the state necessary to implement the
// 'elos cal' command set
type CalCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must be non-null
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified
	UserID string

	// DB is the elos database we interface with.
	data.DB

	// cal is the user's current elos calendar
	cal *models.Calendar
}

// Synopsis is a one-line, short summary of the 'cal' command.
// It is guaranteed to be at most 50 characters.
func (c *CalCommand) Synopsis() string {
	return "Utilities for managing the elos calendrical system"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possibly, a complete
// list of flags the 'cal' command accepts.
func (c *CalCommand) Help() string {
	helpText := `
Usage:
	elos cal <subcommand>

Subcommands:
	next	list the next fixture
	now		list the current fixture
	scheduling {base | weekday | yearday}	modify schedules
	today	list fixtures for today
`
	return strings.TrimSpace(helpText)
}

// Run runs the 'cal' command with the given command-line arguments.
// It returns an exit status when it finishes. 0 indicates a success,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface.
func (c *CalCommand) Run(args []string) int {
	// shortcircuit before hitting the network
	if len(args) == 0 {
		c.UI.Output(c.Help())
		return success
	}

	// initialize
	if i := c.init(); i != success {
		return i
	}

	switch args[0] {
	case "now":
	case "next":
	case "today":
		return c.runToday(args)
	case "scheduling":
		if len(args) == 1 {
			c.UI.Output("Usage: elos cal scheduling { base | weekday | yearday }")
			return success
		}
		switch args[1] {
		case "base":
			return c.runSchedulingBase(args)
		case "weekday":
			return c.runSchedulingWeekday(args)
		case "yearday":
			c.UI.Output("elos cal scheduling yearday not implemented yet")
		}
	}

	return success
}

func (c *CalCommand) runNow(args []string) int {
	return success
}

func (c *CalCommand) runNext(args []string) int {
	return success
}

func (c *CalCommand) runToday(args []string) int {
	fixtures, err := c.cal.FixturesForDate(time.Now(), c.DB)
	if err != nil {
		c.UI.Error(err.Error())
		return failure
	}

	printFixtures(c.UI, fixtures)
	return success
}

func (c *CalCommand) newSchedule(name string) *models.Schedule {
	base := models.NewSchedule()
	base.SetID(c.DB.NewID())
	base.CreatedAt = time.Now()
	base.Name = name
	base.EndTime = base.StartTime.Add(24 * time.Hour)
	log.Print(base.EndTime.Year())
	base.OwnerId = c.UserID
	base.UpdatedAt = time.Now()
	return base
}

func (c *CalCommand) runSchedulingBase(args []string) int {
	base, err := c.cal.BaseSchedule(c.DB)
	if err != nil {
		if err == models.ErrEmptyLink {
			c.UI.Output("It appears you don't have a base schedule, creating one for you now")
			base := c.newSchedule("Base Schedule")

			if err := c.DB.Save(base); err != nil {
				c.UI.Error(err.Error())
				return failure
			}

			c.cal.SetBaseSchedule(base)
			if err := c.DB.Save(c.cal); err != nil {
				c.UI.Error(err.Error())
				return failure
			}
		} else {
			c.UI.Error(err.Error())
			return failure
		}
	}

	fixtures, err := base.Fixtures(c.DB)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error retrieving the fixtures of your base schedule %s", err))
		return failure
	}

	c.UI.Output("Base Schedule Fixtures:")
	printFixtures(c.UI, fixtures)
	return success
}

func (c *CalCommand) runSchedulingWeekday(args []string) int {
	i := -1
	var err error
	for !models.ValidWeekday(i) {
		i, err = intInput(c.UI, "For which weekday?")

		if err != nil {
			c.UI.Error(fmt.Sprintf("Error with input: %s", err))
			return failure
		}
	}

	scheduleID, ok := c.cal.WeekdaySchedules[string(i)]
	schedule := models.NewSchedule()
	if ok {
		schedule.Id = scheduleID
		if err := c.DB.PopulateByID(schedule); err != nil {
			c.UI.Error(fmt.Sprintf("Error populating weekday schedule: %s", err))
			return 1
		}
	} else {
		c.UI.Output("Looks like you don't have a schedule for that day, creating one now...")
		weekday := c.newSchedule("Weekday Schedule")

		if err := c.DB.Save(weekday); err != nil {
			c.UI.Error(err.Error())
			return failure
		}

		c.cal.WeekdaySchedules[string(i)] = weekday.Id

		if err = c.DB.Save(c.cal); err != nil {
			c.UI.Error(err.Error())
			return failure
		}

		schedule = weekday
	}

	fixtures, err := schedule.Fixtures(c.DB)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error retrieving the fixtures of your weekday schedule %s", err))
		return 1
	}
	c.UI.Output(fmt.Sprintf("%s Schedule Fixtures:", time.Weekday(i)))
	printFixtures(c.UI, fixtures)

	b, err := yesNo(c.UI, "Would you like to add a fixture now?")
	if err != nil {
		return 1
	}

	if b {
		f, err := createFixture(c.UI, c.UserID, c.DB)
		if err != nil {
			return 1
		}

		schedule.IncludeFixture(f)
		err = c.DB.Save(schedule)
		if err != nil {
			return 1
		}
	}
	return success
}

type byStartTime []*models.Fixture

// Len is the number of elements in the collection.
func (b byStartTime) Len() int {
	return len(b)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (b byStartTime) Less(i, j int) bool {
	return b[i].StartTime.Before(b[j].StartTime)
}

// Swap swaps the elements with indexes i and j.
func (b byStartTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func printFixtures(ui cli.Ui, fixtures []*models.Fixture) {
	if len(fixtures) == 0 {
		ui.Output(" -- No fixtures")
		return
	}
	sort.Sort(byStartTime(fixtures))
	for _, f := range fixtures {
		var output string
		if f.Label {
			output = fmt.Sprintf("* %s [Label]", f.Name)
		} else {
			output = fmt.Sprintf(`
			 * %s [%s - %s]
			`, f.Name, f.StartTime.Format("15:04"), f.EndTime.Format("15:04"))
		}

		ui.Output(strings.TrimSpace(output))
	}
}

func createFixture(ui cli.Ui, ownerID string, db data.DB) (fixture *models.Fixture, err error) {
	ui.Output("Creating a fixture")

	fixture = models.NewFixture()
	fixture.SetID(db.NewID())
	fixture.OwnerId = ownerID
	fixture.CreatedAt = time.Now()

	if fixture.Name, err = stringInput(ui, "Name of the fixture:"); err != nil {
		return
	}

	if fixture.Label, err = boolInput(ui, "Is this a label?"); err != nil {
		return
	}

	if !fixture.Label {
		if fixture.StartTime, err = timeInput(ui, "Start time of fixture?"); err != nil {
			return
		}

		if fixture.EndTime, err = timeInput(ui, "End time of fixture?"); err != nil {
			return
		}
	}

	fixture.UpdatedAt = time.Now()

	err = db.Save(fixture)
	return
}

// errorf is a IO function which performs the equivalent of log.Errorf
// in the standard lib, except using the cli.Ui interface with which
// the CalCommand was provided.
func (c *CalCommand) errorf(s string, values ...interface{}) {
	c.UI.Error("[elos cal] Error: " + fmt.Sprintf(s, values...))
}

func (c *CalCommand) init() int {
	if c.UI == nil {
		return failure // we can't c.errorf because the user interface isn't defined
	}

	if c.DB == nil {
		c.errorf("no database")
		return failure
	}

	if c.UserID == "" {
		c.errorf("no user id")
		return failure
	}

	c.cal = models.NewCalendar()

	if err := c.DB.PopulateByField("owner_id", c.UserID, c.cal); err != nil {
		if err == data.ErrNotFound {
			createOneNow, err := yesNo(c.UI, "It appears you do not have a calendar, would you like to create one?")
			if err != nil {
				c.UI.Error(err.Error())
				return failure
			}

			if createOneNow {
				cal, err := newCalendar(c.DB, c.UserID)
				if err != nil {
					c.UI.Error(err.Error())
					return failure
				}
				c.cal = cal
			} else {
				c.UI.Output("Ok, you will have to create one eventually in order to use the 'elos cal' commands")
				return failure
			}
		} else {
			c.UI.Error(fmt.Sprintf("Error looking for calendar: %s", err))
			return failure
		}
	}

	if c.cal.WeekdaySchedules == nil {
		c.cal.WeekdaySchedules = make(map[string]string)
	}

	if c.cal.YeardaySchedules == nil {
		c.cal.YeardaySchedules = make(map[string]string)
	}

	return success
}

func newCalendar(db data.DB, userID string) (*models.Calendar, error) {
	cal := models.NewCalendar()

	cal.SetID(db.NewID())
	cal.CreatedAt = time.Now()
	cal.Name = "Main"
	cal.OwnerId = userID
	cal.UpdatedAt = time.Now()

	return cal, db.Save(cal)
}
