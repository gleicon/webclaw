//go:build js && wasm

package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gleicon/webclaw/internal/tools"
)

// NewListTool creates the calendar_list tool
func NewListTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "calendar_list",
		Description: "List upcoming calendar events",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"days": map[string]interface{}{
					"type":        "integer",
					"description": "Number of days to look ahead (default 7)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Max events to return (default 20)",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			daysFloat, _ := params["days"].(float64)
			days := int(daysFloat)
			if days <= 0 {
				days = 7
			}
			if days > 365 {
				days = 365 // Limit to 1 year
			}

			countFloat, _ := params["count"].(float64)
			count := int(countFloat)
			if count <= 0 || count > 2500 {
				count = 20
			}

			// Calculate time range
			now := time.Now()
			timeMin := now.Format(time.RFC3339)
			timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

			events, err := client.ListEvents(timeMin, timeMax, count)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to view calendar",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "calendar_list",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to list events: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to list events: %s", errStr),
					IsError:        true,
					ToolName:       "calendar_list",
					Status:         "error",
				}, nil
			}

			if events.Items == nil || len(events.Items) == 0 {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("No upcoming events for the next %d days.", days),
					DisplayContent: fmt.Sprintf("No events in the next %d days", days),
					IsError:        false,
					ToolName:       "calendar_list",
					Status:         "done",
				}, nil
			}

			content := formatEventList(events.Items, days)
			display := fmt.Sprintf("📅 %d events in the next %d days", len(events.Items), days)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "calendar_list",
				Status:         "done",
			}, nil
		},
	}
}

// NewCreateTool creates the calendar_create tool
func NewCreateTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "calendar_create",
		Description: "Create a new calendar event. Time formats: ISO 8601 (2024-01-15T14:00:00Z), RFC 3339, or natural language like 'tomorrow 2pm'",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Event title",
				},
				"start_time": map[string]interface{}{
					"type":        "string",
					"description": "Start time (ISO 8601, RFC 3339, or natural language like 'tomorrow 2pm')",
				},
				"duration_minutes": map[string]interface{}{
					"type":        "integer",
					"description": "Duration in minutes (default 60)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Event description (optional)",
				},
				"location": map[string]interface{}{
					"type":        "string",
					"description": "Location (optional)",
				},
			},
			"required": []string{"title", "start_time"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			title, _ := params["title"].(string)
			startTimeStr, _ := params["start_time"].(string)
			durationFloat, _ := params["duration_minutes"].(float64)
			description, _ := params["description"].(string)
			location, _ := params["location"].(string)

			if title == "" {
				return &tools.ToolResult{
					Content:        "Error: 'title' parameter is required",
					DisplayContent: "Failed: event title is required",
					IsError:        true,
					ToolName:       "calendar_create",
					Status:         "error",
				}, nil
			}

			if startTimeStr == "" {
				return &tools.ToolResult{
					Content:        "Error: 'start_time' parameter is required",
					DisplayContent: "Failed: start time is required",
					IsError:        true,
					ToolName:       "calendar_create",
					Status:         "error",
				}, nil
			}

			duration := int(durationFloat)
			if duration <= 0 {
				duration = 60 // Default 1 hour
			}

			// Parse start time
			start, err := ParseNaturalTime(startTimeStr, "")
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to parse start time: %s. Expected formats: ISO 8601 (2024-01-15T14:00:00Z), RFC 3339, or simple date/time", err.Error()),
					DisplayContent: fmt.Sprintf("Invalid start time: %s", err.Error()),
					IsError:        true,
					ToolName:       "calendar_create",
					Status:         "error",
				}, nil
			}

			// Calculate end time
			end, err := CalculateEndTime(start, duration)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to calculate end time: %s", err.Error()),
					DisplayContent: fmt.Sprintf("Failed to calculate end time: %s", err.Error()),
					IsError:        true,
					ToolName:       "calendar_create",
					Status:         "error",
				}, nil
			}

			event, err := client.CreateEvent(title, description, location, start, end)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to create events",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "calendar_create",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to create event: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to create event: %s", errStr),
					IsError:        true,
					ToolName:       "calendar_create",
					Status:         "error",
				}, nil
			}

			content := fmt.Sprintf("Event created successfully!\n\n%s\n\nEvent ID: %s\nLink: %s",
				FormatEvent(event),
				event.ID,
				GetEventURL(event))

			display := fmt.Sprintf("✓ Created: %s", title)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "calendar_create",
				Status:         "done",
			}, nil
		},
	}
}

