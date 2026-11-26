package golem

import (
	"strings"
	"testing"
	"time"
)

// talkToCalendar returns an ask function for a session of the CalDAV
// example AIML against a fake server, which gives the reply's text
func talkToCalendar(t *testing.T) (*fakeCalDAV, func(string) string) {
	t.Helper()
	f, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	session := g.CreateSession("test")
	return f, func(input string) string {
		t.Helper()
		response, err := g.ProcessInput(input, session)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		return RichText(response)
	}
}

func expectReply(t *testing.T, ask func(string) string, input, want string) {
	t.Helper()
	if got := ask(input); got != want {
		t.Errorf("%q:\n got  %q\n want %q", input, got, want)
	}
}

func eventCount(f *fakeCalDAV) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func TestCalendarAsksWhen(t *testing.T) {
	f, ask := talkToCalendar(t)
	expectReply(t, ask, "create event dentist", `When should I put "dentist" on your calendar?`)
	if eventCount(f) != 0 {
		t.Fatal("an event was created without a day or time")
	}
	expectReply(t, ask, "whenever", `I didn't catch a day or time. When should I put "dentist" on your calendar?`)
	expectReply(t, ask, "Friday at 3pm", `Okay, I've added "dentist" to your calendar on Friday at 3 PM.`)
	if eventCount(f) != 1 {
		t.Errorf("events: %d", eventCount(f))
	}

	expectReply(t, ask, "create event haircut", `When should I put "haircut" on your calendar?`)
	expectReply(t, ask, "never mind", "Okay, I won't add it.")
	if eventCount(f) != 1 {
		t.Errorf("events after never mind: %d", eventCount(f))
	}

	// A date alone is an all-day event, as before
	expectReply(t, ask, "create event holiday on December 24", `Okay, I've added "holiday" to your calendar on Thursday, December 24 (all day).`)
}

func TestCalendarUndo(t *testing.T) {
	f, ask := talkToCalendar(t)
	expectReply(t, ask, "create event lunch tomorrow at noon", `Okay, I've added "lunch" to your calendar tomorrow at noon.`)
	expectReply(t, ask, "undo", `Okay, I've taken "lunch" off your calendar.`)
	if eventCount(f) != 0 {
		t.Errorf("events after undo: %d", eventCount(f))
	}
	expectReply(t, ask, "calendar undo", "There's nothing to undo.")

	// The Undo button names its event, so it undoes that one
	ask("create event gym tomorrow at 7am")
	ask("create event swim tomorrow at 8am")
	f.mu.Lock()
	var gymUID string
	for path, data := range f.events {
		if strings.Contains(data, "SUMMARY:gym") {
			gymUID = strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".ics")
		}
	}
	f.mu.Unlock()
	expectReply(t, ask, "CALENDAR UNDO "+gymUID, `Okay, I've taken "gym" off your calendar.`)
	expectReply(t, ask, "what is on my calendar tomorrow", "Tomorrow, you have swim at 8 AM.")
}

func TestCalendarFindsEventsLoosely(t *testing.T) {
	_, ask := talkToCalendar(t)
	ask("create event Dentist on Friday at 2pm")
	for _, name := range []string{"dentist", "my dentist appointment", "the dentist", "the Dentist meeting"} {
		expectReply(t, ask, "get event "+name, `"Dentist" is on Friday at 2 PM until 3 PM.`)
	}
}

func TestCalendarSeveralMatches(t *testing.T) {
	_, ask := talkToCalendar(t)
	ask("create event Lunch with Sam tomorrow at noon")
	ask("create event Lunch with Ana on Friday at 1pm")
	got := ask("get event lunch")
	want := "You have 2 events matching \"lunch\":\n• Lunch with Sam tomorrow at noon\n• Lunch with Ana on Friday at 1 PM"
	if got != want {
		t.Errorf("get:\n got  %q\n want %q", got, want)
	}
	expectReply(t, ask, "delete event lunch", "You have 2 events matching \"lunch\":\n• Lunch with Sam tomorrow at noon\n• Lunch with Ana on Friday at 1 PM\nWhich one should I remove?")
}

