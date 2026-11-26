package golem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The caldav SRAIX handler talks to a CalDAV server (such as Radicale) for a
// service configured with handler "caldav". The service's URL template is
// the calendar collection, e.g. ${RADICALE_URL}/${RADICALE_USER}/{calendar}/,
// and its operation is one of:
//
//	today    - today's events, e.g. "Standup at 9:00 AM; Lunch at 12:00 PM"
//	upcoming - events from now to the end of the seventh day from today
//	range    - events on the day or in the period the input names, e.g.
//	           "tomorrow", "Friday", "December 3", "next week", "this
//	           weekend", "the next 3 days"; sets the calendar_range
//	           predicate to a label such as "on Friday, November 27", or to
//	           "unknown" (and sends no request) when it can't read the input
//	create   - creates an event from text such as "Lunch tomorrow at noon"
//	get      - the details of the event whose title (or UID) the input names
//	delete   - deletes the event whose title (or UID) the input names
//
// today and upcoming return "" when there are no events. Every failure,
// including get or delete finding no event or more than one, is an error,
// so that the tag falls back. Times are shown in the bot's local time zone.

// caldavOperations are the operations of the caldav handler
var caldavOperations = map[string]bool{"today": true, "upcoming": true, "range": true, "create": true, "get": true, "delete": true, "move": true}

// caldavSearchPast and caldavSearchFuture bound the events get and delete
// search for the one the input names
const (
	caldavSearchPast   = 30 * 24 * time.Hour
	caldavSearchFuture = 365 * 24 * time.Hour
)

// calendarEvent is an event (VEVENT) on a calendar
type calendarEvent struct {
	href        string // the event's resource, from a calendar query
	uid         string
	summary     string
	location    string
	description string
	start, end  time.Time
	allDay      bool // start and end are dates; end is exclusive
	repeats     bool
	// recurrence is how an event created from text repeats; nil for one
	// that doesn't (events read from a calendar only set repeats)
	recurrence *eventRecurrence

	// Read from a calendar: how the event repeats, as written
	rrule        string         // the RRULE value; "" for none
	exdates      []time.Time    // occurrences left out
	recurrenceID time.Time      // for an occurrence that was changed: the one it replaces
	zone         *time.Location // the zone DTSTART is written in, which occurrences keep
	raw          string         // the iCalendar data it came from

	// Parsed from text: whether it named a date and a time
	dated, timed bool
}

// caldavClient makes the requests of one caldav operation
type caldavClient struct {
	sm          *SRAIXManager
	config      *SRAIXConfig
	ctx         context.Context
	params      map[string]string
	calendarURL *url.URL
	logURL      string // calendarURL with secrets masked
}

