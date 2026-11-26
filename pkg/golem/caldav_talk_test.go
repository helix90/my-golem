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
	expectReply(t, ask, "move event yoga to Tuesday", `"yoga" repeats, and I can't move a repeating event yet. You can remove it and add it again.`)
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
