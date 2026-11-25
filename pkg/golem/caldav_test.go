package golem

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// caldavNow is the time the CalDAV tests run at: Tuesday 17 November 2026,
// 10:00 UTC
var caldavNow = time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

func TestParseEventText(t *testing.T) {
	day := func(month time.Month, d, hour, minute int) time.Time {
		year := 2026
		if month < time.November {
			year = 2027
		}
		return time.Date(year, month, d, hour, minute, 0, 0, time.UTC)
	}
	tests := []struct {
		text       string
		title      string
		start, end time.Time
		allDay     bool
	}{
		{"Lunch with Sam tomorrow at noon", "Lunch with Sam", day(11, 18, 12, 0), day(11, 18, 13, 0), false},
		{"Dentist on Nov 20th, 2026 at 9:15 a.m.", "Dentist", day(11, 20, 9, 15), day(11, 20, 10, 15), false},
		{"Team sync tomorrow at 14:00 for 30 minutes", "Team sync", day(11, 18, 14, 0), day(11, 18, 14, 30), false},
		{"Bob's party on Friday", "Bob's party", day(11, 20, 0, 0), day(11, 21, 0, 0), true},
		{"Review on 2026-12-01 at 2:30pm for 2 hours", "Review", day(12, 1, 14, 30), day(12, 1, 16, 30), false},
		{"Call mom at 9am", "Call mom", day(11, 18, 9, 0), day(11, 18, 10, 0), false},
		{"Call mom at 3", "Call mom", day(11, 17, 15, 0), day(11, 17, 16, 0), false},
		{"Holiday on 25 December", "Holiday", day(12, 25, 0, 0), day(12, 26, 0, 0), true},
		{"Retro next Tuesday at 4pm for an hour", "Retro", day(11, 24, 16, 0), day(11, 24, 17, 0), false},
		{"Planning this Tuesday at 4 PM", "Planning", day(11, 17, 16, 0), day(11, 17, 17, 0), false},
		{"Party on January 5", "Party", day(1, 5, 0, 0), day(1, 6, 0, 0), true},
		{"Team Meeting", "Team Meeting", day(11, 17, 0, 0), day(11, 18, 0, 0), true},
		{"Launch at midnight", "Launch", day(11, 18, 0, 0), day(11, 18, 1, 0), false},
		{"tomorrow at 8am", "Event", day(11, 18, 8, 0), day(11, 18, 9, 0), false},
		{"Trip the day after tomorrow", "Trip", day(11, 19, 0, 0), day(11, 20, 0, 0), true},
	}
	for _, tt := range tests {
		event, err := parseEventText(tt.text, caldavNow)
		if err != nil {
			t.Errorf("%q: %v", tt.text, err)
			continue
		}
		if event.summary != tt.title || !event.start.Equal(tt.start) || !event.end.Equal(tt.end) || event.allDay != tt.allDay {
			t.Errorf("%q: got %q %v to %v (all day %v), expected %q %v to %v (all day %v)",
				tt.text, event.summary, event.start, event.end, event.allDay, tt.title, tt.start, tt.end, tt.allDay)
		}
	}

	for _, text := range []string{"Party on Feb 30", "Meeting at 13pm", "Call at 25:00", "Sync on 2026-13-01"} {
		if _, err := parseEventText(text, caldavNow); err == nil {
			t.Errorf("%q: expected an error", text)
		}
	}
}

func TestParseICalendar(t *testing.T) {
	const data = "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTIMEZONE\r\nTZID:America/New_York\r\nBEGIN:STANDARD\r\nDTSTART:19701101T020000\r\nEND:STANDARD\r\nEND:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\nUID:a@x\r\nSUMMARY:Planning\\, part 2: a very long title that the server has folded across\r\n  two lines\r\n" +
		"DTSTART;TZID=America/New_York:20261117T090000\r\nDURATION:PT1H30M\r\nLOCATION:Room 4\r\nDESCRIPTION:Bring notes\\nand coffee\r\n" +
		"RRULE:FREQ=WEEKLY\r\nBEGIN:VALARM\r\nDESCRIPTION:Reminder\r\nTRIGGER:-PT10M\r\nEND:VALARM\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:b@x\r\nSUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20261120\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:c@x\r\nSUMMARY:Call\r\nDTSTART:20261117T150000Z\r\nDTEND:20261117T153000Z\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events := parseICalendar(data, time.UTC)
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d: %+v", len(events), events)
	}
	planning, holiday, call := events[0], events[1], events[2]
	if planning.summary != "Planning, part 2: a very long title that the server has folded across two lines" {
		t.Errorf("Unexpected summary %q", planning.summary)
	}
	if want := time.Date(2026, 11, 17, 14, 0, 0, 0, time.UTC); !planning.start.Equal(want) || !planning.end.Equal(want.Add(90*time.Minute)) {
		t.Errorf("Expected 14:00-15:30 UTC, got %v to %v", planning.start, planning.end)
	}
	if planning.location != "Room 4" || planning.description != "Bring notes\nand coffee" || !planning.repeats {
		t.Errorf("Unexpected location, description or repeats: %+v", planning)
	}
	if !holiday.allDay || !holiday.start.Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) || !holiday.end.Equal(time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Unexpected all-day event: %+v", holiday)
	}
	if call.uid != "c@x" || call.end.Sub(call.start) != 30*time.Minute {
		t.Errorf("Unexpected event: %+v", call)
	}
}