func TestCalendarSeveralMatchesButtons(t *testing.T) {
	f, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	session := g.CreateSession("test")
	g.ProcessInput("create event Lunch with Sam tomorrow at noon", session)
	g.ProcessInput("create event Lunch with Ana on Friday at 1pm", session)
	response, _ := g.ProcessInput("delete event lunch", session)
	buttons := ParseRichResponse(response).Buttons
	if len(buttons) != 2 || buttons[1].Text != "Lunch with Ana on Friday at 1 PM" || !strings.HasPrefix(buttons[1].Postback, "DELETE EVENT ") {
		t.Fatalf("buttons: %+v", buttons)
	}
	// The button asks about that event
	response, _ = g.ProcessInput(buttons[1].Postback, session)
	if got := RichText(response); got != `Should I remove "Lunch with Ana" on Friday at 1 PM from your calendar?` {
		t.Errorf("button: %q", got)
	}
	if response, _ = g.ProcessInput("yes", session); RichText(response) != `Okay, I've removed "Lunch with Ana" on Friday at 1 PM from your calendar.` {
		t.Errorf("yes: %q", response)
	}
	if eventCount(f) != 1 {
		t.Errorf("events: %d", eventCount(f))
	}
}

func TestCalendarMove(t *testing.T) {
	f, ask := talkToCalendar(t)
	ask("create event Dentist on Friday at 2pm")
	expectReply(t, ask, "move event dentist to 4pm", `Okay, "Dentist" is now on Friday at 4 PM.`)
	expectReply(t, ask, "move event dentist to Monday", `Okay, "Dentist" is now on Monday at 4 PM.`)
	expectReply(t, ask, "move event dentist to tomorrow at 9:30am", `Okay, "Dentist" is now tomorrow at 9:30 AM.`)
	expectReply(t, ask, "move event dentist to back an hour", `Okay, "Dentist" is now tomorrow at 10:30 AM.`)
	expectReply(t, ask, "move event dentist to 30 minutes earlier", `Okay, "Dentist" is now tomorrow at 10 AM.`)
	expectReply(t, ask, "make it 11am", `Okay, "Dentist" is now tomorrow at 11 AM.`)
	expectReply(t, ask, "get event dentist", `"Dentist" is tomorrow at 11 AM until noon.`)
	if eventCount(f) != 1 {
		t.Errorf("events: %d", eventCount(f))
	}

	expectReply(t, ask, "create event haircut on Saturday at 10am", `Okay, I've added "haircut" to your calendar on Saturday at 10 AM.`)
	expectReply(t, ask, "make it 2pm", `Okay, "haircut" is now on Saturday at 2 PM.`)
	expectReply(t, ask, "move event dentist to whenever", `When should I move "Dentist" to?`)
	expectReply(t, ask, "Thursday at noon", `Okay, "Dentist" is now on Thursday at noon.`)
	expectReply(t, ask, "move event dentist", `When should I move "Dentist" to?`)
	expectReply(t, ask, "never mind", "Okay, I'll leave it where it is.")

	expectReply(t, ask, "move event unicorns to Friday", `I couldn't find "unicorns" on your calendar.`)
	ask("create event yoga every Monday at 6pm")
	expectReply(t, ask, "move event yoga to Tuesday", `"yoga" repeats. Should I move just the next one, on Monday at 6 PM, or every one?`)
}

