# CalDAV/Radicale Integration for Golem

## Overview

This integration allows Golem AIML bots to interact with CalDAV calendar servers, specifically Radicale. You can list events, create new calendar entries, update existing events, and delete events through natural language AIML patterns.

## What is CalDAV?

**CalDAV** (Calendaring Extensions to WebDAV) is an Internet standard allowing client access to calendar information on a remote server. **Radicale** is a lightweight, open-source CalDAV/CardDAV server written in Python.

## Features

- ✅ List today's events
- ✅ List upcoming events (next 7 days)
- ✅ Create new calendar events
- ✅ Get specific event details
- ✅ Delete calendar events
- ✅ Natural language event parsing
- ✅ iCalendar (RFC 5545) format support
- ✅ WebDAV multistatus response parsing

## Prerequisites

### 1. Install Radicale

```bash
# Using pip
pip install radicale

# Using your package manager (Debian/Ubuntu)
sudo apt install radicale

# Using Docker
docker run -d -p 5232:5232 tomsquest/docker-radicale
```

### 2. Start Radicale Server

```bash
# Basic start
radicale

# With custom port
radicale --port 5232

# With authentication
radicale --auth-type htpasswd --htpasswd-file /path/to/users
```

Default URL: `http://localhost:5232`

### 3. Create a Calendar

1. Access Radicale web interface: `http://localhost:5232`
2. Log in (if authentication is enabled)
3. Create a new calendar (e.g., "personal")

## Setup

### 1. Environment Variables

Set the following environment variables:

```bash
export RADICALE_URL="http://localhost:5232"
export RADICALE_USER="your-username"
export RADICALE_PASSWORD="your-password"
export RADICALE_CALENDAR="personal"
```

For Docker deployments, add these to your `.env` file or docker-compose.yml.

### 2. Load Configuration

The CalDAV configuration is stored in `testdata/caldav-config.properties`. Ensure this file is loaded when initializing Golem:

```go
package main

import (
	"github.com/helix90/golem/pkg/golem"
)

func main() {
	g := golem.New(true)

	// Load CalDAV configuration
	kb := golem.NewAIMLKnowledgeBase()
	g.LoadPropertiesFromFile("testdata/caldav-config.properties", kb)

	// Load CalDAV AIML templates
	g.LoadAIMLFile("testdata/caldav-examples.aiml", kb)

	g.SetKnowledgeBase(kb)

	// ... rest of your code
}
```

### 3. Verify Configuration

Test that CalDAV is configured correctly:

```go
session := g.CreateSession("test")
response, _ := g.ProcessInput("is caldav enabled", session)
fmt.Println(response)
// Expected: "Yes, CalDAV integration is enabled..."
```

## Usage

### Multiple Calendars

Each user session can have its own calendar. The system supports switching between multiple calendars per user.

#### Calendar Selection

```
User: Use calendar work
Bot:  Switched to calendar: work

User: Use calendar personal
Bot:  Switched to calendar: personal

User: What calendar am I using?
Bot:  You are currently using the personal calendar.
```

#### How It Works

- **Session Variable**: Each session stores its calendar selection in the `calendar` session variable
- **Fallback Chain**: Session calendar → `RADICALE_CALENDAR` environment variable → Default ("personal")
- **Per-User State**: Different users can use different calendars simultaneously
- **Persistence**: Calendar selection persists throughout the session

#### Setting Up Multiple Calendars

1. Create multiple calendars in Radicale (e.g., "personal", "work", "family")
2. Users can switch between them using natural language:
   - "Use calendar work"
   - "Switch to calendar personal"
   - "Set my calendar to family"

All calendar operations (list, create, delete) use the currently selected calendar.

### List Today's Events

```
User: What is on my calendar today?
Bot:  Here are your events for today: Team Meeting at 2:00 PM; Lunch with Sarah at 12:00 PM

User: Show my calendar
Bot:  Here are your events for today: ...
```

### List Upcoming Events

```
User: What is on my calendar?
Bot:  Here are your upcoming events: Project Review on Monday at 9:00 AM; ...

User: Show my upcoming events
Bot:  Here are your upcoming events: ...
```

### Create Events

```
User: Create event Team Meeting
Bot:  Event created successfully: Team Meeting

User: Schedule Lunch tomorrow at noon
Bot:  Event created successfully: Lunch tomorrow at noon

User: Create event called Project Review on Monday at 9am
Bot:  Created event: "Project Review" scheduled for Monday at 9am
```

