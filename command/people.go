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

// PeopleCommand contains the state necessary to implement the
// 'elos people' command set.
//
// It implements the cli.Command interface
type PeopleCommand struct {
	// UI is used to communicate (fo IO) with the user
	// It must not be nil.
	UI cli.Ui

	// UserID is the id of theuser we are acting on behalf of.
	// It must be specified.
	UserID string

	// DB is the elos database we interface with.
	// It must not be nil.
	data.DB

	// people is the list of this user's persons.
	people []*models.Person
}

// Synopsis is a one-line, short summary of the 'people' command.
// It is guaranteed to be at most 50 characters
func (c *PeopleCommand) Synopsis() string {
	return "Utilities for managing notes on people"
}

func (c *PeopleCommand) Help() string {
	helpText := `
Usage:
	elos people <subcommand>

Subcommands:
	list	list all of the people
	delete	delete a person
	new	create a new person
	note	add a note to a person
	stream	stream notes for a person
`
	return strings.TrimSpace(helpText)
}

// Runs runs the 'people' command with the given command-line arguments.
// It returns an exit status when it finishes. 0 indicates a success,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface
func (c *PeopleCommand) Run(args []string) int {
	// short circuit to avoid loading people
	if len(args) == 0 {
		c.UI.Output(c.Help())
		return success
	}

	// fully initialize the command, and bail if not a success
	if i := c.init(); i != success {
		return i
	}

	switch args[0] {
	case "list":
		c.runList(args)
	case "delete":
		c.runDelete(args)
	case "new":
		c.runNew(args)
	case "note":
		c.runNote(args)
	case "stream":
		c.runStream(args)
	default:
		c.UI.Output(c.Help())
	}
	return success
}

// removePerson removes the person at the given index.
// You may use this for removing a person after they have
// been deleted
func (c *PeopleCommand) removePerson(index int) {
	c.people = append(c.people[index:], c.people[index+1:]...)
}

// errorf calls UI.Error with a formatted, prefixed error string
// always use it to print an error, avoid using UI.Error directly
func (c *PeopleCommand) errorf(format string, values ...interface{}) {
	c.UI.Error(fmt.Sprintf("(elos people) Error: "+format, values...))
}

// printf calls UI.Output with the formmated string
// always prefer printf over c.UI.Output
func (c *PeopleCommand) printf(format string, values ...interface{}) {
	c.UI.Output(fmt.Sprintf(format, values...))
}

// init performs the necessary initliazation for the *PeopleCommand
// Notable, it ensures we have a UI, DB and UserID, so those can be treated
// as invariants throughought the rest of the code.
//
// Additionally it loads this user's people list.
func (c *PeopleCommand) init() int {
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

	q := c.DB.Query(models.PersonKind)
	q.Select(data.AttrMap{
		"owner_id": c.UserID,
	})
	iter, err := q.Execute()
	if err != nil {
		c.errorf("while querying for people: %s", err)
		return failure
	}

	c.people = make([]*models.Person, 0)
	person := models.NewPerson()
	for iter.Next(person) {
		c.people = append(c.people, person)
		person = models.NewPerson()
	}

	if err := iter.Close(); err != nil {
		c.errorf("while querying for people: %s", err)
		return failure
	}

	return success
}

// printPeopleList prints a numbered list of the people in the people slice
func (c *PeopleCommand) printPeopleList() {
	for i, p := range c.people {
		c.printf("%d) %s %s", i, p.FirstName, p.LastName)
	}
}

// promptSelectPerson prompts the user to select a person from their list
// of people.
//
// Use promptNewPerson for any subcommand which acts on a person.
// Retrieve the person:
//		person, index := promptSelectPerson()
//		if index < 0 {
//			return failure
//		}
//
// As you can see in the example above, a negative index indicates failure.
//
// NOTE: the integer here is not a status code, but rather the index of the person
// in the command's people slice
func (c *PeopleCommand) promptSelectPerson() (*models.Person, int) {
	if len(c.people) == 0 {
		c.UI.Warn("You do not have any people")
		return nil, -1
	}

	c.printPeopleList()

	var (
		indexOfCurrent int
		err            error
	)

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil, -1
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.people)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.people)-1))
		return nil, -1 // to indicate the parent command to exit
	}

	return c.people[indexOfCurrent], indexOfCurrent
}

// promptNewPerson provides the input prompts necessary to construct a new person.
//
// Use this to implement the 'new' subcommand, and for any subcommand which requires
// creating a new user as part of it's functionality.
//
// It returns a person and status code. A 0 status code indicates success, any other
// status indicates failure, and the caller should exit immediately. promptNewPerson
// will have taken care of printing the error output.
func (c *PeopleCommand) promptNewPerson() (*models.Person, int) {
	p := models.NewPerson()
	p.SetID(c.DB.NewID())
	p.CreatedAt = time.Now()

	var inputErr error

	if p.FirstName, inputErr = stringInput(c.UI, "First Name:"); inputErr != nil {
		c.errorf("input error: %s", inputErr)
		return nil, failure
	}

	if p.LastName, inputErr = stringInput(c.UI, "Last Name:"); inputErr != nil {
		c.errorf("input error: %s", inputErr)
		return nil, failure
	}

	p.OwnerId = c.UserID
	p.UpdatedAt = time.Now()

	if err := c.DB.Save(p); err != nil {
		c.errorf("error saving person: %s", err)
		return nil, failure
	}

	return p, success
}