func TestMoveEventKeepsTheRest(t *testing.T) {
	const data = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:x@y\r\nDTSTAMP:20261101T000000Z\r\n" +
		"DTSTART;TZID=Europe/London:20261120T140000\r\nDURATION:PT45M\r\nSEQUENCE:2\r\nSUMMARY:Dentist\r\n" +
		"LOCATION:High St\r\nBEGIN:VALARM\r\nTRIGGER:-PT30M\r\nACTION:DISPLAY\r\nDESCRIPTION:Dentist soon\r\nEND:VALARM\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	event := parseICalendar(data, time.UTC)[0]
	moved, ok, err := moveEvent(event, "4pm", caldavNow)
	if err != nil || !ok {
		t.Fatalf("move: %v %v", ok, err)
	}
	out, err := rewriteEventTimes(data, moved, caldavNow)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"DTSTART:20261120T160000Z\r\n", "DTEND:20261120T164500Z\r\n", "SEQUENCE:3\r\n", "LOCATION:High St\r\n", "TRIGGER:-PT30M\r\n", "DESCRIPTION:Dentist soon\r\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"DURATION", "TZID=", "SEQUENCE:2", "20261101T000000Z"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("%q left in\n%s", unwanted, out)
		}
	}
	if again := parseICalendar(out, time.UTC); len(again) != 1 || again[0].summary != "Dentist" || again[0].end.Sub(again[0].start) != 45*time.Minute {
		t.Errorf("reparsed: %+v", again)
	}
}

func TestEventListLayouts(t *testing.T) {
	at := func(day, hour int) time.Time { return time.Date(2026, 11, day, hour, 0, 0, 0, time.UTC) }
	ev := func(title string, day, hour int) calendarEvent {
		return calendarEvent{summary: title, start: at(day, hour), end: at(day, hour+1)}
	}
	text, layout := formatEventList([]calendarEvent{ev("standup", 17, 9), ev("lunch", 17, 12)}, caldavNow, false)
	if text != "standup at 9 AM and lunch at noon" || layout != "inline" {
		t.Errorf("inline: %q %s", text, layout)
	}
	events := []calendarEvent{ev("standup", 18, 9), ev("lunch", 18, 12), ev("review", 20, 14), {summary: "bins", start: at(26, 0), end: at(27, 0), allDay: true}}
	text, layout = formatEventList(events, caldavNow, true)
	if want := "\n• Tomorrow: standup at 9 AM, lunch at noon\n• Friday: review at 2 PM\n• Thursday, November 26: bins (all day)"; text != want || layout != "lines" {
		t.Errorf("lines:\n got  %q\n want %q", text, want)
	}
}

func TestEventQueries(t *testing.T) {
	got := strings.Join(eventQueries("my next dentist appointment"), "|")
	if got != "my next dentist appointment|dentist appointment|dentist" {
		t.Errorf("got %q", got)
	}
	if got := strings.Join(eventQueries("Bob"), "|"); got != "Bob" {
		t.Errorf("got %q", got)
	}
}

func TestCalendarMoveOneOccurrence(t *testing.T) {
	f, ask := talkToCalendar(t)
	ask("create event yoga every Monday at 6pm")
	expectReply(t, ask, "move event yoga to Tuesday", `"yoga" repeats. Should I move just the next one, on Monday at 6 PM, or every one?`)
	expectReply(t, ask, "just that one", `Okay, just that once: "yoga" is now on Tuesday, November 24 at 6 PM.`)
	expectReply(t, ask, "what do I have next week", "Next week, you have yoga on Tuesday, November 24 at 6 PM.")
	expectReply(t, ask, "what is on my calendar for November 30", "On Monday, November 30, you have yoga at 6 PM.")

	// Moving the moved one again changes it, not the series
	expectReply(t, ask, "move event yoga to 8pm", `"yoga" repeats. Should I move just the next one, on Tuesday, November 24 at 6 PM, or every one?`)
	expectReply(t, ask, "this one", `Okay, just that once: "yoga" is now on Tuesday, November 24 at 8 PM.`)

	// An event with a day names that day's one
	expectReply(t, ask, "move event yoga on November 30 to 7pm", `Okay, just that once: "yoga" is now on Monday, November 30 at 7 PM.`)
	expectReply(t, ask, "what is on my calendar for December 7", "On Monday, December 7, you have yoga at 6 PM.")

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) != 1 {
		t.Fatalf("events: %d", len(f.events))
	}
	for _, data := range f.events {
		if n := strings.Count(data, "BEGIN:VEVENT"); n != 3 {
			t.Errorf("want the series and 2 moved occurrences, got %d VEVENTs:\n%s", n, data)
		}
	}
}

