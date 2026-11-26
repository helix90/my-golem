package golem

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// eventListResponse is the reply of the today, upcoming and range
// operations: the events (see formatEventList), with calendar_layout set to
// inline or lines and calendar_count to how many there are
func eventListResponse(events []calendarEvent, now time.Time, withDay bool, set map[string]string) sraixResponse {
	text, layout := formatEventList(events, now, withDay)
	if set == nil {
		set = map[string]string{}
	}
	set["calendar_layout"] = layout
	set["calendar_count"] = strconv.Itoa(len(events))
	return sraixResponse{text: text, set: set}
}

// eventPredicates are what replies about one event use: its title, when it
// is, its UID and whether it repeats
func eventPredicates(event calendarEvent, now time.Time) map[string]string {
	repeats := "no"
	if event.repeats || event.recurrence != nil {
		repeats = "yes"
	}
	return map[string]string{
		"calendar_event_title":   eventTitle(event),
		"calendar_event_when":    eventWhen(event, now, true),
		"calendar_event_uid":     event.uid,
		"calendar_event_repeats": repeats,
	}
}

// severalEventsError is find's error when more than one event matches
type severalEventsError struct {
	query  string
	events []calendarEvent
}

func (e severalEventsError) Error() string {
	return fmt.Sprintf("%d events match %q", len(e.events), e.query)
}

// response says which events matched: text SEVERAL EVENTS, with
// calendar_matches listing them one per line and calendar_match_buttons a
// button for each, whose postback is action (default GET EVENT) and the
// event's UID
func (e severalEventsError) response(now time.Time, action string) sraixResponse {
	if action == "" {
		action = "GET EVENT"
	}
	var list, buttons strings.Builder
	for _, event := range e.events {
		description := eventTitle(event) + " " + eventWhen(event, now, true)
		list.WriteString("\n• " + description)
		fmt.Fprintf(&buttons, "<button><text>%s</text><postback>%s</postback></button>",
			html.EscapeString(description), html.EscapeString(action+" "+event.uid))
	}
	return sraixResponse{text: "SEVERAL EVENTS", set: map[string]string{
		"calendar_matches":       list.String(),
		"calendar_match_buttons": buttons.String(),
		"calendar_match_count":   strconv.Itoa(len(e.events)),
	}}
}

var (
	eventQueryLeadRe  = regexp.MustCompile(`(?i)^(?:my|the|our|a|an|this|next|upcoming)\s+`)
	eventQueryTrailRe = regexp.MustCompile(`(?i)\s+(?:appointment|meeting|event|thing|session)s?$`)
)

// eventQueries are the ways find tries the name of an event as typed: as
// it is, without "my"/"the"/"next", and without a trailing "appointment",
// "meeting" or "event", so "my dentist appointment" finds "Dentist"
func eventQueries(input string) []string {
	queries := []string{input}
	add := func(q string) {
		q = strings.TrimSpace(q)
		if q != "" && q != queries[len(queries)-1] {
			queries = append(queries, q)
		}
	}
	trimmed := input
	for {
		next := eventQueryLeadRe.ReplaceAllString(trimmed, "")
		if next == trimmed {
			break
		}
		trimmed = next
	}
	add(trimmed)
	add(eventQueryTrailRe.ReplaceAllString(trimmed, ""))
	return queries
}

// eventShiftRe matches moving an event by an amount: "back an hour",
// "30 minutes later", "forward by 2 hours", "an hour earlier"
var eventShiftRe = regexp.MustCompile(`(?i)^\s*(?:(back|later|forward|earlier)\s+)?(?:by\s+)?(an?|one|half\s+an|\d+)\s*(hours?|hrs?|minutes?|mins?)(?:\s+(later|earlier|back|forward))?\s*$`)

