package command_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elos/elos/command"
	"github.com/mitchellh/cli"
)

func TestHostChange(t *testing.T) {
	f, err := ioutil.TempFile("", "configtest")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	conf := &command.Config{
		Path: f.Name(),
	}

	// Remove the file to test the scenario in which it doesn't exist
	os.Remove(f.Name())

	ui := new(cli.MockUi)

	c := &command.ConfCommand{
		Ui:     ui,
		Config: conf,
	}

	newHost := "0.0.0.0:8000"

	ui.InputReader = bytes.NewBufferString(newHost + "\n")

	c.Run([]string{"host", "edit"})

	writtenConf, err := command.ParseConfigFile(conf.Path)
	if err != nil {
		t.Fatalf("ParseConfigFile: %s", err)
	}

	if writtenConf.Host != newHost {
		t.Fatalf("Host should be %s, not %s", newHost, writtenConf.Host)
	}

	os.Remove(writtenConf.Path)
}
