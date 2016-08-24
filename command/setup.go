package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/elos/x/models"
	"github.com/mitchellh/cli"
)

// SetupCommand contains the state necessary to implement the
// 'elos setup' command.
//
// It implements the cli.Command interface
type SetupCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must not be nil.
	UI cli.Ui

	// Config is the elos command configuration block, used and
	// modified by the setup command
	Config *Config
}

// Synopsis is a one-line, short summary of the 'setup' command.
// It is guaranteed to be at most 50 characters
func (c *SetupCommand) Synopsis() string {
	return "Utility to setup the elos command line interface"
}

func (c *SetupCommand) Help() string {
	return c.Synopsis()
}

// Run runs the 'setup' command with the given command-line arguments.
// It returns an exit status when it finishes. 0 indicates a success,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface
func (c *SetupCommand) Run(args []string) int {
	if c.UI == nil {
		log.Print("(elos setup): no ui")
		return failure
	}

	if c.Config.Host == "" {
		if i := c.promptNewHost(); i != success {
			return i
		}
	}

	if alreadyUser, err := yesNo(c.UI, "Do you already have an elos account?"); err != nil {
		c.errorf("input error: %s", err)
		return failure
	} else if alreadyUser {
		return c.setupCurrentUser()

	} else {
		return c.setupNewUser()
	}
	return success
}

// errorf calls UI.Error with a formatted, prefixed error string
// always use it to print an error, avoid using UI.Error directly
func (c *SetupCommand) errorf(format string, values ...interface{}) {
	c.UI.Error(fmt.Sprintf("(elos setup) Error: "+format, values...))
}

// printf calls UI.Output with the formmated string
// always prefer printf over c.UI.Output
func (c *SetupCommand) printf(format string, values ...interface{}) {
	c.UI.Output(fmt.Sprintf(format, values...))
}

func (c *SetupCommand) setConfig(username, password, id string) int {
	c.Config.UserID = id
	c.Config.PublicCredential = username
	c.Config.PrivateCredential = password
	if err := WriteConfigFile(c.Config); err != nil {
		c.errorf("failed to persist configuration change: %s", err)
		return failure
	}
	return success
}

func (c *SetupCommand) promptNewHost() int {
	host, err := stringInput(c.UI, "What host would you like to connect to?")
	if err != nil {
		c.errorf("input: %s", err)
		return failure
	}

	c.Config.Host = host
	if err := WriteConfigFile(c.Config); err != nil {
		c.errorf("failed to persist configuration change: %s", err)
		return failure
	}

	return success
}

func (c *SetupCommand) setupCurrentUser() int {
	var inputErr error
	var username, password, id string

	if username, inputErr = stringInput(c.UI, "Username"); inputErr != nil {
		c.errorf("input: %s", inputErr)
		return failure
	}
	if password, inputErr = stringInput(c.UI, "Password"); inputErr != nil {
		c.errorf("input: %s", inputErr)
		return failure
	}
	if id, inputErr = stringInput(c.UI, "ID"); inputErr != nil {
		c.errorf("input: %s", inputErr)
		return failure
	}

	if i := c.setConfig(username, password, id); i != success {
		return i
	}

	c.printf("We have configured your command line. Welcome back.")
	return success
}

func (c *SetupCommand) promptNewUser() (*models.User, string, string, int) {
	var inputError error
	var username, password string

	if username, inputError = stringInput(c.UI, "What username would you like to use?"); inputError != nil {
		c.errorf("input error: %s", inputError)
		return nil, "", "", failure
	}

	if password, inputError = stringInput(c.UI, "What password (use a fake one) would you like to use?"); inputError != nil {
		c.errorf("input error: %s", inputError)
		return nil, "", "", failure
	}

	if username == "" {
		c.errorf("username can't be empty")
		return nil, "", "", failure
	}

	if password == "" {
		c.errorf("password can't be empty")
		return nil, "", "", failure
	}

	params := url.Values{}
	params.Set("username", username)
	params.Set("password", password)
	url := c.Config.Host + "/register/?" + params.Encode()
	resp, err := http.Post(url, "", nil)

	if err != nil || resp.StatusCode != http.StatusCreated {
		if err != nil {
			c.errorf("error on POST to /register/: %s", err)
		} else {
			c.errorf("bade status code on POST to /register/: %d", resp.StatusCode)
		}
		return nil, "", "", failure
	}

	u := new(models.User)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.errorf("reading response body: %s", err)
		return nil, "", "", failure
	}

	if err := json.Unmarshal(body, u); err != nil {
		c.errorf("unmarshalling response into user: %s", err)
		return nil, "", "", failure
	}

	return u, username, password, success
}

func (c *SetupCommand) setupNewUser() int {
	c.printf("Welcome to elos!")
	c.printf("Later there may be more details to setting up a user account, but for now it is quite straight forward")
	u, username, password, i := c.promptNewUser()
	if i != success {
		return i
	}

	if i := c.setConfig(username, password, u.Id); i != success {
		return i
	}

	c.printf("We have created you an account. Welcome home.")
	return success
}
