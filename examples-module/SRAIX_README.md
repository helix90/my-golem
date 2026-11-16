# SRAIX (Substitute, Resubstitute, and Input eXternal) Support

SRAIX allows your AIML bot to communicate with external HTTP/HTTPS services, enabling dynamic responses based on real-time data from APIs, other bots, or web services.

## Overview

SRAIX extends the concept of SRAI (Substitute, Resubstitute, and Input) by allowing your bot to send input to external services and process their responses. This enables your AIML bot to:

- Get real-time weather information
- Translate text using external translation services
- Query AI services like ChatGPT or Claude
- Access any HTTP/HTTPS API
- Integrate with other chatbots or services

## Basic Usage

### AIML Template Syntax

```xml
<sraix service="service_name">Input to send to external service</sraix>
```

### Example

```xml
<category>
  <pattern>WHAT IS THE WEATHER IN *</pattern>
  <template>
    <sraix service="weather_service">What is the weather in <star/>?</sraix>
  </template>
</category>
```

## Tag Attributes

- **`service`** - Name of the configured SRAIX service
- **`bot`** - Bot identifier for bot-to-bot calls (used as the service name when `service` is absent)
- **`botid`** - Specific bot ID, sent in the request body
- **`host`** - Host specification, sent in the request body
- **`default`** - Response to use when the service is missing or the request fails
- **`hint`** - Context text for the external service, sent in the request body

Attribute values support variable and wildcard processing:

```xml
<sraix service="translator"
       hint="Translate to <get name="target_language"/>"
       default="I can't translate that">Translate: <star/></sraix>
```

```xml
<sraix service="mathbot"
       bot="calculator"
       botid="calc123"
       host="math.example.com"
       default="I can't calculate that"
       hint="Mathematical calculation">What is 5*6?</sraix>
```

Multiple SRAIX tags can appear in a single template.

## Configuration

### From Properties Files (Recommended)

Services can be configured in any `.properties` file using keys of the form `sraix.<servicename>.<property>`. They are configured automatically when the knowledge base is loaded:

```json
[
  ["sraix.weather.urltemplate", "https://api.pirateweather.net/forecast/${PIRATE_WEATHER_API_KEY}/{lat},{lon}"],
  ["sraix.weather.method", "GET"],
  ["sraix.weather.responseformat", "json"],
  ["sraix.weather.responsepath", "currently.summary"],
  ["sraix.weather.fallback", "Weather service unavailable"]
]
```

Available properties: `baseurl`, `urltemplate`, `queryparam`, `method`, `timeout`, `responseformat`, `responsepath`, `fallback`, `includewildcards`, `apikey`, `header.<HeaderName>`.

URL template placeholders:
- `${ENV_VAR}` - Environment variables
- `{input}` - The SRAIX input text
- `{apikey}` - API key
- `{lat}`, `{lon}` - From session variables `latitude`/`longitude`
- `{location}` - From session variable `location`
- `{STAR1}`, `{STAR2}`, ... - Pattern wildcard values

Placeholders are replaced in a single pass, and every value except `${ENV_VAR}` and `{apikey}` is URL-encoded, so user input cannot change the URL. Verbose logs mask environment variables and the API key.

See `testdata/WEATHER_README.md` for a complete working example and `secure-sraix-demo/SECURE_API_KEYS.md` for handling API keys.

### From JSON Configuration Files

```json
[
  {
    "name": "weather_service",
    "base_url": "https://api.weather.com/v1/current",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer YOUR_API_KEY",
      "Content-Type": "application/json"
    },
    "timeout": 10,
    "response_format": "json",
    "response_path": "data.conditions",
    "fallback_response": "Sorry, I couldn't get the weather information right now.",
    "include_wildcards": true
  }
]
```

```go
g := golem.New(true)
err := g.LoadSRAIXConfigsFromFile("sraix_config.json")
// or
err = g.LoadSRAIXConfigsFromDirectory("./configs")
```

### Programmatically

```go
config := &golem.SRAIXConfig{
    Name:           "my_service",
    BaseURL:        "https://api.example.com",
    Method:         "POST",
    ResponseFormat: "json",
    ResponsePath:   "data.message",
}
err := g.AddSRAIXConfig(config)
```

### Configuration Fields

