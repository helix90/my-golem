package golem

import (
	"os"
	"strings"
	"testing"
)

func TestListHandlerHelper_BuildLoginPayload(t *testing.T) {
	lh := NewListHandlerHelper()

	payload := lh.BuildLoginPayload("testuser", "testpass")

	expected := "username=testuser&password=testpass"
	if payload != expected {
		t.Errorf("Expected %q, got %q", expected, payload)
	}
}

func TestListHandlerHelper_BuildLoginPayload_SpecialChars(t *testing.T) {
	lh := NewListHandlerHelper()

	payload := lh.BuildLoginPayload("test@user", "p@ss&word")

	// URL encoding should escape special characters
	if !strings.Contains(payload, "username=test%40user") {
		t.Error("Username special chars should be URL encoded")
	}
	if !strings.Contains(payload, "password=p%40ss%26word") {
		t.Error("Password special chars should be URL encoded")
	}
}

func TestListHandlerHelper_BuildCreateListPayload(t *testing.T) {
	lh := NewListHandlerHelper()

	payload, err := lh.BuildCreateListPayload("Shopping List", "Items to buy")
	if err != nil {
		t.Fatalf("BuildCreateListPayload failed: %v", err)
	}

	if !strings.Contains(payload, `"name":"Shopping List"`) {
		t.Error("Payload should contain list name")
	}
	if !strings.Contains(payload, `"description":"Items to buy"`) {
		t.Error("Payload should contain description")
	}
}

func TestListHandlerHelper_BuildCreateItemPayload(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name        string
		content     string
		isCompleted int
	}{
		{"Uncompleted item", "Buy milk", 0},
		{"Completed item", "Pay bills", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := lh.BuildCreateItemPayload(tt.content, tt.isCompleted)
			if err != nil {
				t.Fatalf("BuildCreateItemPayload failed: %v", err)
			}

			if !strings.Contains(payload, tt.content) {
				t.Errorf("Payload should contain content %q", tt.content)
			}
		})
	}
}

func TestListHandlerHelper_ParseLoginResponse(t *testing.T) {
	lh := NewListHandlerHelper()

	jsonData := `{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9","token_type":"bearer"}`

	response, err := lh.ParseLoginResponse(jsonData)
	if err != nil {
		t.Fatalf("ParseLoginResponse failed: %v", err)
	}

	if response.AccessToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" {
		t.Errorf("Expected access token, got %q", response.AccessToken)
	}
	if response.TokenType != "bearer" {
		t.Errorf("Expected token type bearer, got %q", response.TokenType)
	}
}

func TestListHandlerHelper_ParseList(t *testing.T) {
	lh := NewListHandlerHelper()

	jsonData := `{
		"id": 1,
		"user_id": 1,
		"name": "Shopping",
		"description": "Grocery items",
		"created_at": "2024-01-01T00:00:00"
	}`

	list, err := lh.ParseList(jsonData)
	if err != nil {
		t.Fatalf("ParseList failed: %v", err)
	}

	if list.ID != 1 {
		t.Errorf("Expected ID 1, got %d", list.ID)
	}
	if list.Name != "Shopping" {
		t.Errorf("Expected name Shopping, got %q", list.Name)
	}
	if list.Description != "Grocery items" {
		t.Errorf("Expected description, got %q", list.Description)
	}
}

func TestListHandlerHelper_ParseLists(t *testing.T) {
	lh := NewListHandlerHelper()

	jsonData := `[
		{"id": 1, "user_id": 1, "name": "Shopping", "description": "", "created_at": "2024-01-01T00:00:00"},
		{"id": 2, "user_id": 1, "name": "Work", "description": "Tasks", "created_at": "2024-01-01T00:00:00"}
	]`

	lists, err := lh.ParseLists(jsonData)
	if err != nil {
		t.Fatalf("ParseLists failed: %v", err)
	}

	if len(lists) != 2 {
		t.Fatalf("Expected 2 lists, got %d", len(lists))
	}

	if lists[0].Name != "Shopping" {
		t.Errorf("Expected first list name Shopping, got %q", lists[0].Name)
	}
	if lists[1].Name != "Work" {
		t.Errorf("Expected second list name Work, got %q", lists[1].Name)
	}
}

func TestListHandlerHelper_ParseItem(t *testing.T) {
	lh := NewListHandlerHelper()

	jsonData := `{
		"id": 1,
		"list_id": 1,
		"content": "Buy milk",
		"is_completed": 0,
		"created_at": "2024-01-01T00:00:00"
	}`

	item, err := lh.ParseItem(jsonData)
	if err != nil {
		t.Fatalf("ParseItem failed: %v", err)
	}

	if item.ID != 1 {
		t.Errorf("Expected ID 1, got %d", item.ID)
	}
	if item.Content != "Buy milk" {
		t.Errorf("Expected content 'Buy milk', got %q", item.Content)
	}
	if item.IsCompleted != 0 {
		t.Errorf("Expected is_completed 0, got %d", item.IsCompleted)
	}
}