// sendCalDAV performs a caldav service's operation
func (sm *SRAIXManager) sendCalDAV(config *SRAIXConfig, input string, params map[string]string) (sraixResponse, error) {
	calendarURL, logURL, err := sm.substituteURLTemplate(config.URLTemplate, input, params, config.Headers)
	if err != nil {
		return sraixResponse{}, err
	}
	if !strings.HasSuffix(calendarURL, "/") {
		calendarURL += "/"
		logURL += "/"
	}
	parsed, err := url.Parse(calendarURL)
	if err != nil {
		return sraixResponse{}, fmt.Errorf("invalid calendar URL: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()
	c := &caldavClient{sm: sm, config: config, ctx: ctx, params: params, calendarURL: parsed, logURL: logURL}

	now := sm.now()
	today := startOfDay(now)
	input = strings.TrimSpace(input)
	switch config.Operation {
	case "today":
		events, err := c.query(today, today.AddDate(0, 0, 1))
		if err != nil {
			return sraixResponse{}, err
		}
		return eventListResponse(events, now, false, nil), nil
	case "upcoming":
		events, err := c.query(now, today.AddDate(0, 0, 8))
		if err != nil {
			return sraixResponse{}, err
		}
		return eventListResponse(events, now, true, nil), nil
	case "range":
		start, end, singleDay, label, err := parseDateRange(input, now)
		if err != nil {
			return sraixResponse{set: map[string]string{"calendar_range": "unknown"}}, nil
		}
		events, err := c.query(start, end)
		if err != nil {
			return sraixResponse{}, err
		}
		// The label names the day, so a single day's events show only
		// their times
		if singleDay {
			return eventListResponse(events, start, false, map[string]string{"calendar_range": label}), nil
		}
		return eventListResponse(events, now, true, map[string]string{"calendar_range": label}), nil
	case "create":
		event, err := parseEventText(input, now)
		if err != nil {
			return sraixResponse{}, err
		}
		// Nothing says when: ask rather than guess today
		if !event.dated && !event.timed && event.recurrence == nil {
			return sraixResponse{text: "NEEDS WHEN", set: map[string]string{
				"calendar_event_title": event.summary, "calendar_event_when": "missing"}}, nil
		}
		if event.uid, err = newEventUID(); err != nil {
			return sraixResponse{}, err
		}
		if err := c.put(event, now); err != nil {
			return sraixResponse{}, err
		}
		when := eventWhen(event, now, true)
		if event.recurrence != nil {
			when = event.recurrence.describe(event, now)
		}
		return sraixResponse{text: event.summary + " " + when, set: map[string]string{
			"calendar_event_title": event.summary, "calendar_event_when": when, "calendar_event_uid": event.uid}}, nil
	case "get":
		event, err := c.find(input, now)
		if several, ok := err.(severalEventsError); ok {
			return several.response(now, params["match_action"]), nil
		}
		if _, ok := err.(noEventError); ok {
			return sraixResponse{text: "Event not found"}, nil
		}
		if err != nil {
			return sraixResponse{}, err
		}
		return sraixResponse{text: eventDetails(event, now), set: eventPredicates(event, now)}, nil
	case "delete":
		event, err := c.find(input, now)
		if err != nil {
			return sraixResponse{}, err
		}
		if err := c.delete(event); err != nil {
			return sraixResponse{}, err
		}
		return sraixResponse{text: event.summary + " " + eventWhen(event, now, true), set: eventPredicates(event, now)}, nil
	case "move":
		// The event is the param "event", which may name one occurrence
		// ("yoga on Monday"); the input says when to. scope is "one" or
		// "all" for a repeating event, or "" to ask.
		event, named, err := c.findOccurrence(strings.TrimSpace(params["event"]), now)
		if several, ok := err.(severalEventsError); ok {
			return several.response(now, params["match_action"]), nil
		}
		if _, ok := err.(noEventError); ok {
			return sraixResponse{text: "NOT FOUND"}, nil
		}
		if err != nil {
			return sraixResponse{}, err
		}
		set := eventPredicates(event, now)
		scope := strings.ToLower(strings.TrimSpace(params["scope"]))
		if event.repeats && scope == "" && named {
			scope = "one"
		}
		if event.repeats && scope != "one" && scope != "all" {
			return sraixResponse{text: "REPEATING", set: set}, nil
		}
		// Moving every one is measured from an occurrence where the series
		// puts it, not one that was moved on its own
		if event.repeats && scope == "all" && !event.recurrenceID.IsZero() {
			if regular, found, err := c.nextRegularOccurrence(event, now); err != nil {
				return sraixResponse{}, err
			} else if found {
				event = regular
			}
		}
		moved, ok, err := moveEvent(event, input, now)
		if err != nil {
			return sraixResponse{}, err
		}
		if !ok {
			return sraixResponse{text: "NO WHEN", set: set}, nil
		}
		if event.repeats {
			err = c.updateSeries(event, moved, scope, now)
			if err == errCantMoveSeries {
				return sraixResponse{text: "CANT MOVE ALL", set: set}, nil
			}
		} else {
			err = c.update(event, moved, now)
		}
		if err != nil {
			return sraixResponse{}, err
		}
		set = eventPredicates(moved, now)
		set["calendar_moved_scope"] = scope
		return sraixResponse{text: moved.summary + " " + set["calendar_event_when"], set: set}, nil
	}
	return sraixResponse{}, fmt.Errorf("unknown caldav operation %q", config.Operation)
}

// do sends a request to the calendar, or to one of its events, and returns
// the response status and body
func (c *caldavClient) do(method, resource, body string, headers map[string]string) (int, []byte, error) {
	target := c.calendarURL
	logTarget := c.logURL
	if resource != "" {
		ref, err := url.Parse(resource)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid event reference %q: %v", resource, err)
		}
		target = c.calendarURL.ResolveReference(ref)
		logTarget = target.Path
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.ctx, method, target.String(), bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create CalDAV request: %v", err)
	}
	if err := c.sm.setHeaders(req, c.config, "", c.params); err != nil {
		return 0, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.sm.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("CalDAV request failed: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSRAIXResponseSize+1))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read CalDAV response: %v", err)
	}
	if len(data) > maxSRAIXResponseSize {
		return 0, nil, fmt.Errorf("CalDAV response exceeds %d bytes", maxSRAIXResponseSize)
	}
	if c.sm.verbose {
		c.sm.logger.Printf("CalDAV %s %s (%s): %s, %d bytes", method, logTarget, c.config.Name, resp.Status, len(data))
	}
	return resp.StatusCode, data, nil
}

// caldavTime formats a time as a UTC date-time for CalDAV and iCalendar
func caldavTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// query returns the events between start and end, sorted by start time.
// Recurring events are expanded into their occurrences by the server.
func (c *caldavClient) query(start, end time.Time) ([]calendarEvent, error) {
	body := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="%[1]s" end="%[2]s"/>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`, caldavTime(start.AddDate(0, 0, -1)), caldavTime(end.AddDate(0, 0, 1)))

	status, data, err := c.do("REPORT", "", body, map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
		"Depth":        "1",
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusMultiStatus {
		return nil, fmt.Errorf("calendar query failed with status %d", status)
	}
	events, err := parseMultiStatus(data, start.Location())
	if err != nil {
		return nil, err
	}
	events = expandEvents(events, start, end)
	sort.SliceStable(events, func(i, j int) bool { return events[i].start.Before(events[j].start) })
	return events, nil
}

// find returns the one event within the search window whose UID is the
// input, or whose title is the input (ignoring case), or else whose title
// contains it. Finding none, or several, is an error.
func (c *caldavClient) find(input string, now time.Time) (calendarEvent, error) {
	if input == "" {
		return calendarEvent{}, fmt.Errorf("no event named")
	}
	events, err := c.query(now.Add(-caldavSearchPast), now.Add(caldavSearchFuture))
	if err != nil {
		return calendarEvent{}, err
	}

	// An event that repeats appears once per occurrence; the first upcoming
	// one stands for it
	byHref := make(map[string]calendarEvent)
	var hrefs []string
	for _, event := range events {
		existing, seen := byHref[event.href]
		if !seen {
			hrefs = append(hrefs, event.href)
		}
		if !seen || (existing.end.Before(now) && !event.end.Before(now)) {
			byHref[event.href] = event
		}
	}

	var candidates []calendarEvent
	for _, href := range hrefs {
		candidates = append(candidates, byHref[href])
	}
	return matchEvent(candidates, input)
}

// matchEvent chooses the one event input names: by UID, then title, then
// part of the title, trying each of eventQueries in turn
func matchEvent(candidates []calendarEvent, input string) (calendarEvent, error) {
	for _, query := range eventQueries(input) {
		lowered := strings.ToLower(query)
		for _, match := range []func(calendarEvent) bool{
			func(e calendarEvent) bool { return e.uid == query },
			func(e calendarEvent) bool { return strings.ToLower(e.summary) == lowered },
			func(e calendarEvent) bool { return strings.Contains(strings.ToLower(e.summary), lowered) },
		} {
			var found []calendarEvent
			for _, e := range candidates {
				if match(e) {
					found = append(found, e)
				}
			}
			switch len(found) {
			case 0:
				continue
			case 1:
				return found[0], nil
			default:
				return calendarEvent{}, severalEventsError{query: query, events: found}
			}
		}
	}
	return calendarEvent{}, noEventError{input}
}

// noEventError is find's error when no event matches
type noEventError struct{ query string }

func (e noEventError) Error() string { return fmt.Sprintf("no event matches %q", e.query) }

// put creates an event, failing rather than replacing one with its UID
func (c *caldavClient) put(event calendarEvent, now time.Time) error {
	status, _, err := c.do(http.MethodPut, url.PathEscape(event.uid)+".ics", eventICalendar(event, now), map[string]string{
		"Content-Type":  "text/calendar; charset=utf-8",
		"If-None-Match": "*",
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("creating the event failed with status %d", status)
	}
	return nil
}

// delete deletes an event found by a calendar query
func (c *caldavClient) delete(event calendarEvent) error {
	if event.href == "" {
		return fmt.Errorf("the event has no address")
	}
	status, _, err := c.do(http.MethodDelete, event.href, "", nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("deleting the event failed with status %d", status)
	}
	return nil
}

// multiStatus is a WebDAV multistatus response to a calendar query
type multiStatus struct {
	Responses []struct {
		Href     string `xml:"href"`
		PropStat []struct {
			CalendarData string `xml:"prop>calendar-data"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// parseMultiStatus returns the events in a calendar query's response
func parseMultiStatus(data []byte, loc *time.Location) ([]calendarEvent, error) {
	var status multiStatus
	if err := xml.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse calendar query response: %v", err)
	}
	var events []calendarEvent
	for _, response := range status.Responses {
		for _, propStat := range response.PropStat {
			for _, event := range parseICalendar(propStat.CalendarData, loc) {
				event.href = strings.TrimSpace(response.Href)
				event.raw = propStat.CalendarData
				events = append(events, event)
			}
		}
	}
	return events, nil
}

// parseICalendar returns the events (VEVENTs) in iCalendar data. Times with
// no zone are taken to be in loc.
func parseICalendar(data string, loc *time.Location) []calendarEvent {
	// Unfold lines continued with a leading space or tab
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\n ", "")
	data = strings.ReplaceAll(data, "\n\t", "")

	var events []calendarEvent
	var event *calendarEvent
	var duration time.Duration
	hasEnd, hasDuration := false, false
	// depth counts components nested in the VEVENT, such as VALARM
	depth := 0
	for _, line := range strings.Split(data, "\n") {
		name, params, value, ok := parseContentLine(strings.TrimRight(line, "\r"))
		if !ok {
			continue
		}
		switch {
		case name == "BEGIN" && strings.EqualFold(value, "VEVENT") && event == nil:
			event = &calendarEvent{}
			hasEnd, hasDuration, depth = false, false, 0
			continue
		case name == "BEGIN" && event != nil:
			depth++
			continue
		case name == "END" && event != nil && depth > 0:
			depth--
			continue
		case name == "END" && strings.EqualFold(value, "VEVENT") && event != nil:
			if !event.start.IsZero() {
				if !hasEnd {
					switch {
					case hasDuration:
						event.end = event.start.Add(duration)
					case event.allDay:
						event.end = event.start.AddDate(0, 0, 1)
					default:
						event.end = event.start
					}
				}
				events = append(events, *event)
			}
			event = nil
			continue
		}
		if event == nil || depth > 0 {
			continue
		}

		switch name {
		case "UID":
			event.uid = value
		case "SUMMARY":
			event.summary = unescapeICalText(value)
		case "LOCATION":
			event.location = unescapeICalText(value)
		case "DESCRIPTION":
			event.description = unescapeICalText(value)
		case "RRULE":
			event.repeats, event.rrule = true, value
		case "RDATE":
			event.repeats = true
		case "EXDATE":
			for _, part := range strings.Split(value, ",") {
				if t, _, err := parseICalTime(strings.TrimSpace(part), params, loc); err == nil {
					event.exdates = append(event.exdates, t)
				}
			}
		case "RECURRENCE-ID":
			if t, _, err := parseICalTime(value, params, loc); err == nil {
				event.repeats, event.recurrenceID = true, t
			}
		case "DTSTART":
			if t, allDay, err := parseICalTime(value, params, loc); err == nil {
				event.start, event.allDay = t, allDay
				event.zone = icalTimeZone(value, params, loc)
			}
		case "DTEND":
			if t, _, err := parseICalTime(value, params, loc); err == nil {
				event.end, hasEnd = t, true
			}
		case "DURATION":
			if d, err := parseICalDuration(value); err == nil {
				duration, hasDuration = d, true
			}
		}
	}
	return events
}

// parseContentLine splits an iCalendar line, NAME;PARAM=VALUE:value, into
// its upper-case name, its parameters (with upper-case names) and its value
func parseContentLine(line string) (string, map[string]string, string, bool) {
	// The value starts at the first colon outside a quoted parameter value
	quoted := false
	colon := -1
	for i, r := range line {
		if r == '"' {
			quoted = !quoted
		} else if r == ':' && !quoted {
			colon = i
			break
		}
	}
	if colon <= 0 {
		return "", nil, "", false
	}
	parts := strings.Split(line[:colon], ";")
	params := make(map[string]string, len(parts)-1)
	for _, param := range parts[1:] {
		if key, value, found := strings.Cut(param, "="); found {
			params[strings.ToUpper(key)] = strings.Trim(value, `"`)
		}
	}
	return strings.ToUpper(parts[0]), params, line[colon+1:], true
}

// parseICalTime parses a DATE or DATE-TIME value: UTC (ending in Z), in the
// zone its TZID names, or else in loc. It reports whether it is a date.
// icalTimeZone returns the zone a date-time is written in: its TZID, UTC
// for a time ending in Z, or loc for a floating time or a date
func icalTimeZone(value string, params map[string]string, loc *time.Location) *time.Location {
	if strings.HasSuffix(value, "Z") {
		return time.UTC
	}
	if tzid := params["TZID"]; tzid != "" {
		if named, err := time.LoadLocation(strings.TrimPrefix(tzid, "/")); err == nil {
			return named
		}
	}
	return loc
}

func parseICalTime(value string, params map[string]string, loc *time.Location) (time.Time, bool, error) {
	if params["VALUE"] == "DATE" || len(value) == 8 {
		t, err := time.ParseInLocation("20060102", value, loc)
		return t, true, err
	}
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		return t.In(loc), false, err
	}
	zone := loc
	if tzid := params["TZID"]; tzid != "" {
		if named, err := time.LoadLocation(strings.TrimPrefix(tzid, "/")); err == nil {
			zone = named
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, zone)
	return t.In(loc), false, err
}

// icalDurationRe matches an iCalendar duration such as PT1H30M or P1D
var icalDurationRe = regexp.MustCompile(`^([+-])?P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// parseICalDuration parses an iCalendar duration
func parseICalDuration(value string) (time.Duration, error) {
	m := icalDurationRe.FindStringSubmatch(value)
	if m == nil {
		return 0, fmt.Errorf("invalid duration %q", value)
	}
	var d time.Duration
	for i, unit := range []time.Duration{7 * 24 * time.Hour, 24 * time.Hour, time.Hour, time.Minute, time.Second} {
		if m[i+2] != "" {
			n, _ := strconv.Atoi(m[i+2])
			d += time.Duration(n) * unit
		}
	}
	if m[1] == "-" {
		d = -d
	}
	return d, nil
}

// unescapeICalText undoes the escaping of iCalendar TEXT values
func unescapeICalText(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			i++
			switch value[i] {
			case 'n', 'N':
				b.WriteByte('\n')
			default:
				b.WriteByte(value[i])
			}
			continue
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

// escapeICalText escapes a TEXT value for iCalendar
func escapeICalText(value string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(value)
}

// foldICalLine folds a content line into lines of at most 75 bytes, without
// splitting a UTF-8 character
func foldICalLine(line string) string {
	var b strings.Builder
	limit := 75
	for len(line) > limit {
		cut := limit
		for cut > 0 && line[cut]&0xC0 == 0x80 {
			cut--
		}
		b.WriteString(line[:cut])
		b.WriteString("\r\n ")
		line = line[cut:]
		limit = 74 // the leading space counts
	}
	b.WriteString(line)
	b.WriteString("\r\n")
	return b.String()
}

// eventICalendar returns an event as an iCalendar object
func eventICalendar(event calendarEvent, now time.Time) string {
	var b strings.Builder
	write := func(line string) { b.WriteString(foldICalLine(line)) }
	write("BEGIN:VCALENDAR")
	write("VERSION:2.0")
	write("PRODID:-//Golem//CalDAV SRAIX//EN")
	// A recurring event with a time is written in its time zone, so that it
	// keeps its local time across daylight saving time changes
	zone := ""
	if event.recurrence != nil && !event.allDay {
		zone = eventTimeZoneName(event.start.Location())
	}
	if zone != "" {
		for _, line := range vtimezone(zone, event.start.Location(), event.start) {
			write(line)
		}
	}
	write("BEGIN:VEVENT")
	write("UID:" + event.uid)
	write("DTSTAMP:" + caldavTime(now))
	switch {
	case event.allDay:
		write("DTSTART;VALUE=DATE:" + event.start.Format("20060102"))
		write("DTEND;VALUE=DATE:" + event.end.Format("20060102"))
	case zone != "":
		write("DTSTART;TZID=" + zone + ":" + event.start.Format("20060102T150405"))
		write("DTEND;TZID=" + zone + ":" + event.end.Format("20060102T150405"))
	default:
		write("DTSTART:" + caldavTime(event.start))
		write("DTEND:" + caldavTime(event.end))
	}
	if event.recurrence != nil {
		write("RRULE:" + event.recurrence.rrule(event, zone == ""))
	}
	write("SUMMARY:" + escapeICalText(event.summary))
	write("END:VEVENT")
	write("END:VCALENDAR")
	return b.String()
}

// newEventUID returns a random UID for a new event
func newEventUID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("failed to make an event UID: %v", err)
	}
	return hex.EncodeToString(id[:]) + "@golem", nil
}

// startOfDay returns midnight at the start of t's day, in t's zone
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// eventWhen describes when an event is, relative to now: "at 2:00 PM"
// (today, when withDay is false), "today at 2:00 PM", "tomorrow (all day)",
// "on Mon Nov 23 at 9:00 AM"
func eventWhen(event calendarEvent, now time.Time, withDay bool) string {
	start := event.start.In(now.Location())
	day := ""
	if phrase := dayPhrase(start, now); phrase != "today" || withDay {
		day = phrase
		if phrase != "today" && phrase != "tomorrow" {
			day = "on " + phrase
		}
	}
	when := "at " + clockPhrase(start)
	if event.allDay {
		when = "(all day)"
	}
	if day == "" {
		return when
	}
	return day + " " + when
}

// clockPhrase says a time of day: "9 AM", "9:30 AM", "noon" or "midnight"
func clockPhrase(t time.Time) string {
	switch {
	case t.Hour() == 12 && t.Minute() == 0:
		return "noon"
	case t.Hour() == 0 && t.Minute() == 0:
		return "midnight"
	case t.Minute() == 0:
		return t.Format("3 PM")
	}
	return t.Format("3:04 PM")
}

// formatEventList lists events for a reply. Up to three read as a sentence
// ("standup at 9 AM, lunch at noon and gym at 6 PM", with their days when
// withDay is set); more go one per line, or with withDay one line per day
// ("\n• Tomorrow: standup at 9 AM, lunch at noon\n• Friday: ..."). It
// returns "" when there are none, and the layout: "inline" or "lines".
func formatEventList(events []calendarEvent, now time.Time, withDay bool) (string, string) {
	if len(events) == 0 {
		return "", "inline"
	}
	if len(events) <= 3 {
		descriptions := make([]string, len(events))
		for i, event := range events {
			descriptions[i] = eventTitle(event) + " " + eventWhen(event, now, withDay)
		}
		return joinWithAnd(descriptions), "inline"
	}
	var b strings.Builder
	if !withDay {
		for _, event := range events {
			b.WriteString("\n• " + eventTitle(event) + " " + eventWhen(event, now, false))
		}
		return b.String(), "lines"
	}
	var days []string
	byDay := map[string][]string{}
	for _, event := range events {
		day := dayPhrase(event.start.In(now.Location()), now)
		if _, seen := byDay[day]; !seen {
			days = append(days, day)
		}
		at := "at " + clockPhrase(event.start.In(now.Location()))
		if event.allDay {
			at = "(all day)"
		}
		byDay[day] = append(byDay[day], eventTitle(event)+" "+at)
	}
	for _, day := range days {
		b.WriteString("\n• " + strings.ToUpper(day[:1]) + day[1:] + ": " + strings.Join(byDay[day], ", "))
	}
	return b.String(), "lines"
}

// eventTitle returns an event's summary, or "Untitled event"
func eventTitle(event calendarEvent) string {
	if event.summary == "" {
		return "Untitled event"
	}
	return event.summary
}

// eventDetails describes an event in full, as a sentence: its title, when
// it is, where, and its description
func eventDetails(event calendarEvent, now time.Time) string {
	var b strings.Builder
	b.WriteString(`"` + eventTitle(event) + `" is ` + eventWhen(event, now, true))
	if !event.allDay && event.end.After(event.start) {
		end := event.end.In(now.Location())
		if startOfDay(end).Equal(startOfDay(event.start.In(now.Location()))) {
			b.WriteString(" until " + clockPhrase(end))
		} else {
			b.WriteString(" until " + dayPhrase(end, now) + " at " + clockPhrase(end))
		}
	}
	if event.location != "" {
		b.WriteString(", at " + event.location)
	}
	b.WriteString(".")
	if event.repeats {
		b.WriteString(" It repeats.")
	}
	if event.description != "" {
		b.WriteString(" " + strings.TrimSpace(event.description))
	}
	return b.String()
}

// Patterns for parseEventText. Each matches a phrase that is removed from
// the event's title.
var (
	eventMonths      = `(january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|jun|jul|aug|sept|sep|oct|nov|dec)\.?`
	eventDayAfterRe  = regexp.MustCompile(`(?i)\b(?:on\s+)?the\s+day\s+after\s+tomorrow\b`)
	eventRelativeRe  = regexp.MustCompile(`(?i)\b(?:on\s+)?(today|tonight|tomorrow)\b`)
	eventWeekdayRe   = regexp.MustCompile(`(?i)\b(?:on\s+)?(?:(this|next)\s+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`)
	eventISODateRe   = regexp.MustCompile(`(?i)\b(?:on\s+)?(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	eventMonthDayRe  = regexp.MustCompile(`(?i)\b(?:on\s+)?` + eventMonths + `\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?\b`)
	eventDayMonthRe  = regexp.MustCompile(`(?i)\b(?:on\s+)?(?:the\s+)?(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?` + eventMonths + `(?:,?\s+(\d{4}))?\b`)
	eventClockRe     = regexp.MustCompile(`(?i)(?:\bat\s+)?\b(\d{1,2})(?:[:.](\d{2}))?\s*(a\.?m\.?|p\.?m\.?)(?:\s|$|[,;!?])`)
	event24HourRe    = regexp.MustCompile(`(?i)(?:\bat\s+)?\b(\d{1,2}):(\d{2})\b`)
	eventNamedTimeRe = regexp.MustCompile(`(?i)(?:\bat\s+)?\b(noon|midday|midnight)\b`)
	eventBareHourRe  = regexp.MustCompile(`(?i)\bat\s+(\d{1,2})\b`)
	eventDurationRe  = regexp.MustCompile(`(?i)\bfor\s+(?:(\d+)\s*(minutes?|mins?|hours?|hrs?)|(an|one|half\s+an)\s+hour)\b`)
	eventWeekdays    = map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}
)

// takeEventDate finds the date in text, as parseEventText describes, and
// removes it with remove. It reports whether there was one.
func takeEventDate(remove func(*regexp.Regexp) []string, today time.Time) (date time.Time, hasDate bool, err error) {
	loc := today.Location()
	if m := remove(eventDayAfterRe); m != nil {
		date, hasDate = today.AddDate(0, 0, 2), true
	} else if m := remove(eventRelativeRe); m != nil {
		date, hasDate = today, true
		if strings.EqualFold(m[1], "tomorrow") {
			date = today.AddDate(0, 0, 1)
		}
	} else if m := remove(eventISODateRe); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		date, hasDate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc), true
		if date.Month() != time.Month(month) || date.Day() != day {
			return time.Time{}, false, fmt.Errorf("invalid date %s", strings.TrimSpace(m[0]))
		}
	} else if m := remove(eventMonthDayRe); m != nil {
		var err error
		if date, err = monthDayDate(m[1], m[2], m[3], today); err != nil {
			return time.Time{}, false, err
		}
		hasDate = true
	} else if m := remove(eventDayMonthRe); m != nil {
		var err error
		if date, err = monthDayDate(m[2], m[1], m[3], today); err != nil {
			return time.Time{}, false, err
		}
		hasDate = true
	} else if m := remove(eventWeekdayRe); m != nil {
		days := (int(eventWeekdays[strings.ToLower(m[2])]) - int(today.Weekday()) + 7) % 7
		if days == 0 && !strings.EqualFold(m[1], "this") {
			days = 7
		}
		date, hasDate = today.AddDate(0, 0, days), true
	}
	return date, hasDate, nil
}

