package golem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Reminders: the bot keeps them in a ReminderStore and a scheduler hands each
// one, when it is due, to a delivery function the app provides (the Telegram
// bot sends it to the chat it was set in). AIML reaches them through SRAIX
// services with handler "reminders", whose operation is one of:
//
//	add          - "call mom at 5pm", "in 20 minutes to stretch",
//	               "every day at 8am to take vitamins"
//	list         - the user's reminders, numbered
//	delete       - deletes the reminder the input names, by number or text
//	briefing     - sends the user's daily briefing at the time the input
//	               gives, e.g. "7am"; the app delivers it by answering the
//	               input DAILY BRIEFING
//	briefing_off - stops the daily briefing
//
// The services take two params: user (the user's ID, e.g. the telegram_user
// predicate) and chat (where to deliver, e.g. telegram_chat). The handler's
// replies are meant to be shown as they are.

// Reminder is one reminder, or the daily briefing
type Reminder struct {
	ID   string `json:"id"`
	User string `json:"user"`
	Chat string `json:"chat"`
	// Text is what to remind the user of
	Text string `json:"text,omitempty"`
	// Input, when set, is answered by the bot in the user's session at the
	// due time, and the answer delivered instead of Text
	Input   string    `json:"input,omitempty"`
	Due     time.Time `json:"due"`
	Repeat  string    `json:"repeat,omitempty"` // "" or "daily"
	Created time.Time `json:"created"`
}

// BriefingInput is the input a daily briefing reminder answers
const BriefingInput = "DAILY BRIEFING"

// ReminderStore keeps reminders in a JSON file
type ReminderStore struct {
	mu        sync.Mutex
	path      string
	reminders []Reminder
}

// OpenReminderStore opens the reminders kept in the file at path, which is
// created when the first reminder is added
func OpenReminderStore(path string) (*ReminderStore, error) {
	store := &ReminderStore{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read reminders: %v", err)
	}
	if err := json.Unmarshal(data, &store.reminders); err != nil {
		return nil, fmt.Errorf("failed to parse reminders %s: %v", path, err)
	}
	return store, nil
}

// Add saves a reminder, giving it an ID
func (rs *ReminderStore) Add(r Reminder) (Reminder, error) {
	if r.User == "" || r.Chat == "" {
		return Reminder{}, fmt.Errorf("a reminder needs a user and a chat")
	}
	id := make([]byte, 6)
	if _, err := rand.Read(id); err != nil {
		return Reminder{}, err
	}
	r.ID = hex.EncodeToString(id)
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.reminders = append(rs.reminders, r)
	return r, rs.save()
}

// List returns a user's reminders in the order they are due. With
// briefings false, the daily briefing is left out.
func (rs *ReminderStore) List(user string, briefings bool) []Reminder {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	var list []Reminder
	for _, r := range rs.reminders {
		if r.User == user && (briefings || r.Input != BriefingInput) {
			list = append(list, r)
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Due.Before(list[j].Due) })
	return list
}

// Delete removes a reminder. It reports whether it existed.
func (rs *ReminderStore) Delete(id string) (bool, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	for i, r := range rs.reminders {
		if r.ID == id {
			rs.reminders = append(rs.reminders[:i], rs.reminders[i+1:]...)
			return true, rs.save()
		}
	}
	return false, nil
}

// TakeDue returns the reminders due at now and updates the store: a
// one-time reminder is removed and a daily one moves to its next time
func (rs *ReminderStore) TakeDue(now time.Time) ([]Reminder, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	var due []Reminder
	kept := rs.reminders[:0]
	for _, r := range rs.reminders {
		if r.Due.After(now) {
			kept = append(kept, r)
			continue
		}
		due = append(due, r)
		if r.Repeat == "daily" {
			for !r.Due.After(now) {
				r.Due = r.Due.AddDate(0, 0, 1)
			}
			kept = append(kept, r)
		}
	}
	rs.reminders = kept
	if len(due) == 0 {
		return nil, nil
	}
	return due, rs.save()
}

