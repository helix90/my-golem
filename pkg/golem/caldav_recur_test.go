package golem

import (
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseRecurringEvents(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("no time zone data")
	}
	// Tuesday, November 17, 2026, 10:00 in London
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, london)
	at := func(month time.Month, day, hour, minute int) time.Time {
		return time.Date(2026, month, day, hour, minute, 0, 0, london)
	}
	tests := []struct {
		text    string
		title   string
		start   time.Time
		allDay  bool
		rrule   string
		describ string
	}{
		{"yoga every Monday at 6pm", "yoga", at(11, 23, 18, 0), false, "FREQ=WEEKLY;BYDAY=MO",
			"every Monday at 6:00 PM, starting Mon Nov 23"},
		{"standup every weekday at 9:30am", "standup", at(11, 18, 9, 30), false, "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			"every weekday at 9:30 AM, starting tomorrow"},
		// 9am today has passed, so tomorrow is the first weekday
		{"standup every weekday at 9am", "standup", at(11, 18, 9, 0), false, "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			"every weekday at 9:00 AM, starting tomorrow"},
		{"gym every Tuesday and Thursday at 7pm", "gym", at(11, 17, 19, 0), false, "FREQ=WEEKLY;BYDAY=TU,TH",
			"every Tuesday and Thursday at 7:00 PM, starting today"},
		{"piano lesson on Mondays and Wednesdays at 4pm", "piano lesson", at(11, 18, 16, 0), false, "FREQ=WEEKLY;BYDAY=MO,WE",
			"every Monday and Wednesday at 4:00 PM, starting tomorrow"},
		{"run every Mon, Wed and Fri at 7am", "run", at(11, 18, 7, 0), false, "FREQ=WEEKLY;BYDAY=MO,WE,FR",
			"every Monday, Wednesday and Friday at 7:00 AM, starting tomorrow"},
		{"bins out every Monday", "bins out", at(11, 23, 0, 0), true, "FREQ=WEEKLY;BYDAY=MO",
			"every Monday, starting Mon Nov 23"},
		{"team sync every other Friday at 2pm", "team sync", at(11, 20, 14, 0), false, "FREQ=WEEKLY;INTERVAL=2;BYDAY=FR",
			"every other Friday at 2:00 PM, starting Fri Nov 20"},
		{"1:1 every 2 weeks on Thursday at 3pm", "1:1", at(11, 19, 15, 0), false, "FREQ=WEEKLY;INTERVAL=2",
			"every other Thursday at 3:00 PM, starting Thu Nov 19"},
		{"vitamins every day at 8am", "vitamins", at(11, 18, 8, 0), false, "FREQ=DAILY",
			"every day at 8:00 AM, starting tomorrow"},
		{"book club monthly on December 3 at 7pm", "book club", at(12, 3, 19, 0), false, "FREQ=MONTHLY",
			"every month on the 3rd at 7:00 PM, starting Thu Dec 3"},
		{"Mum's birthday every year on December 3", "Mum's birthday", at(12, 3, 0, 0), true, "FREQ=YEARLY",
			"every year on December 3, starting Thu Dec 3"},
		{"swimming every Saturday at 10am until December 19", "swimming", at(11, 21, 10, 0), false,
			"FREQ=WEEKLY;BYDAY=SA;UNTIL=20261219T235959Z", "every Saturday at 10:00 AM, starting Sat Nov 21, until Sat Dec 19"},
		{"physio every Wednesday at noon for 6 weeks", "physio", at(11, 18, 12, 0), false,
			"FREQ=WEEKLY;BYDAY=WE;UNTIL=20261229T235959Z", "every Wednesday at 12:00 PM, starting tomorrow, until Tue Dec 29"},
		{"course every Monday at 6pm 10 times", "course", at(11, 23, 18, 0), false,
			"FREQ=WEEKLY;BYDAY=MO;COUNT=10", "every Monday at 6:00 PM, starting Mon Nov 23, 10 times"},
		{"yoga every Monday at 6pm starting December 7", "yoga", at(12, 7, 18, 0), false, "FREQ=WEEKLY;BYDAY=MO",
			"every Monday at 6:00 PM, starting Mon Dec 7"},
		{"meeting weekly on Friday at 9am for an hour", "meeting", at(11, 20, 9, 0), false, "FREQ=WEEKLY",
			"every Friday at 9:00 AM, starting Fri Nov 20"},
	}
	for _, tt := range tests {
		event, err := parseEventText(tt.text, now)
		if err != nil {
			t.Errorf("%q: %v", tt.text, err)
			continue
		}
		if event.recurrence == nil {
			t.Errorf("%q: not read as repeating", tt.text)
			continue
		}
		rrule := event.recurrence.rrule(event, false)
		if event.summary != tt.title || !event.start.Equal(tt.start) || event.allDay != tt.allDay || rrule != tt.rrule {
			t.Errorf("%q = %q %s allDay=%v %s\n want %q %s allDay=%v %s", tt.text,
				event.summary, event.start.Format("Mon Jan 2 15:04"), event.allDay, rrule,
				tt.title, tt.start.Format("Mon Jan 2 15:04"), tt.allDay, tt.rrule)
		}
		if got := event.recurrence.describe(event, now); got != tt.describ {
			t.Errorf("%q describes as %q, want %q", tt.text, got, tt.describ)
		}
	}

	// One-off events stay one-off
	for _, text := range []string{"dentist on Monday at 2pm", "lunch tomorrow at noon", "review on Friday", "Monday planning at 10am"} {
		event, err := parseEventText(text, now)
		if err != nil || event.recurrence != nil {
			t.Errorf("%q should not repeat (err %v)", text, err)
		}
	}
	for _, text := range []string{"class every Monday until yesterday", "class every Monday until whenever"} {
		if _, err := parseEventText(text, now); err == nil {
			t.Errorf("%q should be refused", text)
		}
	}
}

// In UTC, an evening event in a zone behind UTC falls on the next UTC day,
// so BYDAY names that day
func TestRecurrenceInUTCShiftsWeekdays(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone data")
	}
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, newYork)
	event, err := parseEventText("league every Monday and Friday at 9pm", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := event.recurrence.rrule(event, true); got != "FREQ=WEEKLY;BYDAY=TU,SA" {
		t.Errorf("UTC rrule = %s, want BYDAY=TU,SA (9pm in New York is 02:00 UTC the next day)", got)
	}
	if got := event.recurrence.rrule(event, false); got != "FREQ=WEEKLY;BYDAY=MO,FR" {
		t.Errorf("local rrule = %s", got)
	}
}