### Get Event Details

```
User: Get event event-123
Bot:  Event details: [event information]
```

### Delete Events

```
User: Delete event event-123
Bot:  Event deleted successfully

User: Cancel event meeting-456
Bot:  Event deleted successfully
```

### Calendar Management

```
User: Use calendar work
Bot:  Switched to calendar: work

User: What calendar am I using?
Bot:  You are currently using the work calendar.

User: Switch to calendar personal
Bot:  Switched to calendar: personal
```

### Get Help

```
User: Calendar help
Bot:  I can help you manage your calendar. Try these commands:
      - "What is on my calendar today"
      - "Show my upcoming events"
      - "Create event called Meeting on tomorrow at 2pm"
      - "Delete event [event-id]"
      - "Use calendar [name]" - Switch to a different calendar
      - "What calendar am I using" - Show current calendar
      - "Calendar status"
```

## Architecture

### SRAIX Configuration

The integration uses Golem's SRAIX (external service integration) system. Five SRAIX services are configured:

1. **caldav_list** - List upcoming events (REPORT method)
2. **caldav_today** - List today's events (REPORT method)
3. **caldav_create** - Create new events (PUT method)
4. **caldav_get** - Get specific event (GET method)
5. **caldav_delete** - Delete events (DELETE method)

### CalDAV Helper

The `CalDAVHelper` class provides utilities for:

- Building CalDAV XML queries (calendar-query REPORT)
- Creating iCalendar events (RFC 5545 format)
- Parsing iCalendar data
- Parsing WebDAV multistatus responses
- Natural language event parsing
- Authentication (Basic Auth)

### AIML Templates

The `caldav-examples.aiml` file contains patterns for natural language calendar interactions, using:

- `<sraix service="caldav_*">` tags to call CalDAV services
- `<think>` and `<set>` for variable management
- `<condition>` for response handling
- `<srai>` for pattern reuse

## API Reference

### CalDAVHelper Methods

```go
// Calendar management
GetSessionCalendar(session *ChatSession) string
GetDefaultCalendar() string
BuildCalendarURL(calendarName string) string
BuildEventURL(calendarName, eventUID string) string

// Create calendar queries
BuildCalendarQuery(startTime, endTime time.Time) string
BuildTodayCalendarQuery() string
BuildUpcomingCalendarQuery() string

// Create events
BuildICalendarEvent(summary, description, location string, startTime, endTime time.Time, uid string) string

// Parse responses
ParseICalendarEvent(icalData string) (map[string]string, error)
ParseMultiStatusResponse(xmlData string) ([]map[string]string, error)
FormatEventsList(events []map[string]string) string

// Authentication
GetBasicAuth(username, password string) string
GetRadicaleAuthFromEnv() string

// Natural language parsing
ParseEventDetails(input string) (summary string, startTime, endTime time.Time, err error)
```

## Configuration Properties

All CalDAV services use the `sraix.{service}.{property}` format:

| Property | Description | Example |
|----------|-------------|---------|
| `urltemplate` | URL with placeholders | `${RADICALE_URL}/${RADICALE_USER}/${RADICALE_CALENDAR}/` |
| `method` | HTTP method | `REPORT`, `PUT`, `GET`, `DELETE` |
| `timeout` | Request timeout (seconds) | `30` |
| `responseformat` | Response format | `xml`, `text` |
| `responsepath` | JSON/XML path to extract | `multistatus.response` |
| `fallback` | Fallback response on error | `"Unable to retrieve events"` |
| `header.*` | Custom HTTP headers | `header.Content-Type`, `header.Authorization` |

### Environment Variable Substitution

Properties support `${ENV_VAR}` syntax for environment variables:

- `${RADICALE_URL}` - Server URL
- `${RADICALE_USER}` - Username
- `${RADICALE_PASSWORD}` - Password
- `${RADICALE_CALENDAR}` - Calendar name
- `${RADICALE_AUTH}` - Pre-computed Basic Auth header

## CalDAV Protocol Details

### REPORT Method (List Events)

```xml
<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <C:calendar-data/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="20240101T000000Z" end="20240108T000000Z"/>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>
```

### PUT Method (Create Event)