// Run checks for due reminders every interval until ctx ends, passing each
// to deliver
func (rs *ReminderStore) Run(ctx context.Context, interval time.Duration, deliver func(Reminder)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		due, err := rs.TakeDue(time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "reminders: %v\n", err)
		}
		for _, r := range due {
			deliver(r)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// save writes the reminders to the file. The caller must hold rs.mu.
func (rs *ReminderStore) save() error {
	data, err := json.MarshalIndent(rs.reminders, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(rs.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".reminders-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), rs.path)
}

// SetReminderStore lets SRAIX services with handler "reminders" use store
func (g *Golem) SetReminderStore(store *ReminderStore) {
	g.sraixMgr.mu.Lock()
	defer g.sraixMgr.mu.Unlock()
	g.sraixMgr.reminders = store
}

// reminderOperations are the operations of the reminders handler
var reminderOperations = map[string]bool{"add": true, "list": true, "delete": true, "briefing": true, "briefing_off": true}

var (
	reminderInRe    = regexp.MustCompile(`(?i)\bin\s+(an?|one|\d+)\s+(minutes?|mins?|hours?|hrs?|days?)\b`)
	reminderEveryRe = regexp.MustCompile(`(?i)\b(?:every\s*day|daily|each\s+day)\b`)
	reminderLeadRe  = regexp.MustCompile(`(?i)^(?:me\s+)?(?:to|that|about)\s+`)
)

// parseReminderText reads the task and the time of a reminder from text
// such as "call mom at 5pm", "in 20 minutes to stretch" or "every day at
// 8am to take vitamins"
func parseReminderText(text string, now time.Time) (task string, due time.Time, daily bool, err error) {
	text = strings.TrimSpace(text)
	if loc := reminderEveryRe.FindStringIndex(text); loc != nil {
		daily = true
		text = text[:loc[0]] + " " + text[loc[1]:]
	}

	if m := reminderInRe.FindStringSubmatchIndex(text); m != nil && !daily {
		countText := strings.ToLower(text[m[2]:m[3]])
		count := 1
		if n, convErr := strconv.Atoi(countText); convErr == nil {
			count = n
		}
		unit := strings.ToLower(text[m[4]:m[5]])
		switch {
		case strings.HasPrefix(unit, "m"):
			due = now.Add(time.Duration(count) * time.Minute)
		case strings.HasPrefix(unit, "h"):
			due = now.Add(time.Duration(count) * time.Hour)
		default:
			due = now.AddDate(0, 0, count)
		}
		if count <= 0 {
			return "", time.Time{}, false, fmt.Errorf("that time is in the past")
		}
		task = cleanReminderTask(text[:m[0]] + " " + text[m[1]:])
		return task, due.Truncate(time.Second), false, nil
	}

	event, parseErr := parseEventText(text, now)
	if parseErr != nil {
		return "", time.Time{}, false, parseErr
	}
	task = cleanReminderTask(event.summary)
	if event.allDay {
		if cleanEventTitle(text) == event.summary {
			return task, time.Time{}, daily, fmt.Errorf("no time given")
		}
		// A date without a time: the morning of that day
		event.start = time.Date(event.start.Year(), event.start.Month(), event.start.Day(), 9, 0, 0, 0, now.Location())
	}
	due = event.start
	if !due.After(now) {
		if daily {
			due = due.AddDate(0, 0, 1)
		} else {
			return task, time.Time{}, false, fmt.Errorf("that time is in the past")
		}
	}
	return task, due, daily, nil
}

// cleanReminderTask tidies the task of a reminder, e.g. "to call mom" into
// "call mom"
func cleanReminderTask(task string) string {
	task = cleanEventTitle(task)
	if task == "Event" {
		return ""
	}
	return strings.TrimSpace(reminderLeadRe.ReplaceAllString(task, ""))
}

// reminderWhen says when a reminder is due, e.g. "today at 5:00 PM"
func reminderWhen(r Reminder, now time.Time) string {
	when := eventWhen(calendarEvent{start: r.Due}, now, true)
	if r.Repeat == "daily" {
		return "every day at " + r.Due.In(now.Location()).Format("3:04 PM")
	}
	return when
}

// sendReminders performs a reminders service's operation
func (sm *SRAIXManager) sendReminders(config *SRAIXConfig, input string, params map[string]string) (sraixResponse, error) {
	sm.mu.RLock()
	store := sm.reminders
	sm.mu.RUnlock()
	if store == nil {
		return sraixResponse{}, fmt.Errorf("reminders are not set up")
	}
	user, chat := params["user"], params["chat"]
	if user == "" {
		return sraixResponse{}, fmt.Errorf("reminders need the user's ID")
	}
	now := sm.now()
	input = strings.TrimSpace(input)
	reply := func(text string) (sraixResponse, error) { return sraixResponse{text: text}, nil }

	switch config.Operation {
	case "add":
		if chat == "" {
			return sraixResponse{}, fmt.Errorf("reminders need the chat to deliver to")
		}
		task, due, daily, err := parseReminderText(input, now)
		switch {
		case err != nil && err.Error() == "no time given":
			return reply("When should I remind you? Try something like \"remind me to call mom at 5pm\" or \"remind me in 20 minutes to stretch\".")
		case err != nil && err.Error() == "that time is in the past":
			return reply("That time has already passed. When should I remind you?")
		case err != nil:
			return reply("I couldn't work out when you meant: " + err.Error() + ".")
		case task == "":
			return reply("What should I remind you about?")
		}
		r := Reminder{User: user, Chat: chat, Text: task, Due: due, Created: now}
		if daily {
			r.Repeat = "daily"
		}
		if r, err = store.Add(r); err != nil {
			return sraixResponse{}, err
		}
		return reply(fmt.Sprintf("Okay, I'll remind you to %s %s.", task, reminderWhen(r, now)))

	case "list":
		list := store.List(user, false)
		if len(list) == 0 {
			return reply("You don't have any reminders.")
		}
		lines := make([]string, len(list))
		for i, r := range list {
			lines[i] = fmt.Sprintf("%d. %s, %s", i+1, r.Text, reminderWhen(r, now))
		}
		return reply("Your reminders:\n" + strings.Join(lines, "\n"))

	case "delete":
		list := store.List(user, false)
		var match []Reminder
		if n, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(input), "number ")); err == nil {
			if n >= 1 && n <= len(list) {
				match = []Reminder{list[n-1]}
			}
		} else {
			want := strings.ToLower(cleanReminderTask(input))
			for _, r := range list {
				if strings.Contains(strings.ToLower(r.Text), want) {
					match = append(match, r)
				}
			}
		}
		switch len(match) {
		case 0:
			return reply("I couldn't find that reminder. Say \"show my reminders\" to see them.")
		case 1:
			if _, err := store.Delete(match[0].ID); err != nil {
				return sraixResponse{}, err
			}
			return reply(fmt.Sprintf("Deleted the reminder to %s.", match[0].Text))
		}
		return reply(fmt.Sprintf("%d reminders match that. Which number? Say \"show my reminders\" to see them.", len(match)))

	case "briefing":
		if chat == "" {
			return sraixResponse{}, fmt.Errorf("reminders need the chat to deliver to")
		}
		_, due, _, err := parseReminderText("briefing every day "+input, now)
		if err != nil {
			return reply("What time should I send your daily briefing? For example, \"send me a daily briefing at 7am\".")
		}
		for _, r := range store.List(user, true) {
			if r.Input == BriefingInput {
				if _, err := store.Delete(r.ID); err != nil {
					return sraixResponse{}, err
				}
			}
		}
		r := Reminder{User: user, Chat: chat, Input: BriefingInput, Text: "daily briefing", Due: due, Repeat: "daily", Created: now}
		if _, err := store.Add(r); err != nil {
			return sraixResponse{}, err
		}
		return reply(fmt.Sprintf("Okay, I'll send you a briefing every day at %s, starting %s.", due.Format("3:04 PM"), eventWhen(calendarEvent{start: due}, now, true)))

	case "briefing_off":
		stopped := false
		for _, r := range store.List(user, true) {
			if r.Input == BriefingInput {
				if _, err := store.Delete(r.ID); err != nil {
					return sraixResponse{}, err
				}
				stopped = true
			}
		}
		if !stopped {
			return reply("You don't have a daily briefing set up.")
		}
		return reply("Okay, I've stopped your daily briefing.")
	}
	return sraixResponse{}, fmt.Errorf("unknown reminders operation %q", config.Operation)
}
