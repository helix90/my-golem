# Assistant features

Who can use the Telegram bot, what it remembers about each user, reminders
and the daily briefing, buttons, Wikipedia lookups, and sums and searches with SearXNG.

## Who can use the bot

The bot answers only the Telegram users in `TELEGRAM_ALLOWED_USERS`, a
comma-separated list of user IDs. It can read and change your calendar and
lists, so it is private by default: with the list empty, it answers no one.

Anyone else, in a private chat or a group, gets "Sorry, this is a private
assistant. Your Telegram user ID is 123456789; ask the bot's owner to add it
to TELEGRAM_ALLOWED_USERS." (at most once every 10 minutes), and the bot logs
the ID. That is also the easiest way to find your own ID: start the bot,
message it, and copy the number into `TELEGRAM_ALLOWED_USERS`.

- `TELEGRAM_ADMINS` (IDs, each also allowed) may run `/reload`; by default
  every allowed user may.
- `TELEGRAM_ALLOW_ALL=true` lets everyone in. Use it only for a bot with
  nothing private behind it; `/reload` then needs `TELEGRAM_ADMINS`.
- Messages, commands, button taps and shared locations are all checked, and
  reminders of users who were removed from the list are not sent.
- A user removed from the list keeps their stored data until it is deleted;
  they can no longer reach `/forgetme`.

## What the bot remembers about users

Every predicate whose name starts with `user_` (`user_name`,
`user_location`, `user_list_username`, ...) is kept for the person who set
it, in a **user store**: a JSON file at `USER_STORE_PATH` (default
`data/users.json`). AIML needs nothing special: `<set name="user_name">` saves
it and `<get name="user_name"/>` reads it, in any later session and after a
restart.

How it works: the Telegram bot sets each session's `UserID` to the sender of
the message. Before golem answers, it loads that user's `user_*` predicates
into the session; afterwards it saves any that changed. Setting a predicate to
`""` deletes it. In a group chat each message gets its sender's values.

Nothing in the store can be matched by input, so no user can read another's
data. (Earlier versions saved this with `<learnf>` as categories like
`GET LIST PASSWORD FOR <telegram id>`, which any user could ask for.)

- **Secrets.** Predicates matching `*password*`, `*secret*` or `*token*` are
  secret: `/session` shows them as `(hidden)`, and with `USER_STORE_KEY` set
  they are encrypted in the file (AES-256-GCM). Set `USER_STORE_KEY` to a long
  random string and keep it; without it the bot can't read saved passwords.
- **Logs.** Even with `VERBOSE=true`, secret values and passwords typed in
  credential commands are masked as `********` in golem's log.
- **Passwords in chat.** When someone types `save my list credentials USER
  PASS` or `list login USER PASS`, the bot deletes that message from the chat
  and removes the password from the session's history.
- **Choosing what persists.** The bot properties `userstore.persist` and
  `userstore.secret` take comma-separated globs and replace the defaults
  (`user_*`, and `*password*,*secret*,*token*`).
- **Moving old data.** At startup (and on `/reload`) the bot moves the values
  saved by the old `GET NAME/LOCATION/LATITUDE/LONGITUDE/LIST USERNAME/LIST
  PASSWORD FOR <id>` categories into the store, and deletes those categories
  from the knowledge base, `learned_categories.json` and its backups. Its log
  says how many it moved.
