package golem

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Recurring events: "every Monday", "every weekday", "every other week",
// "monthly", "every Tuesday and Thursday until December 18", "every day for
// 2 weeks", "10 times". parseEventText removes the phrase from the title and
// sets the event's recurrence; the event is written with an RRULE, so the
// calendar server repeats it.
//
// A recurring event with a time is written in the bot's time zone (TZID and
// a VTIMEZONE), so that it keeps its local time across daylight saving time
// changes. When the zone has no IANA name (TZ unset and the system zone not
// known), it is written in UTC, which keeps the weekday right but not the
// local time across a change.

// eventRecurrence is how an event repeats
type eventRecurrence struct {
	freq     string // DAILY, WEEKLY, MONTHLY or YEARLY
	interval int    // 1 for every, 2 for every other, ...
	byDay    []time.Weekday
	until    time.Time // the last day it can occur on; zero for no end
	count    int       // how many times; 0 for no limit
	forN     int       // "for 6 weeks": until is computed from the start
	forUnit  string    // days, weeks, months or years
}

const recurDayWord = `(?:(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday)s?|mon|tue|tues|wed|weds|thu|thur|thurs|fri|sat|sun)`

var (
	recurDaysRe   = regexp.MustCompile(`(?i)\b(every|each|on)\s+(?:(other|second|alternate)\s+)?(` + recurDayWord + `(?:\s*(?:,\s*and|,|and|&|/|or)\s*` + recurDayWord + `)*)\b`)
	recurDayRe    = regexp.MustCompile(`(?i)\b` + recurDayWord + `\b`)
	recurEveryRe  = regexp.MustCompile(`(?i)\b(?:every|each)\s+(?:(other|second|alternate)\s+|(\d+|two|three|four|five|six)\s+)?(day|weekday|weekend|week|fortnight|month|year)s?\b`)
	recurAdverbRe = regexp.MustCompile(`(?i)\b(daily|weekly|fortnightly|biweekly|monthly|yearly|annually)\b`)
	recurUntilRe  = regexp.MustCompile(`(?i)\b(?:until|till|through|thru)\b`)
	recurTimesRe  = regexp.MustCompile(`(?i)\b(?:for\s+)?(\d+|two|three|four|five|six|seven|eight|nine|ten|twelve)\s+times\b`)
	recurForRe    = regexp.MustCompile(`(?i)\bfor\s+(?:the\s+next\s+)?(\d+|two|three|four|five|six|seven|eight|nine|ten|twelve)\s+(days|weeks|months|years)\b`)
	recurStartRe  = regexp.MustCompile(`(?i)\b(?:starting|beginning)(?:\s+(?:on|from))?\b`)
	recurNumbers  = map[string]int{"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "twelve": 12}
	icalDays      = map[time.Weekday]string{time.Monday: "MO", time.Tuesday: "TU", time.Wednesday: "WE", time.Thursday: "TH", time.Friday: "FR", time.Saturday: "SA", time.Sunday: "SU"}
)

func recurNumber(text string) int {
	if n, err := strconv.Atoi(text); err == nil {
		return n
	}
	return recurNumbers[strings.ToLower(text)]
}

// recurWeekday reads a day word such as "Mon", "tues" or "Fridays"
func recurWeekday(word string) time.Weekday {
	switch strings.ToLower(word)[:2] {
	case "mo":
		return time.Monday
	case "tu":
		return time.Tuesday
	case "we":
		return time.Wednesday
	case "th":
		return time.Thursday
	case "fr":
		return time.Friday
	case "sa":
		return time.Saturday
	}
	return time.Sunday
}

// takeEventRecurrence finds how an event repeats in *title and removes it.
// It returns nil when the text doesn't say. today is the start of the
// current day.
func takeEventRecurrence(title *string, today time.Time) (*eventRecurrence, error) {
	cut := func(loc []int) {
		*title = (*title)[:loc[0]] + " " + (*title)[loc[1]:]
	}
	var rec *eventRecurrence

	// every Monday, each Tue and Thu, on Mondays
	for _, m := range recurDaysRe.FindAllStringSubmatchIndex(*title, -1) {
		keyword, days := (*title)[m[2]:m[3]], (*title)[m[6]:m[7]]
		words := recurDayRe.FindAllString(days, -1)
		if strings.EqualFold(keyword, "on") && m[4] < 0 {
			// "on Monday" is one day; only "on Mondays" repeats
			plural := true
			for _, w := range words {
				if !strings.HasSuffix(strings.ToLower(w), "days") {
					plural = false
				}
			}
			if !plural {
				continue
			}
		}
		rec = &eventRecurrence{freq: "WEEKLY", interval: 1}
		if m[4] >= 0 {
			rec.interval = 2
		}
		seen := map[time.Weekday]bool{}
		for _, w := range words {
			if day := recurWeekday(w); !seen[day] {
				seen[day] = true
				rec.byDay = append(rec.byDay, day)
			}
		}
		cut(m[:2])
		break
	}

	// every day, every weekday, every other week, every 3 months
	if rec == nil {
		if m := recurEveryRe.FindStringSubmatchIndex(*title); m != nil {
			rec = &eventRecurrence{interval: 1}
			if m[2] >= 0 {
				rec.interval = 2
			} else if m[4] >= 0 {
				rec.interval = recurNumber((*title)[m[4]:m[5]])
			}
			switch unit := strings.ToLower((*title)[m[6]:m[7]]); unit {
			case "day":
				rec.freq = "DAILY"
			case "weekday":
				rec.freq, rec.byDay = "WEEKLY", []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
			case "weekend":
				rec.freq, rec.byDay = "WEEKLY", []time.Weekday{time.Saturday, time.Sunday}
			case "week":
				rec.freq = "WEEKLY"
			case "fortnight":
				rec.freq, rec.interval = "WEEKLY", 2*rec.interval
			case "month":
				rec.freq = "MONTHLY"
			case "year":
				rec.freq = "YEARLY"
			}
			cut(m[:2])
		}
	}

	// daily, weekly, monthly
	if rec == nil {
		if m := recurAdverbRe.FindStringSubmatchIndex(*title); m != nil {
			rec = &eventRecurrence{interval: 1}
			switch strings.ToLower((*title)[m[2]:m[3]]) {
			case "daily":
				rec.freq = "DAILY"
			case "weekly":
				rec.freq = "WEEKLY"
			case "fortnightly", "biweekly":
				rec.freq, rec.interval = "WEEKLY", 2
			case "monthly":
				rec.freq = "MONTHLY"
			default:
				rec.freq = "YEARLY"
			}
			cut(m[:2])
		}
	}
	if rec == nil {
		return nil, nil
	}
	if rec.interval < 1 || rec.interval > 99 {
		return nil, fmt.Errorf("invalid repeat interval")
	}

	// How long it goes on: until a date, N times, or for N weeks
	if m := recurUntilRe.FindStringIndex(*title); m != nil {
		tail := (*title)[m[1]:]
		remove := func(re *regexp.Regexp) []string {
			idx := re.FindStringSubmatchIndex(tail)
			if idx == nil {
				return nil
			}
			groups := make([]string, len(idx)/2)
			for i := range groups {
				if idx[2*i] >= 0 {
					groups[i] = tail[idx[2*i]:idx[2*i+1]]
				}
			}
			tail = tail[:idx[0]] + " " + tail[idx[1]:]
			return groups
		}
		until, ok, err := takeEventDate(remove, today)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("no date after %q", strings.TrimSpace((*title)[m[0]:m[1]]))
		}
		rec.until = until
		*title = (*title)[:m[0]] + " " + tail
	}
	if m := recurTimesRe.FindStringSubmatchIndex(*title); m != nil {
		rec.count = recurNumber((*title)[m[2]:m[3]])
		cut(m[:2])
	} else if m := recurForRe.FindStringSubmatchIndex(*title); m != nil {
		rec.forN, rec.forUnit = recurNumber((*title)[m[2]:m[3]]), strings.ToLower((*title)[m[4]:m[5]])
		cut(m[:2])
	}
	if rec.count < 0 || rec.count > 999 || rec.forN < 0 || rec.forN > 999 {
		return nil, fmt.Errorf("too many repeats")
	}
	return rec, nil
}

