# List Handler API Integration for Golem

## Overview

This integration allows Golem AIML bots to interact with the List Handler API for managing todo lists and items. Users can create lists, add items, mark items as complete, and delete lists/items through natural language AIML patterns. Each user session maintains its own lists independently.

## What is List Handler?

**List Handler** is a REST API for managing todo lists and items with the following features:
- User authentication with JWT tokens
- Per-user list management
- Todo items with completion status
- Full CRUD operations on lists and items

**GitHub**: https://github.com/helix90/list-handler

## Features

- ✅ User authentication (JWT-based)
- ✅ Per-user list management
- ✅ Create, read, update, delete lists
- ✅ Add, view, update, delete items
- ✅ Toggle item completion status
- ✅ Session-based authentication storage
- ✅ Natural language interface
- ✅ Formatted list/item display

## Prerequisites

### 1. Install and Run List Handler

```bash
# Clone the repository
git clone https://github.com/helix90/list-handler.git
cd list-handler

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Run the server
uvicorn main:app --reload --port 8088
```

Default URL: `http://localhost:8088`

### 2. Verify Installation

```bash
# Check health endpoint
curl http://localhost:8088/health
# Should return: {"status": "healthy"}

# View API documentation
open http://localhost:8088/docs
```

## Setup

### 1. Environment Variables

Set the following environment variables:

```bash
export LIST_HANDLER_URL="http://localhost:8088"
export LIST_HANDLER_USERNAME="your-default-username"  # Optional
export LIST_HANDLER_PASSWORD="your-default-password"  # Optional
```

For Docker deployments, add these to your `.env` file or docker-compose.yml.

### 2. Register a User

Before using the list handler, users need to register:

```bash
curl -X POST http://localhost:8088/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"testpass"}'
```

### 3. Load Configuration

The List Handler configuration is stored in `testdata/list-handler-config.properties`. Load it when initializing Golem:

```go
package main

import (
	"github.com/helix90/golem/pkg/golem"
)

func main() {
	g := golem.New(true)

	// Load List Handler configuration
	kb := golem.NewAIMLKnowledgeBase()
	g.LoadPropertiesFromFile("testdata/list-handler-config.properties", kb)

	// Load List Handler AIML templates
	g.LoadAIMLFile("testdata/list-handler-examples.aiml", kb)

	g.SetKnowledgeBase(kb)

	// ... rest of your code
}
```

### 4. Verify Configuration

Test that List Handler is configured correctly:

```go
session := g.CreateSession("test")
response, _ := g.ProcessInput("is list handler enabled", session)
fmt.Println(response)
// Expected: "Yes, List Handler integration is enabled..."
```

## Usage

### Authentication

Users must login before accessing their lists:

```
User: list login testuser testpass
Bot:  Logged in successfully as testuser. You can now manage your lists.

User: list status
Bot:  You are logged in as testuser (User ID: 1)

User: list logout
Bot:  Logged out successfully.
```

### Create Lists

```
User: list create Shopping
Bot:  Created list 'Shopping' successfully (ID: 1)

User: create list Work
Bot:  Created list 'Work' successfully (ID: 2)

User: list create Groceries description Items to buy at store
Bot:  Created list 'Groceries' with description successfully (ID: 3)
```

### View Lists

```
User: show my lists
Bot:  Your lists: Shopping (ID: 1); Work (ID: 2) - Tasks; Groceries (ID: 3) - Items to buy at store

User: list show all
Bot:  Your lists: Shopping (ID: 1); Work (ID: 2) - Tasks

User: list show 1
Bot:  List details: Shopping (ID: 1)
      Items (2):
        [ ] Buy milk (ID: 1)
        [✓] Pay bills (ID: 2)
```

### Delete Lists

```
User: list delete 1
Bot:  List 1 deleted successfully.

User: delete list 2
Bot:  List 2 deleted successfully.

User: remove list 3
Bot:  List 3 deleted successfully.
```

### Add Items to Lists

