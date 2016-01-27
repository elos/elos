package command

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
)

func NewSession(user *models.User, db data.DB, input <-chan string, output chan<- string, bail func()) *Session {
	return &Session{
		user:   user,
		db:     db,
		input:  input,
		Output: output,
		bail:   bail,
	}
}

type Session struct {
	// the user interacting with the session
	user *models.User

	// the db to user to execute the commands
	db data.DB

	// channel to read string input on
	input <-chan string

	// channel to send string output on
	Output chan<- string

	// function to call indicating failure, or exit
	// for example, on timeout, on errors
	bail func()
}

func (s *Session) Start() {
	if s.user == nil {
		s.Output <- "Looks like you don't have an account, sorry :("
		s.bail()
	}

	for i := range s.input {
		// we block so that the text ui can read in our absence
		s.run(strings.Split(i, " "))
	}
}

func (s *Session) run(args []string) {
	// construct a new CLI with name and version
	c := cli.NewCLI("elos", "0.1")
	c.Args = args
	ui := NewTextUI(s.input, s.Output)
	c.Commands = map[string]cli.CommandFactory{
		"todo": func() (cli.Command, error) {
			return &TodoCommand{
				UI:     ui,
				UserID: s.user.Id,
				DB:     s.db,
			}, nil
		},
	}

	_, err := c.Run()
	if err != nil {
		log.Printf("command session error: %s", err)
	}
}

// A TextUI is used for making command line interfaces
// more suitable for a medium in which you can only ccommunicate
// strings, i.e., text messaging
type TextUI struct {
	in  <-chan string
	out chan<- string
}

// Constructs a new text ui
func NewTextUI(in <-chan string, out chan<- string) *TextUI {
	return &TextUI{
		in:  in,
		out: out,
	}
}

// send is abstraction for sending out
func (u *TextUI) send(txt string) {
	u.out <- txt
}

// Ask asks the user for input using the given query. The response is
// returned as the given string, or an error.
func (u *TextUI) Ask(s string) (string, error) {
	u.send(s)
	select {
	case msg := <-u.in:
		return msg, nil
	case <-time.After(5 * time.Minute):
		u.out <- "timeout"
		return "", fmt.Errorf("TextUI Ask, timeout")
	}
}

// AskSecret asks the user for input using the given query, but does not echo
// the keystrokes to the terminal.
func (u *TextUI) AskSecret(s string) (string, error) {
	return u.Ask(s)
}

// Output is called for normal standard output.
func (u *TextUI) Output(s string) {
	u.send(s)
}

// Info is called for information related to the previous output.
// In general this may be the exact same as Output, but this gives
// UI implementors some flexibility with output formats.
func (u *TextUI) Info(s string) {
	u.send(s)
}

func (u *TextUI) Error(s string) {
	u.send(s)
}

func (u *TextUI) Warn(s string) {
	u.send(s)
}