// applyRecurrence fixes a recurring event's first occurrence and end once its
// start is known: an event on given weekdays starts on the first of them on
// or after its date
func applyRecurrence(event *calendarEvent, rec *eventRecurrence) error {
	if len(rec.byDay) > 0 {
		for i := 0; i < 7; i++ {
			day := event.start.AddDate(0, 0, i)
			if containsWeekday(rec.byDay, day.Weekday()) {
				shift := day.Sub(event.start)
				event.start, event.end = day, event.end.Add(shift)
				break
			}
		}
	}
	if rec.forN > 0 {
		end := startOfDay(event.start)
		switch rec.forUnit {
		case "days":
			end = end.AddDate(0, 0, rec.forN)
		case "weeks":
			end = end.AddDate(0, 0, 7*rec.forN)
		case "months":
			end = end.AddDate(0, rec.forN, 0)
		default:
			end = end.AddDate(rec.forN, 0, 0)
		}
		rec.until = end.AddDate(0, 0, -1)
	}
	if !rec.until.IsZero() && rec.until.Before(startOfDay(event.start)) {
		return fmt.Errorf("it would end before it starts")
	}
	event.recurrence = rec
	event.repeats = true
	return nil
}

func containsWeekday(days []time.Weekday, day time.Weekday) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

// eventTimeZoneName returns the IANA name of loc, such as "Europe/London",
// or "" when it has none that LoadLocation knows. time.Local is named by TZ.
func eventTimeZoneName(loc *time.Location) string {
	name := loc.String()
	if loc == time.Local || name == "Local" {
		name = strings.TrimPrefix(os.Getenv("TZ"), ":")
	}
	if name == "" || name == "UTC" {
		return ""
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ""
	}
	return name
}