func TestListHandlerHelper_ParseItems(t *testing.T) {
	lh := NewListHandlerHelper()

	jsonData := `[
		{"id": 1, "list_id": 1, "content": "Buy milk", "is_completed": 0, "created_at": "2024-01-01T00:00:00"},
		{"id": 2, "list_id": 1, "content": "Pay bills", "is_completed": 1, "created_at": "2024-01-01T00:00:00"}
	]`

	items, err := lh.ParseItems(jsonData)
	if err != nil {
		t.Fatalf("ParseItems failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	if items[0].Content != "Buy milk" {
		t.Errorf("Expected first item 'Buy milk', got %q", items[0].Content)
	}
	if items[1].IsCompleted != 1 {
		t.Errorf("Expected second item completed, got %d", items[1].IsCompleted)
	}
}

func TestListHandlerHelper_FormatListsSummary(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name     string
		lists    []List
		expected string
	}{
		{
			name:     "Empty list",
			lists:    []List{},
			expected: "No lists found",
		},
		{
			name: "Single list",
			lists: []List{
				{ID: 1, Name: "Shopping", Description: "Groceries"},
			},
			expected: "Shopping (ID: 1) - Groceries",
		},
		{
			name: "Multiple lists",
			lists: []List{
				{ID: 1, Name: "Shopping", Description: "Groceries"},
				{ID: 2, Name: "Work", Description: ""},
			},
			expected: "Shopping (ID: 1) - Groceries; Work (ID: 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lh.FormatListsSummary(tt.lists)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestListHandlerHelper_FormatItemsSummary(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name     string
		items    []Item
		expected string
	}{
		{
			name:     "Empty items",
			items:    []Item{},
			expected: "No items found",
		},
		{
			name: "Single uncompleted item",
			items: []Item{
				{ID: 1, Content: "Buy milk", IsCompleted: 0},
			},
			expected: "[ ] Buy milk (ID: 1)",
		},
		{
			name: "Single completed item",
			items: []Item{
				{ID: 1, Content: "Pay bills", IsCompleted: 1},
			},
			expected: "[✓] Pay bills (ID: 1)",
		},
		{
			name: "Multiple items",
			items: []Item{
				{ID: 1, Content: "Buy milk", IsCompleted: 0},
				{ID: 2, Content: "Pay bills", IsCompleted: 1},
			},
			expected: "[ ] Buy milk (ID: 1); [✓] Pay bills (ID: 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lh.FormatItemsSummary(tt.items)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestListHandlerHelper_FormatListDetails(t *testing.T) {
	lh := NewListHandlerHelper()

	list := &List{
		ID:          1,
		Name:        "Shopping",
		Description: "Grocery items",
		Items: []Item{
			{ID: 1, Content: "Buy milk", IsCompleted: 0},
			{ID: 2, Content: "Pay bills", IsCompleted: 1},
		},
	}

	result := lh.FormatListDetails(list)

	if !strings.Contains(result, "Shopping (ID: 1)") {
		t.Error("Should contain list name and ID")
	}
	if !strings.Contains(result, "Grocery items") {
		t.Error("Should contain description")
	}
	if !strings.Contains(result, "[ ] Buy milk") {
		t.Error("Should contain uncompleted item")
	}
	if !strings.Contains(result, "[✓] Pay bills") {
		t.Error("Should contain completed item")
	}
}

func TestListHandlerHelper_FormatListDetails_NoItems(t *testing.T) {
	lh := NewListHandlerHelper()

	list := &List{
		ID:          1,
		Name:        "Empty List",
		Description: "",
		Items:       []Item{},
	}

	result := lh.FormatListDetails(list)

	if !strings.Contains(result, "No items in this list") {
		t.Error("Should indicate no items")
	}
}

func TestListHandlerHelper_GetBaseURL(t *testing.T) {
	lh := NewListHandlerHelper()

	// Save original env var
	origURL := os.Getenv("LIST_HANDLER_URL")
	defer os.Setenv("LIST_HANDLER_URL", origURL)

	// Test with env var set
	os.Setenv("LIST_HANDLER_URL", "http://example.com:8088")
	url := lh.GetBaseURL()
	if url != "http://example.com:8088" {
		t.Errorf("Expected http://example.com:8088, got %q", url)
	}

	// Test with trailing slash
	os.Setenv("LIST_HANDLER_URL", "http://example.com:8088/")
	url = lh.GetBaseURL()
	if url != "http://example.com:8088" {
		t.Errorf("Expected trailing slash removed, got %q", url)
	}

	// Test with env var empty - should use default
	os.Setenv("LIST_HANDLER_URL", "")
	url = lh.GetBaseURL()
	if url != "http://localhost:8088" {
		t.Errorf("Expected default localhost:8088, got %q", url)
	}
}

func TestListHandlerHelper_BuildListURL(t *testing.T) {
	lh := NewListHandlerHelper()

	// Save original env var
	origURL := os.Getenv("LIST_HANDLER_URL")
	defer os.Setenv("LIST_HANDLER_URL", origURL)

	os.Setenv("LIST_HANDLER_URL", "http://localhost:8088")

	tests := []struct {
		name        string
		userID      string
		listID      string
		expectedURL string
	}{
		{
			name:        "All lists for user",
			userID:      "1",
			listID:      "",
			expectedURL: "http://localhost:8088/users/1/lists",
		},
		{
			name:        "Specific list",
			userID:      "1",
			listID:      "5",
			expectedURL: "http://localhost:8088/users/1/lists/5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := lh.BuildListURL(tt.userID, tt.listID)
			if url != tt.expectedURL {
				t.Errorf("Expected %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

func TestListHandlerHelper_BuildItemURL(t *testing.T) {
	lh := NewListHandlerHelper()

	// Save original env var
	origURL := os.Getenv("LIST_HANDLER_URL")
	defer os.Setenv("LIST_HANDLER_URL", origURL)

	os.Setenv("LIST_HANDLER_URL", "http://localhost:8088")

	tests := []struct {
		name        string
		userID      string
		listID      string
		itemID      string
		expectedURL string
	}{
		{
			name:        "All items in list",
			userID:      "1",
			listID:      "5",
			itemID:      "",
			expectedURL: "http://localhost:8088/users/1/lists/5/items",
		},
		{
			name:        "Specific item",
			userID:      "1",
			listID:      "5",
			itemID:      "10",
			expectedURL: "http://localhost:8088/users/1/lists/5/items/10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := lh.BuildItemURL(tt.userID, tt.listID, tt.itemID)
			if url != tt.expectedURL {
				t.Errorf("Expected %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

func TestListHandlerHelper_BuildAuthHeader(t *testing.T) {
	lh := NewListHandlerHelper()

	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	header := lh.BuildAuthHeader(token)

	expected := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	if header != expected {
		t.Errorf("Expected %q, got %q", expected, header)
	}
}

func TestListHandlerHelper_GetSessionUserID(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name       string
		session    *ChatSession
		expectedID string
	}{
		{
			name:       "Nil session",
			session:    nil,
			expectedID: "",
		},
		{
			name:       "Session without user_id",
			session:    &ChatSession{Variables: make(map[string]string)},
			expectedID: "",
		},
		{
			name: "Session with user_id",
			session: &ChatSession{
				Variables: map[string]string{"list_user_id": "123"},
			},
			expectedID: "123",
		},
		{
			name: "Session with empty user_id",
			session: &ChatSession{
				Variables: map[string]string{"list_user_id": ""},
			},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := lh.GetSessionUserID(tt.session)
			if userID != tt.expectedID {
				t.Errorf("Expected %q, got %q", tt.expectedID, userID)
			}
		})
	}
}

func TestListHandlerHelper_GetSessionToken(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name          string
		session       *ChatSession
		expectedToken string
	}{
		{
			name:          "Nil session",
			session:       nil,
			expectedToken: "",
		},
		{
			name:          "Session without token",
			session:       &ChatSession{Variables: make(map[string]string)},
			expectedToken: "",
		},
		{
			name: "Session with token",
			session: &ChatSession{
				Variables: map[string]string{"list_access_token": "token123"},
			},
			expectedToken: "token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := lh.GetSessionToken(tt.session)
			if token != tt.expectedToken {
				t.Errorf("Expected %q, got %q", tt.expectedToken, token)
			}
		})
	}
}

func TestListHandlerHelper_IsSessionAuthenticated(t *testing.T) {
	lh := NewListHandlerHelper()

	tests := []struct {
		name     string
		session  *ChatSession
		expected bool
	}{
		{
			name:     "Nil session",
			session:  nil,
			expected: false,
		},
		{
			name:     "Session without credentials",
			session:  &ChatSession{Variables: make(map[string]string)},
			expected: false,
		},
		{
			name: "Session with token only",
			session: &ChatSession{
				Variables: map[string]string{"list_access_token": "token123"},
			},
			expected: false,
		},
		{
			name: "Session with user_id only",
			session: &ChatSession{
				Variables: map[string]string{"list_user_id": "1"},
			},
			expected: false,
		},
		{
			name: "Session with both token and user_id",
			session: &ChatSession{
				Variables: map[string]string{
					"list_access_token": "token123",
					"list_user_id":      "1",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAuth := lh.IsSessionAuthenticated(tt.session)
			if isAuth != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, isAuth)
			}
		})
	}
}
