# Changelog

All notable changes to Golem are recorded here.

## Unreleased

### Security

- **Saved credentials, names and locations could be read by any user.** The persistence AIML saved them with `<learnf>` as categories such as `GET LIST PASSWORD FOR <telegram id>`, which anyone could match to get another user's password. They are now kept in a per-user store (`UserStore`) that input cannot reach, and the bot moves existing values there at startup, deleting the old categories from the knowledge base, `learned_categories/` and its backups.
- Saved passwords and other secrets are encrypted in the store when `USER_STORE_KEY` is set, and hidden in `/session`. A message that carries a password is deleted from the Telegram chat and removed from the session's history.

### Upgrading: behaviour changes

- **The Telegram bot only answers the users in `TELEGRAM_ALLOWED_USERS`** (comma-separated Telegram user IDs). With it unset, it answers no one, and tells each person who writes their user ID so they can be added. Set `TELEGRAM_ALLOW_ALL=true` to keep it open. `/reload` is for `TELEGRAM_ADMINS` (default: every allowed user).
- **The location predicates are renamed** `user_location`, `user_latitude` and `user_longitude` (were `location`, `latitude`, `longitude`), and the list credentials `user_list_username` and `user_list_password` (were `list_username`, `list_password`). Every `user_*` predicate is now saved per user when the bot has a user store and the session's `UserID` is set.
- **"Delete event" asks first.** It replies with Yes/No buttons; `CONFIRM DELETE EVENT <title>` (or answering "yes") deletes.
- The Telegram bot keeps its data in `/app/data` (Docker volume `golem-data`); set `USER_STORE_KEY`.

### Added

