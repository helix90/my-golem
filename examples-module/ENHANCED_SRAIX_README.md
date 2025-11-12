# Enhanced SRAIX Support

This document describes the enhanced SRAIX (Substitute, Resubstitute, and Input eXternal) functionality in Golem, which provides comprehensive support for external service integration with advanced attributes.

## Overview

SRAIX allows AIML bots to make external HTTP requests to web services, APIs, or other bots to enhance their capabilities. The enhanced implementation supports all major SRAIX attributes as specified in AIML2.

## Supported Attributes

### Core Attributes
- **`service`** - Name of the configured SRAIX service
- **`bot`** - Bot identifier for external bot calls
- **`botid`** - Specific bot ID for authentication
- **`host`** - Host specification for the external service
- **`default`** - Fallback response when the service fails
- **`hint`** - Hint text to provide context to the external service

## Basic Usage

### Simple Service Call
```xml
<sraix service="weather">What's the weather like?</sraix>
```

### Bot-to-Bot Communication
```xml
<sraix bot="alice" botid="alice123" host="api.example.com">Hello, how are you?</sraix>
```

### With Fallback Response
```xml
<sraix service="calculator" default="I can't calculate that">What is 2+2?</sraix>
```

### With Hint Text
```xml
<sraix service="translator" hint="Translate to Spanish">Hello world</sraix>
```

### Complete Example
```xml
<sraix service="mathbot" 
       bot="calculator" 
       botid="calc123" 
       host="math.example.com" 
       default="I can't calculate that" 
       hint="Mathematical calculation">What is 5*6?</sraix>
```

## Configuration

### Adding SRAIX Services

```go
// Create a new SRAIX configuration
config := &SRAIXConfig{
    Name:             "weather",
    BaseURL:          "https://api.weather.com/v1/query",
    Method:           "POST",
    Headers:          map[string]string{"Authorization": "Bearer token123"},
    Timeout:          30,
    ResponseFormat:   "json",
    ResponsePath:     "data.forecast",
    FallbackResponse: "Weather service unavailable",
    IncludeWildcards: true,
}

// Add the configuration
golem.AddSRAIXConfig(config)
```

### Configuration Options

- **`Name`** - Unique identifier for the service
- **`BaseURL`** - Base URL for the service endpoint
- **`Method`** - HTTP method (GET, POST, PUT, etc.)
- **`Headers`** - Custom HTTP headers
- **`Timeout`** - Request timeout in seconds
- **`ResponseFormat`** - Expected response format (json, xml, text)
- **`ResponsePath`** - JSONPath to extract response data
- **`FallbackResponse`** - Default response on failure
- **`IncludeWildcards`** - Include wildcard data in requests

## Advanced Features

### Variable Processing
SRAIX content, default responses, and hint text all support variable processing:

```xml
<sraix service="greeting" default="Hello <get name="name"/>, I can't help">Hello <get name="name"/></sraix>
```

### Wildcard Support
Wildcards are automatically processed in SRAIX content:

```xml
<sraix service="search" default="I don't know about <star/>">Tell me about <star/></sraix>
```

### Multiple SRAIX Tags
You can use multiple SRAIX tags in a single template:

```xml
<sraix service="service1" default="Response 1">Question 1</sraix> and <sraix service="service2" default="Response 2">Question 2</sraix>
```

### Content-Type Handling

SRAIX intelligently handles different Content-Type headers for POST/PUT requests:

**Form-Urlencoded Requests** (OAuth2, login endpoints):
```go
config := &SRAIXConfig{
    Name:    "login",
    URLTemplate: "https://api.example.com/auth/login",
    Method:  "POST",
    Headers: map[string]string{
        "Content-Type": "application/x-www-form-urlencoded",
    },
    ResponseFormat: "json",
    ResponsePath:   "access_token",
}
```

When `Content-Type: application/x-www-form-urlencoded` is set in headers:
- The SRAIX input is sent **directly** as the request body (not wrapped in JSON)
- This is essential for FastAPI OAuth2 password flow and similar endpoints
- Example AIML: `<sraix service="login">username=user&password=pass</sraix>`

**JSON Requests** (default):
```go
config := &SRAIXConfig{
    Name:    "chat",
    BaseURL: "https://api.example.com/chat",
    Method:  "POST",
    // No Content-Type header, or set to "application/json"
    ResponseFormat: "json",
}
```

When Content-Type is not specified or set to `application/json`:
- The SRAIX input is wrapped in a JSON object: `{"input": "your text here"}`
- Additional fields (wildcards, botid, host, hint) are included if configured
- This is the default behavior for modern REST APIs

## Error Handling

### Service Not Found
When a service is not configured, SRAIX will:
1. Use the `default` response if provided
2. Leave the tag unchanged if no default is provided

### Network Errors
When a network error occurs, SRAIX will:
1. Use the `default` response if provided
2. Use the service's `FallbackResponse` if configured
3. Leave the tag unchanged if no fallback is available

### Missing Attributes
When neither `service` nor `bot` is specified:
1. Use the `default` response if provided
2. Leave the tag unchanged if no default is provided

## Request Format

### GET Requests
For GET requests, the input is appended as a query parameter:
```
https://api.example.com/query?input=Hello+world
```

### POST/PUT Requests
For POST/PUT requests, the input is sent as JSON:
```json
{
  "input": "Hello world",
  "wildcards": {"star": "cats"},
  "botid": "12345",
  "host": "api.example.com",
  "hint": "This is a test question"
}
```

## Testing

The enhanced SRAIX implementation includes comprehensive tests:

```bash
# Run all SRAIX tests
go test ./pkg/golem -v -run TestEnhancedSRAIX

# Run specific test categories
go test ./pkg/golem -v -run TestEnhancedSRAIXTags
go test ./pkg/golem -v -run TestSRAIXAttributeParsing
go test ./pkg/golem -v -run TestEnhancedSRAIXErrorHandling
```

## Examples

### Weather Service
```xml
<sraix service="weather" default="I can't check the weather right now">What's the weather in <star/>?</sraix>
```

### Translation Service
```xml
<sraix service="translator" hint="Translate to <get name="target_language"/>" default="I can't translate that">Translate: <star/></sraix>
```

### Math Calculator
```xml
<sraix service="calculator" bot="mathbot" botid="calc123" default="I can't calculate that">Calculate: <star/></sraix>
```

### Knowledge Base Query
```xml
<sraix service="knowledge" host="kb.example.com" hint="User <get name="name"/> is asking about <star/>">What is <star/>?</sraix>
```

## Best Practices

1. **Always provide a default response** for better user experience
2. **Use hint text** to provide context to external services
3. **Configure appropriate timeouts** based on service response times
4. **Include wildcard data** when it might be useful for the external service
5. **Test error scenarios** to ensure graceful degradation
6. **Use meaningful service names** for easier configuration management

## Compatibility

The enhanced SRAIX implementation is fully compatible with:
- AIML2 specification
- Pandorabots SRAIX format
- Custom service integrations
- Bot-to-bot communication protocols

## Performance Considerations

- SRAIX requests are made asynchronously when possible
- Response caching can be implemented at the service level
- Timeout settings should be configured based on service requirements
- Consider rate limiting for high-volume applications

## Security Notes

- Always validate external service responses
- Use HTTPS for sensitive data transmission
- Implement proper authentication mechanisms
- Sanitize user input before sending to external services
- Consider implementing request signing for critical operations
