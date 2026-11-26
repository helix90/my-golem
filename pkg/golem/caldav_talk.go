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
	// "standups", "yoga classes": the singular
	last := queries[len(queries)-1]
	lower := strings.ToLower(last)
	switch {
	case strings.HasSuffix(lower, "ies") && len(last) > 4:
		add(last[:len(last)-3] + "y")
	case strings.HasSuffix(lower, "sses") || strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes") || strings.HasSuffix(lower, "xes"):
		add(last[:len(last)-2])
	case strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") && len(last) > 3:
		add(last[:len(last)-1])
	}
	return queries
}

// moveBareWeekdayRe matches a move to a weekday with no "next" or "this"
var moveBareWeekdayRe = regexp.MustCompile(`(?i)^\s*(?:on\s+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)s?\b`)

// day0 is the start of t's day, in t's location
func day0(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
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
		// A bare weekday ("to Wednesday") is that day of the event's own
		// week (Monday to Sunday), unless that has passed
		if m := moveBareWeekdayRe.FindStringSubmatch(text); m != nil {
			monday := day0(old).AddDate(0, 0, -((int(old.Weekday()) + 6) % 7))
			candidate := monday.AddDate(0, 0, (int(eventWeekdays[strings.ToLower(m[1])])+6)%7)
			if !candidate.Before(startOfDay(now)) {
				day = candidate
			}
		}
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

// findOccurrence finds the event text names. Text with a day in it ("yoga
// on Monday", "standup tomorrow") names that day's occurrence of it, and
// named is then true; otherwise it is find's event, the next occurrence of
// a repeating one.
func (c *caldavClient) findOccurrence(text string, now time.Time) (event calendarEvent, named bool, err error) {
	event, err = c.find(text, now)
	if _, ok := err.(noEventError); !ok {
		return event, false, err
	}
	parsed, parseErr := parseEventText(text, now)
	if parseErr != nil || !parsed.dated || parsed.summary == "Event" {
		return calendarEvent{}, false, err
	}
	day := startOfDay(parsed.start.In(now.Location()))
	events, queryErr := c.query(day, day.AddDate(0, 0, 1))
	if queryErr != nil {
		return calendarEvent{}, false, queryErr
	}
	event, err = matchEvent(events, parsed.summary)
	return event, err == nil, err
}

// errCantMoveSeries is updateSeries' error for a rule it can't move to
// another day, such as "the first Monday of the month"
var errCantMoveSeries = fmt.Errorf("can't move every occurrence of this event")

// updateSeries moves an occurrence of a repeating event: just it (scope
// "one", by writing or changing the occurrence's own VEVENT with a
// RECURRENCE-ID), or the whole series (scope "all", by the same change of
// day and time, with its rule, exceptions and changed occurrences)
func (c *caldavClient) updateSeries(occurrence, moved calendarEvent, scope string, now time.Time) error {
	if occurrence.href == "" || occurrence.raw == "" {
		return fmt.Errorf("the event has no address")
	}
	var data string
	var err error
	if scope == "all" {
		data, err = moveSeries(occurrence.raw, occurrence, moved, now)
	} else {
		data, err = moveOccurrence(occurrence.raw, occurrence, moved, now)
	}
	if err != nil {
		return err
	}
	status, _, err := c.do(http.MethodPut, occurrence.href, data, map[string]string{"Content-Type": "text/calendar; charset=utf-8"})
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("moving the event failed with status %d", status)
	}
	return nil
}

// icalPart is a run of iCalendar lines: one VEVENT (with what it nests), or
// the lines between them
type icalPart struct {
	lines   []string
	vevent  bool
	master  bool      // a VEVENT with no RECURRENCE-ID
	replace time.Time // its RECURRENCE-ID
}