// NewDeleteTool creates the calendar_delete tool
func NewDeleteTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "calendar_delete",
		Description: "Delete a calendar event by ID",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"event_id": map[string]interface{}{
					"type":        "string",
					"description": "Calendar event ID",
				},
			},
			"required": []string{"event_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			eventID, _ := params["event_id"].(string)

			if eventID == "" {
				return &tools.ToolResult{
					Content:        "Error: 'event_id' parameter is required",
					DisplayContent: "Failed: event ID is required",
					IsError:        true,
					ToolName:       "calendar_delete",
					Status:         "error",
				}, nil
			}

			err := client.DeleteEvent(eventID)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to delete events",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "calendar_delete",
						Status:         "error",
					}, nil
				}

				if strings.Contains(errStr, "404") {
					return &tools.ToolResult{
						Content:        "Event not found. It may have already been deleted.",
						DisplayContent: "Event not found",
						IsError:        true,
						ToolName:       "calendar_delete",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to delete event: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to delete: %s", errStr),
					IsError:        true,
					ToolName:       "calendar_delete",
					Status:         "error",
				}, nil
			}

			return &tools.ToolResult{
				Content:        fmt.Sprintf("Event deleted successfully. ID: %s", eventID),
				DisplayContent: "✓ Event deleted",
				IsError:        false,
				ToolName:       "calendar_delete",
				Status:         "done",
			}, nil
		},
	}
}

// NewTodayTool creates the calendar_today convenience tool
func NewTodayTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "calendar_today",
		Description: "Get today's calendar events",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Get today's time range
			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endOfDay := startOfDay.AddDate(0, 0, 1)

			timeMin := startOfDay.Format(time.RFC3339)
			timeMax := endOfDay.Format(time.RFC3339)

			events, err := client.ListEvents(timeMin, timeMax, 50)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to view calendar",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "calendar_today",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to list events: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to list events: %s", errStr),
					IsError:        true,
					ToolName:       "calendar_today",
					Status:         "error",
				}, nil
			}

			dayStr := now.Format("Monday, January 2, 2006")

			if events.Items == nil || len(events.Items) == 0 {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("No events scheduled for %s.", dayStr),
					DisplayContent: fmt.Sprintf("No events for %s", now.Format("Jan 2")),
					IsError:        false,
					ToolName:       "calendar_today",
					Status:         "done",
				}, nil
			}

			content := formatTodayEvents(events.Items, dayStr)
			display := fmt.Sprintf("📅 %d events today", len(events.Items))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "calendar_today",
				Status:         "done",
			}, nil
		},
	}
}

// formatEventList formats a list of events for display
func formatEventList(events []*Event, days int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Upcoming events (next %d days):\n\n", days))

	for i, event := range events {
		// Format time
		timeStr := FormatEventTime(event.Start, event.End)

		// Title with index
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, event.Summary))
		b.WriteString(fmt.Sprintf("   🕐 %s\n", timeStr))

		// Location if present
		if event.Location != "" {
			b.WriteString(fmt.Sprintf("   📍 %s\n", truncateString(event.Location, 50)))
		}

		// ID
		b.WriteString(fmt.Sprintf("   ID: %s\n", event.ID))

		if i < len(events)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatTodayEvents formats events for today
func formatTodayEvents(events []*Event, dayStr string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Events for %s:\n\n", dayStr))

	for i, event := range events {
		// Format time for today (just show time, not full date)
		timeStr := formatTimeOnly(event.Start, event.End)

		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, event.Summary))
		b.WriteString(fmt.Sprintf("   🕐 %s\n", timeStr))

		if event.Location != "" {
			b.WriteString(fmt.Sprintf("   📍 %s\n", truncateString(event.Location, 50)))
		}

		if i < len(events)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatTimeOnly returns just the time portion for today's events
func formatTimeOnly(start, end *EventDateTime) string {
	if start == nil {
		return "Time TBD"
	}

	// All-day event
	if start.Date != "" {
		return "All day"
	}

	// Timed event
	if start.DateTime != "" {
		startTime, err := time.Parse(time.RFC3339, start.DateTime)
		if err != nil {
			return start.DateTime
		}

		startStr := startTime.Format("3:04 PM")

		if end != nil && end.DateTime != "" {
			endTime, err := time.Parse(time.RFC3339, end.DateTime)
			if err == nil {
				return fmt.Sprintf("%s - %s", startStr, endTime.Format("3:04 PM"))
			}
		}

		return startStr
	}

	return "Time TBD"
}

// truncateString truncates a string to maxLen with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
