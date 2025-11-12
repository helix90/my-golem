package golem

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ListHandlerHelper provides utilities for List Handler API operations
type ListHandlerHelper struct{}

// NewListHandlerHelper creates a new List Handler helper
func NewListHandlerHelper() *ListHandlerHelper {
	return &ListHandlerHelper{}
}

// LoginRequest represents the login request structure
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response from the API
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// List represents a list object
type List struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Items       []Item `json:"items,omitempty"`
}

// Item represents a list item
type Item struct {
	ID          int    `json:"id"`
	ListID      int    `json:"list_id"`
	Content     string `json:"content"`
	IsCompleted int    `json:"is_completed"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// CreateListRequest represents request to create a new list
type CreateListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateItemRequest represents request to create a new item
type CreateItemRequest struct {
	Content     string `json:"content"`
	IsCompleted int    `json:"is_completed"`
}

// GetSessionUserID gets the user ID from session
func (lh *ListHandlerHelper) GetSessionUserID(session *ChatSession) string {
	if session == nil {
		return ""
	}
	if userID, exists := session.Variables["list_user_id"]; exists && userID != "" {
		return userID
	}
	return ""
}

// GetSessionToken gets the access token from session
func (lh *ListHandlerHelper) GetSessionToken(session *ChatSession) string {
	if session == nil {
		return ""
	}
	if token, exists := session.Variables["list_access_token"]; exists && token != "" {
		return token
	}
	return ""
}

// GetSessionUsername gets the username from session
func (lh *ListHandlerHelper) GetSessionUsername(session *ChatSession) string {
	if session == nil {
		return ""
	}
	if username, exists := session.Variables["list_username"]; exists && username != "" {
		return username
	}
	return ""
}

// IsSessionAuthenticated checks if the session has valid authentication
func (lh *ListHandlerHelper) IsSessionAuthenticated(session *ChatSession) bool {
	return lh.GetSessionToken(session) != "" && lh.GetSessionUserID(session) != ""
}

// BuildLoginPayload creates form-urlencoded login payload
// FastAPI OAuth2PasswordRequestForm requires grant_type field
func (lh *ListHandlerHelper) BuildLoginPayload(username, password string) string {
	return fmt.Sprintf("username=%s&password=%s&grant_type=password",
		url.QueryEscape(username),
		url.QueryEscape(password))
}

// BuildCreateListPayload creates JSON payload for creating a list
func (lh *ListHandlerHelper) BuildCreateListPayload(name, description string) (string, error) {
	req := CreateListRequest{
		Name:        name,
		Description: description,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal create list request: %w", err)
	}
	return string(data), nil
}

// BuildCreateItemPayload creates JSON payload for creating an item
func (lh *ListHandlerHelper) BuildCreateItemPayload(content string, isCompleted int) (string, error) {
	req := CreateItemRequest{
		Content:     content,
		IsCompleted: isCompleted,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal create item request: %w", err)
	}
	return string(data), nil
}

// BuildUpdateItemPayload creates JSON payload for updating an item
func (lh *ListHandlerHelper) BuildUpdateItemPayload(content string, isCompleted int) (string, error) {
	req := CreateItemRequest{
		Content:     content,
		IsCompleted: isCompleted,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal update item request: %w", err)
	}
	return string(data), nil
}

// ParseLoginResponse parses the login response JSON
func (lh *ListHandlerHelper) ParseLoginResponse(jsonData string) (*LoginResponse, error) {
	var response LoginResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}
	return &response, nil
}

// ParseList parses a single list from JSON
func (lh *ListHandlerHelper) ParseList(jsonData string) (*List, error) {
	var list List
	err := json.Unmarshal([]byte(jsonData), &list)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list: %w", err)
	}
	return &list, nil
}

// ParseLists parses an array of lists from JSON
func (lh *ListHandlerHelper) ParseLists(jsonData string) ([]List, error) {
	var lists []List
	err := json.Unmarshal([]byte(jsonData), &lists)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lists: %w", err)
	}
	return lists, nil
}

// ParseItem parses a single item from JSON
func (lh *ListHandlerHelper) ParseItem(jsonData string) (*Item, error) {
	var item Item
	err := json.Unmarshal([]byte(jsonData), &item)
	if err != nil {
		return nil, fmt.Errorf("failed to parse item: %w", err)
	}
	return &item, nil
}

// ParseItems parses an array of items from JSON
func (lh *ListHandlerHelper) ParseItems(jsonData string) ([]Item, error) {
	var items []Item
	err := json.Unmarshal([]byte(jsonData), &items)
	if err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}
	return items, nil
}

// FormatListsSummary creates a user-friendly summary of lists
func (lh *ListHandlerHelper) FormatListsSummary(lists []List) string {
	if len(lists) == 0 {
		return "No lists found"
	}

	var result strings.Builder
	for i, list := range lists {
		if i > 0 {
			result.WriteString("; ")
		}
		result.WriteString(fmt.Sprintf("%s (ID: %d)", list.Name, list.ID))
		if list.Description != "" {
			result.WriteString(fmt.Sprintf(" - %s", list.Description))
		}
	}
	return result.String()
}

// FormatItemsSummary creates a user-friendly summary of items
func (lh *ListHandlerHelper) FormatItemsSummary(items []Item) string {
	if len(items) == 0 {
		return "No items found"
	}

	var result strings.Builder
	for i, item := range items {
		if i > 0 {
			result.WriteString("; ")
		}
		status := "[ ]"
		if item.IsCompleted == 1 {
			status = "[✓]"
		}
		result.WriteString(fmt.Sprintf("%s %s (ID: %d)", status, item.Content, item.ID))
	}
	return result.String()
}

// FormatListDetails creates a detailed view of a list with items
func (lh *ListHandlerHelper) FormatListDetails(list *List) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("List: %s (ID: %d)\n", list.Name, list.ID))
	if list.Description != "" {
		result.WriteString(fmt.Sprintf("Description: %s\n", list.Description))
	}

	if len(list.Items) == 0 {
		result.WriteString("No items in this list")
	} else {
		result.WriteString(fmt.Sprintf("Items (%d):\n", len(list.Items)))
		for _, item := range list.Items {
			status := "[ ]"
			if item.IsCompleted == 1 {
				status = "[✓]"
			}
			result.WriteString(fmt.Sprintf("  %s %s (ID: %d)\n", status, item.Content, item.ID))
		}
	}
	return result.String()
}

// GetBaseURL returns the base URL from environment or default
func (lh *ListHandlerHelper) GetBaseURL() string {
	baseURL := os.Getenv("LIST_HANDLER_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8088"
	}
	// Ensure no trailing slash
	return strings.TrimSuffix(baseURL, "/")
}

// BuildListURL constructs the URL for list operations
func (lh *ListHandlerHelper) BuildListURL(userID string, listID string) string {
	baseURL := lh.GetBaseURL()
	if listID == "" {
		return fmt.Sprintf("%s/users/%s/lists", baseURL, userID)
	}
	return fmt.Sprintf("%s/users/%s/lists/%s", baseURL, userID, listID)
}

// BuildItemURL constructs the URL for item operations
func (lh *ListHandlerHelper) BuildItemURL(userID string, listID string, itemID string) string {
	baseURL := lh.GetBaseURL()
	if itemID == "" {
		return fmt.Sprintf("%s/users/%s/lists/%s/items", baseURL, userID, listID)
	}
	return fmt.Sprintf("%s/users/%s/lists/%s/items/%s", baseURL, userID, listID, itemID)
}

// BuildAuthHeader creates the Bearer token authorization header
func (lh *ListHandlerHelper) BuildAuthHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

// GetDefaultUsername returns the default username from environment
func (lh *ListHandlerHelper) GetDefaultUsername() string {
	return os.Getenv("LIST_HANDLER_USERNAME")
}

// GetDefaultPassword returns the default password from environment
func (lh *ListHandlerHelper) GetDefaultPassword() string {
	return os.Getenv("LIST_HANDLER_PASSWORD")
}