- **Moving repeating events.** Moving a repeating event asks "just the next one, or every one?" (`MOVE ONE EVENT` and `MOVE EVERY EVENT` don't ask; "yoga on December 14" names one). One occurrence is written as its own VEVENT with a RECURRENCE-ID; every one moves the series in its own time zone, with its weekdays, UNTIL, left-out dates and moved occurrences. A move to a bare weekday is that day of the event's own week. Event names are also tried in the singular ("standups").
- **Conversational calendar replies.** "Okay, I've added "dentist" to your calendar on Friday at 3 PM.", "Today you have standup at 9 AM and lunch at noon.", "On Monday, November 30, you have review at 2 PM.", and more than three events one line per day. Times read "9 AM", "9:30 AM", "noon"; days "tomorrow", "Friday" in the coming week, else "Friday, December 4". Reminders use the same times. The today, upcoming and range services set `calendar_layout` (inline or lines) and `calendar_count`.
- **Asking when.** "create event dentist" with no day or time asks "When should I put "dentist" on your calendar?" instead of adding an all-day event today; the create service sets `calendar_event_when` to `missing`. Adding has an Undo button, and "undo" or "make it 4pm" right after work.
- **Moving events**: `MOVE EVENT <event> TO <when>` (service `caldav_move`, operation `move`) moves an event to a new day, time or both, or by "back an hour"; the rest of the event is kept.
- **Finding events** tries the name without "my", "the", "next" and a trailing "appointment"/"meeting": "my dentist appointment" finds "Dentist". Deleting names the event it found before asking, and says when it can't find one. When several events match, the reply lists them with a button for each. caldav_get answers "Event not found" itself and falls back to "Unable to retrieve calendar events" when the calendar can't be reached.
- **Plain-English list replies.** Lists read "Your groceries list has 2 items:" with one ☐/☑ line per item, and "You have 2 lists: groceries and hardware." with a button for each list. Item and list IDs are no longer shown. "Added "milk" to your groceries list.", "Created a list called "groceries".", "Deleted your groceries list." `<jsonformat type="lists">` takes `button="LIST SHOW {list}"`, and `only="buttons"` for buttons alone.
- **Missing lists.** Adding to or showing a list that doesn't exist asks "Should I create it?" (and add the item), with Yes/No buttons; checking off on one says so, with a button for each list and one to create it. New helpers for phrase packs: `LIST LOOKUP <name>` (service `list_lookup`), `LIST NOT FOUND`, `LIST CHOICE BUTTONS`, `LIST ADD PENDING TO <list>`. Showing or changing a list sets `list_current`, the list in use. Lists are found by name loosely with list-handler 0.6 ("Grocery" finds "groceries").
- **User store**: `OpenUserStore`, `Golem.SetUserStore`, `ChatSession.UserID`, `Golem.MigrateLegacyUserData`; `/forgetme` in the Telegram bot.
- **Reminders, timers and a daily briefing**: the `reminders` SRAIX handler, `ReminderStore`, `testdata/reminders.aiml`. The Telegram bot delivers them when due.
- **Buttons**: AIML 2.1 rich media (`<button>`, `<reply>`, `<location/>`) become Telegram inline buttons and suggested replies; `ParseRichResponse`. `<jsonformat button="...">` adds a tick button per list item. Weather offers to share a location, which the bot saves.
- The Telegram bot stops cleanly on SIGTERM.
- The daily briefing greets by the time of day (good morning, afternoon or evening, or hello at night) instead of always "good morning", using `timeofday.map`.
- Repeating calendar events: "create event yoga every Monday at 6pm", "every weekday", "every other Friday", "every Tuesday and Thursday until December 18", "monthly", "for 6 weeks", "10 times", "starting December 7". They are written with an RRULE, in the bot's time zone (`TZ`) with a VTIMEZONE, so they keep their local time across daylight saving time changes.
- List items by name: `LIST <list> CHECK OFF <item>`, `UNCHECK` and `REMOVE` (and "check off milk on list groceries", "remove milk from list groceries") find the item with the list handler's new `items/search` (list-handler 0.5) and say which item they changed. "Check off" always marks done, so repeating it doesn't undo it.
- Calendar questions about a day or a period: "what do I have on Friday", "what's on my calendar tomorrow / next week / this weekend / the next 3 days / next month", `CALENDAR FOR <when>`. The `caldav` handler's new `range` operation reads the date.
- SRAIX `body`: a request body template whose values are escaped for its Content-Type (URL-encoded for forms, escaped for JSON).
- Log redaction: every golem log line masks the values of secret predicates and the passwords in credential commands; `AddLogRedaction` adds patterns.
- SRAIX `reauth`: an input to answer when a service replies 401 Unauthorized, after which the request is retried once if the service's params changed. The list services use it to log in again when their token expires.
- An allowlist for the Telegram bot: `TELEGRAM_ALLOWED_USERS`, `TELEGRAM_ADMINS`, `TELEGRAM_ALLOW_ALL`.

### Fixed

- Repeating events (for example made in Thunderbird) showed an hour off after a daylight saving time change: Radicale expands them at their first occurrence's UTC time. The bot now fetches them unexpanded and expands them itself, in their own time zone, with EXDATE and changed occurrences (RECURRENCE-ID), using github.com/teambition/rrule-go.

- With no location saved, weather replies used the catch-all reply as the place name ("The weather in I'm not sure I understand..."), and a weather service failure was pasted into the sentence. Both now give a clear message.
- "Ask openai ...", "chat ...", "translate ... to French" and similar replied by repeating the user's words: `testdata/sraix-config-example.aiml`, an example for services the bot doesn't have, was loaded with the bot's AIML. It and `geocode-config-example.properties` (a stale copy of the geocode service) are moved to `examples-module/`, and a test checks that every service `testdata/` uses is configured there.
- "Show my lists" with no lists, or the items of an empty list, replied `[]`. They now say "You have no lists yet." and "No items in this list.", and `<jsonformat>` shows any other empty array as "There's nothing here yet."
- Adding, creating, renaming or changing list items and lists failed when the text contained `"` or `\` (e.g. `12" pizza`): the JSON body was built in AIML. The list services now build their bodies with `body` templates from the predicates the AIML sets.
- List login failed for passwords containing `&`, `+`, `=` or `%`: the form body was built in AIML without URL-encoding. `list_login` now uses a `body` template.
- With verbose logging, passwords and list access tokens were written to the log (the raw input, predicate lookups and sets). They are now masked.
- List commands failed once the list handler's token expired (30 minutes), saying for example that the list could not be found, until the user logged in again. They now log in again with the saved credentials and retry, or say that the login has expired.
- A failed list login left the fallback text in `list_access_token`, so later list commands sent it as the token.
- List items always showed as not done: the list handler's `is_completed` field was ignored.
- `<that>` matched against rich media markup in the bot's previous reply.
- `/session` showed the user's last message as the bot's last response.

- Loading a directory merges its `.properties` files in load (lexical path) order, so a property set in several files takes its value from the file loaded last, as categories do. Before, the order was random, and the bot's name or other properties could change between runs.

## v1.6.11 - 2026-11-17

### Upgrading: behaviour changes

- **`<id/>` returns the client ID, as AIML 2.0 specifies.** This is the ID of the session being answered (e.g. `telegram_42`), or empty without a session. It used to return the bot's `id` property, or `golem`. Use the new `<botid/>` tag for that.
- **The regex-based template processors are removed.** The tree processor was already the only one in use. These methods are gone: `EnableTreeProcessing`, `DisableTreeProcessing`, `IsTreeProcessingEnabled`, `GetConsolidatedProcessor`, `GetProcessorMetrics`, `SetProcessingOrder`, `GetProcessor`, `GetProcessorsByType`, and the stats, clear and invalidate methods of the regex, template, pattern-matching and that-pattern caches, which were never filled. `GetProcessorStats`, `ResetProcessorMetrics` and `GetProcessingOrder` remain.
- **`<shuffle>` gives a fresh order each time.** Before, its result was cached, so the same words always came back in the same order.
- **A `<sraix>` tag's `default` attribute now comes before the service's configured `fallback`.** Before, a configured fallback hid the tag's default.
- **SRAIX requests that cannot be completed are not sent.** A URL or header placeholder with no value, or an unset `${ENV_VAR}`, now fails the request and falls back. Before, the placeholder was sent as it was, or the variable as an empty string.
- **SRAIX services get only the session values they name.** `<sraix>` no longer passes `location`, `latitude`/`longitude`, `_coords`, `list_access_token` and `list_user_id` to every service. Name what a service needs with `["sraix.<service>.param.<placeholder>", "<predicate>"]` (or `Params` in a `SRAIXConfig`), or pass values in the input as JSON, e.g. `{"lat":"..","lon":".."}`. `testdata/list-handler-config.properties` and `testdata/weather.aiml` are updated.
- **A `hint` attribute no longer fills `{lat}` and `{lon}`.** Any hint with one comma, such as "Paris, France", used to be split into coordinates. Pass coordinates in the input instead.
- **`geocode_lon` is gone from `testdata`.** `geocode` now sets the `geocode_lon` predicate from the same reply, so geocoding takes one request instead of two; use `<get name="geocode_lon"/>`.
- **The CalDAV configuration has changed.** Set `RADICALE_PASSWORD` instead of a pre-encoded `RADICALE_AUTH`. "Delete event" and "get event" take the event's title. The unused `CalDAVHelper` type is removed; its work is done by the `caldav` SRAIX handler.
- **`SRAIXManager.GetServiceConfig` is removed.** Use `GetConfig`. `AddConfig` stores a copy of the config, and `ListConfigs` returns copies.
- **SRAIX request bodies follow the configured `Content-Type`.** GET, HEAD, DELETE and OPTIONS send no body. JSON input is sent as it is when no Content-Type is configured, not wrapped in `{"input": ...}`. Any Content-Type other than JSON (e.g. `text/calendar`, `application/xml`) gets the input as it is.

### Added

- `<botid/>` returns the bot's `id` property, or `golem` when it is not set.
- SRAIX `param.<placeholder>` names the predicate that fills a placeholder, and `set.<predicate>` sets predicates from several paths of one JSON reply.
- SRAIX `default.<placeholder>` gives a placeholder a value when nothing else does; `username` and `password` send basic authentication; `${ENV_VAR:-fallback}` gives an unset environment variable a value.
- **CalDAV calendars work.** The new SRAIX `caldav` handler lists today's and the coming week's events, creates events from text such as "Lunch with Sam tomorrow at noon", and shows or deletes an event by its title. Each session can pick its own calendar. `testdata/caldav-config.properties` and `testdata/caldav-examples.aiml` use it; see `CALDAV_INTEGRATION.md`. Before, the CalDAV services sent malformed requests and could not list, create or delete events.

### Fixed

- `ProcessTemplate`, `ProcessTemplateWithContext` and `ProcessTemplateWithSession` hold the knowledge base lock, so they no longer race with `SetKnowledgeBase` or loading.
- `<split>` works with its `limit` attribute written before `delimiter`.
- SRAIX:
  - A `204 No Content` reply is a success, not a fallback, so a successful DELETE no longer reports failure.
  - JSON numbers keep their digits: `{"list_id": 12345678}` no longer becomes `1.2345678e+07`, in the URL or in a response. A response path that points at an object or array returns JSON.
  - Values given in the input (as JSON or form fields) take precedence over wildcards and session values of the same name.
  - Form input with one field (`list_id=1`) is read, and form values are URL-decoded once rather than encoded twice.
  - The input no longer fills a URL placeholder when it is also the body, so `caldav_create` no longer puts the event text in the event's URL.
  - Values in the URL path are escaped as a path segment (`%20`, not `+`).
  - The SRAIX content and replies, request and response bodies, and response headers are no longer logged, since they carry credentials and tokens. Error messages include at most 200 bytes of the response.
  - A `responsepath` on a service whose response format is not json logs a warning; it is ignored.
  - Reloading properties removes the SRAIX services whose properties are gone.
  - `sraix list` and `sraix test` show the URL template of template-based services, not an empty base URL.
  - `WEATHER FORECAST` in `testdata/weather.aiml` formats the forecast instead of showing raw JSON.
  - A SRAIX request no longer holds the knowledge base lock while it waits. Before, a slow service blocked loading and reloading for up to its timeout, and once a load was waiting, every other conversation too.

## v1.6.0 - 2026-11-16

This release replaces Golem's pattern matcher with an AIML 2.0 graphmaster. It also fixes many template, loader, concurrency and SRAIX bugs. Some changes alter how existing bots answer, so read **Upgrading** before you update.

### Upgrading: behaviour changes

- **`*` and `_` now need at least one word.** `HELLO *` no longer matches a bare `hello`. If you relied on that, add a `HELLO` category or use `^` or `#`, which match zero or more words.
- **`_` can match more than one word.** Before, it matched exactly one.
- **Wildcards only match whole words.** `HELLO *` no longer matches `helloworld`, and `* LOVE` no longer matches `glove`.
- **Input is split into sentences.** Each sentence is matched and answered in turn, and the replies are joined. `hi. how are you?` now gets two answers.
- **`<that>` matches the last sentence of the bot's previous response.** Before, it matched the raw template text, and the match was not anchored. Now `<that>LIKE CATS</that>` no longer matches "Do you like cats?". `<that>` also works after templates that use `<srai>`, `<random>` or `<get>`.
- **`<set>` outputs the value it sets**, as the spec requires. Wrap the `<set>` in `<think>` where you want it silent.
- **`<learn>` evaluates only `<eval>`, plus wildcards when the teaching category has them.** Other tags, such as `<get>` and `<set>`, are stored and run when the learned category is used.
- **Responses are plain text, not XML-escaped.** `&amp;` in a template is output as `&`. Escape responses yourself if you insert them into HTML.
- **Alternation groups such as `(A|B)` are no longer matched.** They were not part of AIML 2.0.
- **SRAIX no longer returns hard-coded English replies when it fails.** It uses the configured fallback, then the `default` attribute, then the tag's content.

### Pattern matching (AIML 2.0 graphmaster)

- Categories form a graphmaster tree (pattern, then `<that>`, then `<topic>`) and are matched word by word. At each word Golem tries, in order: `$WORD`, `#`, `_`, `WORD` (or `<bot name="..."/>`), `<set>`, `^`, `*`. The earlier word decides.
- Fixed: a catch-all `*` or `TEACH *` could outrank `TEACH * MEANS *`.
- Fixed: `HELLO _` beat `_ WORLD`.
- Fixed: `<that>` could outweigh the input pattern.
- Fixed: a lone `$` was treated as a wildcard.
- `<bot name="..."/>` now matches inside patterns.
- `<that>` patterns may start with a wildcard, and `^` works in `<that>`.
- `<topicstar/>` returns the words matched by wildcards in `<topic>`.
- Patterns may be lowercase and may contain accents, apostrophes, `&`, `@` and other punctuation.
- Matching is deterministic. Equally good matches go to the category loaded first.
- Wildcard values keep the user's original case and punctuation.
- Learned categories are added to the tree in place, so `<learn>` does not rebuild it.

### Templates

- `<random>` runs only the chosen `<li>`. Before, every item's `<set>`, `<srai>` and `<learn>` ran. An empty `<li>` is now a valid choice.
- `<condition>`:
  - supports `name` or `var` on each `<li>`;
  - supports `value="*"`, which is true when the variable has a value;
  - supports `<loop/>`, bounded by a maximum number of iterations.
- Attributes can be written as child elements, for example `<get><name>x</name></get>`.
- Local variables set with `<set var>` stay in their own category, including across `<srai>`.
- `<srai>` and `<sr/>` match against the current topic and `<that>`.
- `<sr/>` is exactly `<srai><star/></srai>`. It no longer overwrites predicates whose names start with `star`, such as `starsign`.
- Inside a template reached by `<srai>`, `<star/>` no longer shows the caller's wildcards.
- XML entities in template text and attribute values are decoded.
- Unknown tags, such as HTML links, keep their attributes.

### Loading

- Categories with the same pattern but a different `<that>` or topic no longer overwrite each other.
- `<topic name="...">` blocks are honoured.
- A `<category>` inside `<learn>` no longer leaks its `<that>` or `<topic>` into the category that contains it.
- XML comments that span several lines are removed.
- An invalid category is dropped with a warning instead of rejecting its whole file.
- Fixed a parser panic on an unterminated attribute value.
- Fuzz targets were added for the loader and the AST parser.

### Concurrency and learning

- Fixed data races between concurrent sessions, which could crash with "concurrent map writes" or write one session's variables into another. The full test suite passes under `-race`.
- `<learn>` and `<unlearn>` are safe while other sessions are matching.
- After `<unlearn>`, other patterns no longer answer with a neighbouring category's template.

### SRAIX

- User input is URL-escaped and placeholders are substituted in a single pass. User input can no longer add query parameters or expose the API key.
- `{STAR1}`, `{STAR2}`, ... and `{location}` placeholders now have values.
- An empty or unparseable response goes through the fallback chain.
- Configured timeouts longer than 30s are honoured.
- Responses are capped at 1 MB.
- Lower-case methods such as `get` are accepted.
- Verbose logs mask secrets.

### Performance and housekeeping

- The LRU caches are now O(1), which makes each lookup 6–11x faster.
- Regexes are compiled once instead of on every call.
- Removed the regex pattern matcher, unused code and obsolete documentation.
- The version is read from the `VERSION` file.

**Full changelog:** https://github.com/helix90/golem/compare/v1.5.0...v1.6.0
