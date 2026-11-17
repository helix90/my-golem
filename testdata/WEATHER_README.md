# Weather Integration with Pirate Weather API

This example demonstrates how to integrate weather services using the Pirate Weather API with Golem's SRAIX functionality.

## Features

- **Query weather by location name**: Automatic geocoding converts city names to coordinates
- **Set default location**: Save your location for quick weather queries
- **Secure API key management**: Use environment variables to protect your API key
- **Free geocoding**: Uses OpenStreetMap Nominatim (no API key needed)

## Setup

### 1. Get a Pirate Weather API Key

1. Visit [https://pirateweather.net](https://pirateweather.net)
2. Sign up for a free account
3. Copy your API key

### 2. Set Environment Variable

**Linux/macOS:**
```bash
export PIRATE_WEATHER_API_KEY="your-api-key-here"
```

**Windows (PowerShell):**
```powershell
$env:PIRATE_WEATHER_API_KEY="your-api-key-here"
```

**Permanent setup (Linux/macOS):**
Add to your `~/.bashrc` or `~/.zshrc`:
```bash
export PIRATE_WEATHER_API_KEY="your-api-key-here"
```

### 3. Load Configuration and AIML

```go
package main

import (
    "fmt"
    "os"
    "github.com/helix90/golem/pkg/golem"
)

func main() {
    // Create Golem instance
    g := golem.New(true)

    // Load weather AIML and configuration
    kb, err := g.LoadAIMLFromFile("testdata/weather.aiml")
    if err != nil {
        fmt.Printf("Error loading AIML: %v\n", err)
        return
    }

    // Load weather properties (with environment variable substitution)
    props, err := golem.LoadPropertiesFile("testdata/weather-config.properties")
    if err != nil {
        fmt.Printf("Error loading properties: %v\n", err)
        return
    }

    // Substitute environment variables in properties
    for key, value := range props {
        if strings.Contains(value, "${") {
            // Simple environment variable substitution
            value = os.ExpandEnv(value)
            props[key] = value
        }
    }

    kb.Properties = props
    g.SetKnowledgeBase(kb)

    // Create session
    session := g.CreateSession("weather-user")

    // Set user location
    response, _ := g.ProcessInput("my location is San Francisco", session)
    fmt.Println(response)

    // Query weather
    response, _ = g.ProcessInput("what is the weather", session)
    fmt.Println(response)

    // Query weather for specific location
    response, _ = g.ProcessInput("what is the weather in Boston", session)
    fmt.Println(response)
}
```

## Usage Examples

### Set Your Location

```
User: My location is Portland Oregon
Bot: I've set your location to Portland Oregon (coordinates: 45.5202, -122.6742).

User: Set location New York
Bot: I've set your location to New York (coordinates: 40.7128, -74.0060).
```

### Query Weather (Default Location)

```
User: What is the weather?
Bot: The weather in Portland Oregon is currently: Partly cloudy with a temperature of 18°C

User: Weather
Bot: The weather in Portland Oregon is currently: Clear skies with a temperature of 22°C
```

### Query Weather (Specific Location)

```
User: What is the weather in Tokyo?
Bot: The weather in Tokyo is currently: Rainy with a temperature of 20°C

User: Weather in London
Bot: The weather in London is currently: Cloudy with a temperature of 15°C
```

### Location Management

```
User: What is my location?
Bot: Your location is set to Portland Oregon (coordinates: 45.5202, -122.6742).

User: Clear my location
Bot: Your location has been cleared.
```

## How It Works

### 1. URL Templates

The Pirate Weather API requires coordinates in the URL path:
```
https://api.pirateweather.net/forecast/{apikey}/{lat},{lon}?units=si
```

The SRAIX configuration (`weather-config.properties`) uses a URL template with placeholders:
```json
["sraix.weather.urltemplate", "https://api.pirateweather.net/forecast/${PIRATE_WEATHER_API_KEY}/{lat},{lon}?units=si"]
```

### 2. Coordinate Passing

Coordinates are passed in the SRAIX input as a JSON object, whose fields fill the placeholders:
```xml
<sraix service="weather">{"lat":"<get name="latitude"/>","lon":"<get name="longitude"/>"}</sraix>
```

A query for another place passes that place's coordinates, so the user's own location is left alone.

### 3. Geocoding

Location names are converted to coordinates using OpenStreetMap Nominatim. One request gives both: `geocode` returns the latitude and sets the `geocode_lon` predicate to the longitude (`["sraix.geocode.set.geocode_lon", "0.lon"]`):
```xml
<set var="lat"><sraix service="geocode">San Francisco</sraix></set>
<set var="lon"><get name="geocode_lon"/></set>
```

### 4. Session Predicates

User location is stored in session predicates:
```xml
<set name="location">San Francisco</set>
<set name="latitude">37.7749</set>
<set name="longitude">-122.4194</set>
```

## Security Best Practices

### DO:
✅ Use environment variables for API keys
✅ Add `.env` files to `.gitignore`
✅ Use placeholder values in committed configuration files
✅ Rotate API keys periodically
✅ Use separate keys for development and production

### DON'T:
❌ Commit API keys to version control
❌ Share API keys in plain text
❌ Use production keys in development
❌ Hard-code API keys in source code
❌ Include API keys in error messages or logs

## Environment Variable Substitution

The configuration file uses `${VARIABLE_NAME}` syntax:
```properties
sraix.weather.apikey=${PIRATE_WEATHER_API_KEY}
```

You can implement environment variable substitution in your code:
```go
import "os"

value := "${PIRATE_WEATHER_API_KEY}"
value = os.ExpandEnv(value)  // Expands to actual API key
```

## API Rate Limits

Pirate Weather free tier:
- 20,000 API calls per month
- ~666 calls per day
- Implement caching for production use

## Troubleshooting

### "Weather information is currently unavailable"
- Check your API key is set correctly
- Verify the API key is valid on pirateweather.net
- Check your internet connection
- Verify you haven't exceeded rate limits

### "Unable to find that location"
- Try more specific location names (e.g., "Portland Oregon" instead of "Portland")
- Include country names for international locations
- Use major cities or full addresses
- Check spelling of location name

### "You haven't set your location yet"
- Use "my location is [city]" to set your default location
- Or query with specific location: "weather in [city]"

## Alternative: OpenWeatherMap

To use OpenWeatherMap instead:

1. Sign up at [https://openweathermap.org/api](https://openweathermap.org/api)
2. Update `weather-config.properties`:
```properties
sraix.weather.urltemplate=https://api.openweathermap.org/data/2.5/weather?q={location}&appid={apikey}&units=metric
sraix.weather.apikey=${OPENWEATHER_API_KEY}
sraix.weather.responsepath=weather.0.description
```

## See Also

- [Pirate Weather API Documentation](https://pirateweather.net/en/latest/)
- [OpenStreetMap Nominatim](https://nominatim.org/)
- [AIML SRAIX Documentation](../../docs/sraix.md)
- [Environment Variables Guide](../../docs/environment.md)