// moveEvent works out where text moves an event to: a new date (keeping
// its time), a new time (keeping its day), both, or a shift such as "back
// an hour". It keeps the event's length. ok is false when text says
// neither.
func moveEvent(event calendarEvent, text string, now time.Time) (moved calendarEvent, ok bool, err error) {
	loc := now.Location()
	moved = event
	length := event.end.Sub(event.start)
	text = strings.TrimSpace(text)

	if m := eventShiftRe.FindStringSubmatch(text); m != nil && (m[1] != "" || m[4] != "") {
		amount := 1
		switch amountWord := strings.ToLower(m[2]); {
		case strings.HasPrefix(amountWord, "half"):
			amount = 30
		case amountWord == "a" || amountWord == "an" || amountWord == "one":
		default:
			amount, _ = strconv.Atoi(amountWord)
		}
		unit := time.Minute
		if strings.HasPrefix(strings.ToLower(m[3]), "h") {
			unit = time.Hour
			if strings.HasPrefix(strings.ToLower(m[2]), "half") {
				amount, unit = 30, time.Minute
			}
		}
		shift := time.Duration(amount) * unit
		direction := strings.ToLower(m[1] + m[4])
		if strings.Contains(direction, "forward") || strings.Contains(direction, "earlier") {
			shift = -shift
		}
		if event.allDay {
			return calendarEvent{}, false, nil
		}
		moved.start, moved.end = event.start.Add(shift), event.end.Add(shift)
		return moved, true, nil
	}

	parsed, err := parseEventText(text, now)
	if err != nil {
		return calendarEvent{}, false, err
	}
	if !parsed.dated && !parsed.timed {
		return calendarEvent{}, false, nil
	}
	old := event.start.In(loc)
	day := time.Date(old.Year(), old.Month(), old.Day(), 0, 0, 0, 0, loc)
	if parsed.dated {
		p := parsed.start.In(loc)
		day = time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, loc)
	}
	switch {
	case parsed.timed:
		p := parsed.start.In(loc)
		if event.allDay {
			length = parsed.end.Sub(parsed.start)
		}
		moved.allDay = false
		moved.start = time.Date(day.Year(), day.Month(), day.Day(), p.Hour(), p.Minute(), 0, 0, loc)
		// "move lunch to 2pm" with no day, once 2 PM has passed today, is
		// tomorrow, as for a new event
		if !parsed.dated && event.allDay {
			moved.start = parsed.start
		}
	case event.allDay:
		moved.start = day
		length = event.end.Sub(event.start)
	default:
		moved.start = time.Date(day.Year(), day.Month(), day.Day(), old.Hour(), old.Minute(), 0, 0, loc)
	}
	moved.end = moved.start.Add(length)
	return moved, true, nil
}

// update writes an event back with new times, changing only its DTSTART,
// DTEND (or DURATION) and SEQUENCE, so that the rest (alarms, attendees,
// notes) is kept
func (c *caldavClient) update(event, moved calendarEvent, now time.Time) error {
	if event.href == "" || event.raw == "" {
		return fmt.Errorf("the event has no address")
	}
	data, err := rewriteEventTimes(event.raw, moved, now)
	if err != nil {
		return err
	}
	status, _, err := c.do(http.MethodPut, event.href, data, map[string]string{"Content-Type": "text/calendar; charset=utf-8"})
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("moving the event failed with status %d", status)
	}
	return nil
}

// rewriteEventTimes replaces the times of the (one) VEVENT in data
func rewriteEventTimes(data string, moved calendarEvent, now time.Time) (string, error) {
	unfolded := strings.NewReplacer("\r\n ", "", "\r\n\t", "", "\n ", "", "\n\t", "").Replace(data)
	lines := strings.Split(strings.ReplaceAll(unfolded, "\r\n", "\n"), "\n")
	var out []string
	inEvent, events, sequence := false, 0, 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		name := strings.ToUpper(line)
		if i := strings.IndexAny(name, ";:"); i >= 0 {
			name = name[:i]
		}
		switch {
		case name == "BEGIN" && strings.EqualFold(line, "BEGIN:VEVENT"):
			inEvent = true
			events++
		case inEvent && (name == "DTSTART" || name == "DTEND" || name == "DURATION" || name == "DTSTAMP"):
			continue
		case inEvent && name == "SEQUENCE":
			sequence, _ = strconv.Atoi(strings.TrimSpace(line[strings.Index(line, ":")+1:]))
			continue
		case inEvent && name == "END" && strings.EqualFold(line, "END:VEVENT"):
			out = append(out, "DTSTAMP:"+caldavTime(now))
			if moved.allDay {
				out = append(out, "DTSTART;VALUE=DATE:"+moved.start.Format("20060102"), "DTEND;VALUE=DATE:"+moved.end.Format("20060102"))
			} else {
				out = append(out, "DTSTART:"+caldavTime(moved.start), "DTEND:"+caldavTime(moved.end))
			}
			out = append(out, "SEQUENCE:"+strconv.Itoa(sequence+1))
			inEvent = false
		}
		out = append(out, line)
	}
	if events != 1 {
		return "", fmt.Errorf("can't move an event with %d parts", events)
	}
	var b strings.Builder
	for _, line := range out {
		b.WriteString(foldICalLine(line))
	}
	return b.String(), nil
}
