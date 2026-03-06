//go:build js && wasm

package calendar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gleicon/webclaw/internal/integrations/google"
)

// Client provides Google Calendar API operations
type Client struct {
	base *google.Client
}

// NewClient creates a new Calendar client
func NewClient(baseClient *google.Client) *Client {
	return &Client{
		base: baseClient,
	}
}

// ListEvents lists events from the primary calendar
// timeMin and timeMax should be in RFC 3339 format
func (c *Client) ListEvents(timeMin, timeMax string, maxResults int) (*ListEventsResponse, error) {
	if maxResults <= 0 || maxResults > 2500 {
		maxResults = 250 // Default per Google API
	}

	// Build URL with query parameters
	url := c.base.BuildCalendarURL("/calendars/primary/events")
	params := []string{
		fmt.Sprintf("maxResults=%d", maxResults),
		"orderBy=startTime",
		"singleEvents=true", // Expand recurring events
	}

	if timeMin != "" {
		params = append(params, "timeMin="+timeMin)
	}

	if timeMax != "" {
		params = append(params, "timeMax="+timeMax)
	}

	url = url + "?" + strings.Join(params, "&")

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result ListEventsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	return &result, nil
}

// GetEvent retrieves a specific event by ID
func (c *Client) GetEvent(id string) (*Event, error) {
	url := c.base.BuildCalendarURL(fmt.Sprintf("/calendars/primary/events/%s", id))

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(resp.Body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	return &event, nil
}

// CreateEvent creates a new event on the primary calendar
func (c *Client) CreateEvent(summary, description, location string, start, end *EventDateTime) (*Event, error) {
	req := CreateEventRequest{
		Summary:     summary,
		Description: description,
		Location:    location,
		Start:       start,
		End:         end,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	url := c.base.BuildCalendarURL("/calendars/primary/events")

	resp, err := c.base.DoRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(resp.Body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse created event: %w", err)
	}

	return &event, nil
}

// DeleteEvent permanently deletes an event
func (c *Client) DeleteEvent(id string) error {
	url := c.base.BuildCalendarURL(fmt.Sprintf("/calendars/primary/events/%s", id))
	_, err := c.base.DoRequest("DELETE", url, nil)
	return err
}

// QuickAddEvent adds an event using the "quick add" feature
// This parses natural language like "Meeting tomorrow at 2pm"
func (c *Client) QuickAddEvent(text string) (*Event, error) {
	// URL-encode the text
	text = strings.ReplaceAll(text, " ", "%20")
	url := c.base.BuildCalendarURL(fmt.Sprintf("/calendars/primary/events/quickAdd?text=%s", text))

	resp, err := c.base.DoRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(resp.Body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	return &event, nil
}

// ListCalendars lists all calendars the user has access to
func (c *Client) ListCalendars(maxResults int) (*ListCalendarsResponse, error) {
	if maxResults <= 0 {
		maxResults = 100
	}

	url := c.base.BuildCalendarURL(fmt.Sprintf("/users/me/calendarList?maxResults=%d", maxResults))

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result ListCalendarsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse calendars: %w", err)
	}

	return &result, nil
}

// ParseNaturalTime attempts to parse natural language time expressions
// Returns an EventDateTime suitable for API calls
// This is a simple implementation - complex expressions should be handled by LLM
func ParseNaturalTime(text string, timezone string) (*EventDateTime, error) {
	if timezone == "" {
		timezone = "UTC"
	}

	// Try RFC 3339 first
	if t, err := time.Parse(time.RFC3339, text); err == nil {
		return &EventDateTime{
			DateTime: t.Format(time.RFC3339),
			TimeZone: timezone,
		}, nil
	}

	// Try common formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 2, 2006 3:04 PM",
		"Jan 2, 2006",
		"Monday, Jan 2, 2006 3:04 PM",
		"01/02/2006 15:04",
		"01/02/2006",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, text, time.Local); err == nil {
			if strings.Contains(format, "15") || strings.Contains(format, "3") {
				// Has time component
				return &EventDateTime{
					DateTime: t.Format(time.RFC3339),
					TimeZone: timezone,
				}, nil
			}
			// Date only (all-day event)
			return &EventDateTime{
				Date: t.Format("2006-01-02"),
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to parse time: %s", text)
}

// CalculateEndTime calculates end time based on start time and duration
func CalculateEndTime(start *EventDateTime, durationMinutes int) (*EventDateTime, error) {
	if durationMinutes <= 0 {
		durationMinutes = 60 // Default 1 hour
	}

	// If start has DateTime (timed event)
	if start.DateTime != "" {
		t, err := time.Parse(time.RFC3339, start.DateTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}

		endTime := t.Add(time.Duration(durationMinutes) * time.Minute)
		return &EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: start.TimeZone,
		}, nil
	}

	// If start has Date (all-day event)
	if start.Date != "" {
		t, err := time.Parse("2006-01-02", start.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %w", err)
		}

		// For all-day events, duration is in days (round up to nearest day)
		days := (durationMinutes + 1439) / 1440 // 1440 minutes in a day
		if days < 1 {
			days = 1
		}

		endTime := t.AddDate(0, 0, days)
		return &EventDateTime{
			Date: endTime.Format("2006-01-02"),
		}, nil
	}

	return nil, fmt.Errorf("start time has neither DateTime nor Date")
}

// FormatEvent returns a human-readable string representation of an event
func FormatEvent(e *Event) string {
	var parts []string

	// Title
	parts = append(parts, fmt.Sprintf("📅 %s", e.Summary))

	// Time
	timeStr := FormatEventTime(e.Start, e.End)
	parts = append(parts, fmt.Sprintf("🕐 %s", timeStr))

	// Location
	if e.Location != "" {
		parts = append(parts, fmt.Sprintf("📍 %s", e.Location))
	}

	// Description (truncated)
	if e.Description != "" {
		desc := e.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		parts = append(parts, fmt.Sprintf("📝 %s", desc))
	}

	// Status
	if e.Status != "" && e.Status != "confirmed" {
		parts = append(parts, fmt.Sprintf("Status: %s", e.Status))
	}

	return strings.Join(parts, "\n")
}

// FormatEventTime returns a formatted time string for an event
func FormatEventTime(start, end *EventDateTime) string {
	if start == nil {
		return "Time TBD"
	}

	// All-day event
	if start.Date != "" {
		if end != nil && end.Date != "" && end.Date != start.Date {
			return fmt.Sprintf("%s - %s (all-day)", formatDate(start.Date), formatDate(end.Date))
		}
		return fmt.Sprintf("%s (all-day)", formatDate(start.Date))
	}

	// Timed event
	if start.DateTime != "" {
		startTime, err := time.Parse(time.RFC3339, start.DateTime)
		if err != nil {
			return start.DateTime
		}

		startStr := startTime.Format("Monday, Jan 2 at 3:04 PM")

		if end != nil && end.DateTime != "" {
			endTime, err := time.Parse(time.RFC3339, end.DateTime)
			if err == nil {
				// Check if same day
				if startTime.Year() == endTime.Year() && startTime.YearDay() == endTime.YearDay() {
					return fmt.Sprintf("%s - %s", startStr, endTime.Format("3:04 PM"))
				}
				return fmt.Sprintf("%s - %s", startStr, endTime.Format("Monday, Jan 2 at 3:04 PM"))
			}
		}

		return startStr
	}

	return "Time TBD"
}

// formatDate converts "2006-01-02" to a friendly format
func formatDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Monday, Jan 2, 2006")
}

// IsAllDay checks if an event is an all-day event
func IsAllDay(e *Event) bool {
	return e.Start != nil && e.Start.Date != ""
}

// GetEventURL returns the Google Calendar web URL for an event
func GetEventURL(e *Event) string {
	if e.HTMLLink != "" {
		return e.HTMLLink
	}
	return fmt.Sprintf("https://calendar.google.com/calendar/event?eid=%s", e.ID)
}
