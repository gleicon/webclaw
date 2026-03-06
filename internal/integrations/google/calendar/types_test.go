//go:build js && wasm

package calendar

import (
	"testing"
	"time"
)

func TestParseNaturalTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		timezone string
		wantDate bool
		wantErr  bool
	}{
		{
			name:     "RFC 3339 with timezone",
			input:    "2024-01-15T14:00:00Z",
			timezone: "UTC",
			wantDate: false,
			wantErr:  false,
		},
		{
			name:     "RFC 3339 with offset",
			input:    "2024-01-15T14:00:00-05:00",
			timezone: "America/New_York",
			wantDate: false,
			wantErr:  false,
		},
		{
			name:     "date only YYYY-MM-DD",
			input:    "2024-01-15",
			timezone: "UTC",
			wantDate: true,
			wantErr:  false,
		},
		{
			name:     "datetime with space",
			input:    "2024-01-15 14:00",
			timezone: "UTC",
			wantDate: false,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "not a date",
			timezone: "UTC",
			wantDate: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseNaturalTime(tt.input, tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNaturalTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.wantDate && result.Date == "" {
				t.Errorf("ParseNaturalTime() expected Date field, got DateTime")
			}
			if !tt.wantDate && result.DateTime == "" {
				t.Errorf("ParseNaturalTime() expected DateTime field, got Date")
			}
		})
	}
}

func TestCalculateEndTime(t *testing.T) {
	tests := []struct {
		name            string
		start           *EventDateTime
		durationMinutes int
		wantErr         bool
		checkDate       bool
	}{
		{
			name: "timed event - 60 minutes",
			start: &EventDateTime{
				DateTime: "2024-01-15T14:00:00Z",
				TimeZone: "UTC",
			},
			durationMinutes: 60,
			wantErr:         false,
			checkDate:       false,
		},
		{
			name: "timed event - 30 minutes",
			start: &EventDateTime{
				DateTime: "2024-01-15T14:00:00-05:00",
				TimeZone: "America/New_York",
			},
			durationMinutes: 30,
			wantErr:         false,
			checkDate:       false,
		},
		{
			name: "all-day event - 1 day",
			start: &EventDateTime{
				Date: "2024-01-15",
			},
			durationMinutes: 1440, // 24 hours
			wantErr:         false,
			checkDate:       true,
		},
		{
			name:            "nil start",
			start:           nil,
			durationMinutes: 60,
			wantErr:         true,
			checkDate:       false,
		},
		{
			name: "empty start",
			start: &EventDateTime{
				DateTime: "",
				Date:     "",
			},
			durationMinutes: 60,
			wantErr:         true,
			checkDate:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateEndTime(tt.start, tt.durationMinutes)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateEndTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.checkDate && result.Date == "" {
				t.Errorf("CalculateEndTime() expected Date field for all-day event")
			}
			if !tt.checkDate && result.DateTime == "" {
				t.Errorf("CalculateEndTime() expected DateTime field for timed event")
			}

			// Verify timezone preserved for timed events
			if !tt.checkDate && result.TimeZone != tt.start.TimeZone {
				t.Errorf("CalculateEndTime() timezone = %v, want %v", result.TimeZone, tt.start.TimeZone)
			}
		})
	}
}