// promptNewNote prompts the user for the information necessary to create a new
// note about the given user.
//
// Use this to implement the 'note' subcommand, or if a subcommand ever needs
// to prompt the user to add a note to a person as a part of it's functionality.
//
// It returns the new note, which has been saved, and a status code. As always,
// 0 indicates success, whereas any other integer indicates failure. When
// a failure has occured the first return argument will be nil. The promptNewNote
// function handles error message outputting itself.
func (c *PeopleCommand) promptNewNote(p *models.Person) (*models.Note, int) {
	n := models.NewNote()
	n.SetID(c.DB.NewID())
	n.CreatedAt = time.Now()

	var inputErr error

	if n.Text, inputErr = stringInput(c.UI, "Content"); inputErr != nil {
		c.errorf("input err: %s", inputErr)
		return nil, failure
	}

	n.OwnerId = c.UserID
	n.UpdatedAt = time.Now()

	if err := c.DB.Save(n); err != nil {
		c.errorf("error saving note: %s", err)
		return nil, failure
	}

	p.IncludeNote(n)

	if err := c.DB.Save(p); err != nil {
		c.errorf("error saving person: %s", err)
		return nil, failure
	}

	return n, success
}

// runDelete runs the 'delete' subcommand with the given arguments.
//
// The 'delete' subcommands prompts the user for a person to delete.
func (c *PeopleCommand) runDelete(args []string) int {
	person, index := c.promptSelectPerson()
	if index < 0 {
		return failure
	}

	if confirm, err := yesNo(c.UI, fmt.Sprintf("Are you sure you want to delete %s %s", person.FirstName, person.LastName)); err != nil {
		c.errorf(err.Error())
	} else if !confirm {
		c.printf("Cancelled")
	}

	if err := c.DB.Delete(person); err != nil {
		c.errorf("%s", err)
		return failure
	}

	c.removePerson(index)
	c.printf("Deleted %s %s", person.FirstName, person.LastName)

	return success
}

// runList runs the 'list' subcommand with the given arguments.
//
// The 'list' subcommand lists all the user's people.
func (c *PeopleCommand) runList(args []string) int {
	if len(c.people) == 0 {
		c.printf("You have no people")
		return success
	}

	c.printf("Here are the people you have notes on:")
	c.printPeopleList()
	return success
}

// runNew runs the 'new' subcommand with the given arguments.
//
// The 'new' subcommand provides prompts to create a new person.
func (c *PeopleCommand) runNew(args []string) int {
	person, out := c.promptNewPerson()
	if out != success {
		return out
	}

	c.printf("Created %s %s", person.FirstName, person.LastName)
	return success
}

// runNotes runs the 'note' subcommand with the given arguments.
//
// The 'note' subcommand allows you to add notes to a person.
func (c *PeopleCommand) runNote(args []string) int {
	person, index := c.promptSelectPerson()
	if index < 0 {
		return failure
	}

Adding:
	for {
		if _, out := c.promptNewNote(person); out != success {
			return out
		}

		c.printf("noted")

		if another, err := yesNo(c.UI, "Would you like to add another note?"); err != nil {
			c.errorf("input error: %s", err)
			return failure
		} else if !another {
			break Adding
		}
	}

	return success
}

// runStream runs the stream command with the given arguments.
//
// The stream command loads all the note on a particular user,
// and allows you to scroll through them by pressing enter.
//
// Interestingly, the current design is that oldest notes come first
// but perhaps this may change.
func (c *PeopleCommand) runStream(args []string) int {
	person, index := c.promptSelectPerson()
	if index < 0 {
		return failure
	}

	// get the notes
	notes, err := person.Notes(c.DB)
	if err != nil {
		c.errorf("error retrieving the notes: %s", err)
		return failure
	}

	// sort the notes
	sort.Sort(byCreatedAt(notes))

	c.printf("press enter to scroll through")
	for i, n := range notes {
		c.printf("%d) %s", i, n.Text)

		if i != len(notes)-1 {
			// block and wait for enter until the next one
			if _, err := c.UI.Ask(""); err != nil {
				c.errorf("input error: %s", err)
				return failure
			}
		}
	}

	return success
}

// byCreated is a type which satisfies the sort.Interface
// and sorts note by the date they were created at,
// oldest first
type byCreatedAt []*models.Note

func (b byCreatedAt) Len() int {
	return len(b)
}

func (b byCreatedAt) Less(i, j int) bool {
	return b[i].CreatedAt.Before(b[j].CreatedAt)
}

func (b byCreatedAt) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