func TestCalendarMoveEveryOccurrence(t *testing.T) {
	_, ask := talkToCalendar(t)
	ask("create event yoga every Monday at 6pm")
	// Wednesday is that of the next one's week
	expectReply(t, ask, "move event yoga to Wednesday at 7pm", `"yoga" repeats. Should I move just the next one, on Monday at 6 PM, or every one?`)
	expectReply(t, ask, "every one", `Okay, every "yoga" has moved: the next one is now on Wednesday, November 25 at 7 PM.`)
	expectReply(t, ask, "what do I have next week", "Next week, you have yoga on Wednesday, November 25 at 7 PM.")
	expectReply(t, ask, "what is on my calendar for December 2", "On Wednesday, December 2, you have yoga at 7 PM.")
	expectReply(t, ask, "what is on my calendar for November 30", "Your calendar is clear on Monday, November 30.")

	// With the scope given, it doesn't ask; a shift moves the time
	expectReply(t, ask, "move every event yoga to back half an hour", `Okay, every "yoga" has moved: the next one is now on Wednesday, November 25 at 7:30 PM.`)
	expectReply(t, ask, "move event yoga to Thursday", `"yoga" repeats. Should I move just the next one, on Wednesday, November 25 at 7:30 PM, or every one?`)
	expectReply(t, ask, "never mind", "Okay, I'll leave it where it is.")
	expectReply(t, ask, "CALENDAR MOVE ALL", `Okay, every "yoga" has moved: the next one is now on Thursday, November 26 at 7:30 PM.`)
}

func TestCalendarMoveEveryOccurrenceKeepsMovedOnes(t *testing.T) {
	_, ask := talkToCalendar(t)
	ask("create event standup every weekday at 9am")
	expectReply(t, ask, "move one event standup to 10am", `Okay, just that once: "standup" is now tomorrow at 10 AM.`)
	// The next one is the moved one; every one moves from where the series
	// puts them
	expectReply(t, ask, "move every event standup to 8:30am", `Okay, every "standup" has moved: the next one is now on Thursday at 8:30 AM.`)
	expectReply(t, ask, "what is on my calendar tomorrow", "Tomorrow, you have standup at 10 AM.")
	expectReply(t, ask, "what is on my calendar for Friday", "On Friday, November 20, you have standup at 8:30 AM.")
}

func TestShiftRRule(t *testing.T) {
	for _, tt := range []struct {
		rule string
		days int
		want string
	}{
		{"FREQ=WEEKLY;BYDAY=MO", 1, "FREQ=WEEKLY;BYDAY=TU"},
		{"FREQ=WEEKLY;BYDAY=SA,SU", 1, "FREQ=WEEKLY;BYDAY=SU,MO"},
		{"FREQ=WEEKLY;BYDAY=MO,WE,FR;UNTIL=20261218T235959Z", -1, "FREQ=WEEKLY;BYDAY=SU,TU,TH;UNTIL=20261217T235959Z"},
		{"FREQ=MONTHLY;BYMONTHDAY=3", 2, "FREQ=MONTHLY;BYMONTHDAY=5"},
		{"FREQ=DAILY;UNTIL=20261218", 3, "FREQ=DAILY;UNTIL=20261221"},
		{"FREQ=WEEKLY;BYDAY=MO", 0, "FREQ=WEEKLY;BYDAY=MO"},
	} {
		if got, err := shiftRRule(tt.rule, tt.days); err != nil || got != tt.want {
			t.Errorf("%s by %d = %q, %v; want %q", tt.rule, tt.days, got, err, tt.want)
		}
	}
	for _, rule := range []string{"FREQ=MONTHLY;BYDAY=1MO", "FREQ=MONTHLY;BYDAY=MO;BYSETPOS=-1", "FREQ=MONTHLY;BYMONTHDAY=30"} {
		if _, err := shiftRRule(rule, 1); err != errCantMoveSeries {
			t.Errorf("%s: err = %v", rule, err)
		}
	}
}

