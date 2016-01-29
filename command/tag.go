package command

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/tag"
	"github.com/mitchellh/cli"
)

type TagCommand struct {
	// UI is used to communicate (for IO) with the user
	// It must be non-nil
	UI cli.Ui

	// UserID is the id of the user we are acting on behalf of.
	// It must be specified
	UserID string

	// DB is the elos database we interface with
	// It must be non-nil
	data.DB

	// The tags of the user given by c.UserID
	//
	// During the lifecycle of the command, and assuming
	// the user is only accessing the elos system through
	// the command prompt, the tag list is complete and
	// definitive (reflects exactly what is in the database).
	tags []*models.Tag
}

// Synopsis is a one-line, short summary of the 'tag' command.
// It is guaranteed to be at most 50 characters.
func (c *TagCommand) Synopsis() string {
	return "Utilities for managing elos tags"
}

// Help is the long-form help text that includes command-line
// usage. It includes the subcommands and, possibly a complete
// list of flags the 'tag' command accepts.
func (c *TagCommand) Help() string {
	helpText := `
Usage:
	elos tag <subcommand>

Subcommands:
	delete		delete a tag
	edit		edit a tag
	list		list all your tags
	new		create a new tag
`
	return strings.TrimSpace(helpText)
}

// Run runs the 'tag' command with the given command-line arguments.
// It returns an exit status when it finishes. 0 indicates a sucess,
// any other integer indicates a failure.
//
// All user interaction is handled by the command using the UI
// interface.
func (c *TagCommand) Run(args []string) int {
	// short circuit to avoid loading tags
	if len(args) == 0 && c.UI != nil {
		c.UI.Output(c.Help())
		return success
	}

	// fully initialize the command, and bail if not a success
	if i := c.init(); i != success {
		return i
	}

	switch len(args) {
	case 1:
		switch args[0] {
		case "e":
		case "edit":
			return c.runEdit(args)
		case "d":
		case "delete":
			return c.runDelete(args)
		case "l":
		case "list":
			return c.runList(args)
		case "n":
		case "new":
			return c.runNew(args)
		default:
			c.UI.Output(c.Help())
		}
	default:
		c.UI.Output(c.Help())
	}

	return success
}

// init performs some verification that the TagCommand object
// is valid (has a non-null database & UI and a user id).
//
// It loads all of the UserID's tags into the tags field of the
// TodoCommand object.
//
// It loads all of the UserID's tags into the tags field of the
// TodoCommand object.
//
// A 0 return value indicates success, a 1 indiciates failure. The
// init command handles appropriate error printing the the UI.
func (c *TagCommand) init() int {
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

	// Load the tags

	iter, err := c.DB.Query(models.TagKind).Select(data.AttrMap{
		"owner_id": c.UserID,
	}).Execute()

	if err != nil {
		c.errorf("data retrieval: querying tags")
		return failure
	}

	t := models.NewTag()
	tags := make([]*models.Tag, 0)
	for iter.Next(t) {
		tags = append(tags, t)
		t = models.NewTag()
	}

	if err := iter.Close(); err != nil {
		c.errorf("data retrieval: querying tags")
		return failure
	}

	c.tags = tags

	sort.Sort(tag.ByName(c.tags))

	return success
}

// errorf is a IO function which performs the equivalent of log.Errorf
// in the standard lib, except using the cli.Ui interface with which
// the Tag was provided.
func (c *TagCommand) errorf(s string, values ...interface{}) {
	c.UI.Error("[elos tag] Error: " + fmt.Sprintf(s, values...))
}

func (c *TagCommand) runEdit(args []string) int {
	tg, index := c.promptSelectTag()
	if index < 0 {
		return failure
	}

	bytes, err := json.MarshalIndent(tg, "", "	")
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
	case "name":
		tg.Name, err = stringInput(c.UI, "Name")
	default:
		c.UI.Warn("That attribute is not recognized/supported")
		return success
	}

	if err != nil {
		c.errorf("(subcommand edit) Input Error %s", err)
		return failure
	}

	if err = c.DB.Save(tg); err != nil {
		c.errorf("(subcommand edit) Error: %s", err)
		return failure
	}

	c.UI.Output("Tag updated")

	return success

}

// runDelete runs the 'delete' subcommand.
//
// It returns an exit status:
// 0 := success
// 1 := failure
func (c *TagCommand) runDelete(args []string) int {
	tg, index := c.promptSelectTag()
	if index < 0 {
		return failure
	}

	if confirm, err := yesNo(c.UI, "Are you sure?"); err != nil {
		c.errorf("Input Error: %s", err)
		return failure
	} else if !confirm {
		c.UI.Info("Cancelled")
		return success
	}

	if err := c.DB.Delete(tg); err != nil {
		c.errorf("(subcommand delete) Error: %s", err)
		return failure
	}

	c.UI.Info(fmt.Sprintf("Deleted '%s'", tg.Name))
	return success
}

// runDelete runs the 'delete' subcommand.
//
// It returns an exit status, always success
func (c *TagCommand) runList(args []string) int {
	if len(c.tags) == 0 {
		c.UI.Output("You don't have any tags")
	} else {
		c.printTagList()
	}
	return success
}

func (c *TagCommand) runNew(args []string) int {
	t := models.NewTag()
	t.SetID(c.DB.NewID())
	t.OwnerId = c.UserID
	var err error
	t.Name, err = stringInput(c.UI, "Name")
	if err != nil {
		c.errorf("Input Error: %s", err)
		return failure
	}

	if err := c.DB.Save(t); err != nil {
		c.errorf("Error saving tag: %s", err)
		return failure
	}

	return success
}

// printTagList prints the list of tags, with deadline and salience info
// the list is numbered, and can be useful for tags that involve the user
// looking at / selecting a particular tag (however use promptSelectTag
// for the case of selecting a single tag from the c.tags)
func (c *TagCommand) printTagList(selectors ...func(*models.Tag) bool) {
PrintLoop:
	for i, t := range c.tags {
		for i := range selectors {
			if !selectors[i](t) {
				continue PrintLoop
			}
		}

		c.UI.Output(fmt.Sprintf("%d) %s", i, t.Name))
	}
}

// promptSelectTag prompts the user to select one of their tags. The
// first return argument is the tag the user selected, and the second is
// the index of that tag. If the index is negative, then there was either an
// error retrieving a tag selection from the user, or the user has no tags,
// in either case the value of the first return argument is undefined.
//
// Use promptSelectTag for todo subcommands which operate on a tag.
func (c *TagCommand) promptSelectTag() (*models.Tag, int) {
	if len(c.tags) == 0 {
		c.UI.Warn("You do not have any tags")
		return nil, -1
	}

	c.printTagList()

	var (
		indexOfCurrent int
		err            error
	)

	if indexOfCurrent, err = intInput(c.UI, "Which number?"); err != nil {
		c.errorf("input error: %s", err)
		return nil, -1
	}

	if indexOfCurrent < 0 || indexOfCurrent > len(c.tags)-1 {
		c.UI.Warn(fmt.Sprintf("%d is not a valid index. Need a # in (0,...,%d)", indexOfCurrent, len(c.tags)-1))
		return nil, -1 // to indicate the parent command to exit
	}

	return c.tags[indexOfCurrent], indexOfCurrent
}
