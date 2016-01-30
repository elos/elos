package command

import (
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
)

// yesNo requests confirmation of something
//
// Use this for deciding what to do, like whether to request
// additional information from a user or whether the user
// intended to do something.
func yesNo(ui cli.Ui, text string) (bool, error) {
	i, err := ui.Ask(text + " [y to confirm]")
	return (i == "y"), err
}

// stringInput requests textual input
//
// Use this where you need to take a string value, or where
// you would want to parse a string input yourself
func stringInput(ui cli.Ui, text string) (string, error) {
	return ui.Ask(text + " [string]:")
}

// stringListInput requests a list of a comma delimitted strings
//
// Use this where you need to take a list of string values,
// such as the case where you want to take a list of strings
//
// It parses the strings based on commas, or double commas,
// if the string input needs to include commas
func stringListInput(ui cli.Ui, text string) ([]string, error) {
	in, err := ui.Ask(text + " [list,of,strings]")
	if err != nil {
		return nil, err
	}

	// if the user used double commas, single commas will be ignored
	if strings.Contains(in, ",,") {
		return strings.Split(in, ",,"), nil
	}

	return strings.Split(in, ","), nil
}

// boolInput requests a boolean input
//
// Use this where you need to take a boolean value. If you are
// looking for a confirmation prompt, however, use 'yesNo'
func boolInput(ui cli.Ui, text string) (bool, error) {
	for {
		input, err := ui.Ask(text + " [boolean]:")
		if err != nil {
			return false, err
		}

		switch input {
		case "yes":
			return true, err
		case "no":
			return false, err
		default:
			b, err := strconv.ParseBool(input)
			if err == nil {
				return b, nil
			}

			out := " Invalid input, please try again. Valid boolean expressions include: true, false, 0, 1 etc."
			ui.Output(strings.TrimSpace(out))
		}
	}
}

// intInput requests an integer input (signed)
//
// Use intInput if you need to retrieve an integer.
func intInput(ui cli.Ui, text string) (int, error) {
	for {
		input, err := ui.Ask(text + " [integer]:")
		if err != nil {
			return 0, err
		}

		i64, err := strconv.ParseInt(input, 10, 64)
		if err == nil {
			return int(i64), nil
		}

		out := "Invalid input, please try again. Valid integer expressions include: 1, 12, -300 etc."
		ui.Output(strings.TrimSpace(out))
	}
}

// timeInput retrieves a time.Time value, but only pays attention
// to the hour and the minute components. It fills in the year 0,
// month 0, day 0, second 0 and nsecond 0. It uses time.Local for
// location information.
//
// Use timeInput if you need to retrieve a time of the form 12:45,
// but if you care also abou the calendrical components, such as
// the year, month and day, use 'dateInput'.
func timeInput(ui cli.Ui, text string) (t time.Time, err error) {

	ui.Output(text + " [time]")
	if useNow, err := yesNo(ui, "Would you like to use the current time?"); err != nil {
		return *new(time.Time), err
	} else if useNow {
		now := time.Now()
		return time.Date(0, 0, 0, now.Hour(), now.Minute(), 0, 0, time.Local), nil
	}

	var (
		inputErr  error
		hour, min int
	)

	if hour, inputErr = intInput(ui, "Hour [e.g., 13]"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	if min, inputErr = intInput(ui, "Minute [e.g., 59]"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	return time.Date(0, 0, 0, hour, min, 0, 0, time.Local), nil
}

// dateInput retrieves a time.Time value as textual input over a
// a series of messages
//
// Use dateInput when you need a full date and time, i.e., 1/1/1 12:00
// If you only need a time, use 'timeInput'.
func dateInput(ui cli.Ui, text string) (time.Time, error) {
	ui.Output(text + " [date]")
	if useNow, err := yesNo(ui, "Would you like to use the current time?"); err != nil {
		return *new(time.Time), err
	} else if useNow {
		return time.Now(), nil
	}

	var (
		inputErr                    error
		year, month, day, hour, min int
	)

	if year, inputErr = intInput(ui, "Year (e.g., 2016)"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	if month, inputErr = intInput(ui, "Month (e.g., 1 for January)"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	if day, inputErr = intInput(ui, "Day [e.g., 1]"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	if hour, inputErr = intInput(ui, "Hour [e.g., 13]"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	if min, inputErr = intInput(ui, "Minute [e.g., 59]"); inputErr != nil {
		return *new(time.Time), inputErr
	}

	return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.Local), nil
}
