package golem

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

// CalDAVHelper provides utilities for CalDAV/iCalendar operations
type CalDAVHelper struct{}

// NewCalDAVHelper creates a new CalDAV helper
func NewCalDAVHelper() *CalDAVHelper {
	return &CalDAVHelper{}
}

// GetSessionCalendar gets the calendar name for a session, or default if not set
func (ch *CalDAVHelper) GetSessionCalendar(session *ChatSession) string {
	if session == nil {
		return ch.GetDefaultCalendar()
	}

	// Check if session has a calendar set
	if calName, exists := session.Variables["calendar"]; exists && calName != "" {
		return calName
	}

	// Fall back to default
	return ch.GetDefaultCalendar()
}

// GetDefaultCalendar returns the default calendar from environment or config
func (ch *CalDAVHelper) GetDefaultCalendar() string {
	calendar := os.Getenv("RADICALE_CALENDAR")
	if calendar == "" {
		calendar = "personal" // Default fallback
	}
	return calendar
}

// BuildCalendarURL constructs the full calendar URL for a specific calendar
func (ch *CalDAVHelper) BuildCalendarURL(calendarName string) string {
	baseURL := os.Getenv("RADICALE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5232"
	}

	user := os.Getenv("RADICALE_USER")
	if user == "" {
		user = "user"
	}

	if calendarName == "" {
		calendarName = ch.GetDefaultCalendar()
	}

	// Ensure trailing slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return fmt.Sprintf("%s%s/%s/", baseURL, user, calendarName)
}

// BuildEventURL constructs the URL for a specific event in a calendar
func (ch *CalDAVHelper) BuildEventURL(calendarName, eventUID string) string {
	calURL := ch.BuildCalendarURL(calendarName)
	return fmt.Sprintf("%s%s.ics", calURL, eventUID)
}

// BuildCalendarQuery creates a REPORT XML payload for calendar queries
func (ch *CalDAVHelper) BuildCalendarQuery(startTime, endTime time.Time) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data>
      <C:comp name="VCALENDAR">
        <C:prop name="VERSION"/>
        <C:comp name="VEVENT">
          <C:prop name="SUMMARY"/>
          <C:prop name="UID"/>
          <C:prop name="DTSTART"/>
          <C:prop name="DTEND"/>
          <C:prop name="DESCRIPTION"/>
          <C:prop name="LOCATION"/>
        </C:comp>
      </C:comp>
    </C:calendar-data>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="%s" end="%s"/>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`,
		startTime.Format("20060102T150405Z"),
		endTime.Format("20060102T150405Z"))
}

// BuildTodayCalendarQuery creates a query for today's events
func (ch *CalDAVHelper) BuildTodayCalendarQuery() string {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return ch.BuildCalendarQuery(startOfDay, endOfDay)
}

// BuildUpcomingCalendarQuery creates a query for upcoming events (next 7 days)
func (ch *CalDAVHelper) BuildUpcomingCalendarQuery() string {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startOfDay.Add(7 * 24 * time.Hour)
	return ch.BuildCalendarQuery(startOfDay, endDate)
}

// BuildICalendarEvent creates an iCalendar event (VEVENT) in iCal format
func (ch *CalDAVHelper) BuildICalendarEvent(summary, description, location string, startTime, endTime time.Time, uid string) string {
	if uid == "" {
		uid = fmt.Sprintf("%d@golem", time.Now().Unix())
	}

	return fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Golem AIML Bot//CalDAV Integration//EN
BEGIN:VEVENT
UID:%s
DTSTAMP:%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
LOCATION:%s
END:VEVENT
END:VCALENDAR`,
		uid,
		time.Now().UTC().Format("20060102T150405Z"),
		startTime.UTC().Format("20060102T150405Z"),
		endTime.UTC().Format("20060102T150405Z"),
		summary,
		description,
		location)
}

// ParseICalendarEvent extracts basic event information from iCalendar format
func (ch *CalDAVHelper) ParseICalendarEvent(icalData string) (map[string]string, error) {
	event := make(map[string]string)
	lines := strings.Split(icalData, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUMMARY:") {
			event["summary"] = strings.TrimPrefix(line, "SUMMARY:")
		} else if strings.HasPrefix(line, "DTSTART:") {
			event["start"] = strings.TrimPrefix(line, "DTSTART:")
		} else if strings.HasPrefix(line, "DTEND:") {
			event["end"] = strings.TrimPrefix(line, "DTEND:")
		} else if strings.HasPrefix(line, "UID:") {
			event["uid"] = strings.TrimPrefix(line, "UID:")
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			event["description"] = strings.TrimPrefix(line, "DESCRIPTION:")
		} else if strings.HasPrefix(line, "LOCATION:") {
			event["location"] = strings.TrimPrefix(line, "LOCATION:")
		}
	}

	return event, nil
}

