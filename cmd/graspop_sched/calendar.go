package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func uploadToGoogleCalendar(showList []Show, credFile string, tokenFile string, calendarName string, tz string, desc string) {
	ctx := context.Background()
	// Authorization credentials for a desktop application. To learn how to create credentials for a desktop application, refer to Create credentials.
	// https://developers.google.com/workspace/guides/create-credentials
	b, err := ioutil.ReadFile(credFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config, tokenFile)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}
	cal := getOrCreateCalendar(*srv, calendarName, tz, desc)
	cleanupCalendar(*srv, *cal)
	populateCalendar(*srv, *cal, showList)
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenFile string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getOrCreateCalendar(svc calendar.Service, name string, tz string, desc string) *calendar.Calendar {
	list, err := svc.CalendarList.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar list: %v", err)
	}
	cal_id := ""
	for i := 0; i < len(list.Items); i++ {
		cal := list.Items[i]
		if cal.Summary == name {
			cal_id = cal.Id
			break
		}
	}
	if cal_id == "" {
		cal := calendar.Calendar{Summary: name, TimeZone: tz, Description: desc}
		res, err := svc.Calendars.Insert(&cal).Do()
		if err != nil {
			log.Fatalf("Unable to insert a new calendar: %v", err)
		}
		return res
	} else {
		cal, err := svc.Calendars.Get(cal_id).Do()
		if err != nil {
			log.Fatalf("Unable to get a calendar: %v", err)
		}
		return cal
	}
}
func cleanupCalendar(svc calendar.Service, cal calendar.Calendar) {
	// removes everything from the calendar
	events, err := svc.Events.List(cal.Id).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve Event list: %v", err)
	}
	for i := 0; i < len(events.Items); i++ {
		svc.Events.Delete(cal.Id, events.Items[i].Id).Do()
		time.Sleep(100 * time.Millisecond)
	}

}
func populateCalendar(svc calendar.Service, cal calendar.Calendar, show_list []Show) {
	for i := 0; i < len(show_list); i++ {
		show := show_list[i]
		event := calendar.Event{
			Summary:  show.name,
			Location: show.scene,
			ColorId:  show.scene_num,
			Start: &calendar.EventDateTime{
				DateTime: time.Unix(show.start_date, 0).Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: time.Unix(show.end_date, 0).Format(time.RFC3339),
			},
			Description: show.url,
		}
		_, err := svc.Events.Insert(cal.Id, &event).Do()
		if err != nil {
			log.Fatalf("Unable to create an event: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