```
User: list 1 add Buy milk
Bot:  Added 'Buy milk' to list 1 (Item ID: 1)

User: add Pay bills to list 1
Bot:  Added 'Pay bills' to list 1 (Item ID: 2)
```

### View Items in Lists

```
User: list 1 items
Bot:  Items in list 1: [ ] Buy milk (ID: 1); [ ] Pay bills (ID: 2)

User: show items in list 1
Bot:  Items in list 1: [ ] Buy milk (ID: 1); [✓] Pay bills (ID: 2)
```

### Toggle Item Completion

```
User: list 1 toggle item 1
Bot:  Toggled completion status for item 1 in list 1.

User: list 1 complete item 2
Bot:  Toggled completion status for item 2 in list 1.

User: list 1 mark item 3 done
Bot:  Toggled completion status for item 3 in list 1.
```

### Delete Items

```
User: list 1 delete item 1
Bot:  Deleted item 1 from list 1.

User: delete item 2 from list 1
Bot:  Deleted item 2 from list 1.
```

### Update Items

```
User: list 1 update item 1 to Buy organic milk
Bot:  Updated item 1 in list 1 to 'Buy organic milk'.
```

### Get Help

```
User: list help
Bot:  List Handler Commands:

      Authentication:
      - 'list login username password' - Login to list handler
      - 'list logout' - Logout
      - 'list status' - Check login status

      List Management:
      - 'list create [name]' - Create a new list
      - 'list show all' / 'show my lists' - Show all your lists
      ...
```

## Architecture

### SRAIX Configuration

The integration uses Golem's SRAIX (external service integration) system. Nine SRAIX services are configured:

1. **list_login** - Authenticate user and get JWT token (POST, form-urlencoded)
2. **list_get_all** - Get all lists for user (GET)
3. **list_create** - Create new list (POST, JSON)
4. **list_get** - Get specific list with items (GET)
5. **list_delete** - Delete list (DELETE)
6. **list_item_add** - Add item to list (POST, JSON)
7. **list_item_get** - Get all items in list (GET)
8. **list_item_delete** - Delete item from list (DELETE)
9. **list_item_toggle** - Toggle item completion (PATCH)
10. **list_item_update** - Update item content (PUT, JSON)

**Content-Type Handling**: The SRAIX system intelligently handles different Content-Type headers:
- **Form-urlencoded** (`application/x-www-form-urlencoded`): When this Content-Type is configured in headers, the SRAIX input is sent directly as the request body without JSON wrapping. This is essential for authentication endpoints that expect form data.
- **JSON** (default): For all other Content-Types (or when not specified), the SRAIX input is wrapped in a JSON object as `{"input": "..."}` and sent with `Content-Type: application/json`.

