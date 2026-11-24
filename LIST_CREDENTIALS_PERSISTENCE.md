# List Handler Credentials Persistence

Users can save their list handler login once; list commands then log in
automatically, in every session and after restarts.

```
save my list credentials USERNAME PASSWORD   saves them and logs in to check them
do I have saved list credentials             says which username is saved
load my list credentials                     logs in again now
forget my list credentials                   deletes them and logs out
```

The credentials are kept in the user store as `user_list_username` and
`user_list_password`, for the Telegram user who saved them (see
[ASSISTANT_FEATURES.md](ASSISTANT_FEATURES.md)). The password is a secret:
encrypted in the store when `USER_STORE_KEY` is set, hidden in `/session`,
and deleted from the chat and the session's history after it is typed. The
access token (`list_access_token`) stays in the session; a failed login
leaves it empty, so the next list command tries again.

Every list command in `testdata/list-handler-examples.aiml` calls `AUTOLOAD
LIST CREDENTIALS` first, which logs in with the saved credentials when the
session has no token.

**Special characters.** The login form is built and URL-encoded by the
engine (the `list_login` service's `body` setting), so passwords may contain
`&`, `+`, `=`, `%` and the like. A password can't contain spaces, or end with
`.`, `!` or `?`, since the bot splits input into words and sentences.

**Logs.** With `VERBOSE=true`, the password and the access token are masked
as `********` in every log line.

**Expired tokens.** The list handler's tokens expire (after 30 minutes by
default). When it rejects one with 401, the list services' `reauth` setting
makes the bot answer `LIST RELOGIN`, which logs in again with the saved
credentials, and the request is retried once with the new token, so the
user notices nothing. Without saved credentials the token is cleared and the
user is told their login has expired.

**Upgrading:** earlier versions saved credentials with `<learnf>` as `GET LIST
USERNAME FOR <id>` and `GET LIST PASSWORD FOR <id>` categories, which any user
could match to read another user's password. The bot moves those into the
user store at startup and deletes them from `learned_categories/` and its
backups.