// rrule returns the event's RRULE value. inUTC says the event's start is
// written in UTC, so that BYDAY must name the UTC weekdays and UNTIL is a
// UTC time; allDay events use dates.
func (rec *eventRecurrence) rrule(event calendarEvent, inUTC bool) string {
	parts := []string{"FREQ=" + rec.freq}
	if rec.interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", rec.interval))
	}
	if len(rec.byDay) > 0 {
		shift := 0
		if inUTC && !event.allDay {
			shift = int(event.start.UTC().Weekday()) - int(event.start.Weekday())
		}
		days := make([]string, len(rec.byDay))
		for i, d := range rec.byDay {
			days[i] = icalDays[time.Weekday((int(d)+shift+7)%7)]
		}
		parts = append(parts, "BYDAY="+strings.Join(days, ","))
	}
	switch {
	case rec.count > 0:
		parts = append(parts, fmt.Sprintf("COUNT=%d", rec.count))
	case !rec.until.IsZero() && event.allDay:
		parts = append(parts, "UNTIL="+rec.until.Format("20060102"))
	case !rec.until.IsZero():
		last := time.Date(rec.until.Year(), rec.until.Month(), rec.until.Day(), 23, 59, 59, 0, event.start.Location())
		parts = append(parts, "UNTIL="+caldavTime(last))
	}
	return strings.Join(parts, ";")
}

// describe says how the event repeats, e.g. "every Monday at 2:00 PM,
// starting tomorrow, until Dec 18"
func (rec *eventRecurrence) describe(event calendarEvent, now time.Time) string {
	start := event.start.In(now.Location())
	var what string
	switch rec.freq {
	case "DAILY":
		what = "every day"
		if rec.interval > 1 {
			what = fmt.Sprintf("every %d days", rec.interval)
		}
	case "WEEKLY":
		days := rec.byDay
		if len(days) == 0 {
			days = []time.Weekday{start.Weekday()}
		}
		sorted := append([]time.Weekday(nil), days...)
		sort.Slice(sorted, func(i, j int) bool { return (sorted[i]+6)%7 < (sorted[j]+6)%7 })
		var names []string
		for _, d := range sorted {
			names = append(names, d.String())
		}
		list := joinWithAnd(names)
		switch {
		case len(sorted) == 5 && !containsWeekday(sorted, time.Saturday) && !containsWeekday(sorted, time.Sunday):
			list = "weekday"
		case len(sorted) == 2 && containsWeekday(sorted, time.Saturday) && containsWeekday(sorted, time.Sunday):
			list = "Saturday and Sunday"
		}
		switch rec.interval {
		case 1:
			what = "every " + list
		case 2:
			what = "every other " + list
		default:
			what = fmt.Sprintf("every %d weeks on %s", rec.interval, list)
		}
	case "MONTHLY":
		what = "every month on the " + ordinal(start.Day())
		if rec.interval > 1 {
			what = fmt.Sprintf("every %d months on the %s", rec.interval, ordinal(start.Day()))
		}
	default:
		what = "every year on " + start.Format("January 2")
		if rec.interval > 1 {
			what = fmt.Sprintf("every %d years on %s", rec.interval, start.Format("January 2"))
		}
	}
	if !event.allDay {
		what += " at " + start.Format("3:04 PM")
	}
	what += ", starting " + dayPhrase(start, now)
	switch {
	case rec.count > 0:
		what += fmt.Sprintf(", %d times", rec.count)
	case !rec.until.IsZero():
		what += ", until " + rec.until.Format("Mon Jan 2")
		if rec.until.Year() != now.Year() {
			what += rec.until.Format(" 2006")
		}
	}
	return what
}