This allows seamless integration with both traditional form-based APIs (like FastAPI's OAuth2 password flow) and modern JSON APIs.

### ListHandlerHelper

The `ListHandlerHelper` class provides utilities for:

- Building request payloads (JSON and form-urlencoded)
- Parsing JSON responses (lists, items, login)
- Formatting display output
- Managing session state (user_id, token, username)
- URL construction for API endpoints
- Authentication header building

### AIML Templates

The `list-handler-examples.aiml` file contains patterns for natural language list interactions, using:

- `<sraix service="list_*">` tags to call List Handler services
- `<think>` and `<set>` for session variable management
- `<condition>` for authentication checking and response handling
- `<srai>` for pattern reuse and aliases

### Session State

Each user session stores:
- `list_user_id` - User's ID in the List Handler system
- `list_access_token` - JWT Bearer token for authentication
- `list_username` - Username for the List Handler system

This enables per-user list management where different users maintain separate lists.

## API Reference

### ListHandlerHelper Methods

```go
// Session management
GetSessionUserID(session *ChatSession) string
GetSessionToken(session *ChatSession) string
GetSessionUsername(session *ChatSession) string
IsSessionAuthenticated(session *ChatSession) bool

// Build request payloads
BuildLoginPayload(username, password string) string
BuildCreateListPayload(name, description string) (string, error)
BuildCreateItemPayload(content string, isCompleted int) (string, error)
BuildUpdateItemPayload(content string, isCompleted int) (string, error)

// Parse responses
ParseLoginResponse(jsonData string) (*LoginResponse, error)
ParseList(jsonData string) (*List, error)
ParseLists(jsonData string) ([]List, error)
ParseItem(jsonData string) (*Item, error)
ParseItems(jsonData string) ([]Item, error)

// Format output
FormatListsSummary(lists []List) string
FormatItemsSummary(items []Item) string
FormatListDetails(list *List) string

// URL construction
GetBaseURL() string
BuildListURL(userID string, listID string) string
BuildItemURL(userID string, listID string, itemID string) string
BuildAuthHeader(token string) string

// Environment defaults
GetDefaultUsername() string
GetDefaultPassword() string
```

### Data Structures

```go
type LoginResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
}

type List struct {
    ID          int    `json:"id"`
    UserID      int    `json:"user_id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at,omitempty"`
    Items       []Item `json:"items,omitempty"`
}

type Item struct {
    ID          int    `json:"id"`
    ListID      int    `json:"list_id"`
    Content     string `json:"content"`
    IsCompleted int    `json:"is_completed"`  // 0 = incomplete, 1 = complete
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at,omitempty"`
}
```

## Configuration Properties

All List Handler services use the `sraix.{service}.{property}` format:

| Property | Description | Example |
|----------|-------------|---------|
| `urltemplate` | URL with placeholders | `${LIST_HANDLER_URL}/users/{user_id}/lists` |
| `method` | HTTP method | `POST`, `GET`, `PUT`, `DELETE`, `PATCH` |
| `timeout` | Request timeout (seconds) | `30` |
| `responseformat` | Response format | `json`, `text` |
| `responsepath` | JSON path to extract | `access_token`, `id` |
| `fallback` | Fallback response on error | `"Unable to retrieve lists"` |
| `header.*` | Custom HTTP headers | `header.Content-Type`, `header.Authorization` |

### Environment Variable Substitution

Properties support `${ENV_VAR}` syntax for environment variables:

- `${LIST_HANDLER_URL}` - Server URL (default: http://localhost:8088)
- `${LIST_HANDLER_USERNAME}` - Default username
- `${LIST_HANDLER_PASSWORD}` - Default password

### URL Template Placeholders

- `{user_id}` - User ID from session variable `list_user_id`
- `{list_id}` - List ID from input
- `{item_id}` - Item ID from input
- `{access_token}` - JWT token from session variable `list_access_token`

## REST API Protocol Details

### Authentication (POST /auth/login)

**Important**: FastAPI (and many Python web frameworks) require `application/x-www-form-urlencoded` for OAuth2 password flow authentication. The configuration automatically handles this.

**Request** (Form-Urlencoded):
```
POST /auth/login
Content-Type: application/x-www-form-urlencoded

username=testuser&password=testpass
```

**Response** (JSON):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer"
}
```

**Note**: The SRAIX configuration correctly sets `Content-Type: application/x-www-form-urlencoded` for the login endpoint. The AIML template provides credentials in form-urlencoded format (`username=X&password=Y`), which is sent directly as the request body without JSON wrapping.

### Create List (POST /users/{userId}/lists)

**Request**:
```
POST /users/1/lists
Authorization: Bearer <token>
Content-Type: application/json

{"name": "Shopping", "description": "Grocery items"}
```

**Response**:
```json
{
  "id": 1,
  "user_id": 1,
  "name": "Shopping",
  "description": "Grocery items",
  "created_at": "2024-01-01T00:00:00"
}
```

### Add Item (POST /users/{userId}/lists/{listId}/items)

**Request**:
```
POST /users/1/lists/1/items
Authorization: Bearer <token>
Content-Type: application/json

{"content": "Buy milk", "is_completed": 0}
```