func TestEventICalendarRecurring(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("no time zone data")
	}
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, london)
	event, err := parseEventText("yoga every Monday at 6pm", now)
	if err != nil {
		t.Fatal(err)
	}
	event.uid = "x@golem"
	ics := eventICalendar(event, now)
	for _, want := range []string{
		"BEGIN:VTIMEZONE\r\nTZID:Europe/London\r\n",
		"BEGIN:DAYLIGHT\r\nDTSTART:20260329T010000\r\nRRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU\r\nTZOFFSETFROM:+0000\r\nTZOFFSETTO:+0100\r\nTZNAME:BST\r\nEND:DAYLIGHT",
		"BEGIN:STANDARD\r\nDTSTART:20261025T020000\r\nRRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU\r\nTZOFFSETFROM:+0100\r\nTZOFFSETTO:+0000\r\nTZNAME:GMT\r\nEND:STANDARD",
		"DTSTART;TZID=Europe/London:20261123T180000\r\n",
		"DTEND;TZID=Europe/London:20261123T190000\r\n",
		"RRULE:FREQ=WEEKLY;BYDAY=MO\r\n",
	} {
		if !strings.Contains(ics, want) {
			t.Errorf("missing %q in\n%s", want, ics)
		}
	}
	// It reads back at the same local time
	read := parseICalendar(ics, time.UTC)
	if len(read) != 1 || !read[0].start.Equal(event.start) || !read[0].repeats {
		t.Errorf("read back %+v", read)
	}

	// One-off events are unchanged: UTC, no time zone
	once, _ := parseEventText("dentist on Friday at 2pm", now)
	once.uid = "y@golem"
	if ics := eventICalendar(once, now); strings.Contains(ics, "VTIMEZONE") || strings.Contains(ics, "RRULE") || !strings.Contains(ics, "DTSTART:20261120T140000Z") {
		t.Errorf("one-off event:\n%s", ics)
	}
}

func TestVTimezone(t *testing.T) {
	tests := []struct {
		zone  string
		wants []string
	}{
		{"America/New_York", []string{"RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=2SU", "RRULE:FREQ=YEARLY;BYMONTH=11;BYDAY=1SU", "TZOFFSETTO:-0400", "TZOFFSETTO:-0500"}},
		// Southern hemisphere: summer time starts in October
		{"Australia/Sydney", []string{"BYMONTH=10;BYDAY=1SU", "BYMONTH=4;BYDAY=1SU", "TZOFFSETTO:+1100", "TZOFFSETTO:+1000"}},
		// The last Sunday of March is the 4th Sunday from 2027 to 2029:
		// still "last", not "4th"
		{"Europe/London@2027", []string{"BYMONTH=3;BYDAY=-1SU", "BYMONTH=10;BYDAY=-1SU"}},
		// No daylight saving time: one offset
		{"Asia/Tokyo", []string{"BEGIN:STANDARD\nDTSTART:19700101T000000\nTZOFFSETFROM:+0900\nTZOFFSETTO:+0900"}},
	}
	for _, tt := range tests {
		zone, year := tt.zone, 2026
		if i := strings.Index(zone, "@"); i >= 0 {
			year, _ = strconv.Atoi(zone[i+1:])
			zone = zone[:i]
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Skip("no time zone data")
		}
		text := strings.Join(vtimezone(zone, loc, time.Date(year, 3, 1, 0, 0, 0, 0, loc)), "\n")
		for _, want := range tt.wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s: missing %q in\n%s", tt.zone, want, text)
			}
		}
	}
}