// splitICalParts unfolds data and splits it into VEVENTs and the rest
func splitICalParts(data string, loc *time.Location) []icalPart {
	unfolded := strings.NewReplacer("\r\n ", "", "\r\n\t", "", "\n ", "", "\n\t", "").Replace(data)
	var parts []icalPart
	current := icalPart{}
	depth := 0
	for _, line := range strings.Split(strings.ReplaceAll(unfolded, "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		name, params, value, _ := parseContentLine(line)
		switch {
		case name == "BEGIN" && strings.EqualFold(value, "VEVENT") && depth == 0:
			if len(current.lines) > 0 {
				parts = append(parts, current)
			}
			current = icalPart{vevent: true, master: true}
			depth = 1
		case name == "BEGIN" && depth > 0:
			depth++
		case name == "END" && depth > 0:
			depth--
		case name == "RECURRENCE-ID" && depth == 1:
			current.master = false
			current.replace, _, _ = parseICalTime(value, params, loc)
		}
		current.lines = append(current.lines, line)
		if current.vevent && depth == 0 {
			parts = append(parts, current)
			current = icalPart{}
		}
	}
	if len(current.lines) > 0 {
		parts = append(parts, current)
	}
	return parts
}

func joinICalParts(parts []icalPart) string {
	var b strings.Builder
	for _, part := range parts {
		for _, line := range part.lines {
			b.WriteString(foldICalLine(line))
		}
	}
	return b.String()
}

// icalProperty returns the first line of a VEVENT's own property (not one
// of a nested VALARM)
func icalProperty(lines []string, property string) (index int, params map[string]string, value string) {
	depth := 0
	for i, line := range lines {
		name, p, v, ok := parseContentLine(line)
		if !ok {
			continue
		}
		switch {
		case name == "BEGIN":
			depth++
		case name == "END":
			depth--
		case depth == 1 && name == property:
			return i, p, v
		}
	}
	return -1, nil, ""
}

// icalTimeLike formats t as property, written the way params and value
// are: a date, a time in its TZID, UTC, or floating
func icalTimeLike(property string, params map[string]string, value string, t time.Time, loc *time.Location) string {
	switch {
	case params["VALUE"] == "DATE" || len(value) == 8:
		return property + ";VALUE=DATE:" + t.Format("20060102")
	case strings.HasSuffix(value, "Z"):
		return property + ":" + caldavTime(t)
	case params["TZID"] != "":
		return property + ";TZID=" + params["TZID"] + ":" + t.In(icalTimeZone(value, params, loc)).Format("20060102T150405")
	}
	return property + ":" + t.In(loc).Format("20060102T150405")
}

// setEventTimes replaces a VEVENT's DTSTART, DTEND and DURATION with the
// event's times, and bumps its SEQUENCE and DTSTAMP
func setEventTimes(lines []string, event calendarEvent, now time.Time) []string {
	sequence := 0
	if i, _, value := icalProperty(lines, "SEQUENCE"); i >= 0 {
		sequence, _ = strconv.Atoi(strings.TrimSpace(value))
	}
	var out []string
	depth := 0
	for _, line := range lines {
		name, _, _, _ := parseContentLine(line)
		switch {
		case name == "BEGIN":
			depth++
		case name == "END":
			depth--
		}
		if depth == 1 && (name == "DTSTART" || name == "DTEND" || name == "DURATION" || name == "DTSTAMP" || name == "SEQUENCE") {
			continue
		}
		if name == "END" && depth == 0 {
			out = append(out, "DTSTAMP:"+caldavTime(now))
			if event.allDay {
				out = append(out, "DTSTART;VALUE=DATE:"+event.start.Format("20060102"), "DTEND;VALUE=DATE:"+event.end.Format("20060102"))
			} else {
				out = append(out, "DTSTART:"+caldavTime(event.start), "DTEND:"+caldavTime(event.end))
			}
			out = append(out, "SEQUENCE:"+strconv.Itoa(sequence+1))
		}
		out = append(out, line)
	}
	return out
}

// moveOccurrence writes one occurrence of a repeating event at its new
// times: it changes the occurrence's VEVENT if it has one (it was moved
// before), or adds one, a copy of the series with a RECURRENCE-ID
func moveOccurrence(data string, occurrence, moved calendarEvent, now time.Time) (string, error) {
	loc := now.Location()
	parts := splitICalParts(data, loc)
	if !occurrence.recurrenceID.IsZero() {
		for i, part := range parts {
			if part.vevent && !part.master && part.replace.Equal(occurrence.recurrenceID) {
				parts[i].lines = setEventTimes(part.lines, moved, now)
				return joinICalParts(parts), nil
			}
		}
		return "", fmt.Errorf("the changed occurrence is missing")
	}
	for i, part := range parts {
		if !part.vevent || !part.master {
			continue
		}
		_, startParams, startValue := icalProperty(part.lines, "DTSTART")
		if startValue == "" {
			return "", fmt.Errorf("the event has no start")
		}
		var copied []string
		depth := 0
		for _, line := range part.lines {
			name, _, _, _ := parseContentLine(line)
			switch {
			case name == "BEGIN":
				depth++
			case name == "END":
				depth--
			}
			if depth == 1 && (name == "RRULE" || name == "RDATE" || name == "EXDATE" || name == "EXRULE") {
				continue
			}
			copied = append(copied, line)
			if depth == 1 && name == "UID" {
				copied = append(copied, icalTimeLike("RECURRENCE-ID", startParams, startValue, occurrence.start, loc))
			}
		}
		override := icalPart{lines: setEventTimes(copied, moved, now), vevent: true}
		rest := append([]icalPart{override}, parts[i+1:]...)
		parts = append(parts[:i+1], rest...)
		return joinICalParts(parts), nil
	}
	return "", fmt.Errorf("the event has no series")
}

var icalWeekdays = []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}

