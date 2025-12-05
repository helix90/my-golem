# User Location Persistence with Telegram Integration

This system uses AIML's `<learnf>` tag to persist user location information across bot restarts, keyed by Telegram username.

## How It Works

1. **User sets location**: "My location is Seattle"
2. **Bot geocodes** the location to get coordinates
3. **Bot learns** three patterns using `<learnf>`:
   - `GET LOCATION FOR username` → "Seattle"
   - `GET LATITUDE FOR username` → "47.6062"
   - `GET LONGITUDE FOR username` → "-122.3321"
4. **Patterns are saved** to `learned_categories/` directory
5. **Next session**: Bot auto-loads user's location when they ask for weather

## Telegram Bot Integration

### Step 1: Set the Telegram Username in Session

When creating or retrieving a session for a Telegram user, set their username as a session variable:

```go
// In your Telegram bot message handler
session := golem.CreateSession(update.Message.From.Username)

// CRITICAL: Set the telegram_user variable
session.Variables["telegram_user"] = update.Message.From.Username
```

### Step 2: Load AIML Files

Make sure to load both the user persistence AIML and weather AIML:

```go
kb, err := golem.LoadAIMLFromDirectory("./aiml_file")
if err != nil {
    log.Fatal(err)
}
golem.SetKnowledgeBase(kb)
```

Your `aiml_file/` directory should contain:
- `user-location-persistence.aiml` (from testdata/)
- `weather.aiml` (from testdata/)
- Other AIML files as needed

### Step 3: Process User Messages

Normal message processing - the AIML patterns handle everything:

```go
response, err := golem.ProcessInput(userMessage, session)
if err != nil {
    log.Printf("Error: %v", err)
}
// Send response back to user
```

## Usage Examples

### Setting Location (First Time)
```
User: My location is Seattle
Bot:  I've set your location to Seattle (coordinates: 47.6062, -122.3321).
      This location has been saved to your profile.
```

### Getting Weather (Auto-loads saved location)
```
User: What's the weather?
Bot:  The weather in Seattle is Partly cloudy with a temperature of 54°F (12°C).
```

### Getting Tomorrow's Weather
```
User: What's the weather tomorrow?
Bot:  In Seattle, Tomorrow will be light rain with a high of 52°F (11°C)
      and a low of 45°F (7°C).
```

### Checking Current Location
```
User: What is my location?
Bot:  Your location is set to Seattle (coordinates: 47.6062, -122.3321).
```

### Loading Saved Location Manually
```
User: Load my location
Bot:  Welcome back! I've loaded your saved location: Seattle.
```

### Forgetting Location
```
User: Forget my location
Bot:  Your location has been cleared from this session and removed from
      your saved profile.
```

## How Auto-Loading Works

When a user asks for weather without setting their location:

1. `WHAT IS THE WEATHER` pattern calls `AUTOLOAD LOCATION FOR WEATHER`
2. Checks if location is already in session
3. If not, uses SRAI to call `GET LOCATION FOR [username]`
4. If found, loads location/lat/lon into session variables
5. Weather query proceeds with loaded location

## Persistence Details

### Storage Location
Learned patterns are saved to: `learned_categories/`

Each learned category is saved as a separate file with pattern and template.

### What Gets Saved
For each Telegram user:
- Location name (city/address)
- Latitude (for SRAIX weather calls)
- Longitude (for SRAIX weather calls)

### Session vs. Persistent Storage
- **Session variables** (`<set name="">`) are lost on bot restart
- **Learned patterns** (`<learnf>`) persist across restarts
- Auto-load brings persistent data into session variables

## Testing Without Telegram

For testing, you can manually set the telegram_user variable:

```go
session := golem.CreateSession("test-session")
session.Variables["telegram_user"] = "testuser123"

// Now "My location is Seattle" will save to testuser123's profile
```

## Advanced: Using for Other Data

This pattern can be extended to save other user preferences:

```xml
<!-- Learn user's preferred temperature unit -->
<learnf>
  <category>
    <pattern>GET TEMP UNIT FOR <get var="telegram_user"/></pattern>
    <template>fahrenheit</template>
  </category>
</learnf>

<!-- Learn user's timezone preference -->
<learnf>
  <category>
    <pattern>GET TIMEZONE FOR <get var="telegram_user"/></pattern>
    <template>America/Los_Angeles</template>
  </category>
</learnf>
```

## Troubleshooting

### Location Not Persisting
- Verify `telegram_user` session variable is set
- Check that `learned_categories/` directory exists and is writable
- Enable verbose logging to see learnf operations

### Location Not Auto-Loading
- Verify the learned patterns exist: check `learned_categories/`
- Try manually: "load my location"
- Check logs for SRAI calls to `GET LOCATION FOR [username]`

### Geocoding Failures
- Ensure geocode and geocode_lon SRAIX services are configured
- Check `geocode-config.properties` is loaded
- Verify internet connection for Nominatim API calls

## Security Considerations

1. **Username Sanitization**: Telegram usernames are already sanitized (alphanumeric + underscore)
2. **File Storage**: Learned categories are plain text JSON files
3. **Privacy**: Each user's data is in separate learned patterns
4. **Cleanup**: Use `FORGET MY LOCATION` to remove user data

## Migration from Session-Only Storage

If you were previously using session predicates only:

1. Deploy the new AIML files
2. Users will need to set their location once more
3. After that, location persists across restarts
4. Old session data is not migrated automatically
