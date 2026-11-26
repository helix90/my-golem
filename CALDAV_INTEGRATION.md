# CalDAV/Radicale Integration for Golem

## Overview

Golem bots can read and change a CalDAV calendar, such as one on a [Radicale](https://radicale.org/) server, through `<sraix>`. The bot can:

- List today's events, or the events in the next seven days
- Create an event from text such as "Lunch with Sam tomorrow at noon"
- Show an event's details, found by its title
- Delete an event, found by its title
- Use a different calendar for each session ("use calendar work")

The work is done by the SRAIX `caldav` handler (`pkg/golem/caldav.go`). It builds the CalDAV requests (`REPORT` calendar queries, `PUT` and `DELETE`), writes and reads iCalendar (RFC 5545), and turns events into text for the bot to say. `testdata/caldav-config.properties` configures one SRAIX service per operation, and `testdata/caldav-examples.aiml` has the categories that use them.

## Setup

### 1. Run a CalDAV server

```bash
pip install radicale
radicale --auth-type htpasswd --htpasswd-file /path/to/users
# or: docker run -d -p 5232:5232 tomsquest/docker-radicale
```

Create a calendar, e.g. `personal`, in the web interface at `http://localhost:5232`.

### 2. Set environment variables

```bash
export RADICALE_URL="http://localhost:5232"   # default http://localhost:5232
export RADICALE_USER="alice"                  # the calendar path and the login
export RADICALE_PASSWORD="secret"             # leave unset if the server has no authentication
export RADICALE_CALENDAR="personal"           # until a session picks one; default personal
```

The user name and password are sent with HTTP basic authentication. Use `https://` in `RADICALE_URL` anywhere but on your own machine.

#### From a calendar URL to these settings

The bot builds the calendar's address from three settings:

```
RADICALE_URL / RADICALE_USER / RADICALE_CALENDAR /
```

Calendar apps and servers (FreedomBox, Thunderbird, Radicale's web page) show
the whole address, with the user and the calendar already in it. Don't put
that in `RADICALE_URL`: split it. For example, FreedomBox shows

```
https://freedombox.example.org/radicale/alice/fb88f562-4b7a-bc77-d37b-aeef7e64dee9/
```

which becomes

```bash
RADICALE_URL=https://freedombox.example.org/radicale     # everything before the user
RADICALE_USER=alice                                      # the account that owns the calendar
RADICALE_PASSWORD=...                                    # that account's password
RADICALE_CALENDAR=fb88f562-4b7a-bc77-d37b-aeef7e64dee9   # the calendar's ID, often not its display name
```

`RADICALE_USER` is both the path segment and the login, so it must be the
account that owns the calendar. If `RADICALE_USER` is empty, the bot sends no
request at all.

To check the address and password without the bot (it asks for the password):

```bash
curl -s -o /dev/null -w "%{http_code}\n" -u alice -X PROPFIND -H "Depth: 0" \
  https://freedombox.example.org/radicale/alice/fb88f562-4b7a-bc77-d37b-aeef7e64dee9/
```

`207` means both are right, `401` a wrong password or user, `404` a wrong
path or calendar ID. With `VERBOSE=true` the bot's log gives the reason for a
failed request in a `SRAIX request failed` line.

Times are read and shown in the bot's local time zone. Set `TZ` (e.g. `TZ=Europe/Paris`) if the bot runs somewhere else, such as in a container that defaults to UTC.

### 3. Load the configuration and AIML

Loading the `testdata` directory loads both files:

```go
g := golem.New(false)
if err := g.Execute("load", []string{"testdata"}); err != nil {
	log.Fatal(err)
}
```

To load only the calendar files:

```go
g := golem.New(false)
kb, err := g.LoadAIML("testdata/caldav-examples.aiml")
if err != nil {
	log.Fatal(err)
}
properties, err := g.LoadPropertiesFromFile("testdata/caldav-config.properties")
if err != nil {
	log.Fatal(err)
}
for key, value := range properties {
	kb.Properties[key] = value
}
g.SetKnowledgeBase(kb) // configures the SRAIX services
```

## Usage

Recorded against a Radicale server with `TZ=Europe/London`, on a Thursday
evening. Buttons are in brackets.

```
User: create event Lunch with Sam tomorrow at noon
Bot:  Okay, I've added "Lunch with Sam" to your calendar tomorrow at noon.
      [Undo]
User: create event Review on Monday at 2pm
Bot:  Okay, I've added "Review" to your calendar on Monday at 2 PM.
      [Undo]
User: schedule Standup at 9pm for 15 minutes
Bot:  Okay, I've added "Standup" to your calendar today at 9 PM.
      [Undo]

User: create event dentist
Bot:  When should I put "dentist" on your calendar?
User: Saturday at 10am
Bot:  Okay, I've added "dentist" to your calendar on Saturday at 10 AM.
      [Undo]
User: make it 11am
Bot:  Okay, "dentist" is now on Saturday at 11 AM.

User: what is on my calendar today
Bot:  Today you have Standup at 9 PM.
User: show my upcoming events
Bot:  Here's what's coming up:
      • Today: Standup at 9 PM
      • Tomorrow: Lunch with Sam at noon
      • Saturday: dentist at 11 AM
      • Monday: Review at 2 PM
User: what do I have on Monday
Bot:  On Monday, November 30, you have Review at 2 PM.
User: what's on my calendar next weekend
Bot:  Your calendar is clear next weekend.
User: get event lunch
Bot:  "Lunch with Sam" is tomorrow at noon until 1 PM.

User: move event review to 4pm
Bot:  Okay, "Review" is now on Monday at 4 PM.
User: move event lunch to back half an hour
Bot:  Okay, "Lunch with Sam" is now tomorrow at 12:30 PM.

User: delete event review
Bot:  Should I remove "Review" on Monday at 4 PM from your calendar?
      [Yes, remove it] [No, keep it]
User: yes
Bot:  Okay, I've removed "Review" on Monday at 4 PM from your calendar.

User: create event retro on Tuesday at 3pm
Bot:  Okay, I've added "retro" to your calendar on Tuesday at 3 PM.
      [Undo]
User: undo
Bot:  Okay, I've taken "retro" off your calendar.

User: use calendar work
Bot:  Switched to calendar: work
```

Up to three events read as a sentence; more go one line per day. Say
"calendar help" for the full list. The my-AIML phrase pack adds everyday
phrasings on top of these commands: "I have a dentist appointment on Friday
at 3pm", "move the dentist to 4pm", "push lunch back half an hour", "I can't
make it to dinner", "when am I seeing Sam".

### Repeating events

```
User: create event yoga every Monday at 6pm
Bot:  Okay, I've added "yoga" to your calendar every Monday at 6 PM, starting Monday.
User: schedule standup every weekday at 9am until December 18
Bot:  Okay, I've added "standup" to your calendar every weekday at 9 AM, starting tomorrow, until December 18.
User: create event team sync every other Friday at 2pm
Bot:  Okay, I've added "team sync" to your calendar every other Friday at 2 PM, starting tomorrow.
User: create event physio every Wednesday at noon for 6 weeks
Bot:  Okay, I've added "physio" to your calendar every Wednesday at noon, starting Wednesday, until January 12, 2027.
User: create event bins every Thursday
Bot:  Okay, I've added "bins" to your calendar every Thursday, starting today.
User: create event Mum's birthday every year on December 3
Bot:  Okay, I've added "Mum's birthday" to your calendar every year on December 3, starting Thursday, December 3.

User: what do I have next week
Bot:  Here's your calendar next week:
      • Monday: standup at 9 AM, yoga at 6 PM
      • Tuesday: standup at 9 AM
      • Wednesday: standup at 9 AM, physio at noon
      • Thursday, December 3: bins (all day), Mum's birthday (all day), standup at 9 AM
      • Friday, December 4: standup at 9 AM

User: move event yoga to Tuesday
Bot:  "yoga" repeats, and I can't move a repeating event yet. You can remove it and add it again.
User: delete event yoga
Bot:  "yoga" repeats, so this removes every one. Should I remove "yoga" from your calendar?
      [Yes, remove it] [No, keep it]
User: yes
Bot:  Okay, I've removed "yoga" and all its repeats from your calendar.

User: delete event standup
Bot:  You have 2 events matching "standup":
      • Standup today at 9 PM
      • standup tomorrow at 9 AM
      Which one should I remove?
      [Standup today at 9 PM] [standup tomorrow at 9 AM]
```

### Asking about a day or a period

"What's on my calendar ...", "what do I have on ...", "show my calendar for
..." and `CALENDAR FOR <when>` take:

| Asking about | Examples |
|------|------------------|
| A day | `today`, `tomorrow`, `Friday`, `next Tuesday`, `December 3`, `the 5th of January`, `2026-12-03` (the same date forms as events) |
| A week | `this week` (today to Sunday), `next week` (Monday to Sunday) |
| A weekend | `this weekend`, `the weekend`, `next weekend` |
| A number of days | `the next 3 days`, `the coming seven days` (up to 62) |
| A month | `this month` (today to its end), `next month` |

Repeating events, whether the bot or another app (such as Thunderbird) created
them, are expanded by the bot in their own time zone, including dates left out
and single occurrences that were moved. The calendar server isn't asked to
expand them: Radicale repeats an event at its first occurrence's UTC time, so
its occurrences after a daylight saving time change came back an hour off.

The `caldav_range` service does the work (operation `range`); it sets the
`calendar_range` predicate to a label such as "on Friday, November 20", or to
"unknown" when the text isn't a day or period it knows.

### Writing events

The text after "create event" (or "add event", "schedule") gives the event's title, date, time and length. Whatever is not a date, time or length is the title.

| Part | Understood forms |
|------|------------------|
| Date | `today`, `tonight`, `tomorrow`, `the day after tomorrow`, a weekday (`Friday`, `next Friday`: the next one after today; `this Tuesday` on a Tuesday is today), `Nov 20`, `November 20th, 2027`, `20 November`, `2026-11-20` |
| Time | `2pm`, `2:30 p.m.`, `14:00`, `noon`, `midnight`, or a bare hour after "at" (`at 3`; 1 to 7 are taken as PM) |
| Length | `for 30 minutes`, `for 2 hours`, `for an hour`, `for half an hour`; one hour if not given |
| Repeats | `every Monday`, `every Tuesday and Thursday`, `every Mon, Wed and Fri`, `on Mondays`, `every weekday`, `every weekend`, `every day`, `every week`, `every other Friday`, `every 2 weeks`, `every month`, `every year`, `daily`, `weekly`, `fortnightly`, `monthly`, `yearly` |
| Repeat end | `until December 18`, `for 6 weeks`, `10 times`; none if not given. `starting December 7` gives the first day |

- A repeating event starts on its first matching day: "yoga every Monday at 6pm" on a Wednesday starts next Monday. "On Monday" without "every" (or a plural "Mondays") is a single event.
- Repeating events with a time are written in the bot's time zone (`TZ`, e.g. `TZ=Europe/London`), so they keep their local time when the clocks change. Without `TZ`, they are written in UTC, which keeps the weekday but moves an hour across a change.
- Deleting a repeating event deletes all its occurrences.
- An event with a date but no time lasts all day.
- An event with a time but no date is today, or tomorrow if that time has passed.
- With neither, the bot asks when ("When should I put "dentist" on your calendar?"); the answer ("Saturday at 10am") is added to the text. `calendar_event_when` is then `missing`.
- A leading "a", "an", "my" or "our" is left out of the title: "a dentist appointment on Friday" is "dentist appointment".
- The reply has an Undo button (`CALENDAR UNDO <uid>`); "undo", "scratch that" or "delete it" right after works too, and "make it 4pm" moves it.
- A month and day with no year is the next such date.

### Finding events

"Get event", "show event", "delete event", "move event" (and the phrase pack's
"when is ...", "cancel ...", "move ...") look for the event among those from
30 days ago to a year ahead. They choose, in order:

1. The event whose UID is the text
2. The event whose title is the text, ignoring case
3. The event whose title contains the text

and try the text as typed, then without a leading "my", "the" or "next", then
without a trailing "appointment", "meeting" or "event": "my dentist
appointment" finds "Dentist". When several events match, the reply lists them
with a button for each. Deleting asks first, naming the event it found; a
repeating event is deleted with all its occurrences.

### Moving events

`MOVE EVENT <event> TO <when>` (service `caldav_move`) moves an event to:

| To | Examples | Keeps |
|----|----------|-------|
| A day | `Friday`, `tomorrow`, `December 3` | the time |
| A time | `4pm`, `9:30am`, `noon` | the day |
| Both | `Friday at 4pm` | |
| A shift | `back an hour`, `back half an hour`, `30 minutes later`, `forward 2 hours`, `an hour earlier` | |

A bare weekday is that day of the event's own week (Monday to Sunday): a
Monday event moved "to Wednesday" goes to the Wednesday after it, even weeks
ahead. The event keeps its length, and everything else in it (alarms,
attendees, notes) is kept: only its start, end and sequence number are
rewritten. With no day or time ("move event dentist"), the bot asks when.

#### Repeating events

A repeating event can move just one occurrence or all of them. "Move event
yoga on December 14 to Wednesday" (the event with a day) moves that day's
one; otherwise the bot asks:

```
User: move event yoga to Tuesday
Bot:  "yoga" repeats. Should I move just the next one, on Monday at 6 PM, or every one?
      [Just that one] [Every one]
User: just that one
Bot:  Okay, just that once: "yoga" is now on Tuesday at 6 PM.

User: move event yoga to 7pm
Bot:  "yoga" repeats. Should I move just the next one, on Tuesday at 6 PM, or every one?
      [Just that one] [Every one]
User: every one
Bot:  Okay, every "yoga" has moved: the next one is now on Monday, December 7 at 7 PM.
```

`MOVE ONE EVENT <event> TO <when>` and `MOVE EVERY EVENT <event> TO <when>`
don't ask (they set `calendar_move_scope` to `one` or `all`, the service's
`scope` param).

- **One** writes the occurrence as its own VEVENT in the event (with a
  `RECURRENCE-ID`), as calendar apps do, so Thunderbird and phones show it
  moved too. Moving it again changes that VEVENT.
- **Every one** moves the series by the same days and time, in its own time
  zone, so it keeps its local time across daylight saving changes: every
  Monday at 6 PM moved to Tuesday at 7 PM is every Tuesday at 7 PM. The rule's
  weekdays (`BYDAY`) and days of the month move with it, and so do its
  left-out dates and `UNTIL`. Occurrences moved on their own stay moved; those
  moved only to another day take the new time, and those given a time of
  their own keep it. The change is measured from an occurrence where the
  series puts it.
- Rules that can't move that way to another day, such as "the first Monday
  of the month" (`BYDAY=1MO`) or `BYSETPOS`, can still move one occurrence
  or change the time; the bot offers just the one.

### Calendars

Each session starts on `RADICALE_CALENDAR` (or `personal`). "Use calendar work" sets the session's `calendar` predicate, and every calendar request then uses `.../alice/work/`. "Use my default calendar" goes back.

## Configuration

Each service in `caldav-config.properties` looks like this:

```json
["sraix.caldav_today.handler", "caldav"],
["sraix.caldav_today.operation", "today"],
["sraix.caldav_today.urltemplate", "${RADICALE_URL:-http://localhost:5232}/${RADICALE_USER}/{calendar}/"],
["sraix.caldav_today.param.calendar", "calendar"],
["sraix.caldav_today.default.calendar", "${RADICALE_CALENDAR:-personal}"],
["sraix.caldav_today.username", "${RADICALE_USER}"],
["sraix.caldav_today.password", "${RADICALE_PASSWORD:-}"],
["sraix.caldav_today.timeout", "30"],
["sraix.caldav_today.fallback", "Unable to retrieve calendar events"]
```

| Property | Meaning |
|----------|---------|
| `handler` | `caldav` |
| `operation` | `today`, `upcoming`, `create`, `get` or `delete` |
| `urltemplate` | The calendar collection. `${VAR:-fallback}` uses the fallback when `VAR` is not set. |
| `param.calendar` | The predicate that fills `{calendar}` |
| `default.calendar` | The calendar when the predicate is not set |
| `username`, `password` | Basic authentication |
| `timeout` | Seconds for the whole operation |
| `fallback` | What `<sraix>` returns when the operation fails |

What each operation returns:

| Operation | Service | Input | Returns |
|-----------|---------|-------|---------|
| `today` | `caldav_today` | ignored | `Standup at 9:00 AM; Lunch at 12:00 PM`, or empty when there are no events |
| `upcoming` | `caldav_list` | ignored | Events from now to the end of the seventh day ahead, with their days, or empty |
| `create` | `caldav_create` | Event text | `Lunch with Sam tomorrow at 12:00 PM` |
| `get` | `caldav_get` | A title or UID | `Lunch with Sam tomorrow at 12:00 PM until 1:00 PM, at Room 4. Bring notes` |
| `delete` | `caldav_delete` | A title or UID | The deleted event, as for `create` |

Any failure gives the fallback, or the tag's `default` attribute if it has one. Failures include:
- the server can't be reached;
- a wrong password, or any other error status;
- event text that can't be read, such as `Feb 30`;
- `get` or `delete` finding no event, or several.

## How it works

- **Listing** sends a `REPORT` calendar query (`Depth: 1`) with a time range. The query asks the server to expand repeating events into their occurrences (`<C:expand>`), so they show at the right times. Servers that do not expand show a repeating event at its first occurrence. Radicale supports expansion in recent versions.
- **Creating** sends `PUT <calendar>/<uid>.ics` with `If-None-Match: *`, so an existing event is never replaced. The event has a random UID ending in `@golem`, UTC times (or dates, for all-day events), and its title as `SUMMARY`.
- **Getting and deleting** query the search window, match as described above, and `DELETE` the matched event's resource.
- The iCalendar reader handles folded lines, escaped text, `TZID` times, dates, `DURATION`, and alarms inside events.

## Testing

```bash
go test ./pkg/golem -run 'CalDAV|ParseEventText|ParseICalendar|EventICalendar' -v
```

`TestCalDAVExamples` runs `caldav-examples.aiml` and `caldav-config.properties` against an in-memory CalDAV server.

## Troubleshooting

Run the bot with `golem.New(true)` for verbose logs. Each CalDAV request is logged with its method, path and status, but not its body or credentials.

- **"I'm having trouble accessing your calendar"**: check `RADICALE_URL`, `RADICALE_USER` and `RADICALE_PASSWORD`, e.g. with `curl -u alice:secret http://localhost:5232/alice/personal/`. A `401` status means the credentials are wrong; `404` means the calendar does not exist.
- **Times are off by some hours**: set `TZ` to your time zone.
- **"I couldn't find a single event"**: use more of the title, or say "show my upcoming events" to see the titles.

## References

- [CalDAV (RFC 4791)](https://tools.ietf.org/html/rfc4791)
- [iCalendar (RFC 5545)](https://tools.ietf.org/html/rfc5545)
- [WebDAV (RFC 4918)](https://tools.ietf.org/html/rfc4918)
- [Radicale](https://radicale.org/)