// moveSeries moves every occurrence of a repeating event by the change
// from occurrence to moved: the same number of days, to the same time of
// day, in the series' own zone (so "every Monday at 6 PM" moved to Tuesday
// at 7 PM is every Tuesday at 7 PM, across daylight saving changes). Its
// rule's weekdays, its UNTIL, its left-out dates and the RECURRENCE-IDs of
// its changed occurrences move with it; changed occurrences keep their own
// times, but those moved only to another day take the new time.
func moveSeries(data string, occurrence, moved calendarEvent, now time.Time) (string, error) {
	loc := now.Location()
	parts := splitICalParts(data, loc)
	for i, part := range parts {
		if !part.vevent || !part.master {
			continue
		}
		startIndex, startParams, startValue := icalProperty(part.lines, "DTSTART")
		if startIndex < 0 {
			return "", fmt.Errorf("the event has no start")
		}
		allDay := startParams["VALUE"] == "DATE" || len(startValue) == 8
		zone := icalTimeZone(startValue, startParams, loc)
		if allDay {
			zone = loc
		}
		from, to := occurrence.start.In(zone), moved.start.In(zone)
		if allDay && !moved.allDay {
			return "", errCantMoveSeries
		}
		days := int(time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC).Sub(time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
		clock := func(t time.Time) time.Duration {
			return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second
		}
		shiftClock := clock(to) - clock(from)
		shift := func(t time.Time) time.Time {
			w := t.In(zone)
			if allDay {
				return time.Date(t.Year(), t.Month(), t.Day()+days, 0, 0, 0, 0, t.Location())
			}
			return time.Date(w.Year(), w.Month(), w.Day()+days, w.Hour(), w.Minute(), w.Second(), 0, zone).Add(shiftClock)
		}

		sequence := 0
		if i, _, value := icalProperty(part.lines, "SEQUENCE"); i >= 0 {
			sequence, _ = strconv.Atoi(strings.TrimSpace(value))
		}
		depth := 0
		var out []string
		for _, line := range part.lines {
			name, params, value, ok := parseContentLine(line)
			switch {
			case name == "BEGIN":
				depth++
			case name == "END":
				depth--
			}
			if !ok || depth != 1 {
				out = append(out, line)
				continue
			}
			switch name {
			case "DTSTART", "DTEND":
				t, _, err := parseICalTime(value, params, loc)
				if err != nil {
					return "", err
				}
				line = icalTimeLike(name, params, value, shift(t), loc)
			case "EXDATE":
				var values []string
				for _, v := range strings.Split(value, ",") {
					t, _, err := parseICalTime(strings.TrimSpace(v), params, loc)
					if err != nil {
						return "", err
					}
					values = append(values, strings.SplitN(icalTimeLike("X", params, v, shift(t), loc), ":", 2)[1])
				}
				prefix := line[:strings.Index(line, ":")+1]
				line = prefix + strings.Join(values, ",")
			case "RRULE":
				rule, err := shiftRRule(value, days)
				if err != nil {
					return "", err
				}
				line = "RRULE:" + rule
			case "SEQUENCE", "DTSTAMP":
				continue
			}
			out = append(out, line)
		}
		parts[i].lines = stampEvent(out, sequence+1, now)

		// The changed occurrences now replace the moved ones. Those moved
		// only to another day (still at the series' time) take the change of
		// time; those given their own time keep it.
		keepsTime := map[int]bool{}
		for j, other := range parts {
			if !other.vevent || other.master || allDay {
				continue
			}
			if _, params, value := icalProperty(other.lines, "DTSTART"); value != "" {
				if t, _, err := parseICalTime(value, params, loc); err == nil && clock(t.In(zone)) == clock(from) {
					keepsTime[j] = true
				}
			}
		}
		for j, other := range parts {
			if !other.vevent || other.master {
				continue
			}
			depth := 0
			for k, line := range other.lines {
				name, params, value, ok := parseContentLine(line)
				switch {
				case name == "BEGIN":
					depth++
				case name == "END":
					depth--
				}
				if !ok || depth != 1 {
					continue
				}
				t, _, err := parseICalTime(value, params, loc)
				if err != nil {
					continue
				}
				switch name {
				case "RECURRENCE-ID":
					parts[j].lines[k] = icalTimeLike(name, params, value, shift(t), loc)
				case "DTSTART", "DTEND":
					if keepsTime[j] && !(params["VALUE"] == "DATE" || len(value) == 8) {
						w := t.In(zone)
						parts[j].lines[k] = icalTimeLike(name, params, value, time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), 0, zone).Add(shiftClock), loc)
					}
				}
			}
		}
		return joinICalParts(parts), nil
	}
	return "", fmt.Errorf("the event has no series")
}