// A series written in its time zone moves in wall-clock time, so it keeps
// its local time across a daylight saving change; its alarms, left-out
// dates and moved occurrences come along
func TestMoveSeriesInItsTimeZone(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("no time zone data")
	}
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, london) // a Tuesday, before the clocks go back
	const data = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:yoga@x\r\nDTSTAMP:20261001T000000Z\r\n" +
		"DTSTART;TZID=Europe/London:20261019T180000\r\nDTEND;TZID=Europe/London:20261019T190000\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=MO\r\nEXDATE;TZID=Europe/London:20261102T180000\r\nSEQUENCE:4\r\nSUMMARY:yoga\r\n" +
		"BEGIN:VALARM\r\nTRIGGER:-PT15M\r\nACTION:DISPLAY\r\nEND:VALARM\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:yoga@x\r\nRECURRENCE-ID;TZID=Europe/London:20261109T180000\r\n" +
		"DTSTART;TZID=Europe/London:20261110T180000\r\nDTEND;TZID=Europe/London:20261110T190000\r\nSUMMARY:yoga\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	events := parseICalendar(data, london)
	occurrence := expandEvents(events, now, now.AddDate(0, 0, 7))[0] // Monday October 26, after the change
	if occurrence.start.Hour() != 18 {
		t.Fatalf("occurrence: %v", occurrence.start)
	}
	moved, ok, err := moveEvent(occurrence, "Tuesday at 7pm", now)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	out, err := moveSeries(data, occurrence, moved, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"DTSTART;TZID=Europe/London:20261020T190000\r\n", "DTEND;TZID=Europe/London:20261020T200000\r\n",
		"RRULE:FREQ=WEEKLY;BYDAY=TU\r\n", "EXDATE;TZID=Europe/London:20261103T190000\r\n", "SEQUENCE:5\r\n",
		"TRIGGER:-PT15M\r\n", "RECURRENCE-ID;TZID=Europe/London:20261110T190000\r\n",
		"DTSTART;TZID=Europe/London:20261110T190000\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	// Every Tuesday at 7 PM London time, on both sides of the change (the
	// clocks go back on October 25), except the left-out November 3, with
	// the moved one on its own day at the new time
	var got []string
	for _, e := range expandEvents(parseICalendar(out, london), now, now.AddDate(0, 0, 28)) {
		got = append(got, e.start.In(london).Format("Mon Jan 2 15:04"))
	}
	if want := "Tue Oct 20 19:00|Tue Oct 27 19:00|Tue Nov 10 19:00"; strings.Join(got, "|") != want {
		t.Errorf("occurrences %v, want %s", got, want)
	}
}

func TestMoveToAWeekdayOfTheEventsWeek(t *testing.T) {
	at := func(month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, time.UTC)
	}
	event := calendarEvent{summary: "dentist", start: at(12, 14, 15), end: at(12, 14, 16)} // a Monday
	for text, want := range map[string]time.Time{
		"Wednesday":         at(12, 16, 15),
		"on Sunday at 10am": at(12, 20, 10),
		"next Wednesday":    at(11, 18, 15), // "next" counts from today, as for new events
		"Friday the 4th":    at(12, 18, 15),
	} {
		moved, ok, err := moveEvent(event, text, caldavNow)
		if err != nil || !ok || !moved.start.Equal(want) {
			t.Errorf("%q: %v %v %v, want %v", text, moved.start, ok, err, want)
		}
	}
	// This week's has passed: the next one
	soon := calendarEvent{summary: "lunch", start: at(11, 18, 12), end: at(11, 18, 13)} // Wednesday, tomorrow
	if moved, _, _ := moveEvent(soon, "Monday", caldavNow); !moved.start.Equal(at(11, 23, 12)) {
		t.Errorf("Monday: %v", moved.start)
	}
}

func TestEventQueriesSingular(t *testing.T) {
	for input, want := range map[string]string{
		"standups":     "standups|standup",
		"yoga classes": "yoga classes|yoga class",
		"my parties":   "my parties|parties|party",
		"Boss":         "Boss",
	} {
		if got := strings.Join(eventQueries(input), "|"); got != want {
			t.Errorf("%q: %q, want %q", input, got, want)
		}
	}
}