func TestEventTimeZoneName(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	if got := eventTimeZoneName(london); got != "Europe/London" {
		t.Errorf("got %q", got)
	}
	if got := eventTimeZoneName(time.UTC); got != "" {
		t.Errorf("UTC: got %q", got)
	}
	t.Setenv("TZ", "Europe/Paris")
	if got := eventTimeZoneName(time.Local); got != "Europe/Paris" {
		t.Errorf("time.Local with TZ=Europe/Paris: got %q", got)
	}
	t.Setenv("TZ", "")
	if got := eventTimeZoneName(time.Local); got != "" {
		t.Errorf("time.Local without TZ: got %q", got)
	}
}

// Events as Thunderbird writes them, expanded here in their own zone
func TestExpandEvents(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("no time zone data")
	}
	ics := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\nUID:league\r\nDTSTART;TZID=Europe/London:20270301T200000\r\nDTEND;TZID=Europe/London:20270301T210000\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=MO\r\nEXDATE;TZID=Europe/London:20270322T200000\r\nSUMMARY:league\r\nEND:VEVENT\r\n" +
		// The April 12 game moved to 6pm
		"BEGIN:VEVENT\r\nUID:league\r\nRECURRENCE-ID;TZID=Europe/London:20270412T200000\r\nDTSTART;TZID=Europe/London:20270412T180000\r\n" +
		"DTEND;TZID=Europe/London:20270412T190000\r\nSUMMARY:league (early)\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:book\r\nDTSTART;TZID=Europe/London:20270112T190000\r\nDTEND;TZID=Europe/London:20270112T200000\r\n" +
		"RRULE:FREQ=MONTHLY;BYDAY=2TU\r\nSUMMARY:book club\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:camp\r\nDTSTART;VALUE=DATE:20270329\r\nDTEND;VALUE=DATE:20270330\r\nRRULE:FREQ=DAILY;UNTIL=20270402\r\n" +
		"SUMMARY:camp\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	events := parseICalendar(ics, london)
	for i := range events {
		events[i].href = "/cal/" + events[i].uid + ".ics"
	}

	got := func(from, to time.Time) string {
		var out []string
		for _, e := range expandEvents(events, from, to) {
			when := e.start.Format("Mon Jan 2 15:04")
			if e.allDay {
				when = e.start.Format("Mon Jan 2") + " all day"
			}
			out = append(out, e.summary+" "+when)
		}
		sort.Strings(out)
		return strings.Join(out, "; ")
	}
	day := func(m time.Month, d int) time.Time { return time.Date(2027, m, d, 0, 0, 0, 0, london) }

	for _, tt := range []struct {
		from, to time.Time
		want     string
	}{
		// Before the clock change (GMT)
		{day(3, 1), day(3, 2), "league Mon Mar 1 20:00"},
		// Left out by EXDATE
		{day(3, 22), day(3, 23), ""},
		// After the change (BST): still 8pm local
		{day(4, 5), day(4, 6), "league Mon Apr 5 20:00"},
		// The changed occurrence replaces the 8pm one
		{day(4, 12), day(4, 13), "league (early) Mon Apr 12 18:00"},
		// The second Tuesday of each month
		{day(4, 1), day(5, 1), "book club Tue Apr 13 19:00; camp Fri Apr 2 all day; camp Thu Apr 1 all day; league (early) Mon Apr 12 18:00; league Mon Apr 19 20:00; league Mon Apr 26 20:00; league Mon Apr 5 20:00"},
		// A daily all-day series ending on April 2, across the change
		{day(3, 28), day(4, 4), "camp Fri Apr 2 all day; camp Mon Mar 29 all day; camp Thu Apr 1 all day; camp Tue Mar 30 all day; camp Wed Mar 31 all day; league Mon Mar 29 20:00"},
	} {
		if g := got(tt.from, tt.to); g != tt.want {
			t.Errorf("%s..%s:\n got  %s\n want %s", tt.from.Format("Jan 2"), tt.to.Format("Jan 2"), g, tt.want)
		}
	}
}
