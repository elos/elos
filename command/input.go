package command

import (
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
)

func yesNo(ui cli.Ui, text string) (b bool, err error) {
	var i string
	i, err = ui.Ask(text + " [y to confirm]:")
	if err == nil {
		b = (i == "y")
	}
	return
}

func stringInput(ui cli.Ui, text string) (string, error) {
	return ui.Ask(text + " [string]:")
}

func boolInput(ui cli.Ui, text string) (b bool, err error) {
	var input string

	for {
		input, err = ui.Ask(text + " [boolean]:")
		if err != nil {
			return
		}

		if input == "yes" {
			b = true
			return
		}

		if input == "no" {
			b = false
			return
		}

		b, err = strconv.ParseBool(input)

		if err != nil {
			out := `
Invalid input, please try again.

Valid boolean expressions include: true, false, 0, 1 etc.
		`
			ui.Output(strings.TrimSpace(out))
		} else {
			return
		}
	}
}

func intInput(ui cli.Ui, text string) (i int, err error) {
	var input string

	for {
		input, err = ui.Ask(text + " [integer]:")
		if err != nil {
			return
		}

		var i64 int64
		i64, err = strconv.ParseInt(input, 10, 64)

		if err != nil {
			out := `
Invalid input, please try again.

Valid integer expressions include: 1, 12, -300 etc.
		`
			ui.Output(strings.TrimSpace(out))

		} else {
			i = int(i64)
			return
		}
	}
}

func timeInput(ui cli.Ui, text string) (t time.Time, err error) {
	var hour, min int
	var useNow bool

	ui.Output(text)
	useNow, err = yesNo(ui, "You must input a time, would you like to use the current time?")
	if err != nil {
		return
	}

	if useNow {
		t = time.Now()
		return
	}

	hour, err = intInput(ui, "Hour [e.g., 13]")
	if err != nil {
		return
	}
	min, err = intInput(ui, "Minute [e.g., 59]")
	if err != nil {
		return
	}

	t, err = time.Date(0, 0, 0, hour, min, 0, 0, time.Local), nil
	return
}

func dateInput(ui cli.Ui, text string) (t time.Time, err error) {
	var year, month, day, hour, min int
	var useNow bool

	ui.Output(text)
	ui.Output("You must input a datetime...")
	useNow, err = yesNo(ui, "Would you like to use the current time?")
	if err != nil {
		return
	}

	if useNow {
		t = time.Now()
		return
	}

	year, err = intInput(ui, "Year [e.g., 2015]:")
	if err != nil {
		return
	}
	month, err = intInput(ui, "Month [e.g., 1 for January")
	if err != nil {
		return
	}
	day, err = intInput(ui, "Day [e.g., 1]")
	if err != nil {
		return
	}
	hour, err = intInput(ui, "Hour [e.g., 13]")
	if err != nil {
		return
	}
	min, err = intInput(ui, "Minute [e.g., 59]")
	if err != nil {
		return
	}

	t, err = time.Date(year, time.Month(month), day, hour, min, 0, 0, time.Local), nil
	return
}
