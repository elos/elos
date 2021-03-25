package command

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/x/data"
	"github.com/elos/x/models"
	"github.com/mitchellh/cli"
)

func s(ks []models.Kind) []string {
	ns := make([]string, len(ks))
	for i, k := range ks {
		ns[i] = k.String()
	}
	return ns
}

func apply(ss []string, f func(s string) string) []string {
	for i, s := range ss {
		ss[i] = f(s)
	}
	return ss
}

func TestRecords(t *testing.T) {
	cases := map[string]struct {
		args             []string
		code             int
		in, err, out     []byte
		prior, posterior data.State
	}{

		// elos records kinds {{{
		"elos records kinds": {
			args: []string{"kinds"},
			out:  []byte(strings.Join(apply(s(models.Kinds), func(s string) string { return "* " + s }), "\n") + "\n"),
			prior: data.State{
				models.Kind_USER: []*data.Record{
					&data.Record{
						Kind: models.Kind_USER,
						User: &models.User{
							Id: "1",
						},
					},
				},
				models.Kind_CREDENTIAL: []*data.Record{
					&data.Record{
						Kind: models.Kind_CREDENTIAL,
						Credential: &models.Credential{
							Id:      "2",
							Type:    models.Credential_PASSWORD,
							Public:  "pu",
							Private: "pr",
							OwnerId: "1",
						},
					},
				},
			},
		},
		// }}}

		// elos records count {{{
		"elos records count": {
			args: []string{"count"},
			in:   []byte("SESSION\n"),
			out:  []byte("Which kind? [string]:2\n"),
			prior: data.State{
				models.Kind_USER: []*data.Record{
					&data.Record{
						Kind: models.Kind_USER,
						User: &models.User{
							Id: "1",
						},
					},
					&data.Record{
						Kind: models.Kind_USER,
						User: &models.User{
							Id: "2",
						},
					},
				},
				models.Kind_CREDENTIAL: []*data.Record{
					&data.Record{
						Kind: models.Kind_CREDENTIAL,
						Credential: &models.Credential{
							Id:      "3",
							Type:    models.Credential_PASSWORD,
							Public:  "pu",
							Private: "pr",
							OwnerId: "1",
						},
					},
					&data.Record{
						Kind: models.Kind_CREDENTIAL,
						Credential: &models.Credential{
							Id:      "4",
							Type:    models.Credential_PASSWORD,
							Public:  "2pu",
							Private: "pr",
							OwnerId: "2",
						},
					},
				},
				models.Kind_SESSION: []*data.Record{
					&data.Record{
						Kind: models.Kind_SESSION,
						Session: &models.Session{
							Id:           "5",
							AccessToken:  "non-empty",
							ExpiresAt:    models.TimestampFrom(time.Now().Add(5 * time.Minute)).WithoutNanos(),
							OwnerId:      "1",
							CredentialId: "3",
						},
					},
					&data.Record{
						Kind: models.Kind_SESSION,
						Session: &models.Session{
							Id:           "4",
							AccessToken:  "non-empty",
							ExpiresAt:    models.TimestampFrom(time.Now().Add(5 * time.Minute)).WithoutNanos(),
							OwnerId:      "1",
							CredentialId: "3",
						},
					},
					&data.Record{
						Kind: models.Kind_SESSION,
						Session: &models.Session{
							Id:           "4",
							AccessToken:  "non-empty",
							ExpiresAt:    models.TimestampFrom(time.Now().Add(5 * time.Minute)).WithoutNanos(),
							OwnerId:      "2",
							CredentialId: "4",
						},
					},
				},
			},
		},
		// }}}

		// TODO(nclandolfi) test query and changes

	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			db := mem.NewDB()
			dbc, conn, err := data.DBBothLocal(ctx, db)
			if err != nil {
				t.Fatalf("data.DBBothLocal error: %v", err)
			}
			defer conn.Close()
			if err := data.Seed(context.Background(), dbc, c.prior); err != nil {
				t.Fatalf("data.Seed error: %v", err)
			}

			ui := &cli.MockUi{
				InputReader: bytes.NewBuffer(c.in),
			}
			cmd := &RecordsCommand{
				UI:       ui,
				UserID:   c.prior[models.Kind_USER][0].User.Id,
				DBClient: dbc,
			}

			if got, want := cmd.Run(c.args), c.code; got != want {
				t.Log(ui.ErrorWriter.String())
				t.Fatalf("cmd.Run(%v): got %d, want %d", c.args, got, want)
			}

			if got, want := ui.ErrorWriter.String(), string(c.err); got != want {
				t.Fatalf("ui.ErrorWriter.String(): got %q, want %q", got, want)
			}

			if got, want := ui.OutputWriter.String(), string(c.out); got != want {
				t.Fatalf("ui.OutputWriter.String(): got %q, want %q", got, want)
			}

			finalState := c.prior
			if c.posterior != nil {
				finalState = c.posterior
			}

			if got, want := data.CompareState(context.Background(), dbc, finalState), error(nil); got != want {
				t.Fatalf("data.CompareState: got %v, want %v", got, want)
			}
		})
	}
}
