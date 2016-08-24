package command_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/elos/command"
	"github.com/elos/gaia"
	"github.com/elos/gaia/services"
	"github.com/elos/x/auth"
	"github.com/elos/x/data/access"
	"github.com/elos/x/models"
	"github.com/elos/x/records"
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

	mem.WithData(map[data.Kind][]data.Record{
		data.Kind(models.Kind_USER.String()): []data.Record{
			&models.User{
				Id: "1",
			},
		},
		data.Kind(models.Kind_CREDENTIAL.String()): []data.Record{
			&models.Credential{
				Id:      "2",
				Type:    models.Credential_PASSWORD,
				Public:  "public",
				Private: "private",
				OwnerId: "1",
			},
		},
	})

	// yes already account, then public key and private key and user id
	ui.InputReader = bytes.NewBufferString(fmt.Sprintf("y\npublic\nprivate\n%s\n", "1"))

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
	if conf.UserID != "1" {
		t.Fatalf("User id should be: %s", "1")
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

	dbc, closers, err := access.NewTestDB(db)
	if err != nil {
		t.Fatalf("access.NewTestDB error: %v", err)
	}
	defer func(cs []io.Closer) {
		for _, c := range cs {
			c.Close()
		}
	}(closers)

	authc, closers, err := auth.NewTestAuth(db)
	if err != nil {
		t.Fatalf("access.NewTestDB error: %v", err)
	}
	defer func(cs []io.Closer) {
		for _, c := range cs {
			c.Close()
		}
	}(closers)

	webui, conn, err := records.WebUIBothLocal(dbc, authc)
	if err != nil {
		t.Fatalf("records.WebUIBothLocal error: %v", err)
	}
	defer conn.Close()

	g := gaia.New(
		context.Background(),
		&gaia.Middleware{},
		&gaia.Services{
			SMSCommandSessions: services.NewSMSMux(),
			Logger:             services.NewTestLogger(t),
			DB:                 db,
			WebUIClient:        webui,
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

	i, err := db.Query(data.Kind(models.Kind_CREDENTIAL.String())).Select(data.AttrMap{
		"public":  "public",
		"private": "private",
	}).Execute()
	if err != nil {
		t.Fatal(err)
	}

	cred := new(models.Credential)
	if ok := i.Next(cred); !ok {
		t.Fatal("no credentials found")
	}

	if got, want := cred.OwnerId, "1"; got != want {
		t.Fatalf("cred.OwnerId: got %q, want %q", got, want)
	}

	// verify conf was changed
	if got, want := conf.UserID, "1"; got != want {
		t.Fatalf("conf.UserID: got %q, want %q", got, want)
	}

	if conf.PublicCredential != "public" {
		t.Fatalf("public credential should be: public")
	}

	if conf.PrivateCredential != "private" {
		t.Fatalf("private credential should be: private")
	}
}

// --- }}}