func TestEventICalendarRoundTrip(t *testing.T) {
	event := calendarEvent{
		uid:     "1234@golem",
		summary: "Dinner; with Ana, Bo & Chloé at the long-named restaurant by the river \\ harbour",
		start:   time.Date(2026, 11, 18, 19, 0, 0, 0, time.UTC),
		end:     time.Date(2026, 11, 18, 20, 0, 0, 0, time.UTC),
	}
	data := eventICalendar(event, caldavNow)
	for _, line := range strings.Split(strings.TrimSuffix(data, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("Line longer than 75 bytes: %q", line)
		}
	}
	events := parseICalendar(data, time.UTC)
	if len(events) != 1 || events[0].summary != event.summary || !events[0].start.Equal(event.start) || !events[0].end.Equal(event.end) || events[0].uid != event.uid {
		t.Errorf("Round trip changed the event: %+v\n%s", events, data)
	}
}

// fakeCalDAV is a CalDAV server holding events in memory, for user alice
// with password secret
type fakeCalDAV struct {
	mu       sync.Mutex
	events   map[string]string // path -> iCalendar
	requests []string          // "METHOD path"
	headers  []http.Header
}

var timeRangeRe = regexp.MustCompile(`time-range start="(\w+)" end="(\w+)"`)

func newFakeCalDAV(t *testing.T) (*fakeCalDAV, *httptest.Server) {
	f := &fakeCalDAV{events: make(map[string]string)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.headers = append(f.headers, r.Header.Clone())
		if user, password, ok := r.BasicAuth(); !ok || user != "alice" || password != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case "REPORT":
			m := timeRangeRe.FindStringSubmatch(string(body))
			if m == nil || r.Header.Get("Depth") != "1" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			start, _ := time.Parse("20060102T150405Z", m[1])
			end, _ := time.Parse("20060102T150405Z", m[2])
			var paths []string
			for path := range f.events {
				if strings.HasPrefix(path, r.URL.Path) {
					paths = append(paths, path)
				}
			}
			sort.Strings(paths)
			var out strings.Builder
			out.WriteString(`<?xml version="1.0"?><multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">`)
			for _, path := range paths {
				events := parseICalendar(f.events[path], time.UTC)
				// A repeating event that has started may occur in the range,
				// so a server returns it; the client expands it
				repeating := len(events) > 0 && events[0].rrule != "" && events[0].start.Before(end)
				if len(events) == 0 || (!repeating && (!events[0].start.Before(end) || !events[0].end.After(start))) {
					continue
				}
				fmt.Fprintf(&out, `<response><href>%s</href><propstat><prop><getetag>"1"</getetag><C:calendar-data><![CDATA[%s]]></C:calendar-data></prop><status>HTTP/1.1 200 OK</status></propstat></response>`, path, f.events[path])
			}
			out.WriteString(`</multistatus>`)
			w.WriteHeader(http.StatusMultiStatus)
			w.Write([]byte(out.String()))
		case http.MethodPut:
			if _, exists := f.events[r.URL.Path]; exists && r.Header.Get("If-None-Match") == "*" {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
			f.events[r.URL.Path] = string(body)
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			if _, exists := f.events[r.URL.Path]; !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(f.events, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeCalDAV) lastRequest() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

// newCalDAVGolem loads testdata's CalDAV config and AIML for the fake server
func newCalDAVGolem(t *testing.T, serverURL, password string) *Golem {
	t.Setenv("RADICALE_URL", serverURL)
	t.Setenv("RADICALE_USER", "alice")
	t.Setenv("RADICALE_PASSWORD", password)
	t.Setenv("RADICALE_CALENDAR", "")

	g := New(false)
	g.sraixMgr.now = func() time.Time { return caldavNow }
	kb, err := g.LoadAIML("../../testdata/caldav-examples.aiml")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	properties, err := g.LoadPropertiesFromFile("../../testdata/caldav-config.properties")
	if err != nil {
		t.Fatalf("Failed to load properties: %v", err)
	}
	for key, value := range properties {
		kb.Properties[key] = value
	}
	g.SetKnowledgeBase(kb)
	return g
}

// The CalDAV example AIML and config work against a CalDAV server.
func TestCalDAVExamples(t *testing.T) {
	f, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	session := g.CreateSession("test")
	ask := func(input, want string) {
		t.Helper()
		response, err := g.ProcessInput(input, session)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if response != want {
			t.Errorf("%q: expected %q, got %q", input, want, response)
		}
	}

	ask("what is on my calendar today", "You have no events scheduled for today.")
	if got := f.lastRequest(); got != "REPORT /alice/personal/" {
		t.Errorf("Expected a REPORT on the default calendar, got %s", got)
	}

	ask("create event Lunch with Sam tomorrow at noon", "Event created: Lunch with Sam tomorrow at 12:00 PM")
	ask("create event called Review on Friday at 2pm", "Event created: Review on Fri Nov 20 at 2:00 PM")
	ask("schedule Standup at 3pm for 15 minutes", "Event created: Standup today at 3:00 PM")

	f.mu.Lock()
	var lunch string
	for path, data := range f.events {
		if strings.Contains(data, "SUMMARY:Lunch with Sam") {
			lunch = path
			if !strings.Contains(data, "DTSTART:20261118T120000Z\r\n") || !strings.Contains(data, "DTEND:20261118T130000Z\r\n") {
				t.Errorf("Unexpected event data:\n%s", data)
			}
		}
	}
	if !strings.HasPrefix(lunch, "/alice/personal/") || !strings.HasSuffix(lunch, "@golem.ics") {
		t.Errorf("Expected the event in the personal calendar, got %q", lunch)
	}
	if got := f.headers[len(f.headers)-1].Get("If-None-Match"); got != "*" {
		t.Errorf("Expected If-None-Match: *, got %q", got)
	}
	f.mu.Unlock()

	ask("what is on my calendar today", "Here are your events for today: Standup at 3:00 PM")
	ask("show my upcoming events", "Here are your upcoming events: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM; Review on Fri Nov 20 at 2:00 PM")
	ask("get event lunch", "Event details: Lunch with Sam tomorrow at 12:00 PM until 1:00 PM")
	ask("show event nothing like it", `I couldn't find a single event matching "nothing like it".`)

	// Deleting asks first; the Yes button (or typing yes) confirms
	if response, _ := g.ProcessInput("delete event review", session); RichText(response) != `Should I delete the event "review"?` {
		t.Errorf("Expected a question before deleting, got %q", response)
	}
	if f.lastRequest() == "" || strings.HasPrefix(f.lastRequest(), "DELETE") {
		t.Errorf("Nothing should be deleted before confirming, got %s", f.lastRequest())
	}
	ask("yes", "Event deleted: Review on Fri Nov 20 at 2:00 PM")
	if got := f.lastRequest(); !strings.HasPrefix(got, "DELETE /alice/personal/") {
		t.Errorf("Expected a DELETE, got %s", got)
	}
	ask("confirm delete event review", `I couldn't delete that event. Check that exactly one event matches "review".`)

	ask("create event Lunch with Ana on Friday at 1pm", "Event created: Lunch with Ana on Fri Nov 20 at 1:00 PM")
	ask("confirm delete event lunch", `I couldn't delete that event. Check that exactly one event matches "lunch".`)

	ask("use calendar work", "Switched to calendar: work")
	ask("what is on my calendar", "You have no events in the next seven days.")
	if got := f.lastRequest(); got != "REPORT /alice/work/" {
		t.Errorf("Expected a REPORT on the work calendar, got %s", got)
	}
	ask("use my default calendar", "Switched to your default calendar.")
	ask("what is on my calendar", "Here are your upcoming events: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM; Lunch with Ana on Fri Nov 20 at 1:00 PM")
}

// An event whose title is the input is chosen over one whose title only
// contains it.
func TestCalDAVFindPrefersExactTitle(t *testing.T) {
	f, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	session := g.CreateSession("test")
	for _, input := range []string{"create event Standup at 3pm", "create event Standup sync on Friday at 9am"} {
		g.ProcessInput(input, session)
	}
	if response, _ := g.ProcessInput("confirm delete event standup", session); response != "Event deleted: Standup today at 3:00 PM" {
		t.Errorf("Expected the exact title to be deleted, got %q", response)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) != 1 {
		t.Errorf("Expected one event left, got %d", len(f.events))
	}
}

// A failure to reach the calendar, such as a wrong password, gives the
// service's fallback.
func TestCalDAVWrongPassword(t *testing.T) {
	_, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "wrong")
	response, err := g.ProcessInput("what is on my calendar today", g.CreateSession("test"))
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "I'm having trouble accessing your calendar right now." {
		t.Errorf("Expected the trouble message, got %q", response)
	}
}

// RADICALE_CALENDAR picks the calendar until a session chooses one.
func TestCalDAVDefaultCalendarFromEnvironment(t *testing.T) {
	f, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	t.Setenv("RADICALE_CALENDAR", "family")
	g.ProcessInput("what is on my calendar today", g.CreateSession("test"))
	if got := f.lastRequest(); got != "REPORT /alice/family/" {
		t.Errorf("Expected a REPORT on the family calendar, got %s", got)
	}
}