// parseEventText reads an event from text such as "Lunch with Sam tomorrow
// at noon", "Dentist on Nov 20 at 9:15am for 30 minutes" or "Holiday on
// Friday". The date can be today, tonight, tomorrow, the day after
// tomorrow, a weekday (the next one after today), a month and day, or
// YYYY-MM-DD. The time can be 2pm, 2:30 p.m., 14:00, noon or midnight, or a
// bare hour after "at" (1 to 7 are taken as PM). An event with a time lasts
// an hour unless the text says "for 30 minutes" or "for 2 hours"; one with
// only a date lasts all day; one with neither is all day today. A time with
// no date is today, or tomorrow when that time has passed. The rest of the
// text is the title.
func parseEventText(text string, now time.Time) (calendarEvent, error) {
	loc := now.Location()
	today := startOfDay(now)
	title := text
	remove := func(re *regexp.Regexp) []string {
		idx := re.FindStringSubmatchIndex(title)
		if idx == nil {
			return nil
		}
		groups := make([]string, len(idx)/2)
		for i := range groups {
			if idx[2*i] >= 0 {
				groups[i] = title[idx[2*i]:idx[2*i+1]]
			}
		}
		title = title[:idx[0]] + " " + title[idx[1]:]
		return groups
	}

	// How it repeats ("every Monday"), before the date, which would read
	// "Monday" as one day
	rec, err := takeEventRecurrence(&title, today)
	if err != nil {
		return calendarEvent{}, err
	}

	// The date
	date, hasDate, err := takeEventDate(remove, today)
	if err != nil {
		return calendarEvent{}, err
	}
	if rec != nil {
		// "every Monday starting December 7": the date is the first one
		title = recurStartRe.ReplaceAllString(title, " ")
	}

	// The time
	hour, minute := -1, 0
	if m := remove(eventClockRe); m != nil {
		hour, _ = strconv.Atoi(m[1])
		minute, _ = strconv.Atoi(m[2])
		if hour < 1 || hour > 12 {
			return calendarEvent{}, fmt.Errorf("invalid time %s", strings.TrimSpace(m[0]))
		}
		pm := strings.HasPrefix(strings.ToLower(m[3]), "p")
		if hour == 12 {
			hour = 0
		}
		if pm {
			hour += 12
		}
	} else if m := remove(event24HourRe); m != nil {
		hour, _ = strconv.Atoi(m[1])
		minute, _ = strconv.Atoi(m[2])
	} else if m := remove(eventNamedTimeRe); m != nil {
		hour = 12
		if strings.EqualFold(m[1], "midnight") {
			hour = 0
		}
	} else if m := remove(eventBareHourRe); m != nil {
		hour, _ = strconv.Atoi(m[1])
		if hour >= 1 && hour <= 7 {
			hour += 12
		}
	}
	if hour > 23 || minute > 59 {
		return calendarEvent{}, fmt.Errorf("invalid time %02d:%02d", hour, minute)
	}

	duration := time.Hour
	if m := remove(eventDurationRe); m != nil {
		switch {
		case m[1] != "":
			n, _ := strconv.Atoi(m[1])
			unit := time.Minute
			if strings.HasPrefix(strings.ToLower(m[2]), "h") {
				unit = time.Hour
			}
			duration = time.Duration(n) * unit
		case strings.HasPrefix(strings.ToLower(m[3]), "half"):
			duration = 30 * time.Minute
		}
		if duration <= 0 {
			return calendarEvent{}, fmt.Errorf("invalid duration %s", strings.TrimSpace(m[0]))
		}
	}

	event := calendarEvent{summary: cleanEventTitle(title), dated: hasDate, timed: hour >= 0}
	switch {
	case hour < 0:
		if !hasDate {
			date = today
		}
		event.start, event.end, event.allDay = date, date.AddDate(0, 0, 1), true
	default:
		if !hasDate {
			date = today
			if !time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc).After(now) {
				date = today.AddDate(0, 0, 1)
			}
		}
		event.start = time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
		event.end = event.start.Add(duration)
	}
	if rec != nil {
		if err := applyRecurrence(&event, rec); err != nil {
			return calendarEvent{}, err
		}
	}
	return event, nil
}

