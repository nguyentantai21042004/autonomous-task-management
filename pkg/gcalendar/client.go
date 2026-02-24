package gcalendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Calendar API service.
type Client struct {
	service *calendar.Service
}

// NewClientFromCredentialsFile creates a Calendar client from a Service Account JSON file path.
func NewClientFromCredentialsFile(ctx context.Context, credentialsPath string) (*Client, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	return NewClientFromCredentialsJSON(ctx, data)
}

// NewClientFromCredentialsJSON creates a Calendar client from raw Service Account JSON bytes.
func NewClientFromCredentialsJSON(ctx context.Context, credentialsJSON []byte) (*Client, error) {
	// Try service account first
	config, err := google.JWTConfigFromJSON(credentialsJSON, calendar.CalendarScope)
	if err == nil {
		// Service Account path
		tokenSource := config.TokenSource(ctx)
		svc, svcErr := calendar.NewService(ctx, option.WithTokenSource(tokenSource))
		if svcErr != nil {
			return nil, fmt.Errorf("failed to create calendar service: %w", svcErr)
		}
		return &Client{service: svc}, nil
	}

	// Fallback: try OAuth2 installed app credentials
	var oauthCreds struct {
		Installed struct {
			ClientID     string   `json:"client_id"`
			ClientSecret string   `json:"client_secret"`
			RedirectURIs []string `json:"redirect_uris"`
		} `json:"installed"`
	}
	if jsonErr := json.Unmarshal(credentialsJSON, &oauthCreds); jsonErr != nil {
		return nil, fmt.Errorf("unsupported credentials format: %w", err)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     oauthCreds.Installed.ClientID,
		ClientSecret: oauthCreds.Installed.ClientSecret,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}

	// For OAuth2 Desktop app: use a static token if token.json exists
	tokenData, tokenErr := os.ReadFile("token.json")
	if tokenErr != nil {
		return nil, fmt.Errorf("google credentials are OAuth Desktop type but no token.json found: use Service Account instead")
	}

	var tok oauth2.Token
	if jsonErr := json.Unmarshal(tokenData, &tok); jsonErr != nil {
		return nil, fmt.Errorf("failed to parse token.json: %w", jsonErr)
	}

	tokenSource := oauthConfig.TokenSource(ctx, &tok)
	svc, svcErr := calendar.NewService(ctx, option.WithTokenSource(tokenSource))
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create calendar service from OAuth token: %w", svcErr)
	}

	return &Client{service: svc}, nil
}

// NewClientFromHTTP creates a Calendar client from a pre-configured HTTP client.
func NewClientFromHTTP(ctx context.Context, httpClient *http.Client) (*Client, error) {
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}
	return &Client{service: svc}, nil
}

// CreateEvent creates a new Google Calendar event.
func (c *Client) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	event := &calendar.Event{
		Summary:     req.Summary,
		Description: req.Description,
		Start: &calendar.EventDateTime{
			// Use time.RFC3339 to embed timezone info directly (convention fixes recommendation)
			DateTime: req.StartTime.Format(time.RFC3339),
			TimeZone: req.Timezone,
		},
		End: &calendar.EventDateTime{
			DateTime: req.EndTime.Format(time.RFC3339),
			TimeZone: req.Timezone,
		},
	}

	calendarID := req.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	created, err := c.service.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event: %w", err)
	}

	return &Event{
		ID:          created.Id,
		Summary:     created.Summary,
		Description: created.Description,
		HtmlLink:    created.HtmlLink,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}, nil
}

// ListEvents retrieves events from Google Calendar within a time range.
func (c *Client) ListEvents(ctx context.Context, req ListEventsRequest) ([]Event, error) {
	calendarID := req.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 100 // Default limit
	}

	call := c.service.Events.List(calendarID).
		Context(ctx).
		TimeMin(req.TimeMin.Format(time.RFC3339)).
		TimeMax(req.TimeMax.Format(time.RFC3339)).
		MaxResults(maxResults).
		SingleEvents(true).
		OrderBy("startTime")

	eventsResp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events: %w", err)
	}

	events := make([]Event, 0, len(eventsResp.Items))
	for _, item := range eventsResp.Items {
		// Parse start time
		var startTime time.Time
		if item.Start.DateTime != "" {
			startTime, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		} else if item.Start.Date != "" {
			// All-day event
			startTime, _ = time.Parse("2006-01-02", item.Start.Date)
		}

		// Parse end time
		var endTime time.Time
		if item.End.DateTime != "" {
			endTime, _ = time.Parse(time.RFC3339, item.End.DateTime)
		} else if item.End.Date != "" {
			endTime, _ = time.Parse("2006-01-02", item.End.Date)
		}

		events = append(events, Event{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			HtmlLink:    item.HtmlLink,
			StartTime:   startTime,
			EndTime:     endTime,
			Location:    item.Location,
		})
	}

	return events, nil
}
