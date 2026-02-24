package gcalendar_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/pkg/gcalendar"
)

type rewriteTransport struct {
	Transport http.RoundTripper
	Host      string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.Host
	return t.Transport.RoundTrip(req)
}

func TestCalendarClient(t *testing.T) {
	// Constructing fake credentials for local parsing flows
	mockCreds := `{
		"installed": {
			"client_id": "test-client-id.apps.googleusercontent.com",
			"project_id": "test-project",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"client_secret": "test-secret",
			"redirect_uris": ["http://localhost"]
		}
	}`

	t.Run("Initialize with broken JWT/OAuth config", func(t *testing.T) {
		_, err := gcalendar.NewClientFromCredentialsJSON(context.Background(), []byte(`{"broken":true}`))
		if err == nil {
			t.Errorf("expected decoding failure")
		}
	})

	t.Run("Initialize from installed app config", func(t *testing.T) {
		// Native oauth load requires token.json
		os.WriteFile("token.json", []byte(`{"access_token": "dummy", "token_type": "Bearer", "expiry": "2030-01-01T00:00:00Z"}`), 0644)
		defer os.Remove("token.json")

		_, err := gcalendar.NewClientFromCredentialsJSON(context.Background(), []byte(mockCreds))
		if err != nil {
			t.Fatalf("expected parsing to succeed: %v", err)
		}
	})

	t.Run("Initialize from installed app config bad token", func(t *testing.T) {
		os.WriteFile("token.json", []byte(`{"broken": true`), 0644)
		defer os.Remove("token.json")

		_, err := gcalendar.NewClientFromCredentialsJSON(context.Background(), []byte(mockCreds))
		if err == nil {
			t.Fatalf("expected parsing to fail on bad token")
		}
	})

	t.Run("Initialize from File", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "creds.json")
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(`{"broken":true}`)
		tmpFile.Close()

		_, err := gcalendar.NewClientFromCredentialsFile(context.Background(), tmpFile.Name())
		if err == nil {
			t.Errorf("expected failure loading broken file")
		}

		_, err = gcalendar.NewClientFromCredentialsFile(context.Background(), "non-existent-file-path-12345.json")
		if err == nil {
			t.Errorf("expected reading file error")
		}
	})

	t.Run("Create Event E2E", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/calendar/v3/calendars/primary/events" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"id": "event-123",
					"htmlLink": "https://calendar.google.com/event-uri",
					"status": "confirmed"
				}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		tsClient := ts.Client()
		tsClient.Transport = &rewriteTransport{
			Transport: tsClient.Transport,
			Host:      strings.TrimPrefix(ts.URL, "http://"),
		}

		client, err := gcalendar.NewClientFromHTTP(context.Background(), tsClient)
		if err != nil {
			t.Fatalf("unexpected error creating client: %v", err)
		}

		event, err := client.CreateEvent(context.Background(), gcalendar.CreateEventRequest{
			CalendarID:  "primary",
			Summary:     "Title",
			Description: "Desc",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("failed to create event: %v", err)
		}
		if event.HtmlLink != "https://calendar.google.com/event-uri" {
			t.Errorf("unexpected link: %s", event.HtmlLink)
		}
	})

	t.Run("List Events E2E", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/calendar/v3/calendars/test-fail/events" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if r.URL.Path == "/calendar/v3/calendars/primary/events" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"items": [
						{
							"id": "event-123",
							"summary": "Existing Event",
							"start": { "date": "2024-05-01" },
							"end": { "date": "2024-05-01" }
						}
					]
				}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		tsClient := ts.Client()
		tsClient.Transport = &rewriteTransport{
			Transport: tsClient.Transport,
			Host:      strings.TrimPrefix(ts.URL, "http://"),
		}

		client, err := gcalendar.NewClientFromHTTP(context.Background(), tsClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		events, err := client.ListEvents(context.Background(), gcalendar.ListEventsRequest{
			CalendarID: "primary",
			TimeMin:    time.Now(),
			TimeMax:    time.Now().Add(time.Hour * 24),
		})
		if err != nil {
			t.Fatalf("failed to list events: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
		if events[0].Summary != "Existing Event" {
			t.Errorf("unexpected event: %s", events[0].Summary)
		}

		_, err = client.ListEvents(context.Background(), gcalendar.ListEventsRequest{
			CalendarID: "test-fail",
			TimeMin:    time.Now(),
			TimeMax:    time.Now().Add(time.Hour * 24),
		})
		if err == nil {
			t.Fatalf("expected api error on test-fail")
		}
	})

	t.Run("Create Event Error E2E", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/calendar/v3/calendars/primary/events" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}))
		defer ts.Close()

		tsClient := ts.Client()
		tsClient.Transport = &rewriteTransport{
			Transport: tsClient.Transport,
			Host:      strings.TrimPrefix(ts.URL, "http://"),
		}

		client, _ := gcalendar.NewClientFromHTTP(context.Background(), tsClient)
		_, err := client.CreateEvent(context.Background(), gcalendar.CreateEventRequest{
			CalendarID: "primary",
		})
		if err == nil {
			t.Fatalf("expected create event error")
		}
	})
}