// monthDayDate returns the date of a month name and day, in the given year
// or else the next one on or after today
func monthDayDate(monthName, dayText, yearText string, today time.Time) (time.Time, error) {
	month := time.Month(0)
	prefix := strings.ToLower(strings.TrimSuffix(monthName, "."))
	for m := time.January; m <= time.December; m++ {
		if strings.HasPrefix(strings.ToLower(m.String()), prefix[:3]) {
			month = m
			break
		}
	}
	day, _ := strconv.Atoi(dayText)
	year := today.Year()
	if yearText != "" {
		year, _ = strconv.Atoi(yearText)
	}
	date := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
	if month == 0 || date.Day() != day {
		return time.Time{}, fmt.Errorf("invalid date %s %s", monthName, dayText)
	}
	if yearText == "" && date.Before(today) {
		date = date.AddDate(1, 0, 0)
	}
	return date, nil
}

// eventTitleTrimRe matches the words and punctuation left at the ends of a
// title once its date and time are removed
var eventTitleTrimRe = regexp.MustCompile(`(?i)^(?:\s|[,;:.\-]|\b(?:on|at|for|from|the|a|an|my|our)\b)+|(?:\s|[,;:\-]|\b(?:on|at|for|from|the)\b)+$`)

// cleanEventTitle tidies what is left of the text once its date and time are
// removed, or returns "Event" when nothing is
func cleanEventTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	for {
		trimmed := eventTitleTrimRe.ReplaceAllString(title, "")
		if trimmed == title {
			break
		}
		title = trimmed
	}
	if title == "" {
		return "Event"
	}
	return title
}