// dayPhrase names a day: "today", "tomorrow" or "Mon Nov 30"
func dayPhrase(day, now time.Time) string {
	switch days := int(startOfDay(day).Sub(startOfDay(now)).Hours()+12) / 24; {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case day.Year() == now.Year():
		return day.Format("Mon Jan 2")
	}
	return day.Format("Mon Jan 2 2006")
}

func joinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

// vtimezone returns a VTIMEZONE for the zone name (loc), covering times from
// from on. A zone that changes the same way every year (such as the last
// Sunday of March) gets yearly rules; one with no changes, a single offset;
// any other, its changes for twenty years.
func vtimezone(name string, loc *time.Location, from time.Time) []string {
	type transition struct {
		at             time.Time // the first instant with the new offset
		fromOff, toOff int
		dst            bool
		abbr           string
	}
	findTransitions := func(start, end time.Time) []transition {
		var found []transition
		_, off := start.In(loc).Zone()
		for day := start; day.Before(end); day = day.Add(24 * time.Hour) {
			next := day.Add(24 * time.Hour)
			if _, nextOff := next.In(loc).Zone(); nextOff != off {
				lo, hi := day, next
				for hi.Sub(lo) > time.Minute {
					mid := lo.Add(hi.Sub(lo) / 2)
					if _, o := mid.In(loc).Zone(); o == off {
						lo = mid
					} else {
						hi = mid
					}
				}
				hi = hi.Truncate(time.Minute)
				abbr, _ := hi.In(loc).Zone()
				found = append(found, transition{at: hi, fromOff: off, toOff: nextOff, dst: hi.In(loc).IsDST(), abbr: abbr})
				off = nextOff
			}
		}
		return found
	}
	offset := func(seconds int) string {
		sign := "+"
		if seconds < 0 {
			sign, seconds = "-", -seconds
		}
		return fmt.Sprintf("%s%02d%02d", sign, seconds/3600, seconds%3600/60)
	}
	// The wall-clock time a transition starts at, in the offset before it
	wall := func(t transition) time.Time {
		return t.at.UTC().Add(time.Duration(t.fromOff) * time.Second)
	}
	component := func(t transition, extra ...string) []string {
		kind := "STANDARD"
		if t.dst {
			kind = "DAYLIGHT"
		}
		lines := []string{"BEGIN:" + kind, "DTSTART:" + wall(t).Format("20060102T150405")}
		lines = append(lines, extra...)
		return append(lines, "TZOFFSETFROM:"+offset(t.fromOff), "TZOFFSETTO:"+offset(t.toOff), "TZNAME:"+t.abbr, "END:"+kind)
	}

	start := time.Date(from.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	lines := []string{"BEGIN:VTIMEZONE", "TZID:" + name}
	sample := findTransitions(start, start.AddDate(6, 0, 0))
	if len(sample) == 0 {
		abbr, off := from.In(loc).Zone()
		lines = append(lines, "BEGIN:STANDARD", "DTSTART:19700101T000000", "TZOFFSETFROM:"+offset(off),
			"TZOFFSETTO:"+offset(off), "TZNAME:"+abbr, "END:STANDARD")
		return append(lines, "END:VTIMEZONE")
	}

	// A yearly rule per kind of change, when every year follows it
	yearly := func(group []transition) (string, bool) {
		if len(group) < 2 {
			return "", false
		}
		first := wall(group[0])
		nth, allLast := 0, true
		for i, t := range group {
			w := wall(t)
			if w.Month() != first.Month() || w.Weekday() != first.Weekday() || w.Hour() != first.Hour() || w.Minute() != first.Minute() ||
				t.fromOff != group[0].fromOff || t.toOff != group[0].toOff {
				return "", false
			}
			n := (w.Day()-1)/7 + 1
			if i == 0 {
				nth = n
			} else if n != nth {
				nth = 0
			}
			if w.AddDate(0, 0, 7).Month() == w.Month() {
				allLast = false
			}
		}
		// "Last" first: the last Sunday of March is the 4th in some years
		// and the 5th in others
		switch {
		case allLast:
			nth = -1
		case nth > 0 && nth < 5:
		default:
			return "", false
		}
		return fmt.Sprintf("RRULE:FREQ=YEARLY;BYMONTH=%d;BYDAY=%d%s", int(first.Month()), nth, icalDays[first.Weekday()]), true
	}
	var toDST, toStd []transition
	for _, t := range sample {
		if t.dst {
			toDST = append(toDST, t)
		} else {
			toStd = append(toStd, t)
		}
	}
	dstRule, ok1 := yearly(toDST)
	stdRule, ok2 := yearly(toStd)
	if ok1 && ok2 {
		lines = append(lines, component(toDST[0], dstRule)...)
		lines = append(lines, component(toStd[0], stdRule)...)
		return append(lines, "END:VTIMEZONE")
	}
	for _, t := range findTransitions(start, start.AddDate(20, 0, 0)) {
		lines = append(lines, component(t)...)
	}
	return append(lines, "END:VTIMEZONE")
}

// recurUntilDateRe matches an RRULE whose UNTIL is a date, not a date-time
var recurUntilDateRe = regexp.MustCompile(`(?i)(?:^|;)UNTIL=\d{8}(?:;|$)`)

// expandEvents turns events read from a calendar into the occurrences that
// overlap [start, end). A repeating event becomes one event per occurrence,
// computed here in the zone its DTSTART is written in, without the dates
// its EXDATEs leave out or that a changed occurrence (RECURRENCE-ID)
// replaces. The calendar server isn't asked to expand them: Radicale, for
// one, repeats an event at its first occurrence's UTC time, so occurrences
// after a daylight saving time change come back an hour off.
func expandEvents(events []calendarEvent, start, end time.Time) []calendarEvent {
	type series struct{ href, uid string }
	replaced := make(map[series][]time.Time)
	for _, e := range events {
		if !e.recurrenceID.IsZero() {
			k := series{e.href, e.uid}
			replaced[k] = append(replaced[k], e.recurrenceID)
		}
	}
	var out []calendarEvent
	for _, e := range events {
		if e.rrule != "" && e.recurrenceID.IsZero() {
			out = append(out, expandRecurring(e, start, end, replaced[series{e.href, e.uid}])...)
		} else if eventOverlaps(e, start, end) {
			out = append(out, e)
		}
	}
	return out
}

// eventOverlaps reports whether an event is (partly) within [start, end);
// one with no length counts when it starts in the range
func eventOverlaps(e calendarEvent, start, end time.Time) bool {
	if !e.end.After(e.start) {
		return !e.start.Before(start) && e.start.Before(end)
	}
	return e.start.Before(end) && e.end.After(start)
}

// expandRecurring returns the occurrences of a repeating event that overlap
// [start, end), leaving out its EXDATEs and the occurrences in replaced
func expandRecurring(e calendarEvent, start, end time.Time, replaced []time.Time) []calendarEvent {
	loc := start.Location()
	zone := e.zone
	if zone == nil {
		zone = loc
	}
	opt, err := rrule.StrToROption(e.rrule)
	if err != nil {
		// A rule we can't read: show the event as written
		if eventOverlaps(e, start, end) {
			return []calendarEvent{e}
		}
		return nil
	}
	opt.Dtstart = e.start.In(zone)
	if e.allDay && !opt.Until.IsZero() && recurUntilDateRe.MatchString(e.rrule) {
		// UNTIL is the last day it occurs on, in the event's zone
		opt.Until = time.Date(opt.Until.Year(), opt.Until.Month(), opt.Until.Day(), 23, 59, 59, 0, zone)
	}
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		if eventOverlaps(e, start, end) {
			return []calendarEvent{e}
		}
		return nil
	}

	length := e.end.Sub(e.start)
	days := int(length.Hours()/24 + 0.5)
	left := append(append([]time.Time(nil), e.exdates...), replaced...)
	isLeftOut := func(t time.Time) bool {
		for _, x := range left {
			if x.Equal(t) || (e.allDay && x.In(zone).Format("20060102") == t.Format("20060102")) {
				return true
			}
		}
		return false
	}

	var out []calendarEvent
	for _, t := range rule.Between(start.Add(-length), end, true) {
		if isLeftOut(t) {
			continue
		}
		occurrence := e
		occurrence.start = t.In(loc)
		if e.allDay {
			// Dates, so a day stays a day across a clock change
			occurrence.start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
			occurrence.end = occurrence.start.AddDate(0, 0, days)
		} else {
			occurrence.end = t.Add(length).In(loc)
		}
		occurrence.repeats = true
		if eventOverlaps(occurrence, start, end) {
			out = append(out, occurrence)
		}
	}
	return out
}