```
PUT /username/calendar/event-uid.ics HTTP/1.1
Content-Type: text/calendar; charset=utf-8

BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Golem AIML Bot//CalDAV Integration//EN
BEGIN:VEVENT
UID:event-123
DTSTART:20240115T140000Z
DTEND:20240115T150000Z
SUMMARY:Team Meeting
END:VEVENT
END:VCALENDAR
```

## Testing

Run the CalDAV helper tests:

```bash
# Run all CalDAV tests
go test ./pkg/golem -run CalDAV -v

# Run specific test
go test ./pkg/golem -run TestCalDAVHelper_BuildCalendarQuery -v

# Run with coverage
go test ./pkg/golem -run CalDAV -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

### Connection Issues

**Problem**: Cannot connect to Radicale server

**Solutions**:
- Verify `RADICALE_URL` is correct
- Check Radicale is running: `curl http://localhost:5232`
- Verify firewall settings
- Check authentication credentials

### Authentication Errors

**Problem**: 401 Unauthorized responses

**Solutions**:
- Verify `RADICALE_USER` and `RADICALE_PASSWORD`
- Check Radicale authentication settings
- Test credentials manually: `curl -u user:pass http://localhost:5232`

### Empty Event Lists

**Problem**: "No events found" when events exist

**Solutions**:
- Verify calendar name (`RADICALE_CALENDAR`)
- Check time range in queries
- Verify events exist: Access web interface
- Enable verbose logging to see XML responses

### Event Creation Fails

**Problem**: Events not being created

**Solutions**:
- Check calendar permissions
- Verify iCalendar format is valid
- Ensure UID is unique
- Check Content-Type header is set correctly

### Debugging

Enable verbose logging in Golem:

```go
g := golem.New(true) // Enable verbose mode
```

This will log:
- SRAIX requests and responses
- CalDAV XML payloads
- HTTP status codes
- Error messages

## Security Considerations

1. **HTTPS**: Use HTTPS in production (`https://` in `RADICALE_URL`)
2. **Authentication**: Always use authentication for Radicale
3. **Environment Variables**: Never commit credentials to version control
4. **Access Control**: Limit calendar access to authorized users
5. **Input Validation**: Validate event data before creating events

## Advanced Usage

### Custom Event Parsing

Extend `ParseEventDetails()` for more sophisticated natural language processing:

```go
helper := golem.NewCalDAVHelper()
summary, start, end, err := helper.ParseEventDetails("Team standup every Monday at 9am")
// Implement recurring event logic
```

### Multi-Calendar Usage

The CalDAV integration supports multiple calendars per user through session-based calendar selection:

```go
// In your bot implementation
session := g.CreateSession("user123")

// User switches to work calendar
g.ProcessInput("use calendar work", session)
// Calendar stored in session.Variables["calendar"]

// All subsequent calendar operations use the work calendar
g.ProcessInput("what is on my calendar today", session)
g.ProcessInput("create event Team Meeting", session)

// User switches to personal calendar
g.ProcessInput("use calendar personal", session)
// Now all operations use the personal calendar
```

The `GetSessionCalendar()` helper method manages the calendar selection:

```go
helper := golem.NewCalDAVHelper()

// With a session that has calendar set to "work"
calendar := helper.GetSessionCalendar(session) // Returns "work"

// With a session that has no calendar set
calendar := helper.GetSessionCalendar(session) // Returns RADICALE_CALENDAR env var or "personal"

// Build URLs for specific calendars
workURL := helper.BuildCalendarURL("work")
personalURL := helper.BuildCalendarURL("personal")
```

### Custom Response Formatting

Override `FormatEventsList()` for custom formatting:

```go
helper := golem.NewCalDAVHelper()
events := []map[string]string{...}
formatted := helper.FormatEventsList(events)
// Add custom formatting logic
```

## References

- [CalDAV Specification (RFC 4791)](https://tools.ietf.org/html/rfc4791)
- [iCalendar Specification (RFC 5545)](https://tools.ietf.org/html/rfc5545)
- [Radicale Documentation](https://radicale.org/)
- [WebDAV Specification (RFC 4918)](https://tools.ietf.org/html/rfc4918)

## Support

For issues or questions:
1. Check this documentation
2. Review test files for examples
3. Enable verbose logging for debugging
4. Check Radicale server logs
5. Report issues on GitHub

## License

This CalDAV integration is part of the Golem project and follows the same MIT license.
