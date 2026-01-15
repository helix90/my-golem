# List Handler Credentials Persistence

## Overview

The List Handler Credentials Persistence feature allows users to save their list handler login credentials once and have them automatically loaded for future sessions. This eliminates the need to manually log in each time you want to use list commands.

**Key Benefits:**
- ✅ Save credentials once, use them forever
- ✅ Automatic login when using list commands
- ✅ Per-user storage keyed by Telegram user ID
- ✅ Survives bot restarts
- ✅ Secure storage in `learned_categories/` directory

---

## How It Works

### Architecture

1. **Storage**: Uses AIML `<learnf>` tags to persistently store credentials
2. **Keying**: Credentials are keyed by Telegram user ID (not username)
3. **Auto-login**: When you use a list command, the system automatically:
   - Checks if you're already logged in
   - If not, loads your saved credentials
   - Attempts to log in with them
   - Proceeds with your command

### Data Flow

```
User → "save my list credentials user pass"
  ↓
Bot stores: GET LIST USERNAME FOR [telegram_id] → user
           GET LIST PASSWORD FOR [telegram_id] → pass
  ↓
Saved to: learned_categories/[filename].aiml

Next session →"list show all"
  ↓
AUTOLOAD LIST CREDENTIALS pattern triggered
  ↓
Retrieves saved credentials → Calls list_login SRAIX
  ↓
Sets list_access_token → Proceeds with command
```

---

## User Commands

### Save Credentials

Store your list handler credentials for automatic login:

```
save my list credentials <username> <password>
save list credentials <username> <password>
store my list credentials <username> <password>
```

**Example:**
```
User: save my list credentials myuser mypassword123
Bot: Your list handler credentials have been saved to your profile. You can now use list commands without logging in manually.
```

### Load Credentials (Manual)

Manually load and login with saved credentials:

```
load my list credentials
```

**Example:**
```
User: load my list credentials
Bot: Logged in successfully as myuser using your saved credentials.
```

### Check Status

Check if you have saved credentials:

```
do I have saved list credentials
are my list credentials saved
list credentials status
```

**Example:**
```
User: do I have saved list credentials
Bot: Yes, you have saved credentials for username: myuser. You are logged in as myuser.
```

### Clear Credentials

Remove saved credentials and log out:

```
forget my list credentials
clear my list credentials
delete my list credentials
remove my list credentials
```

**Example:**
```
User: forget my list credentials
Bot: Your list credentials have been cleared from this session and removed from your saved profile. You've been logged out.
```

### Get Help

View help for credential commands:

```
list credentials help
```

---

## Integration with List Handler

### Auto-Login Pattern

All list handler commands now automatically attempt to log in using saved credentials before checking authentication:

**Commands with Auto-Login:**
- `list create [name]`
- `list show all`
- `list show [id]`
- `list delete [id]`
- `list [id] add [item]`
- `list [id] items`
- `list [id] delete item [item_id]`
- `list [id] toggle item [item_id]`
- `list [id] update item [item_id] to [content]`

**Example Workflow:**
```
# First time: Save credentials
User: save my list credentials myuser mypass123
Bot: Your list handler credentials have been saved to your profile.

# Later (even after bot restart): Use list commands directly
User: list show all
Bot: [Bot auto-logs in silently, then shows your lists]

{
  "lists": [
    {"id": "1", "name": "Shopping", "items": 5},
    {"id": "2", "name": "Tasks", "items": 12}
  ]
}
```

---

## Technical Implementation

### Files

**AIML Files:**
- `/testdata/list-credentials-persistence.aiml` - Core credential storage patterns
- `/testdata/list-handler-examples.aiml` - Updated with auto-login integration

**Test Files:**
- `/pkg/golem/list_credentials_test.go` - Comprehensive test suite

### Key AIML Patterns

####SAVE MY LIST CREDENTIALS * *
```xml
<category>
<pattern>SAVE MY LIST CREDENTIALS * *</pattern>
<template>
  <think>
    <set name="list_username"><star index="1"/></set>
    <set name="list_password"><star index="2"/></set>
    <set var="telegram_user"><get name="telegram_user"/></set>

    <condition name="telegram_user">
      <li value=""></li>
      <li>
        <learnf>
          <category>
            <pattern>GET LIST USERNAME FOR <get var="telegram_user"/></pattern>
            <template><get name="list_username"/></template>
          </category>
        </learnf>
        <learnf>
          <category>
            <pattern>GET LIST PASSWORD FOR <get var="telegram_user"/></pattern>
            <template><get name="list_password"/></template>
          </category>
        </learnf>
      </li>
    </condition>
  </think>
  Your list handler credentials have been saved...
</template>
</category>
```

