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

```
User: Create event Lunch with Sam tomorrow at noon
Bot:  Event created: Lunch with Sam tomorrow at 12:00 PM

User: Create event called Review on Friday at 2pm
Bot:  Event created: Review on Fri Nov 20 at 2:00 PM

User: Schedule Standup at 3pm for 15 minutes
Bot:  Event created: Standup today at 3:00 PM

User: What is on my calendar today?
Bot:  Here are your events for today: Standup at 3:00 PM

User: Show my upcoming events
Bot:  Here are your upcoming events: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM; Review on Fri Nov 20 at 2:00 PM

User: What do I have on Friday?
Bot:  Here's your calendar on Friday, November 20: Review at 2:00 PM

User: What's on my calendar this weekend?
Bot:  You have nothing on your calendar this weekend.

User: Get event lunch
Bot:  Event details: Lunch with Sam tomorrow at 12:00 PM until 1:00 PM

User: Delete event review
Bot:  Should I delete the event "review"?   [Yes, delete it] [No, keep it]
User: yes
Bot:  Event deleted: Review on Fri Nov 20 at 2:00 PM

User: Use calendar work
Bot:  Switched to calendar: work
```

Say "calendar help" for the full list.

### Repeating events

Recorded against a Radicale server with `TZ=Europe/London`, on a Wednesday:

```
User: create event yoga every Monday at 6pm
Bot:  Event created: yoga every Monday at 6:00 PM, starting Mon Nov 30

User: schedule standup every weekday at 9am until December 18
Bot:  Event created: standup every weekday at 9:00 AM, starting tomorrow, until Fri Dec 18

User: add team sync every other Friday at 2pm to my calendar
Bot:  Event created: team sync every other Friday at 2:00 PM, starting Fri Nov 27

User: create event physio every Wednesday at noon for 6 weeks
Bot:  Event created: physio every Wednesday at 12:00 PM, starting Wed Dec 2, until Tue Jan 12 2027

User: create event bins every Thursday
Bot:  Event created: bins every Thursday, starting tomorrow

User: add Mum's birthday every year on December 3 to my calendar
Bot:  Event created: Mum's birthday every year on December 3, starting Thu Dec 3

User: what do I have next week
Bot:  Here's your calendar next week: standup on Mon Nov 30 at 9:00 AM; yoga on Mon Nov 30 at 6:00 PM; standup on Tue Dec 1 at 9:00 AM; standup on Wed Dec 2 at 9:00 AM; physio on Wed Dec 2 at 12:00 PM; Mum's birthday on Thu Dec 3 (all day); bins on Thu Dec 3 (all day); standup on Thu Dec 3 at 9:00 AM; standup on Fri Dec 4 at 9:00 AM

User: delete event yoga
Bot:  Should I delete the event "yoga"?   [Yes, delete it] [No, keep it]
User: yes
Bot:  Event deleted: yoga on Mon Nov 30 at 6:00 PM
```

("Add ... to my calendar" comes from the my-AIML phrase pack; "create event",
"add event" and "schedule" are built in.) Deleting a repeating event deletes
all its occurrences.

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
- An event with neither is an all-day event today.
- A month and day with no year is the next such date.

### Finding events

"Get event", "show event", "delete event", "remove event" and "cancel event" look for the event among those from 30 days ago to a year ahead. They choose, in order:

1. The event whose UID is the text
2. The event whose title is the text, ignoring case
3. The event whose title contains the text

If no event matches, or more than one does at the first step that finds any, the bot says it could not find a single event. Deleting a repeating event deletes all its occurrences.

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
