package golem

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCalDAVHelper_BuildCalendarQuery(t *testing.T) {
	ch := NewCalDAVHelper()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	query := ch.BuildCalendarQuery(start, end)

	// Verify XML structure
	if !strings.Contains(query, "<?xml version=\"1.0\"") {
		t.Error("Query should contain XML declaration")
	}
	if !strings.Contains(query, "<C:calendar-query") {
		t.Error("Query should contain calendar-query element")
	}
	if !strings.Contains(query, "<C:time-range") {
		t.Error("Query should contain time-range element")
	}
	if !strings.Contains(query, "20240101T000000Z") {
		t.Error("Query should contain formatted start time")
	}
	if !strings.Contains(query, "20240102T000000Z") {
		t.Error("Query should contain formatted end time")
	}
}

func TestCalDAVHelper_BuildTodayCalendarQuery(t *testing.T) {
	ch := NewCalDAVHelper()

	query := ch.BuildTodayCalendarQuery()

	// Verify basic structure
	if !strings.Contains(query, "<C:calendar-query") {
		t.Error("Today query should contain calendar-query element")
	}
	if !strings.Contains(query, "<C:time-range") {
		t.Error("Today query should contain time-range element")
	}
}

func TestCalDAVHelper_BuildUpcomingCalendarQuery(t *testing.T) {
	ch := NewCalDAVHelper()

	query := ch.BuildUpcomingCalendarQuery()

	// Verify basic structure
	if !strings.Contains(query, "<C:calendar-query") {
		t.Error("Upcoming query should contain calendar-query element")
	}
	if !strings.Contains(query, "<C:time-range") {
		t.Error("Upcoming query should contain time-range element")
	}
}

func TestCalDAVHelper_BuildICalendarEvent(t *testing.T) {
	ch := NewCalDAVHelper()

	start := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)

	ical := ch.BuildICalendarEvent(
		"Team Meeting",
		"Weekly team sync",
		"Conference Room A",
		start,
		end,
		"test-event-123",
	)

	tests := []struct {
		name     string
		contains string
	}{
		{"VCALENDAR header", "BEGIN:VCALENDAR"},
		{"Version", "VERSION:2.0"},
		{"VEVENT header", "BEGIN:VEVENT"},
		{"UID", "UID:test-event-123"},
		{"Summary", "SUMMARY:Team Meeting"},
		{"Description", "DESCRIPTION:Weekly team sync"},
		{"Location", "LOCATION:Conference Room A"},
		{"Start time", "DTSTART:20240115T140000Z"},
		{"End time", "DTEND:20240115T150000Z"},
		{"VEVENT footer", "END:VEVENT"},
		{"VCALENDAR footer", "END:VCALENDAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(ical, tt.contains) {
				t.Errorf("iCalendar should contain %q", tt.contains)
			}
		})
	}
}