- `name` (required): Unique identifier for the service
- `base_url` (required unless `url_template` is set): The HTTP/HTTPS endpoint URL
- `url_template`: URL with placeholders; overrides `base_url` and query parameter handling
- `query_param`: Query parameter name for GET requests (default: "input")
- `method`: HTTP method (default: "POST")
- `headers`: Custom headers to include in requests
- `timeout`: Request timeout in seconds (default: 30)
- `response_format`: "json", "xml", or "text" (default: "text")
- `response_path`: JSON path to extract specific data (e.g., "data.message")
- `fallback_response`: Response when the service returns an error
- `include_wildcards`: Whether to include wildcard data in requests (default: false)

## Request Format

### GET Requests

The input is sent as a query parameter:

```
https://api.example.com/echo?input=What+is+the+weather+in+New+York%3F
```

### POST Requests

The input is sent as JSON, along with wildcards (when `include_wildcards` is true) and any `botid`, `host`, and `hint` attributes:

```json
{
  "input": "Weather in New York and London",
  "wildcards": {
    "star1": "New York",
    "star2": "London"
  },
  "botid": "12345",
  "host": "api.example.com",
  "hint": "This is a test question"
}
```

The request body depends on the configured `Content-Type` header:

- **`application/x-www-form-urlencoded`**: the input is sent directly as the body, not wrapped in JSON. Use this for OAuth2 password flows and similar login endpoints, e.g. `<sraix service="login">username=user&amp;password=pass</sraix>`.
- **`application/json`**: input that is already valid JSON (an object or array) is sent as-is; anything else is wrapped as above.
- **Not set**: the input is wrapped as above.

Form-urlencoded (`key=value&key2=value2`) and JSON object input also supply their keys as URL template placeholders, e.g. `{list_id}`. When the input is a single plain value and the URL template has exactly one placeholder, the input fills that placeholder.

Placeholders in header values are substituted too (without URL escaping), e.g. `"Authorization": "Bearer {access_token}"`.

## Response Processing

- **json**: The response is parsed and optionally filtered using `response_path`. With `response_path: "data.message"`, `{"data": {"message": "Sunny and 75°F"}}` yields "Sunny and 75°F".
- **text**: The raw response body is used directly.
- **xml**: The raw XML response is used.

## Error Handling

1. **HTTP errors / timeouts**: The service's `fallback_response` is used if configured
2. **Otherwise, on any failure** (service not configured, network error, no `service` or `bot` attribute): the tag's `default` attribute is used if provided
3. **No default**: The tag's evaluated inner text is returned

## Integration with Other AIML Tags

```xml
<category>
  <pattern>COMPLEX REQUEST *</pattern>
  <template>
    <think>
      <set name="user_request"><star/></set>
    </think>
    I'll help you with that. <sraix service="ai_service"><get name="user_request"/></sraix>
  </template>
</category>
```

## Example Services

### Weather Service
```json
{
  "name": "weather",
  "base_url": "https://api.openweathermap.org/data/2.5/weather",
  "method": "GET",
  "response_format": "json",
  "response_path": "weather.0.description"
}
```

### Translation Service
```json
{
  "name": "translate",
  "base_url": "https://api.mymemory.translated.net/get",
  "method": "GET",
  "response_format": "json",
  "response_path": "responseData.translatedText"
}
```

### AI Chat Service
```json
{
  "name": "ai_chat",
  "base_url": "https://api.openai.com/v1/chat/completions",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer YOUR_API_KEY",
    "Content-Type": "application/json"
  },
  "response_format": "json",
  "response_path": "choices.0.message.content"
}
```

## Security and Best Practices

1. **API Keys**: Use environment variables (`${ENV_VAR}`) and never commit keys to version control
2. **Defaults**: Always provide a `default` attribute or `fallback_response` for graceful degradation
3. **HTTPS**: Always use HTTPS for external service communication
4. **Timeouts**: Set timeouts based on each service's expected response time
5. **Rate Limiting**: Be aware of API rate limits
6. **Input Validation**: Validate external responses and sanitize user input sent to services

## Testing

```bash
go test ./pkg/golem -v -run TestSRAIX
go test ./pkg/golem -v -run TestEnhancedSRAIX
```

## Troubleshooting

1. **Service Not Found**: Check that the service name in the SRAIX tag matches the configuration
2. **Authentication Errors**: Verify API keys and headers are correct
3. **Timeout Errors**: Increase the timeout value or check network connectivity
4. **JSON Path Errors**: Verify `response_path` matches the actual JSON structure

Enable verbose logging to see SRAIX request details, responses, and errors:

```go
g := golem.New(true)
```
