# Release Notes - My-Golem v1.6.11

**Released:** 2026-11-17
**Previous:** v1.6.10

## Overview

v1.6.11 is a large release. Its changes:

- It brings in upstream Golem v1.6.0, which replaces the pattern matcher with an AIML 2.0 graphmaster and fixes many template, loader and concurrency bugs.
- It removes the old regex-based template processors, leaving the tree processor as the only one.
- It reworks SRAIX. Requests are built correctly, credentials stay out of the logs, and a slow service no longer blocks the bot.
- It makes CalDAV calendars work.
- It makes `<id/>` follow the AIML 2.0 spec.

Several changes alter how existing bots answer or how services are configured. **Read [Upgrading](#upgrading) before you deploy.**

The full list of changes is in `CHANGELOG.md`, under `v1.6.11` and `v1.6.0`.

---

## Highlights

### CalDAV calendars work

The calendar services never worked. They sent malformed requests and could not list, create or delete events. A new built-in SRAIX handler, `caldav`, now does the work. It speaks CalDAV and iCalendar itself, and turns events into sentences the bot can say:

```
User: Create event Lunch with Sam tomorrow at noon
Bot:  Event created: Lunch with Sam tomorrow at 12:00 PM

User: Show my upcoming events
Bot:  Here are your upcoming events: Standup today at 3:00 PM; Lunch with Sam tomorrow at 12:00 PM

User: Delete event lunch
Bot:  Event deleted: Lunch with Sam tomorrow at 12:00 PM

User: Use calendar work
Bot:  Switched to calendar: work
```

- **Creating events.** The bot reads dates ("tomorrow", "Friday", "Nov 20", "2026-11-20"), times ("2pm", "14:00", "noon") and lengths ("for 30 minutes") from the text. The rest of the text is the title.
- **Finding events.** "Get event" and "delete event" find an event by its title, as well as by UID.
- **Repeating events.** These are listed at each occurrence.
- **Calendars.** Each session can pick its own calendar.

See `CALDAV_INTEGRATION.md`, `testdata/caldav-config.properties` and `testdata/caldav-examples.aiml`.

### AIML 2.0 graphmaster pattern matching

Patterns are matched word by word through a graphmaster tree: pattern, then `<that>`, then `<topic>`. Priority follows the spec: `$WORD` > `#` > `_` > `WORD` > `<set>` > `^` > `*`. This fixes cases where a broad wildcard beat a more specific pattern, and makes matching deterministic.

### SRAIX overhaul

New service options:

| Property | What it does |
|----------|--------------|
| `param.<placeholder>` | Names the predicate that fills a placeholder, e.g. `{access_token}` |
| `set.<predicate>` | Sets predicates from other paths of the same JSON reply |
| `default.<placeholder>` | Gives a placeholder a value when nothing else does |
| `username`, `password` | Send HTTP basic authentication |
| `handler`, `operation` | Route the service to a built-in handler (`caldav`) |
| `${VAR:-fallback}` | Gives an unset environment variable a value, in templates, defaults and credentials |

Fixes:

- **Request bodies.** Bodies follow the configured `Content-Type`. GET, HEAD, DELETE and OPTIONS send none.
- **Status codes.** `204 No Content` counts as a success, so successful deletes no longer report failure.
- **JSON numbers.** These keep their digits. `12345678` no longer becomes `1.2345678e+07`.
- **Logging.** Request and response bodies, replies and response headers are no longer logged, because they carry credentials and tokens.
- **Locking.** A SRAIX request no longer holds the knowledge base lock while it waits. Before, a slow service could stall loading and, behind it, every other conversation.

### `<id/>` and `<botid/>`

`<id/>` now returns the client (session) ID, such as `telegram_42`, as AIML 2.0 specifies. The new `<botid/>` tag returns the bot's `id` property, which `<id/>` used to return.

---

## Upgrading

### AIML that may answer differently

These come from the Golem v1.6.0 merge unless noted.

| Change | What to do |
|--------|------------|
| `*` and `_` need at least one word, so `HELLO *` no longer matches `hello` | Add a `HELLO` category, or use `^`/`#` |
| Wildcards match whole words only | Nothing, unless you relied on partial-word matches |
| Input is split into sentences, each answered in turn | Nothing; `hi. how are you?` now gets two answers |
| `<that>` matches the last sentence of the bot's previous reply, anchored | Check `<that>` patterns that matched part of a sentence |
| `<set>` outputs the value it sets | Wrap `<set>` in `<think>` where it should be silent |
| `<learn>` evaluates only `<eval>` (and wildcards) when learning | Use `<eval>` for values that must be fixed at learn time |
| Responses are plain text, not XML-escaped | Escape them yourself before inserting into HTML |
| `(A\|B)` alternation is no longer matched | Write one category per alternative |
| `<id/>` returns the client ID (v1.6.11) | Use `<botid/>` where you wanted the bot's ID |
| `<shuffle>` gives a fresh order each time (v1.6.11) | Nothing |

### SRAIX configuration

- **Services only get the session values they name.** `<sraix>` no longer passes `location`, `latitude`/`longitude`, `list_access_token` or `list_user_id` to every service. Either declare what a service needs:
  ```json
  ["sraix.list_get.param.access_token", "list_access_token"]
  ```
  or pass the values in the input as JSON, e.g. `{"lat":"47.6","lon":"-122.3"}`. The `testdata` list and weather files are already updated.
- **Incomplete requests are not sent.** A placeholder with no value, or an unset `${ENV_VAR}`, now fails the request and falls back. Before, an empty or literal value was sent.
- **Fallback order.** A tag's `default` attribute now wins over the service's `fallback`.
- **`hint` no longer fills `{lat}`/`{lon}`.** Pass coordinates in the input.
- **The `geocode_lon` service is gone.** `geocode` now sets the `geocode_lon` predicate from the same reply, so use `<get name="geocode_lon"/>` after it. Geocoding takes one request instead of two.
- **JSON input is sent as it is.** It used to be wrapped in `{"input": ...}`. Non-JSON content types get the raw input.

### CalDAV

- Set `RADICALE_USER` and `RADICALE_PASSWORD`. `RADICALE_AUTH` is no longer used.
- `RADICALE_URL` defaults to `http://localhost:5232`, and `RADICALE_CALENDAR` to `personal`.
- Replace your copies of `caldav-config.properties` and `caldav-examples.aiml` with the new ones.
- Set `TZ` if the bot runs in a different time zone from its users, for example in a container that defaults to UTC.

### Go API

| Removed | Use instead |
|---------|-------------|
| `CalDAVHelper` | The `caldav` SRAIX handler |
| `SRAIXManager.GetServiceConfig` | `GetConfig` (returns a copy) |
| `EnableTreeProcessing`, `DisableTreeProcessing`, `IsTreeProcessingEnabled` | Nothing; the tree processor is always used |
| `GetConsolidatedProcessor`, `GetProcessorMetrics`, `SetProcessingOrder`, `GetProcessor`, `GetProcessorsByType` | `GetProcessorStats`, `ResetProcessorMetrics`, `GetProcessingOrder` |
| Stats, clear and invalidate methods of the regex, template, pattern and that-pattern caches | Nothing; these caches were never filled |

`AddConfig` stores a copy of the config, and `ListConfigs` returns copies.

### Deploying

```bash
go get github.com/helix90/my-golem@v1.6.11
go build ./...
```

Copy any updated `testdata` AIML and properties files you use to the bot's AIML directory, then restart the bot. This applies particularly to the CalDAV, list-handler, weather and geocode files.

---

## Other fixes

- **Concurrency**
  - Data races between concurrent sessions are fixed. The full suite passes under `-race`; run it with `make test-race`.
  - The `ProcessTemplate*` methods hold the knowledge base lock.
  - `ChatSession.WithLock` gives callers safe access to session fields.
- **Templates and loading**
  - `<random>` runs only the chosen item.
  - `<condition>` supports `value="*"` and `<loop/>`.
  - `<sr/>` no longer overwrites predicates whose names start with `star`.
  - `<split>` works when `limit` is written before `delimiter`.
  - Topic blocks are honoured, and multi-line comments are removed.
  - An invalid category is skipped with a warning, instead of rejecting the whole file.
- **SRAIX**
  - URL path values are escaped as path segments.
  - Form input is decoded once.
  - Reloading properties removes services whose properties are gone.
  - `sraix list` shows template-based URLs and handlers.
  - `WEATHER FORECAST` formats the forecast instead of showing raw JSON.
- **Housekeeping**
  - The LRU caches are O(1).
  - Regexes are compiled once.
  - The unused `pkg/golem/processors` module, the legacy code and the regex matcher are removed.
  - `examples-module` builds and vets.

---

## Testing

```bash
make test                  # full suite
make test-race             # under the race detector
go test ./pkg/golem -run 'CalDAV|ParseEventText|ParseICalendar|EventICalendar' -v
go test ./pkg/golem -run TestSRAIX -v
```

`make test` and `make test-race` pass on v1.6.11. The CalDAV tests run the shipped `caldav-examples.aiml` and `caldav-config.properties` against an in-memory CalDAV server. They have not been run against a live Radicale server.

---

## Known limitations

- **CalDAV**
  - Deleting a repeating event deletes the whole series.
  - Repeating events are listed at each occurrence only if the server supports `<C:expand>`.
  - Times use the bot's local time zone.
  - Event text and replies are in English.
- **Logging.** Verbose logging of other tags, such as `<think>`, still shows the values they process. Don't enable verbose logging in production when templates handle credentials.

---

## Statistics

- 47 commits since v1.6.10
- 143 files changed: 9,164 insertions, 23,891 deletions (mostly removed legacy code)