func TestCalDAVHelper_ParseICalendarEvent(t *testing.T) {
	ch := NewCalDAVHelper()

	icalData := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:test-123
DTSTART:20240115T140000Z
DTEND:20240115T150000Z
SUMMARY:Test Meeting
DESCRIPTION:Test description
LOCATION:Room 101
END:VEVENT
END:VCALENDAR`

	event, err := ch.ParseICalendarEvent(icalData)
	if err != nil {
		t.Fatalf("ParseICalendarEvent failed: %v", err)
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"uid", "test-123"},
		{"summary", "Test Meeting"},
		{"start", "20240115T140000Z"},
		{"end", "20240115T150000Z"},
		{"description", "Test description"},
		{"location", "Room 101"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if event[tt.field] != tt.expected {
				t.Errorf("Expected %s = %q, got %q", tt.field, tt.expected, event[tt.field])
			}
		})
	}
}

func TestCalDAVHelper_FormatEventsList(t *testing.T) {
	ch := NewCalDAVHelper()

	tests := []struct {
		name     string
		events   []map[string]string
		expected string
	}{
		{
			name:     "Empty list",
			events:   []map[string]string{},
			expected: "No events found",
		},
		{
			name: "Single event",
			events: []map[string]string{
				{"summary": "Meeting", "start": "20240115T140000Z"},
			},
			expected: "Meeting at 2:00 PM",
		},
		{
			name: "Multiple events",
			events: []map[string]string{
				{"summary": "Morning Standup", "start": "20240115T090000Z"},
				{"summary": "Lunch", "start": "20240115T120000Z"},
			},
			expected: "Morning Standup at 9:00 AM; Lunch at 12:00 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ch.FormatEventsList(tt.events)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCalDAVHelper_GetBasicAuth(t *testing.T) {
	ch := NewCalDAVHelper()

	auth := ch.GetBasicAuth("user", "pass")

	if !strings.HasPrefix(auth, "Basic ") {
		t.Error("Auth should start with 'Basic '")
	}

	// The base64 encoding of "user:pass" is "dXNlcjpwYXNz"
	expected := "Basic dXNlcjpwYXNz"
	if auth != expected {
		t.Errorf("Expected %q, got %q", expected, auth)
	}
}

func TestCalDAVHelper_ParseEventDetails(t *testing.T) {
	ch := NewCalDAVHelper()

	tests := []struct {
		name          string
		input         string
		checkSummary  bool
		expectedWords []string
	}{
		{
			name:          "Simple event",
			input:         "Team meeting",
			checkSummary:  true,
			expectedWords: []string{"Team", "meeting"},
		},
		{
			name:          "Event with tomorrow",
			input:         "Lunch tomorrow at 2pm",
			checkSummary:  true,
			expectedWords: []string{"Lunch", "tomorrow"},
		},
		{
			name:          "Event with specific time",
			input:         "Doctor appointment at noon",
			checkSummary:  true,
			expectedWords: []string{"Doctor", "appointment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, startTime, endTime, err := ch.ParseEventDetails(tt.input)

			if err != nil {
				t.Errorf("ParseEventDetails failed: %v", err)
			}

			if tt.checkSummary {
				for _, word := range tt.expectedWords {
					if !strings.Contains(summary, word) {
						t.Errorf("Summary should contain %q, got %q", word, summary)
					}
				}
			}

			// Verify start time is before end time
			if !startTime.Before(endTime) {
				t.Errorf("Start time should be before end time: start=%v, end=%v", startTime, endTime)
			}

			// Verify duration is reasonable (should be 1 hour by default)
			duration := endTime.Sub(startTime)
			if duration != time.Hour {
				t.Errorf("Expected 1 hour duration, got %v", duration)
			}
		})
	}
}

func TestCalDAVHelper_ParseMultiStatusResponse(t *testing.T) {
	ch := NewCalDAVHelper()

	// Sample multistatus response
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/user/calendar/event1.ics</href>
    <propstat>
      <prop>
        <getetag>"12345"</getetag>
        <C:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:event1
SUMMARY:Test Event 1
DTSTART:20240115T100000Z
DTEND:20240115T110000Z
END:VEVENT
END:VCALENDAR</C:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

	events, err := ch.ParseMultiStatusResponse(xmlData)
	if err != nil {
		t.Fatalf("ParseMultiStatusResponse failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event["uid"] != "event1" {
		t.Errorf("Expected uid=event1, got %q", event["uid"])
	}
	if event["summary"] != "Test Event 1" {
		t.Errorf("Expected summary=Test Event 1, got %q", event["summary"])
	}
}

func TestCalDAVHelper_GetDefaultCalendar(t *testing.T) {
	ch := NewCalDAVHelper()

	// Save original env var
	origCal := os.Getenv("RADICALE_CALENDAR")
	defer os.Setenv("RADICALE_CALENDAR", origCal)

	// Test with env var set
	os.Setenv("RADICALE_CALENDAR", "testcal")
	cal := ch.GetDefaultCalendar()
	if cal != "testcal" {
		t.Errorf("Expected 'testcal', got %q", cal)
	}

	// Test with env var empty - should use default
	os.Setenv("RADICALE_CALENDAR", "")
	cal = ch.GetDefaultCalendar()
	if cal != "personal" {
		t.Errorf("Expected 'personal' as default, got %q", cal)
	}
}

func TestCalDAVHelper_GetSessionCalendar(t *testing.T) {
	ch := NewCalDAVHelper()

	// Save original env var
	origCal := os.Getenv("RADICALE_CALENDAR")
	defer os.Setenv("RADICALE_CALENDAR", origCal)
	os.Setenv("RADICALE_CALENDAR", "default_cal")

	tests := []struct {
		name             string
		session          *ChatSession
		expectedCalendar string
	}{
		{
			name:             "Nil session uses default",
			session:          nil,
			expectedCalendar: "default_cal",
		},
		{
			name:             "Session without calendar variable uses default",
			session:          &ChatSession{Variables: make(map[string]string)},
			expectedCalendar: "default_cal",
		},
		{
			name: "Session with calendar variable",
			session: &ChatSession{
				Variables: map[string]string{"calendar": "work"},
			},
			expectedCalendar: "work",
		},
		{
			name: "Session with empty calendar variable uses default",
			session: &ChatSession{
				Variables: map[string]string{"calendar": ""},
			},
			expectedCalendar: "default_cal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cal := ch.GetSessionCalendar(tt.session)
			if cal != tt.expectedCalendar {
				t.Errorf("Expected %q, got %q", tt.expectedCalendar, cal)
			}
		})
	}
}

func TestCalDAVHelper_BuildCalendarURL(t *testing.T) {
	ch := NewCalDAVHelper()

	// Save original env vars
	origURL := os.Getenv("RADICALE_URL")
	origUser := os.Getenv("RADICALE_USER")
	defer func() {
		os.Setenv("RADICALE_URL", origURL)
		os.Setenv("RADICALE_USER", origUser)
	}()

	os.Setenv("RADICALE_URL", "http://localhost:5232")
	os.Setenv("RADICALE_USER", "testuser")

	tests := []struct {
		name         string
		calendarName string
		expectedURL  string
	}{
		{
			name:         "Personal calendar",
			calendarName: "personal",
			expectedURL:  "http://localhost:5232/testuser/personal/",
		},
		{
			name:         "Work calendar",
			calendarName: "work",
			expectedURL:  "http://localhost:5232/testuser/work/",
		},
		{
			name:         "Empty calendar name uses default",
			calendarName: "",
			expectedURL:  "http://localhost:5232/testuser/personal/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := ch.BuildCalendarURL(tt.calendarName)
			if url != tt.expectedURL {
				t.Errorf("Expected %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

func TestCalDAVHelper_BuildEventURL(t *testing.T) {
	ch := NewCalDAVHelper()

	// Save original env vars
	origURL := os.Getenv("RADICALE_URL")
	origUser := os.Getenv("RADICALE_USER")
	defer func() {
		os.Setenv("RADICALE_URL", origURL)
		os.Setenv("RADICALE_USER", origUser)
	}()

	os.Setenv("RADICALE_URL", "http://localhost:5232")
	os.Setenv("RADICALE_USER", "testuser")

	url := ch.BuildEventURL("work", "event-123")
	expected := "http://localhost:5232/testuser/work/event-123.ics"
	if url != expected {
		t.Errorf("Expected %q, got %q", expected, url)
	}
}