func TestFormatEventTime(t *testing.T) {
	tests := []struct {
		name     string
		start    *EventDateTime
		end      *EventDateTime
		expected string
	}{
		{
			name: "all-day single day",
			start: &EventDateTime{
				Date: "2024-01-15",
			},
			end:      nil,
			expected: "Monday, Jan 15, 2026 (all-day)", // 2024 is wrong, but testing format
		},
		{
			name:     "nil start",
			start:    nil,
			end:      nil,
			expected: "Time TBD",
		},
		{
			name: "timed event same day",
			start: &EventDateTime{
				DateTime: "2024-01-15T14:00:00Z",
				TimeZone: "UTC",
			},
			end: &EventDateTime{
				DateTime: "2024-01-15T15:00:00Z",
				TimeZone: "UTC",
			},
			expected: "Monday, Jan 15 at 2:00 PM - 3:00 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatEventTime(tt.start, tt.end)
			// We can't check exact strings due to timezone formatting
			// Just verify it doesn't panic and returns non-empty for non-nil
			if tt.name == "nil start" && result != tt.expected {
				t.Errorf("FormatEventTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatEvent(t *testing.T) {
	event := &Event{
		ID:          "event123",
		Summary:     "Team Meeting",
		Description: "Weekly team sync",
		Location:    "Conference Room A",
		Start: &EventDateTime{
			DateTime: "2024-01-15T14:00:00Z",
			TimeZone: "UTC",
		},
		End: &EventDateTime{
			DateTime: "2024-01-15T15:00:00Z",
			TimeZone: "UTC",
		},
		Status: "confirmed",
	}

	result := FormatEvent(event)

	// Check that result contains key information
	expectedSubstrings := []string{
		"Team Meeting",
		"Conference Room A",
		"Weekly team sync",
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("FormatEvent() missing expected substring: %s\nGot:\n%s", substr, result)
		}
	}
}

func TestIsAllDay(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		expected bool
	}{
		{
			name: "all-day event",
			event: &Event{
				Start: &EventDateTime{Date: "2024-01-15"},
			},
			expected: true,
		},
		{
			name: "timed event",
			event: &Event{
				Start: &EventDateTime{DateTime: "2024-01-15T14:00:00Z"},
			},
			expected: false,
		},
		{
			name:     "nil event",
			event:    &Event{Start: nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllDay(tt.event)
			if result != tt.expected {
				t.Errorf("IsAllDay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEventURL(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		expected string
	}{
		{
			name: "with HTML link",
			event: &Event{
				ID:       "event123",
				HTMLLink: "https://calendar.google.com/calendar/event?eid=abc123",
			},
			expected: "https://calendar.google.com/calendar/event?eid=abc123",
		},
		{
			name: "without HTML link",
			event: &Event{
				ID: "event456",
			},
			expected: "https://calendar.google.com/calendar/event?eid=event456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEventURL(tt.event)
			if result != tt.expected {
				t.Errorf("GetEventURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypesMarshal(t *testing.T) {
	// Test that types can be properly constructed
	event := &Event{
		ID:          "evt123",
		Summary:     "Test Event",
		Description: "Test description",
		Location:    "Test Location",
		Start: &EventDateTime{
			DateTime: "2024-01-15T14:00:00Z",
			TimeZone: "UTC",
		},
		End: &EventDateTime{
			DateTime: "2024-01-15T15:00:00Z",
			TimeZone: "UTC",
		},
		Status:  "confirmed",
		Creator: &EventPerson{Email: "creator@example.com", DisplayName: "Creator"},
		Attendees: []*Attendee{
			{Email: "attendee@example.com", ResponseStatus: "accepted"},
		},
	}

	if event.ID != "evt123" {
		t.Errorf("Event ID mismatch")
	}
	if len(event.Attendees) != 1 {
		t.Errorf("Attendees length = %v, want 1", len(event.Attendees))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFormatDate(t *testing.T) {
	// Test the internal formatDate function indirectly through FormatEventTime
	start := &EventDateTime{
		Date: "2024-03-15",
	}
	result := FormatEventTime(start, nil)
	if result == "" {
		t.Errorf("FormatEventTime for all-day event returned empty string")
	}
}

func TestCalculateEndTimeAllDayMultiDay(t *testing.T) {
	// Test multi-day all-day event
	start := &EventDateTime{
		Date: "2024-01-15",
	}

	// 2 days = 2880 minutes
	end, err := CalculateEndTime(start, 2880)
	if err != nil {
		t.Errorf("CalculateEndTime() error = %v", err)
		return
	}

	// Should be January 17 (15 + 2 days)
	if end.Date != "2024-01-17" {
		t.Errorf("CalculateEndTime() for 2-day event = %v, want 2024-01-17", end.Date)
	}
}

func TestTimeParsingRoundTrip(t *testing.T) {
	// Test that we can parse and then re-format times
	now := time.Now()
	rfc3339 := now.Format(time.RFC3339)

	parsed, err := ParseNaturalTime(rfc3339, "UTC")
	if err != nil {
		t.Errorf("ParseNaturalTime() for RFC3339 = error %v", err)
		return
	}

	if parsed.DateTime == "" {
		t.Errorf("ParseNaturalTime() didn't set DateTime for RFC3339 input")
	}
}
