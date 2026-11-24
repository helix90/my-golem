# Remembering users

The bot remembers each Telegram user's name, location and saved list
credentials in the user store: every `user_*` predicate is saved for the
person who set it and comes back in later sessions and after restarts. See
[ASSISTANT_FEATURES.md](../ASSISTANT_FEATURES.md).

| File | Predicates |
| --- | --- |
| `user-name-persistence.aiml` | `user_name` |
| `user-location-persistence.aiml` | `user_location`, `user_latitude`, `user_longitude` |
| `list-credentials-persistence.aiml` | `user_list_username`, `user_list_password` |

```
User: My location is Seattle
Bot:  I've set your location to Seattle (coordinates: 47.6062, -122.3321). I'll remember it.
...restart...
User: What's the weather?
Bot:  The weather in Seattle is ...
```

Earlier versions saved these with `<learnf>` as `GET LOCATION FOR <user id>`
categories (and NAME, LATITUDE, LONGITUDE, LIST USERNAME, LIST PASSWORD), which
any user could match. The bot moves them into the user store at startup.