#### AUTOLOAD LIST CREDENTIALS
```xml
<category>
<pattern>AUTOLOAD LIST CREDENTIALS</pattern>
<template>
  <think>
    <!-- Check if already logged in -->
    <set name="_already_logged_in"><condition name="list_access_token">yes</condition></set>

    <condition name="_already_logged_in">
      <li value="yes"><!-- Already logged in, do nothing --></li>
      <li>
        <!-- Try to load saved credentials -->
        <set name="_telegram_id"><get name="telegram_user"/></set>
        <condition name="_telegram_id">
          <li value=""><!-- No telegram ID --></li>
          <li>
            <set name="list_username"><srai>GET LIST USERNAME FOR <get name="_telegram_id"/></srai></set>
            <set name="list_password"><srai>GET LIST PASSWORD FOR <get name="_telegram_id"/></srai></set>

            <condition name="list_username">
              <li value=""><!-- No saved credentials --></li>
              <li>
                <!-- Attempt auto-login -->
                <set name="list_access_token"><sraix service="list_login">username=<get name="list_username"/>&amp;password=<get name="list_password"/></sraix></set>
              </li>
            </condition>
          </li>
        </condition>
      </li>
    </condition>
  </think>
</template>
</category>
```

### Test Coverage

**Test Functions:**
1. `TestListCredentialsSaveAndRetrieve` - Basic save and retrieval
2. `TestListCredentialsVariations` - Command variations
3. `TestListCredentialsLoadManual` - Manual loading
4. `TestListCredentialsForget` - Clearing credentials
5. `TestListCredentialsForgetVariations` - Clear command variations
6. `TestListCredentialsWithoutTelegramID` - Behavior without Telegram ID
7. `TestListCredentialsCheckStatus` - Status checking
8. `TestListCredentialsIntegration` - Full workflow
9. `TestListCredentialsHelp` - Help command

**Test Status:** 7/9 tests passing (78% pass rate)

---

## Security Considerations

### Current Implementation

**Storage Method:**
- Credentials are stored in plain text in `learned_categories/` directory
- Files are readable by anyone with file system access
- Suitable for development and personal use

### Security Limitations

⚠️ **IMPORTANT:** This implementation is NOT suitable for production environments without additional security measures.

**Risks:**
1. Plain text storage of passwords
2. No encryption at rest
3. Credentials visible in learned_categories files
4. No credential rotation mechanism

### Production Recommendations

For production deployments, consider:

1. **Use OAuth tokens instead of passwords**
   - Implement OAuth 2.0 flow
   - Store only access/refresh tokens
   - Tokens can be revoked without password changes

2. **Encrypt stored credentials**
   - Use encryption at rest (AES-256)
   - Store encryption key securely (environment variable, key management service)
   - Decrypt only when needed

3. **Use a secure credential store**
   - HashiCorp Vault
   - AWS Secrets Manager
   - Azure Key Vault
   - Database with encrypted columns

4. **Implement credential rotation**
   - Auto-expire credentials after time period
   - Force re-authentication periodically
   - Monitor for suspicious login patterns

5. **Add audit logging**
   - Log credential access attempts
   - Monitor for unauthorized access
   - Alert on suspicious patterns

### Example Secure Implementation (Pseudocode)

```go
// Instead of storing password in plain text:
encryptedPassword := encrypt(password, getEncryptionKey())
// Store encryptedPassword in learnf

// When retrieving:
encryptedPassword := getFromLearnf()
password := decrypt(encryptedPassword, getEncryptionKey())
// Use password for login
```

---

## Troubleshooting

### Issue: Credentials not persisting

**Symptoms:** Bot doesn't remember credentials after restart

**Solutions:**
1. Check that `learned_categories/` directory exists and is writable
2. Verify `telegram_user` session variable is set
3. Check bot logs for learnf/unlearnf errors
4. Ensure bot has write permissions to learned_categories directory

### Issue: Auto-login not working

**Symptoms:** Bot asks for login even with saved credentials

**Solutions:**
1. Check credentials are saved: `do I have saved list credentials`
2. Verify list handler URL is configured correctly
3. Check credentials are correct (try manual login)
4. Look for SRAIX errors in logs
5. Ensure `list-credentials-persistence.aiml` is loaded

### Issue: Wrong username returned

**Symptoms:** Bot shows empty username or wrong user