var (
	rangeWeekRe     = regexp.MustCompile(`(?i)\b(this|next)\s+week\b`)
	rangeWeekendRe  = regexp.MustCompile(`(?i)\b(?:(this|next)\s+)?weekend\b`)
	rangeMonthRe    = regexp.MustCompile(`(?i)\b(this|next)\s+month\b`)
	rangeNextDaysRe = regexp.MustCompile(`(?i)\b(?:in\s+)?(?:the\s+)?(?:next|coming)\s+(\d{1,2}|two|three|four|five|six|seven|ten|fourteen)\s+days\b`)
	rangeFillerRe   = regexp.MustCompile(`(?i)\b(?:for|on|in|over|during|the|my|calendar|schedule|agenda|events?|what|is|do|i|have|am|free|busy)\b|[?.,!]`)
	rangeNumbers    = map[string]int{"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "ten": 10, "fourteen": 14}
)

// maxCalendarRange bounds how many days a range query covers
const maxCalendarRange = 62

// parseDateRange reads the days text such as "tomorrow", "on Friday",
// "December 3", "this week", "next week", "this weekend", "the next 3 days"
// or "next month" names. It returns the range (end exclusive), whether it
// is a single day, and a label for it, e.g. "on Friday, November 27". Weeks
// run Monday to Sunday; "this week" and "this weekend" start today when
// they have begun. Text with anything but a date is not understood.
func parseDateRange(text string, now time.Time) (start, end time.Time, singleDay bool, label string, err error) {
	today := startOfDay(now)
	rest := text
	remove := func(re *regexp.Regexp) []string {
		idx := re.FindStringSubmatchIndex(rest)
		if idx == nil {
			return nil
		}
		groups := make([]string, len(idx)/2)
		for i := range groups {
			if idx[2*i] >= 0 {
				groups[i] = rest[idx[2*i]:idx[2*i+1]]
			}
		}
		rest = rest[:idx[0]] + " " + rest[idx[1]:]
		return groups
	}
	// Days since Monday
	sinceMonday := (int(today.Weekday()) + 6) % 7

	switch {
	case func() bool {
		m := remove(rangeNextDaysRe)
		if m == nil {
			return false
		}
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			n = rangeNumbers[strings.ToLower(m[1])]
		}
		if n < 1 || n > maxCalendarRange {
			err = fmt.Errorf("too many days")
			return true
		}
		start, end, label = today, today.AddDate(0, 0, n), fmt.Sprintf("in the next %d days", n)
		return true
	}():
	case func() bool {
		m := remove(rangeWeekRe)
		if m == nil {
			return false
		}
		monday := today.AddDate(0, 0, -sinceMonday)
		if strings.EqualFold(m[1], "next") {
			start, end, label = monday.AddDate(0, 0, 7), monday.AddDate(0, 0, 14), "next week"
		} else {
			start, end, label = today, monday.AddDate(0, 0, 7), "this week"
		}
		return true
	}():
	case func() bool {
		m := remove(rangeWeekendRe)
		if m == nil {
			return false
		}
		saturday := today.AddDate(0, 0, 5-sinceMonday) // this week's
		start = saturday
		if today.After(saturday) {
			start = today // Sunday
		}
		end, label = saturday.AddDate(0, 0, 2), "this weekend"
		if strings.EqualFold(m[1], "next") {
			start, end, label = saturday.AddDate(0, 0, 7), saturday.AddDate(0, 0, 9), "next weekend"
		}
		return true
	}():
	case func() bool {
		m := remove(rangeMonthRe)
		if m == nil {
			return false
		}
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		if strings.EqualFold(m[1], "next") {
			start, end, label = first.AddDate(0, 1, 0), first.AddDate(0, 2, 0), "next month"
		} else {
			start, end, label = today, first.AddDate(0, 1, 0), "this month"
		}
		return true
	}():
	default:
		date, hasDate, dateErr := takeEventDate(remove, today)
		if dateErr != nil {
			return time.Time{}, time.Time{}, false, "", dateErr
		}
		if !hasDate {
			return time.Time{}, time.Time{}, false, "", fmt.Errorf("no date in %q", text)
		}
		start, end, singleDay = date, date.AddDate(0, 0, 1), true
		switch days := int(date.Sub(today).Hours() / 24); {
		case days == 0:
			label = "today"
		case days == 1:
			label = "tomorrow"
		case days > 1 && days < 7:
			label = "on " + date.Format("Monday, January 2")
		default:
			label = "on " + date.Format("Monday, January 2, 2006")
			if date.Year() == today.Year() {
				label = "on " + date.Format("Monday, January 2")
			}
		}
	}
	if err != nil {
		return time.Time{}, time.Time{}, false, "", err
	}
	// Anything left must be filler, so that "dentist on Friday" is not read
	// as just "Friday"
	if strings.TrimSpace(rangeFillerRe.ReplaceAllString(rest, " ")) != "" {
		return time.Time{}, time.Time{}, false, "", fmt.Errorf("not a date: %q", text)
	}
	return start, end, singleDay, label, nil
}
