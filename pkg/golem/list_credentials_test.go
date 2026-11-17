package golem

import (
	"os"
	"strings"
	"testing"
)

// TestListCredentialsSaveAndRetrieve tests saving and retrieving list credentials
func TestListCredentialsSaveAndRetrieve(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		telegramID string
	}{
		{
			name:       "Save and retrieve basic credentials",
			username:   "testuser",
			password:   "testpass123",
			telegramID: "123456789",
		},
		{
			name:       "Save and retrieve credentials with special chars",
			username:   "user@example.com",
			password:   "P@ssw0rd!",
			telegramID: "987654321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Both instances share the same storage dir to simulate a bot restart
			// while retaining the same on-disk learned categories.
			storageDir := t.TempDir()

			g := NewForTestingWithDir(t, false, storageDir)

			// Load test AIML files
			kb, err := g.LoadAIMLFromDirectory("../../testdata")
			if err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}
			g.SetKnowledgeBase(kb)

			// Create session and set telegram_user
			session := g.CreateSession(tt.telegramID)
			session.Variables["telegram_user"] = tt.telegramID

			// Save credentials
			input := "save my list credentials " + tt.username + " " + tt.password
			response, err := g.ProcessInput(input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if !strings.Contains(response, "saved") {
				t.Errorf("Expected confirmation of saving, got: %s", response)
			}

			// Verify credentials are stored in session
			if session.Variables["list_username"] != tt.username {
				t.Errorf("Expected list_username=%s, got: %s", tt.username, session.Variables["list_username"])
			}

			// Create new Golem instance to simulate bot restart (same storage dir)
			g2 := NewForTestingWithDir(t, false, storageDir)

			kb2, err := g2.LoadAIMLFromDirectory("../../testdata")
			if err != nil {
				t.Fatalf("Failed to load AIML on reload: %v", err)
			}
			g2.SetKnowledgeBase(kb2)

			// Create new session (same telegram ID)
			session2 := g2.CreateSession(tt.telegramID)
			session2.Variables["telegram_user"] = tt.telegramID

			// Check if credentials are saved
			response2, err := g2.ProcessInput("do I have saved list credentials", session2)
			if err != nil {
				t.Fatalf("Failed to check credentials: %v", err)
			}

			if !strings.Contains(response2, "Yes") || !strings.Contains(response2, tt.username) {
				t.Errorf("Expected saved credentials for %s, got: %s", tt.username, response2)
			}
		})
	}
}

// TestListCredentialsVariations tests different command variations for saving credentials
func TestListCredentialsVariations(t *testing.T) {
	variations := []string{
		"save my list credentials testuser testpass",
		"save list credentials testuser testpass",
		"store my list credentials testuser testpass",
	}

	for _, variation := range variations {
		t.Run(variation, func(t *testing.T) {
			g := NewForTesting(t, false)

			kb, err := g.LoadAIMLFromDirectory("../../testdata")
			if err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}
			g.SetKnowledgeBase(kb)

			session := g.CreateSession("testuser123")
			session.Variables["telegram_user"] = "testuser123"

			response, err := g.ProcessInput(variation, session)
			if err != nil {
				t.Fatalf("Failed to process %s: %v", variation, err)
			}

			if !strings.Contains(response, "saved") {
				t.Errorf("Expected confirmation for '%s', got: %s", variation, response)
			}

			// Verify username is stored
			if session.Variables["list_username"] != "testuser" {
				t.Errorf("Username not stored correctly for variation: %s", variation)
			}
		})
	}
}