// MultiStatusResponse represents a WebDAV multistatus response
type MultiStatusResponse struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []Response `xml:"response"`
}

// Response represents a single WebDAV response
type Response struct {
	Href         string       `xml:"href"`
	PropStat     PropStat     `xml:"propstat"`
	CalendarData CalendarData `xml:"calendar-data"`
}

// PropStat represents property status
type PropStat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

// Prop represents properties
type Prop struct {
	GetEtag      string       `xml:"getetag"`
	CalendarData CalendarData `xml:"calendar-data"`
}

// CalendarData represents calendar data
type CalendarData struct {
	Data string `xml:",chardata"`
}

// ParseMultiStatusResponse parses a WebDAV multistatus XML response
func (ch *CalDAVHelper) ParseMultiStatusResponse(xmlData string) ([]map[string]string, error) {
	var multiStatus MultiStatusResponse
	err := xml.Unmarshal([]byte(xmlData), &multiStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multistatus response: %w", err)
	}

	var events []map[string]string
	for _, response := range multiStatus.Responses {
		// Try to get calendar data from different possible locations
		calData := response.PropStat.Prop.CalendarData.Data
		if calData == "" {
			calData = response.CalendarData.Data
		}

		if calData != "" {
			event, err := ch.ParseICalendarEvent(calData)
			if err == nil && len(event) > 0 {
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// FormatEventsList formats a list of events for display
func (ch *CalDAVHelper) FormatEventsList(events []map[string]string) string {
	if len(events) == 0 {
		return "No events found"
	}

	var result strings.Builder
	for i, event := range events {
		if i > 0 {
			result.WriteString("; ")
		}
		result.WriteString(fmt.Sprintf("%s", event["summary"]))
		if start := event["start"]; start != "" {
			// Parse and format the start time
			if t, err := time.Parse("20060102T150405Z", start); err == nil {
				result.WriteString(fmt.Sprintf(" at %s", t.Format("3:04 PM")))
			}
		}
	}
	return result.String()
}

// GetBasicAuth creates a Basic Authentication header value
func (ch *CalDAVHelper) GetBasicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// GetRadicaleAuthFromEnv creates Basic Auth from environment variables
func (ch *CalDAVHelper) GetRadicaleAuthFromEnv() string {
	user := os.Getenv("RADICALE_USER")
	password := os.Getenv("RADICALE_PASSWORD")
	if user == "" || password == "" {
		return ""
	}
	return ch.GetBasicAuth(user, password)
}

// ParseEventDetails parses natural language event details
// Example: "Meeting on tomorrow at 2pm" or "Lunch with John on Friday at noon"
func (ch *CalDAVHelper) ParseEventDetails(input string) (summary string, startTime time.Time, endTime time.Time, err error) {
	// This is a simplified parser - a real implementation would use more sophisticated NLP
	summary = input

	// Default to 1 hour duration
	startTime = time.Now().Add(1 * time.Hour)
	endTime = startTime.Add(1 * time.Hour)

	// Look for time indicators
	lower := strings.ToLower(input)

	// Parse "tomorrow"
	if strings.Contains(lower, "tomorrow") {
		tomorrow := time.Now().Add(24 * time.Hour)
		startTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, tomorrow.Location())
		endTime = startTime.Add(1 * time.Hour)
	}

	// Parse "today"
	if strings.Contains(lower, "today") {
		today := time.Now()
		startTime = time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, today.Location())
		endTime = startTime.Add(1 * time.Hour)
	}

	// Parse specific times (simplified)
	if strings.Contains(lower, "at 2pm") || strings.Contains(lower, "at 2 pm") {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 14, 0, 0, 0, startTime.Location())
		endTime = startTime.Add(1 * time.Hour)
	} else if strings.Contains(lower, "at noon") {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 12, 0, 0, 0, startTime.Location())
		endTime = startTime.Add(1 * time.Hour)
	}

	return summary, startTime, endTime, nil
}