// stampEvent adds a DTSTAMP and SEQUENCE to a VEVENT whose own were
// removed, before its END
func stampEvent(lines []string, sequence int, now time.Time) []string {
	out := make([]string, 0, len(lines)+2)
	for i, line := range lines {
		if i == len(lines)-1 {
			out = append(out, "DTSTAMP:"+caldavTime(now), "SEQUENCE:"+strconv.Itoa(sequence))
		}
		out = append(out, line)
	}
	return out
}

// shiftRRule moves a rule's weekdays (BYDAY) and days of the month
// (BYMONTHDAY) by days, and its UNTIL with them. Rules it can't move that
// way, such as BYDAY=1MO or BYSETPOS, give errCantMoveSeries.
func shiftRRule(rule string, days int) (string, error) {
	if days == 0 {
		return rule, nil
	}
	parts := strings.Split(rule, ";")
	for i, part := range parts {
		key, value, _ := strings.Cut(part, "=")
		switch strings.ToUpper(key) {
		case "BYDAY":
			var codes []string
			for _, code := range strings.Split(value, ",") {
				code = strings.ToUpper(strings.TrimSpace(code))
				index := -1
				for d, name := range icalWeekdays {
					if code == name {
						index = d
					}
				}
				if index < 0 {
					return "", errCantMoveSeries // e.g. 1MO, -1FR
				}
				codes = append(codes, icalWeekdays[((index+days)%7+7)%7])
			}
			parts[i] = "BYDAY=" + strings.Join(codes, ",")
		case "BYMONTHDAY":
			var values []string
			for _, v := range strings.Split(value, ",") {
				n, err := strconv.Atoi(strings.TrimSpace(v))
				if err != nil || n < 1 || n+days < 1 || n+days > 28 {
					return "", errCantMoveSeries
				}
				values = append(values, strconv.Itoa(n+days))
			}
			parts[i] = "BYMONTHDAY=" + strings.Join(values, ",")
		case "BYSETPOS", "BYYEARDAY", "BYWEEKNO":
			return "", errCantMoveSeries
		case "UNTIL":
			layout := "20060102T150405Z"
			if len(value) == 8 {
				layout = "20060102"
			}
			t, err := time.Parse(layout, value)
			if err != nil {
				return "", errCantMoveSeries
			}
			parts[i] = "UNTIL=" + t.AddDate(0, 0, days).Format(layout)
		}
	}
	return strings.Join(parts, ";"), nil
}

// nextRegularOccurrence returns the next occurrence of event's series that
// wasn't moved on its own
func (c *caldavClient) nextRegularOccurrence(event calendarEvent, now time.Time) (calendarEvent, bool, error) {
	events, err := c.query(now, now.Add(caldavSearchFuture))
	if err != nil {
		return calendarEvent{}, false, err
	}
	for _, e := range events {
		if e.href == event.href && e.uid == event.uid && e.recurrenceID.IsZero() && !e.end.Before(now) {
			return e, true, nil
		}
	}
	return calendarEvent{}, false, nil
}