**Response**:
```json
{
  "id": 1,
  "list_id": 1,
  "content": "Buy milk",
  "is_completed": 0,
  "created_at": "2024-01-01T00:00:00"
}
```

## Testing

Run the List Handler helper tests:

```bash
# Run all List Handler tests
go test ./pkg/golem -run TestListHandlerHelper -v

# Run specific test
go test ./pkg/golem -run TestListHandlerHelper_ParseList -v

# Run with coverage
go test ./pkg/golem -run TestListHandlerHelper -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

### Connection Issues

**Problem**: Cannot connect to List Handler server

**Solutions**:
- Verify `LIST_HANDLER_URL` is correct
- Check List Handler is running: `curl http://localhost:8088/health`
- Verify firewall settings
- Check server logs for errors

### Authentication Errors

**Problem**: 401 Unauthorized responses

**Solutions**:
- Verify username and password are correct
- Check if user is registered
- Test credentials manually: `curl -X POST http://localhost:8088/auth/login -d "username=X&password=Y"`
- Ensure token is being stored in session

### Permission Errors

**Problem**: 403 Forbidden when accessing lists

**Solutions**:
- Verify `list_user_id` in session matches authenticated user
- Check that lists belong to the authenticated user
- Ensure token is valid (not expired)

### Empty Lists

**Problem**: "No lists found" when lists exist

**Solutions**:
- Verify user is logged in: check `list_status`
- Verify correct user_id is being used
- Check that lists were created for this user
- Enable verbose logging to see API responses

### Login Issues

**Problem**: Login fails or doesn't store credentials

**Solutions**:
- Verify user is registered in List Handler
- Check that AIML templates are storing session variables
- Manually check session variables after login
- Verify the login SRAIX service configuration

### Debugging

Enable verbose logging in Golem:

```go
g := golem.New(true) // Enable verbose mode
```

This will log:
- SRAIX requests and responses
- HTTP status codes
- Session variable changes
- Error messages

## Security Considerations

1. **HTTPS**: Use HTTPS in production (`https://` in `LIST_HANDLER_URL`)
2. **Password Security**: Never hardcode passwords; use environment variables
3. **Token Security**: Tokens are stored in session; ensure session security
4. **Input Validation**: Validate list/item names before creating
5. **Access Control**: List Handler enforces user-based access control
6. **Token Expiration**: Implement token refresh if needed for long sessions

## Advanced Usage

### Programmatic Usage

Use the helper directly in your Go code:

```go
helper := golem.NewListHandlerHelper()
session := g.CreateSession("user123")

// Check if authenticated
if !helper.IsSessionAuthenticated(session) {
    // Handle login
}

// Build request payloads
payload, _ := helper.BuildCreateListPayload("Shopping", "Groceries")

// Parse responses
lists, _ := helper.ParseLists(responseJSON)
formatted := helper.FormatListsSummary(lists)
```

### Custom Formatting

Override formatting methods for custom display:

```go
helper := golem.NewListHandlerHelper()
items := []golem.Item{...}

// Use helper formatting
formatted := helper.FormatItemsSummary(items)

// Or create custom formatting
for _, item := range items {
    // Custom logic
}
```

### Multiple Users

Each session maintains separate lists:

```go
session1 := g.CreateSession("user1")
session2 := g.CreateSession("user2")

// User 1 creates lists
g.ProcessInput("list login user1 pass1", session1)
g.ProcessInput("list create Shopping", session1)

// User 2 has separate lists
g.ProcessInput("list login user2 pass2", session2)
g.ProcessInput("list create Work", session2)

// Lists are completely independent per user
```

## References

- [List Handler GitHub](https://github.com/helix90/list-handler)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)
- [JWT Authentication](https://jwt.io/)
- [REST API Best Practices](https://restfulapi.net/)

## Support

For issues or questions:
1. Check this documentation
2. Review test files for examples
3. Enable verbose logging for debugging
4. Check List Handler server logs
5. Report issues on GitHub

## License

This List Handler integration is part of the Golem project and follows the same MIT license.
