package golem

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeExpiringListHandler is a list handler whose tokens can be expired:
// a token is valid until expire is called. With alwaysRefuse it rejects
// every token.
type fakeExpiringListHandler struct {
	mu           sync.Mutex
	issued       int
	valid        string
	logins       int
	listRequests int
	alwaysRefuse bool
}

func (f *fakeExpiringListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/auth/login":
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		f.logins++
		if form.Get("username") != "alice" || form.Get("password") != "pw" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.issued++
		f.valid = fmt.Sprintf("token-%d", f.issued)
		fmt.Fprintf(w, `{"access_token":%q,"token_type":"bearer"}`, f.valid)
	case strings.HasPrefix(r.URL.Path, "/users/me/lists/"):
		f.listRequests++
		if f.alwaysRefuse || r.Header.Get("Authorization") != "Bearer "+f.valid {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"Could not validate credentials"}`))
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/users/me/lists/")
		fmt.Fprintf(w, `{"id":1,"name":%q,"description":"","items":[{"id":5,"content":"milk","is_completed":0}]}`, name)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeExpiringListHandler) expire() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.valid = ""
}

func (f *fakeExpiringListHandler) counts() (logins, listRequests int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins, f.listRequests
}

func newListExpiryBot(t *testing.T) (*Golem, *ChatSession, *fakeExpiringListHandler) {
	t.Helper()
	fake := &fakeExpiringListHandler{}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	t.Setenv("LIST_HANDLER_URL", server.URL)
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	return g, telegramSession(g, "111"), fake
}

func TestExpiredListTokenLogsInAgain(t *testing.T) {
	g, s, fake := newListExpiryBot(t)
	ask(t, g, s, "save my list credentials alice pw")
	if got := ask(t, g, s, "show list groceries"); !strings.Contains(got, "milk") {
		t.Fatalf("before expiry: %q", got)
	}

	fake.expire()
	loginsBefore, requestsBefore := fake.counts()
	if got := ask(t, g, s, "show list groceries"); !strings.Contains(got, "milk") {
		t.Errorf("after expiry the list should still show: %q", got)
	}
	logins, requests := fake.counts()
	if logins-loginsBefore != 1 || requests-requestsBefore != 2 {
		t.Errorf("want 1 login and 2 list requests (refused, then retried), got %d and %d", logins-loginsBefore, requests-requestsBefore)
	}
	if s.Variables["list_access_token"] != "token-2" {
		t.Errorf("token = %q, want the new token-2", s.Variables["list_access_token"])
	}
}

func TestExpiredListTokenWithoutSavedCredentials(t *testing.T) {
	g, s, fake := newListExpiryBot(t)
	if got := ask(t, g, s, "list login alice pw"); !strings.Contains(got, "Logged in successfully") {
		t.Fatalf("login: %q", got)
	}
	fake.expire()
	got := ask(t, g, s, "show list groceries")
	if !strings.Contains(got, "Your list login has expired") || strings.Contains(got, "couldn't find") {
		t.Errorf("got %q", got)
	}
	if s.Variables["list_access_token"] != "" {
		t.Errorf("the expired token should be cleared, got %q", s.Variables["list_access_token"])
	}
	// The next command asks for a login, and the message is not repeated
	if got := ask(t, g, s, "show list groceries"); !strings.Contains(got, "Please login first") {
		t.Errorf("next command: %q", got)
	}
}

func TestListTokenRefusedAgainRetriesOnce(t *testing.T) {
	g, s, fake := newListExpiryBot(t)
	ask(t, g, s, "save my list credentials alice pw")
	fake.mu.Lock()
	fake.alwaysRefuse = true
	fake.mu.Unlock()

	_, requestsBefore := fake.counts()
	got := ask(t, g, s, "show list groceries")
	if !strings.Contains(got, "couldn't find a list") {
		t.Errorf("got %q", got)
	}
	if _, requests := fake.counts(); requests-requestsBefore != 2 {
		t.Errorf("want 2 list requests (one retry), got %d", requests-requestsBefore)
	}
}

// Only 401 Unauthorized starts a new login
func TestSRAIXReauthOnlyOnUnauthorized(t *testing.T) {
	for status, want := range map[int]bool{http.StatusUnauthorized: true, http.StatusForbidden: false, http.StatusNotFound: false, http.StatusInternalServerError: false} {
		err := error(&sraixStatusError{status: status})
		if got := isUnauthorized(err); got != want {
			t.Errorf("isUnauthorized(%d) = %v, want %v", status, got, want)
		}
		if isUnauthorized(fmt.Errorf("wrapped: %w", err)) != want {
			t.Errorf("wrapped %d", status)
		}
	}
	if isUnauthorized(fmt.Errorf("connection refused")) {
		t.Error("a network error is not unauthorized")
	}
}
