package golem

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// reminderTestNow is a Tuesday afternoon
var reminderTestNow = time.Date(2026, 11, 24, 14, 30, 0, 0, time.Local)

func TestParseReminderText(t *testing.T) {
	at := func(day, hour, minute int) time.Time {
		return time.Date(2026, 11, day, hour, minute, 0, 0, time.Local)
	}
	tests := []struct {
		text  string
		task  string
		due   time.Time
		daily bool
		err   string
	}{
		{text: "to call mom at 5pm", task: "call mom", due: at(24, 17, 0)},
		{text: "call mom at 5pm", task: "call mom", due: at(24, 17, 0)},
		{text: "at 5pm to call Mom", task: "call Mom", due: at(24, 17, 0)},
		{text: "in 20 minutes to stretch", task: "stretch", due: reminderTestNow.Add(20 * time.Minute)},
		{text: "to stretch in an hour", task: "stretch", due: reminderTestNow.Add(time.Hour)},
		{text: "in 2 days to pay rent", task: "pay rent", due: reminderTestNow.AddDate(0, 0, 2)},
		{text: "tomorrow at 9am to take the bins out", task: "take the bins out", due: at(25, 9, 0)},
		{text: "on friday to buy flowers", task: "buy flowers", due: at(27, 9, 0)},
		{text: "at 10am to water plants", task: "water plants", due: at(25, 10, 0)}, // 10am today has passed
		{text: "every day at 8am to take vitamins", task: "take vitamins", due: at(25, 8, 0), daily: true},
		{text: "to call mom", task: "call mom", err: "no time given"},
		{text: "today at 9am to call mom", task: "call mom", err: "that time is in the past"},
	}
	for _, tt := range tests {
		task, due, daily, err := parseReminderText(tt.text, reminderTestNow)
		if tt.err != "" {
			if err == nil || err.Error() != tt.err {
				t.Errorf("%q: err = %v, want %q", tt.text, err, tt.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tt.text, err)
			continue
		}
		if task != tt.task || !due.Equal(tt.due) || daily != tt.daily {
			t.Errorf("%q = (%q, %v, %v), want (%q, %v, %v)", tt.text, task, due, daily, tt.task, tt.due, tt.daily)
		}
	}
}

func newReminderTestBot(t *testing.T, path string) (*Golem, *ReminderStore) {
	t.Helper()
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	store, err := OpenReminderStore(path)
	if err != nil {
		t.Fatal(err)
	}
	g.SetReminderStore(store)
	g.sraixMgr.now = func() time.Time { return reminderTestNow }
	return g, store
}

func reminderSession(g *Golem, user, chat string) *ChatSession {
	s := telegramSession(g, user)
	s.Variables["telegram_chat"] = chat
	return s
}

func TestRemindersThroughAIML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reminders.json")
	g, store := newReminderTestBot(t, path)
	s := reminderSession(g, "111", "5001")

	steps := []struct{ input, want string }{
		{"what are my reminders", "don't have any reminders"},
		{"remind me to call mom at 5pm", "remind you to call mom today at 5:00 PM"},
		{"Remind me in 20 minutes to stretch", "remind you to stretch today at 2:50 PM"},
		{"set a timer for 10 minutes", "remind you to check the timer today at 2:40 PM"},
		{"remind me every day at 8am to take vitamins", "take vitamins every day at 8:00 AM"},
		{"remind me to call mom", "When should I remind you?"},
		{"show my reminders", "1. check the timer, today at 2:40 PM\n2. stretch, today at 2:50 PM\n3. call mom, today at 5:00 PM\n4. take vitamins, every day at 8:00 AM"},
		{"cancel the timer", "Deleted the reminder to check the timer"},
		{"delete reminder 1", "Deleted the reminder to stretch"},
		{"delete reminder mom", "Deleted the reminder to call mom"},
		{"delete reminder 7", "couldn't find that reminder"},
		{"send me a daily briefing at 7am", "briefing every day at 7:00 AM, starting tomorrow"},
	}
	for _, step := range steps {
		if got := ask(t, g, s, step.input); !strings.Contains(got, step.want) {
			t.Errorf("%q:\n got  %q\n want %q", step.input, got, step.want)
		}
	}

	// Another user sees none of them
	if got := ask(t, g, reminderSession(g, "222", "5002"), "show my reminders"); !strings.Contains(got, "don't have any") {
		t.Errorf("another user: %q", got)
	}

	// Due tomorrow at 8: the vitamins (moved to the next day) and not the
	// briefing, which was at 7 and has moved on too
	due, err := store.TakeDue(time.Date(2026, 11, 25, 8, 0, 30, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 2 || due[0].Chat != "5001" {
		t.Fatalf("due = %+v", due)
	}
	left := store.List("111", true)
	if len(left) != 2 {
		t.Fatalf("daily reminders should remain, got %+v", left)
	}
	for _, r := range left {
		if !r.Due.After(time.Date(2026, 11, 25, 8, 0, 30, 0, time.Local)) || r.Due.Day() != 26 {
			t.Errorf("%s not moved to the next day: %v", r.Text, r.Due)
		}
	}

	// Reminders survive a restart
	reopened, err := OpenReminderStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(reopened.List("111", true)); got != 2 {
		t.Errorf("after reopening: %d reminders, want 2", got)
	}

	if got := ask(t, g, s, "stop my daily briefing"); !strings.Contains(got, "stopped your daily briefing") {
		t.Errorf("stop: %q", got)
	}
	if got := ask(t, g, s, "stop my daily briefing"); !strings.Contains(got, "don't have a daily briefing") {
		t.Errorf("stop again: %q", got)
	}
}

func TestRemindersNeedTelegram(t *testing.T) {
	g, _ := newReminderTestBot(t, filepath.Join(t.TempDir(), "reminders.json"))
	if got := ask(t, g, g.CreateSession("cli"), "remind me to call mom at 5pm"); !strings.Contains(got, "only work in the Telegram chat") {
		t.Errorf("got %q", got)
	}
}

func TestDailyBriefing(t *testing.T) {
	g, _ := newReminderTestBot(t, filepath.Join(t.TempDir(), "reminders.json"))
	s := reminderSession(g, "111", "5001")
	ask(t, g, s, "my name is Alice")

	// The greeting follows the time of day (timeofday.map)
	for hour, greeting := range map[int]string{
		8: "Good morning, Alice!", 14: "Good afternoon, Alice!", 19: "Good evening, Alice!",
		23: "Hello, Alice!", 4: "Hello, Alice!", 5: "Good morning, Alice!",
	} {
		at := time.Date(2026, 11, 24, hour, 30, 0, 0, time.Local)
		if hour == 4 {
			at = at.Add(29 * time.Minute) // 4:59, the last minute of night
		}
		g.clock = func() time.Time { return at }
		got := ask(t, g, s, "daily briefing")
		if !strings.HasPrefix(got, greeting+" Today is Tuesday, November 24.") {
			t.Errorf("at %s: %q, want it to start %q", at.Format("15:04"), got, greeting)
		}
		// No calendar or location is set up, so neither is mentioned
		for _, unwanted := range []string{"calendar right now", "trouble", "location", "rephrase"} {
			if strings.Contains(got, unwanted) {
				t.Errorf("briefing mentions %q: %q", unwanted, got)
			}
		}
	}

	// Without a name
	g.clock = func() time.Time { return time.Date(2026, 11, 24, 15, 0, 0, 0, time.Local) }
	if got := ask(t, g, reminderSession(g, "222", "5002"), "brief me"); !strings.HasPrefix(got, "Good afternoon! Today is") {
		t.Errorf("no name: %q", got)
	}
}
