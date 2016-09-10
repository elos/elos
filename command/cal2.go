package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/elos/x/data"
	"github.com/elos/x/models"
	"github.com/elos/x/models/cal"
	"github.com/mitchellh/cli"
)

const clientSecret = `
{
    "installed": {
        "client_id":"255269529674-kur0rf76f40277knn48p7hquv2urqv5e.apps.googleusercontent.com",
        "project_id":"elos-cal",
        "auth_uri":"https://accounts.google.com/o/oauth2/auth",
        "token_uri":"https://accounts.google.com/o/oauth2/token",
        "auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs",
        "client_secret":"jYXa9vtnsLvww1R_bre4eK97",
        "redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]
    }
}
`

type Cal2Command struct {
	// UI is used to communicate (for IO) with the user.
	UI cli.Ui

	// UserID is the id of the user on whose behalf this
	// command acts
	UserID string

	// The client to the database
	data.DBClient
}

func (c *Cal2Command) Synopsis() string {
	return "Utilities for managing the [new] elos scheduling system"
}
func (c *Cal2Command) Help() string {
	return `
Usage:
	elos cal2 <subcommand>

Subcommands:
	day		list the events for today
	week	list the events for this week
	google	sync with google
`
}

func (c *Cal2Command) Run(args []string) int {
	if len(args) == 0 {
		c.UI.Output(c.Help())
		return success
	}

	switch args[0] {
	case "day":
		return c.runDay(args[1:])
	case "week":
		return c.runWeek(args[1:])
	case "google":
		return c.runGoogle(args[1:])
	default:
		c.UI.Output(c.Help())
		return success
	}
}

func (c *Cal2Command) runDay(args []string) int {
	return c.runListDays(args, 1)
}

func (c *Cal2Command) runWeek(args []string) int {
	return c.runListDays(args, 7)
}

func (c *Cal2Command) runListDays(args []string, num int) int {
	results, err := c.DBClient.Query(context.Background(), &data.Query{
		Kind: models.Kind_FIXTURE,
		Filters: []*data.Filter{
			{
				Op:    data.Filter_EQ,
				Field: "owner_id",
				Reference: &data.Filter_String_{
					String_: c.UserID,
				},
			},
		},
	})
	if err != nil {
		c.UI.Error(fmt.Sprintf("w.db.Query error: %v", err))
		return 1
	}

	fixtures := make([]*models.Fixture, 0)
	for {
		rec, err := results.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.UI.Error(fmt.Sprintf("results.Recv error: %v", err))
			return 1
		}
		fixtures = append(fixtures, rec.Fixture)
	}

	firstDay := cal.DateFrom(time.Now())
	es := cal.EventsWithin(firstDay.Time(), firstDay.Time().AddDate(0, 0, num), fixtures)
	for _, e := range es {
		c.UI.Output(fmt.Sprintf(" - %s [%s-%s]", e.Name, e.Start.Time().Local().Format(time.Kitchen), e.End.Time().Local().Format(time.Kitchen)))
	}
	return 0
}

func ingestEvent(ctx context.Context, dbc data.DBClient, uid string, e *calendar.Event) (*models.Fixture, error) {
	log.Printf("ingesting %s", e.Summary)
	f, err := models.UnmarshalGoogleEvent(e)
	if err != nil {
		return nil, err
	}
	results, err := dbc.Query(ctx, &data.Query{
		Kind: models.Kind_FIXTURE,
		Filters: []*data.Filter{
			&data.Filter{
				Op:    data.Filter_EQ,
				Field: "labels.google/event/id",
				Reference: &data.Filter_String_{
					String_: e.Id,
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	rec, err := results.Recv()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if err := results.CloseSend(); err != nil {
		return nil, err
	}

	if err == io.EOF {
		f.OwnerId = uid
		if rec, err = dbc.Mutate(ctx, &data.Mutation{
			Op: data.Mutation_CREATE,
			Record: &data.Record{
				Kind:    models.Kind_FIXTURE,
				Fixture: f,
			},
		}); err != nil {
			return nil, err
		} else {
			return rec.Fixture, nil
		}
	} else {
		f.Id = rec.Fixture.Id
		if rec, err = dbc.Mutate(ctx, &data.Mutation{
			Op: data.Mutation_UPDATE,
			Record: &data.Record{
				Kind:    models.Kind_FIXTURE,
				Fixture: f,
			},
		}); err != nil {
			return nil, err
		} else {
			return rec.Fixture, nil
		}
	}
}

func (c *Cal2Command) runGoogle(args []string) int {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	config, err := google.ConfigFromJSON([]byte(clientSecret), calendar.CalendarScope)
	if err != nil {
		c.UI.Error(fmt.Sprintf("unable to parse client secrete file to config: %v", err))
		return failure
	}

	u, err := stringInput(c.UI, "Username:")
	if err != nil {
		c.UI.Error(err.Error())
		return failure
	}

	client := getClient(ctx, config, u)
	srv, err := calendar.New(client)
	if err != nil {
		c.UI.Error(fmt.Sprintf("unable to retrieve calendar client %v", err))
		return failure
	}
	events, err := srv.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(time.Now().AddDate(0, -1, 0).Format(time.RFC3339)).
		OrderBy("startTime").Do()
	if err != nil {
		c.UI.Error(fmt.Sprintf("unable to retrieve user events: $v", err))
		return failure
	}

	n := 0
	recurring := map[string]bool{}
	for _, e := range events.Items {
		if e.RecurringEventId != "" {
			recurring[e.RecurringEventId] = true
			continue // don't ingest recurring instances
		}
		c.UI.Output(fmt.Sprintf("Processing: %v", e.Summary))
		_, err := ingestEvent(ctx, c.DBClient, c.UserID, e)
		if err != nil {
			c.UI.Error(err.Error())
			return failure
		}
		n++
	}
	for id := range recurring {
		e, err := srv.Events.Get("primary", id).Do()
		if err != nil {
			c.UI.Error(err.Error())
			return failure
		}
		_, err = ingestEvent(ctx, c.DBClient, c.UserID, e)
		if err != nil {
			c.UI.Error(err.Error())
			return failure
		}
	}
	return success
}

func mainID(srv *calendar.Service) (string, error) {
	cl, err := srv.CalendarList.List().Do()
	if err != nil {
		return "", err
	}

	for _, i := range cl.Items {
		if i.Primary {
			return i.Id, nil
		}
	}

	return "", io.EOF
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config, u string) *http.Client {
	cacheFile, err := tokenCacheFile(u)
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile(u string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials", u)
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("elos.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