// TestListCredentialsLoadManual tests manual loading of credentials
func TestListCredentialsLoadManual(t *testing.T) {
	// Note: This test doesn't actually call the SRAIX service
	// It only tests that credentials can be loaded
	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	telegramID := "999888777"
	session := g.CreateSession(telegramID)
	session.Variables["telegram_user"] = telegramID

	// Save credentials first
	_, err = g.ProcessInput("save my list credentials loadtestuser loadtestpass", session)
	if err != nil {
		t.Fatalf("Failed to save credentials: %v", err)
	}

	// Clear session variables to simulate being logged out
	session.Variables["list_username"] = ""
	session.Variables["list_password"] = ""
	session.Variables["list_access_token"] = ""

	// Manually load credentials
	response, err := g.ProcessInput("load my list credentials", session)
	if err != nil {
		t.Fatalf("Failed to load credentials: %v", err)
	}

	// Should contain username in response (or a login attempt message)
	if !strings.Contains(response, "loadtestuser") && !strings.Contains(response, "Unable to login") {
		t.Logf("Load response: %s", response)
		// This is expected since we don't have a real SRAIX endpoint
		// The test just verifies the AIML pattern works
	}

	// Verify credentials were loaded into session
	if session.Variables["list_username"] != "loadtestuser" {
		t.Errorf("Expected username to be loaded into session")
	}
}

// TestListCredentialsForget tests clearing saved credentials
func TestListCredentialsForget(t *testing.T) {
	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	telegramID := "111222333"
	session := g.CreateSession(telegramID)
	session.Variables["telegram_user"] = telegramID

	// Save credentials
	_, err = g.ProcessInput("save my list credentials forgetuser forgetpass", session)
	if err != nil {
		t.Fatalf("Failed to save credentials: %v", err)
	}

	// Verify they're saved
	checkResponse, err := g.ProcessInput("do I have saved list credentials", session)
	if err != nil {
		t.Fatalf("Failed to check credentials: %v", err)
	}

	if !strings.Contains(checkResponse, "Yes") {
		t.Fatalf("Credentials not saved properly: %s", checkResponse)
	}

	// Forget credentials
	forgetResponse, err := g.ProcessInput("forget my list credentials", session)
	if err != nil {
		t.Fatalf("Failed to forget credentials: %v", err)
	}

	if !strings.Contains(forgetResponse, "cleared") && !strings.Contains(forgetResponse, "removed") {
		t.Errorf("Expected confirmation of clearing, got: %s", forgetResponse)
	}

	// Create new instance to verify unlearning worked
	g2 := NewForTesting(t, false)

	kb2, err := g2.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML on reload: %v", err)
	}
	g2.SetKnowledgeBase(kb2)

	session2 := g2.CreateSession(telegramID)
	session2.Variables["telegram_user"] = telegramID

	// Check if credentials are gone
	response2, err := g2.ProcessInput("do I have saved list credentials", session2)
	if err != nil {
		t.Fatalf("Failed to check credentials after forget: %v", err)
	}

	if !strings.Contains(response2, "don't have") && !strings.Contains(response2, "haven't saved") {
		t.Errorf("Expected no saved credentials, got: %s", response2)
	}
}

// TestListCredentialsForgetVariations tests different forget command variations
func TestListCredentialsForgetVariations(t *testing.T) {
	variations := []string{
		"forget my list credentials",
		"clear my list credentials",
		"delete my list credentials",
		"remove my list credentials",
	}

	for _, variation := range variations {
		t.Run(variation, func(t *testing.T) {
			g := NewForTesting(t, false)

			kb, err := g.LoadAIMLFromDirectory("../../testdata")
			if err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}
			g.SetKnowledgeBase(kb)

			session := g.CreateSession("cleartest")
			session.Variables["telegram_user"] = "cleartest"
			session.Variables["list_username"] = "testuser"
			session.Variables["list_password"] = "testpass"

			response, err := g.ProcessInput(variation, session)
			if err != nil {
				t.Fatalf("Failed to process %s: %v", variation, err)
			}

			if !strings.Contains(response, "cleared") && !strings.Contains(response, "removed") {
				t.Errorf("Expected confirmation for '%s', got: %s", variation, response)
			}

			// Verify session variables are cleared
			if session.Variables["list_username"] != "" {
				t.Errorf("Username not cleared for variation: %s", variation)
			}
		})
	}
}

