//go:build js && wasm

package calendar

// Event represents a Google Calendar event
type Event struct {
	ID               string         `json:"id"`
	Summary          string         `json:"summary"` // Title
	Description      string         `json:"description"`
	Location         string         `json:"location"`
	Start            *EventDateTime `json:"start"`
	End              *EventDateTime `json:"end"`
	Status           string         `json:"status"`  // confirmed, tentative, cancelled
	Created          string         `json:"created"` // RFC 3339
	Updated          string         `json:"updated"`
	Creator          *EventPerson   `json:"creator"`
	Organizer        *EventPerson   `json:"organizer"`
	Attendees        []*Attendee    `json:"attendees"`
	HTMLLink         string         `json:"htmlLink"`
	ICalUID          string         `json:"iCalUID"`
	RecurringEventID string         `json:"recurringEventId,omitempty"`
}

// EventDateTime represents the start or end time of an event
type EventDateTime struct {
	DateTime string `json:"dateTime,omitempty"` // RFC 3339 with timezone, e.g., "2024-01-15T14:00:00-05:00"
	TimeZone string `json:"timeZone,omitempty"` // e.g., "America/New_York"
	Date     string `json:"date,omitempty"`     // For all-day events, e.g., "2024-01-15"
}

// EventPerson represents a person (creator or organizer)
type EventPerson struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Self        bool   `json:"self,omitempty"`
}

// Attendee represents an event attendee
type Attendee struct {
	Email          string `json:"email"`
	DisplayName    string `json:"displayName,omitempty"`
	Organizer      bool   `json:"organizer,omitempty"`
	Self           bool   `json:"self,omitempty"`
	ResponseStatus string `json:"responseStatus"` // needsAction, declined, tentative, accepted
}

// ListEventsResponse is the response from listing events
type ListEventsResponse struct {
	Items         []*Event `json:"items"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
	NextSyncToken string   `json:"nextSyncToken,omitempty"`
}

// CreateEventRequest is used to create a new event
type CreateEventRequest struct {
	Summary     string         `json:"summary"`
	Description string         `json:"description,omitempty"`
	Location    string         `json:"location,omitempty"`
	Start       *EventDateTime `json:"start"`
	End         *EventDateTime `json:"end"`
}

// Calendar represents a calendar
type Calendar struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
}

// ListCalendarsResponse is the response from listing calendars
type ListCalendarsResponse struct {
	Items         []*Calendar `json:"items"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
}
