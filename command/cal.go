package command

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

type CalCommand struct {
	UI cli.Ui
	*Config
	data.DB
}

func (c *CalCommand) Help() string {
	helpText := `
Subcommands:
	* today
	* scheduling
		* base
		* weekday
		* yearday
`
	return strings.TrimSpace(helpText)
}

func (c *CalCommand) Run(args []string) int {
	if c.Config.UserID == "" {
		c.UI.Error("No user id")
		return 1
	}

	if c.DB == nil {
		c.UI.Error("No database")
		return 1
	}

	cal := models.NewCalendar()
	if err := c.DB.PopulateByField("owner_id", c.Config.UserID, cal); err != nil {
		if err == data.ErrNotFound {
			b, err := yesNo(c.UI, "It appears you do not have a calendar, would you like to create one?")
			if err != nil {
				c.UI.Error(err.Error())
				return 1
			}

			if b {
				cal = models.NewCalendar()
				cal.SetID(c.DB.NewID())
				cal.CreatedAt = time.Now()
				cal.Name = "Main"
				cal.OwnerId = c.Config.UserID
				cal.UpdatedAt = time.Now()

				if err := c.DB.Save(cal); err != nil {
					c.UI.Error(fmt.Sprintf("Error creating calendar: %s", err))
					return 1
				}
			}
		} else {
			c.UI.Error(fmt.Sprintf("Error looking for calendar: %s", err))
			return 1
		}
	}

	switch len(args) {
	case 0:
		c.UI.Output(fmt.Sprintf("Today is %s", time.Now()))
		return 0
	case 1:
		switch args[0] {
		case "today":

			schedules := cal.SchedulesForDate(time.Now(), c.DB)
			fixtures, err := models.MergedFixtures(c.DB, schedules...)
			if err != nil {
				c.UI.Error(err.Error())
				return 1
			}
			fixtures = models.RelevantFixtures(time.Now(), fixtures)
		case "scheduling":
			c.UI.Output("Usage: elos cal scheduling { base | weekday | yearday }")
		}
	case 2:

		switch args[0] {
		case "scheduling":
			switch args[1] {
			case "base":
				base, err := cal.BaseScheduleOrCreate(c.DB)
				if err != nil {
					c.UI.Error(fmt.Sprintf("Error retrieving base schedule: %s", err))
					return 1
				}

				fixtures, err := base.Fixtures(c.DB)
				if err != nil {
					c.UI.Error(fmt.Sprintf("Error retrieving the fixtures of your base schedule %s", err))
					return 1
				}
				c.UI.Output("Base Schedule Fixtures:")
				printFixtures(c.UI, fixtures)
			case "weekday":
				i, err := intInput(c.UI, "For which weekday?")
				if err != nil {
					c.UI.Error(fmt.Sprintf("Error with input: %s", err))
					return 1
				}

				scheduleID, ok := cal.WeekdaySchedules[string(i)]
				schedule := models.NewSchedule()
				if ok {
					schedule.Id = scheduleID
					if err := c.DB.PopulateByID(schedule); err != nil {
						c.UI.Error(fmt.Sprintf("Error populating weekday schedule: %s", err))
						return 1
					}
				} else {
					b, err := yesNo(c.UI, "Looks like you don't have a schedule for that day, would you like to create one?")
					if err != nil {
						c.UI.Error(fmt.Sprintf("Error asking question: %s", err))
						return 1
					}

					if !b {
						break
					}

					schedule.SetID(c.DB.NewID())
					schedule.CreatedAt = time.Now()
					schedule.OwnerId = cal.OwnerId
					schedule.UpdatedAt = time.Now()

					err = c.DB.Save(schedule)
					if err != nil {
						c.UI.Error(err.Error())
						return 1
					}

					cal.WeekdaySchedules[string(i)] = schedule.Id

					err = c.DB.Save(cal)
					if err != nil {
						c.UI.Error(err.Error())
						return 1
					}
				}
				fixtures, err := schedule.Fixtures(c.DB)
				if err != nil {
					c.UI.Error(fmt.Sprintf("Error retrieving the fixtures of your base schedule %s", err))
					return 1
				}
				c.UI.Output(fmt.Sprintf("%s Schedule Fixtures:", time.Weekday(i)))
				printFixtures(c.UI, fixtures)

				b, err := yesNo(c.UI, "Would you like to add a fixture now?")
				if err != nil {
					return 1
				}

				if b {
					f, err := createFixture(c.UI, c.Config.UserID, c.DB)
					if err != nil {
						return 1
					}

					schedule.IncludeFixture(f)
					err = c.DB.Save(schedule)
					if err != nil {
						return 1
					}
				}
			case "yearday":
			default:
				c.UI.Output("Usage: elos cal scheduling { base | weekday | yearday }")
			}
		}
	}

	return 0
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

func (c *CalCommand) Synopsis() string {
	return "Calendar utilities"
}