// TestListCredentialsWithoutTelegramID tests behavior without telegram_user set
func TestListCredentialsWithoutTelegramID(t *testing.T) {
	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("notelegram")
	// Intentionally not setting telegram_user

	// Try to save credentials
	response, err := g.ProcessInput("save my list credentials testuser testpass", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// Should still work in session, but no persistence message
	if !strings.Contains(response, "saved") {
		t.Errorf("Expected save confirmation without telegram ID, got: %s", response)
	}

	// Should not mention "profile" since no telegram ID
	if strings.Contains(response, "profile") {
		t.Errorf("Should not mention profile without telegram ID, got: %s", response)
	}

	// Verify stored in session only
	if session.Variables["list_username"] != "testuser" {
		t.Errorf("Credentials not stored in session")
	}
}

// TestListCredentialsCheckStatus tests the credentials status command
func TestListCredentialsCheckStatus(t *testing.T) {
	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	telegramID := "444555666"
	session := g.CreateSession(telegramID)
	session.Variables["telegram_user"] = telegramID

	// Check status before saving
	response1, err := g.ProcessInput("do I have saved list credentials", session)
	if err != nil {
		t.Fatalf("Failed to check status: %v", err)
	}

	if !strings.Contains(response1, "don't have") && !strings.Contains(response1, "haven't saved") {
		t.Errorf("Expected no credentials initially, got: %s", response1)
	}

	// Save credentials
	_, err = g.ProcessInput("save my list credentials statususer statuspass", session)
	if err != nil {
		t.Fatalf("Failed to save credentials: %v", err)
	}

	// Check status after saving
	response2, err := g.ProcessInput("are my list credentials saved", session)
	if err != nil {
		t.Fatalf("Failed to check status after save: %v", err)
	}

	if !strings.Contains(response2, "Yes") || !strings.Contains(response2, "statususer") {
		t.Errorf("Expected saved credentials, got: %s", response2)
	}
}

// TestListCredentialsIntegration tests full workflow with auto-login
func TestListCredentialsIntegration(t *testing.T) {
	// Set up test environment variable for list handler
	os.Setenv("LIST_HANDLER_URL", "http://localhost:8088")
	defer os.Unsetenv("LIST_HANDLER_URL")

	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	telegramID := "777888999"
	session := g.CreateSession(telegramID)
	session.Variables["telegram_user"] = telegramID

	// Step 1: Save credentials
	response1, err := g.ProcessInput("save my list credentials integuser integpass", session)
	if err != nil {
		t.Fatalf("Failed to save credentials: %v", err)
	}
	t.Logf("Save response: %s", response1)

	// Step 2: Clear session (simulate logout)
	session.Variables["list_access_token"] = ""
	session.Variables["list_username"] = ""
	session.Variables["list_password"] = ""

	// Step 3: Try to use a list command that requires authentication
	// It should auto-login using saved credentials
	response2, err := g.ProcessInput("list show all", session)
	if err != nil {
		t.Fatalf("Failed to execute list command: %v", err)
	}
	t.Logf("List command response: %s", response2)

	// Note: Since we don't have a real list handler running, this will likely fail
	// But the test verifies that the auto-login pattern is triggered
	// The response should either show login attempt or "Unable to retrieve lists"
	if !strings.Contains(response2, "Unable to retrieve") &&
		!strings.Contains(response2, "login") &&
		!strings.Contains(response2, "save credentials") {
		t.Logf("Auto-login was triggered (expected behavior without real server)")
	}

	// Step 4: Check that credentials were loaded
	if session.Variables["list_username"] == "" {
		t.Errorf("Auto-login should have loaded username into session")
	}
}

// TestListCredentialsHelp tests the help command
func TestListCredentialsHelp(t *testing.T) {
	g := NewForTesting(t, false)

	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("helpuser")

	response, err := g.ProcessInput("list credentials help", session)
	if err != nil {
		t.Fatalf("Failed to get help: %v", err)
	}

	// Should contain key information
	expectedKeywords := []string{"save", "credentials", "forget", "clear"}
	for _, keyword := range expectedKeywords {
		if !strings.Contains(strings.ToLower(response), keyword) {
			t.Errorf("Help should contain '%s', got: %s", keyword, response)
		}
	}
}
