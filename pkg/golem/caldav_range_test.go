package golem

import (
	"strings"
	"testing"
	"time"
)

func TestParseDateRange(t *testing.T) {
	day := func(month time.Month, d, year int) time.Time { return time.Date(year, month, d, 0, 0, 0, 0, time.UTC) }
	nov := func(d int) time.Time { return day(time.November, d, 2026) }
	// caldavNow is Tuesday, November 17, 2026, 10:00
	sunday := time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC)
	tests := []struct {
		text       string
		now        time.Time
		start, end time.Time
		single     bool
		label      string
	}{
		{"today", caldavNow, nov(17), nov(18), true, "today"},
		{"tomorrow", caldavNow, nov(18), nov(19), true, "tomorrow"},
		{"on Friday", caldavNow, nov(20), nov(21), true, "on Friday, November 20"},
		{"friday?", caldavNow, nov(20), nov(21), true, "on Friday, November 20"},
		{"next tuesday", caldavNow, nov(24), nov(25), true, "on Tuesday, November 24"},
		{"December 3", caldavNow, day(time.December, 3, 2026), day(time.December, 4, 2026), true, "on Thursday, December 3"},
		{"the 5th of January", caldavNow, day(time.January, 5, 2027), day(time.January, 6, 2027), true, "on Tuesday, January 5, 2027"},
		{"2026-11-30", caldavNow, nov(30), day(time.December, 1, 2026), true, "on Monday, November 30"},
		{"this week", caldavNow, nov(17), nov(23), false, "this week"},
		{"for next week", caldavNow, nov(23), nov(30), false, "next week"},
		{"this weekend", caldavNow, nov(21), nov(23), false, "this weekend"},
		{"the weekend", caldavNow, nov(21), nov(23), false, "this weekend"},
		{"next weekend", caldavNow, nov(28), nov(30), false, "next weekend"},
		{"in the next 3 days", caldavNow, nov(17), nov(20), false, "in the next 3 days"},
		{"the coming seven days", caldavNow, nov(17), nov(24), false, "in the next 7 days"},
		{"this month", caldavNow, nov(17), day(time.December, 1, 2026), false, "this month"},
		{"next month", caldavNow, day(time.December, 1, 2026), day(time.January, 1, 2027), false, "next month"},
		// On a Sunday, this week and this weekend are what is left of them
		{"this week", sunday, nov(22), nov(23), false, "this week"},
		{"this weekend", sunday, nov(22), nov(23), false, "this weekend"},
		{"next week", sunday, nov(23), nov(30), false, "next week"},
	}
	for _, tt := range tests {
		start, end, single, label, err := parseDateRange(tt.text, tt.now)
		if err != nil {
			t.Errorf("%q: %v", tt.text, err)
			continue
		}
		if !start.Equal(tt.start) || !end.Equal(tt.end) || single != tt.single || label != tt.label {
			t.Errorf("%q = %s..%s single=%v %q, want %s..%s single=%v %q", tt.text,
				start.Format("Jan 2"), end.Format("Jan 2"), single, label,
				tt.start.Format("Jan 2"), tt.end.Format("Jan 2"), tt.single, tt.label)
		}
	}

	for _, text := range []string{"", "dentist on friday", "whenever", "the next 100 days", "february 30"} {
		if _, _, _, _, err := parseDateRange(text, caldavNow); err == nil {
			t.Errorf("%q should not be read as a date", text)
		}
	}
}

func TestCalDAVCalendarFor(t *testing.T) {
	_, srv := newFakeCalDAV(t)
	g := newCalDAVGolem(t, srv.URL, "secret")
	session := g.CreateSession("test")
	for _, input := range []string{
		"create event Standup at 3pm",
		"create event Lunch with Sam tomorrow at noon",
		"create event Review on Friday at 2pm",
		"create event Planning next Tuesday at 9am",
	} {
		if response, _ := g.ProcessInput(input, session); !strings.HasPrefix(response, "Event created") {
			t.Fatalf("%q: %q", input, response)
		}
	}

	for input, want := range map[string]string{
		"what is on my calendar tomorrow":     "Here's your calendar tomorrow: Lunch with Sam at 12:00 PM",
		"what do I have on Friday":            "Here's your calendar on Friday, November 20: Review at 2:00 PM",
		"what is on my calendar this week":    "Here's your calendar this week: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM; Review on Fri Nov 20 at 2:00 PM",
		"what do I have next week":            "Here's your calendar next week: Planning on Tue Nov 24 at 9:00 AM",
		"what do I have this weekend":         "You have nothing on your calendar this weekend.",
		"show my calendar for December 3":     "You have nothing on your calendar on Thursday, December 3.",
		"what is on my calendar for Thursday": "You have nothing on your calendar on Thursday, November 19.",
		"calendar for the cat":                `I'm not sure which day you mean. Try "tomorrow", "Friday", "December 3", "next week" or "this weekend".`,
		// Unchanged: today, and the coming seven days
		"what is on my calendar today": "Here are your events for today: Standup at 3:00 PM",
		"what is on my calendar":       "Here are your upcoming events: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM; Review on Fri Nov 20 at 2:00 PM; Planning on Tue Nov 24 at 9:00 AM",
	} {
		response, err := g.ProcessInput(input, session)
		if err != nil {
			t.Errorf("%q: %v", input, err)
			continue
		}
		if response = strings.Join(strings.Fields(response), " "); response != want {
			t.Errorf("%q:\n got  %q\n want %q", input, response, want)
		}
	}
}

func TestCalDAVCalendarForUnreachable(t *testing.T) {
	g := newCalDAVGolem(t, "http://127.0.0.1:1", "secret")
	session := g.CreateSession("test")
	if response, _ := g.ProcessInput("what is on my calendar tomorrow", session); strings.TrimSpace(response) != "I'm having trouble accessing your calendar right now." {
		t.Errorf("got %q", response)
	}
}