- **Forgetting.** `/forgetme` deletes everything stored about the user and
  their reminders ("🗑️ Done. I've deleted everything I remembered about you
  and 2 reminder(s)."). `/clear` only clears the conversation.

Predicates used by the bundled AIML:

| Predicate | Set by |
| --- | --- |
| `user_name` | "my name is ...", "call me ..." |
| `user_location`, `user_latitude`, `user_longitude` | "my location is ...", or sharing a location |
| `user_list_username`, `user_list_password` (secret) | "save my list credentials USER PASS" |

Library use:

```go
store, err := golem.OpenUserStore("data/users.json", os.Getenv("USER_STORE_KEY"))
g.SetUserStore(store)
moved, err := g.MigrateLegacyUserData() // after loading the AIML
session.UserID = "123456789"           // whoever is talking
```

## Reminders, timers and the daily briefing

`testdata/reminders.aiml` and `testdata/reminders-config.properties`.

```
remind me to call mom at 5pm
remind me in 20 minutes to stretch
remind me tomorrow at 9am to take the bins out
remind me every day at 8am to take vitamins
set a timer for 10 minutes
show my reminders
delete reminder 2            (or: delete reminder mom)
send me a daily briefing at 7am
brief me
stop my daily briefing
```

Times use the date parser of the calendar (today, tomorrow, weekdays, dates,
"at 5pm", "noon", ...) plus "in N minutes/hours/days" and "every day". A date
with no time means 9 AM. Times are in the bot's time zone (`TZ`).

For example, on a Wednesday evening:

```
User: remind me to call mom at 5pm
Bot:  Okay, I'll remind you to call mom tomorrow at 5 PM.

User: remind me in 20 minutes to stretch
Bot:  Okay, I'll remind you to stretch today at 6:05 PM.

User: set a timer for 10 minutes
Bot:  Okay, I'll remind you to check the timer today at 5:55 PM.

User: remind me every day at 8am to take vitamins
Bot:  Okay, I'll remind you to take vitamins every day at 8 AM.

User: remind me to call mom
Bot:  When should I remind you? Try something like "remind me to call mom at 5pm" or "remind me in 20 minutes to stretch".

User: show my reminders
Bot:  Your reminders:
      1. check the timer, today at 5:55 PM
      2. stretch, today at 6:05 PM
      3. take vitamins, every day at 8 AM
      4. call mom, tomorrow at 5 PM

User: delete reminder 2
Bot:  Deleted the reminder to stretch.

User: cancel the timer
Bot:  Deleted the reminder to check the timer.

  ... at 5 PM the next day ...

Bot:  ⏰ Reminder: call mom

User: send me a daily briefing at 7am
Bot:  Okay, I'll send you a briefing every day at 7 AM, starting tomorrow at 7 AM.

User: brief me
Bot:  Good evening, Alice! Today is Wednesday, November 25. Your calendar is clear today.

User: stop my daily briefing
Bot:  Okay, I've stopped your daily briefing.
```

The greeting follows the time of day in the bot's time zone (`TZ`): good
morning (5:00 to 11:59), afternoon (12:00 to 16:59) or evening (17:00 to
21:59), and hello at night. With events today the briefing lists them ("Today
you have standup at 9 AM and lunch at noon."), and with a saved location and
a weather service it adds the weather.

Reminders are kept in `REMINDERS_PATH` (default `data/reminders.json`) and
survive restarts; one that came due while the bot was down is sent when it
starts. The bot checks every 20 seconds and sends each reminder to the chat it
was set in.

The **daily briefing** is sent by answering the input `DAILY BRIEFING` in the
user's session, so its content is AIML: it greets the user, gives the date,
today's calendar events (when a calendar is set up) and the weather (when the
user has a location). Edit the `DAILY BRIEFING` category to change it.

The SRAIX handler is `reminders`, with operations `add`, `list`, `delete`,
`briefing` and `briefing_off`; it needs the `user` and `chat` params, which
the Telegram bot provides as the `telegram_user` and `telegram_chat`
predicates. Library use: `g.SetReminderStore(store)` and
`go store.Run(ctx, 20*time.Second, deliver)`.

## Buttons

Templates offer choices with the rich media elements of AIML 2.1:

```xml
Should I delete the event "<star/>"?
<button><text>Yes, delete it</text><postback>CONFIRM DELETE EVENT <star/></postback></button>
<button><text>No, keep it</text><postback>KEEP THE EVENT</postback></button>

<button><text>Open the map</text><url>https://example.com/map</url></button>
<reply><text>Today</text><postback>WHAT IS ON MY CALENDAR TODAY</postback></reply>
<reply><text>📍 Share my location</text><location/></reply>
```

- `<button>` becomes an inline button under the message. Tapping it sends its
  `<postback>` (or its text) as the user's next input, and removes the buttons
  from that message so a question is answered once.
- `<reply>` becomes a suggested reply above the keyboard; tapping it sends its
  postback. `<location/>` makes it ask Telegram for the user's location,
  which arrives as `MY COORDINATES ARE <lat> <lon>`.
- `<that>` sees only the text, so typed answers still work: the delete
  question above also accepts "yes" and "no".

In a conversation (buttons in brackets, suggested replies in parentheses):

```
User: delete event dentist
Bot:  Should I remove "dentist" on Friday at 3 PM from your calendar?
      [Yes, remove it] [No, keep it]
User: no                                   (or tap "No, keep it")
Bot:  Okay, I'll leave it on your calendar.

User: what is the weather
Bot:  You haven't set your location yet. Tell me by saying "My location is [city name]", or share it.
      (📍 Share my location)
      ... the user taps it and Telegram shares their location ...
Bot:  Thanks! I've saved your location (51.5072, -0.1276), so you can ask me about the weather there.

User: show list groceries
Bot:  Your groceries list has 2 items:
      ☐ milk
      ☐ oat milk
      [☐ milk] [☐ oat milk]

User: show my lists
Bot:  You have 2 lists: groceries and hardware.
      [groceries] [hardware]
```

`golem.ParseRichResponse` splits a response into its text, buttons and
replies, for other clients.

The bundled AIML uses buttons for:

- **Deleting a calendar event**, which now asks first.
- **Lists**: `list groceries check off milk` and `list groceries remove
  milk` change an item by its name. `show my lists` shows a button for each
  list. Adding to or showing a list that isn't there asks whether to create
  it, with Yes/No buttons. `show list groceries` shows a ☐/☑ button for each item, and
  tapping one ticks it and shows the list again. `<jsonformat
  button="LIST {list} TOGGLE ITEM {id}">` adds them; `{list}`, `{id}` and
  `{content}` are filled in per item.
- **Weather** without a location, which offers to share one.

## Looking things up in Wikipedia

`testdata/wikipedia.aiml` and `testdata/wikipedia-config.properties`. With
the my-AIML phrase pack, "what is ...", "who was ...", "tell me about ..."
and "what does ... mean" all look the topic up; without it, say
`wikipedia <topic>` or `look up <topic>`. No key is needed.

```
User: who was Ada Lovelace
Bot:  Augusta Ada King, Countess of Lovelace (née Byron; 10 December 1815 – 27 November 1852), also known as Ada Lovelace, was an English mathematician and writer chiefly known for work on Charles Babbage's proposed mechanical general-purpose computer, the analytical engine. She was the first to recognise the machine had applications beyond pure calculation.
      [📖 Read more on Wikipedia]

User: tell me about Lisbon
Bot:  Lisbon is the capital and most populous city of Portugal, with an estimated population of 658,236 as of 2025, within its administrative limits and 3,353,000 within the metropolis, as of 2025. The city lies in the western part of the Iberian Peninsula, fronting the northern shore of the River Tagus.
      [📖 Read more on Wikipedia]

User: what is mercury
Bot:  "mercury" could mean a few things. Which one?
      [Freddie Mercury – British rock singer and songwriter (1946–1991)] [Mercury (element)]
      [Mercury (planet)] [Mercury Records – American record label]

User: what is xqzzvbn florp
Bot:  I couldn't find anything on Wikipedia about "xqzzvbn florp".
```

- The reply is the first one or two sentences of the article's introduction
  (about 400 characters at most), without pronunciations, and a button
  linking to the article.
- The article is the one whose title is the topic, else Wikipedia's top
  search result. A disambiguation page ("Mercury") offers the other results
  as buttons, which send `WIKIPEDIA PAGE <title>`.
- **Set `WIKIPEDIA_USER_AGENT`** to something with your contact details, e.g.
  `my-assistant/1.0 (you@example.com)`. Wikimedia throttles clients without
  one; when it says "too many requests" the bot waits a moment and tries once
  more, then says it can't reach Wikipedia.
- `WIKIPEDIA_URL` picks another language, e.g.
  `https://de.wikipedia.org/w/api.php`.
- Questions with their own answers ("what is your name", "what is the
  weather", "what's on my list") still get them; only the rest go to
  Wikipedia.

## Sums, conversions and web searches with SearXNG

`testdata/searx.aiml` and `testdata/searx-config.properties`, for your own
[SearXNG](https://docs.searxng.org/) instance. Set `SEARX_URL` (e.g.
`http://192.168.1.10:8888`) and allow its JSON API in its `settings.yml`:

```yaml
search:
  formats:
    - html
    - json
```

Without that, SearXNG answers 403 and the bot says it can't reach the
calculator.

```
User: what is 2 + 2
Bot:  2 + 2 is 4.
User: what's 15 percent of 80
Bot:  15 percent of 80 is 12.
User: what is 5 miles in km
Bot:  5 miles in km is 8.04672 km.
User: calculate 1,234 * 5,678
Bot:  1,234 * 5,678 is 7006652.
User: search for best pizza in Lisbon
Bot:  The 10 best pizza places in Lisbon: Our pick of the best pizza in Lisbon.
      [The 10 best pizza places in Lisbon] [Pizza – Lisbon forum] [Lisbon food guide]
```

- **Sums** (`calculate ...`, `how much is ...`, and with the my-AIML pack
  "what is ..."): numbers with `+ - * / ^ %` and parentheses, or the words
  plus, minus, times, multiplied by, divided by, over, squared, cubed, to
  the power of, and "15% of 80" / "20 percent of 45". They go to SearXNG's
  Calculator plugin, which must be enabled (it is by default).
- **Conversions**: "5 miles in km", "72 fahrenheit to celsius", "100 usd in
  eur", "how many ounces in a pound", from SearXNG's unit converter and
  currency engine.
- Only sums and conversions are sent: "what is a quasar" goes straight to
  Wikipedia, so everyday questions don't reach the search engines behind
  SearXNG. A sum never goes to Wikipedia (whose top hit for "2 + 2" is the
  article on *2 + 2 = 5*): without SearXNG the bot says it can't reach the
  calculator.
- **Web searches**: `search for ...`, `search the web for ...`, `google ...`
  reply with the top result and buttons linking to the first three.
- Behind basic auth? Add `username` and `password` lines to the two
  services in `searx-config.properties` (they can use `${ENV_VAR}`).
