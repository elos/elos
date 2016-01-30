package command_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/elos/command"
	"github.com/elos/gaia"
	"github.com/elos/gaia/services"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"github.com/mitchellh/cli"
	"golang.org/x/net/context"
)

func newMockSetupCommand(t *testing.T) (*cli.MockUi, *command.Config, *command.SetupCommand) {
	ui := new(cli.MockUi)
	c := &command.Config{}

	return ui, c, &command.SetupCommand{
		UI:     ui,
		Config: c,
	}
}

// --- 'elos setup'  (context: already have an account) {{{
func TestSetupCurrentUser(t *testing.T) {
	f, err := ioutil.TempFile("", "conf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ui, conf, c := newMockSetupCommand(t)
	conf.Path = f.Name()
	conf.Host = "fake" // not needed here because doesn't hit api

	db := mem.NewDB()

	t.Log("Creating test user")
	u, _, err := user.Create(db, "public", "private")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	// yes already account, then public key and private key and user id
	ui.InputReader = bytes.NewBufferString(fmt.Sprintf("y\npublic\nprivate\n%s\n", u.Id))

	t.Log("running: `elos setup`")
	code := c.Run([]string{})
	t.Log("command `setup` terminated")

	t.Log("Reading outputs")
	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != 0 {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "account") {
		t.Fatalf("Output should have contained a 'account' for saying something about an account")
	}

	t.Log("Configuration:\n%+v", conf)

	// verify conf was changed
	if conf.UserID != u.Id {
		t.Fatalf("User id should be: %s", u.Id)
	}

	if conf.PublicCredential != "public" {
		t.Fatalf("public credential should be: public")
	}

	if conf.PrivateCredential != "private" {
		t.Fatalf("private credential should be: private")
	}
}

// --- }}}

// --- 'elos setup'  (context: need a new account) {{{
func TestSetupNewUser(t *testing.T) {

	f, err := ioutil.TempFile("", "conf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	db := mem.NewDB()

	g := gaia.New(
		context.Background(),
		&gaia.Middleware{},
		&gaia.Services{
			SMSCommandSessions: services.NewSMSMux(),
			Logger:             services.NewTestLogger(t),
			DB:                 db,
		},
	)

	s := httptest.NewServer(g)
	defer s.Close()

	ui, conf, c := newMockSetupCommand(t)
	conf.Path = f.Name()

	// no already account, then username input and password input
	ui.InputReader = bytes.NewBufferString(fmt.Sprintf("%s\nn\npublic\nprivate\n", s.URL))

	t.Log("running: `elos setup`")
	code := c.Run([]string{})
	t.Log("command `setup` terminated")

	t.Log("Reading outputs")
	errput := ui.ErrorWriter.String()
	output := ui.OutputWriter.String()
	t.Logf("Error output:\n%s", errput)
	t.Logf("Output:\n%s", output)

	// verify there were no errors
	if errput != "" {
		t.Fatalf("Expected no error output, got: %s", errput)
	}

	// verify success
	if code != 0 {
		t.Fatalf("Expected successful exit code along with empty error output.")
	}

	// verify some of the output
	if !strings.Contains(output, "account") {
		t.Fatalf("Output should have contained a 'account' for saying something about an account")
	}

	cred, err := access.Authenticate(db, "public", "private")
	if err != nil {
		t.Fatal(err)
	}

	u, err := cred.Owner(db)
	if err != nil {
		t.Fatal(err)
	}

	// verify conf was changed
	if conf.UserID != u.Id {
		t.Fatalf("User id should be: %s", u.Id)
	}

	if conf.PublicCredential != "public" {
		t.Fatalf("public credential should be: public")
	}

	if conf.PrivateCredential != "private" {
		t.Fatalf("private credential should be: private")
	}
}

// --- }}}