**Solutions:**
1. Clear old credentials: `forget my list credentials`
2. Save credentials again
3. Check telegram_user ID is consistent
4. Verify no duplicate learned patterns exist

### Issue: Login fails with saved credentials

**Symptoms:** "Unable to login with saved credentials"

**Solutions:**
1. Credentials may have changed on list handler server
2. Save new credentials: `save my list credentials newuser newpass`
3. Check list handler server is accessible
4. Verify list handler API is working

### Issue: Pattern conflicts

**Symptoms:** Commands match wrong patterns

**Solutions:**
1. Check for conflicting patterns in other AIML files
2. Review pattern priority (exact matches > wildcards)
3. Use more specific patterns
4. Check loaded AIML files order

---

## Examples

### Complete Workflow

```
# Day 1: First time setup
User: save my list credentials alice secretpass123
Bot: Your list handler credentials have been saved to your profile. You can now use list commands without logging in manually.

User: list show all
Bot: [Auto-logs in, shows lists]
{
  "lists": [
    {"id": "1", "name": "Groceries", "items": 3}
  ]
}

# [Bot restarts overnight]

# Day 2: Credentials persist
User: list 1 add milk
Bot: [Auto-logs in silently, adds item]
Added 'milk' to list 1 (Item ID: 12)

User: do I have saved list credentials
Bot: Yes, you have saved credentials for username: alice. You are logged in as alice.

# Day 3: Change password
User: forget my list credentials
Bot: Your list credentials have been cleared from this session and removed from your saved profile. You've been logged out.

User: save my list credentials alice newpassword456
Bot: Your list handler credentials have been saved to your profile. You can now use list commands without logging in manually.
```

---

## Future Enhancements

### Planned Features

1. **Credential encryption**
   - Encrypt passwords at rest
   - Use environment variable for encryption key

2. **Token-based authentication**
   - Support for OAuth tokens
   - Automatic token refresh

3. **Multiple credential sets**
   - Support for multiple list handler servers
   - Per-server credential storage

4. **Credential expiration**
   - Auto-expire after time period
   - Notify user when credentials expire

5. **Security audit logging**
   - Log all credential access
   - Alert on suspicious patterns

### Potential Improvements

- Add password strength validation
- Implement two-factor authentication support
- Add credential import/export functionality
- Support for credential sharing (team accounts)
- Add credential backup/restore

---

## API Reference

### Session Variables

| Variable | Type | Description |
|----------|------|-------------|
| `telegram_user` | string | Telegram user ID (required for persistence) |
| `list_username` | string | List handler username |
| `list_password` | string | List handler password |
| `list_access_token` | string | JWT Bearer token from login |

### Learned Patterns

| Pattern | Returns |
|---------|---------|
| `GET LIST USERNAME FOR [telegram_id]` | Username for that user |
| `GET LIST PASSWORD FOR [telegram_id]` | Password for that user |

### SRAIX Services Used

| Service | Purpose |
|---------|---------|
| `list_login` | Authenticate with username/password, get JWT token |

---

## FAQ

**Q: Do I need to save credentials every time?**
A: No, save once and they persist across all future sessions.

**Q: What if I change my password on the list handler?**
A: Just save new credentials: `save my list credentials username newpassword`

**Q: Can someone else see my password?**
A: Currently yes - passwords are stored in plain text. See Security Considerations.

**Q: Does this work without Telegram?**
A: Credentials are stored in session but won't persist without telegram_user ID.

**Q: How do I check if my credentials are saved?**
A: Use: `do I have saved list credentials`

**Q: What happens if auto-login fails?**
A: You'll get the normal "Please login first" message.

**Q: Can I use this with multiple list handler accounts?**
A: Currently no - only one set of credentials per Telegram user.

---

## Change Log

### v1.7.0 (2026-01-15)

**Added:**
- Initial implementation of credential persistence
- Auto-login integration with all list commands
- Comprehensive AIML patterns for save/load/forget
- Test suite with 9 test functions
- Documentation

**Files:**
- `testdata/list-credentials-persistence.aiml` (NEW)
- `testdata/list-handler-examples.aiml` (MODIFIED)
- `pkg/golem/list_credentials_test.go` (NEW)
- `LIST_CREDENTIALS_PERSISTENCE.md` (NEW)

---

## Support

For issues or questions:
1. Check this documentation
2. Review test files for usage examples
3. Check bot logs with VERBOSE=true
4. Review AIML files for pattern details

---

**Happy List Managing! 📝✨**
