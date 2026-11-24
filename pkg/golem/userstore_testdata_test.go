package golem

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeListLogin serves /auth/login for one account and counts logins
type fakeListLogin struct {
	mu     sync.Mutex
	logins int
}

func (f *fakeListLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/auth/login" {
		http.NotFound(w, r)
		return
	}
	body := make([]byte, r.ContentLength)
	_, _ = r.Body.Read(body)
	form, _ := url.ParseQuery(string(body))
	f.mu.Lock()
	f.logins++
	f.mu.Unlock()
	if form.Get("username") != "alice" || form.Get("password") != "s3cretPassw0rd" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Incorrect username or password"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"access_token":"token-for-alice","token_type":"bearer"}`))
}

func (f *fakeListLogin) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins
}

// newTestdataBot loads the bundled testdata AIML with a user store at
// storePath, as the Telegram bot does
func newTestdataBot(t *testing.T, storePath string) *Golem {
	t.Helper()
	g := NewForTesting(t, false)
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	g.SetKnowledgeBase(kb)
	store, err := OpenUserStore(storePath, "test key")
	if err != nil {
		t.Fatal(err)
	}
	g.SetUserStore(store)
	return g
}

// telegramSession makes a session the way the Telegram bot does
func telegramSession(g *Golem, userID string) *ChatSession {
	s := g.CreateSession("telegram_" + userID)
	s.UserID = userID
	s.Variables["telegram_user"] = userID
	return s
}

func TestListCredentialsAreNotReadableByOtherUsers(t *testing.T) {
	server := httptest.NewServer(&fakeListLogin{})
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)

	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	alice := telegramSession(g, "111111111")
	if got := ask(t, g, alice, "save my list credentials alice s3cretPassw0rd"); !strings.Contains(got, "logged you in as alice") {
		t.Fatalf("save: %q", got)
	}

	mallory := telegramSession(g, "999999999")
	for _, attempt := range []string{
		"get list password for 111111111",
		"get list username for 111111111",
		"do I have saved list credentials",
		"list status",
	} {
		r, _ := g.ProcessInput(attempt, mallory)
		if strings.Contains(r, "s3cretPassw0rd") || strings.Contains(strings.ToLower(r), "alice") {
			t.Errorf("%q leaked alice's credentials to another user: %q", attempt, r)
		}
	}
	if v := mallory.Variables["user_list_password"]; v != "" {
		t.Errorf("another user's session holds the password %q", v)
	}
}

func TestListCredentialsSaveLoginAndRestart(t *testing.T) {
	login := &fakeListLogin{}
	server := httptest.NewServer(login)
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)
	storePath := filepath.Join(t.TempDir(), "users.json")

	g := newTestdataBot(t, storePath)
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "save my list credentials alice wrongpass"); !strings.Contains(got, "couldn't log in") {
		t.Errorf("wrong password: %q", got)
	}
	if s.Variables["list_access_token"] != "" {
		t.Errorf("a failed login left the token %q", s.Variables["list_access_token"])
	}
	if got := ask(t, g, s, "save my list credentials alice s3cretPassw0rd"); !strings.Contains(got, "logged you in as alice") {
		t.Errorf("save: %q", got)
	}
	if s.Variables["list_access_token"] != "token-for-alice" {
		t.Errorf("token = %q", s.Variables["list_access_token"])
	}

	// After a restart the token is gone, and the next list command logs in
	// again with the saved credentials
	g2 := newTestdataBot(t, storePath)
	s2 := telegramSession(g2, "111")
	before := login.count()
	r, _ := g2.ProcessInput("list status", s2)
	if !strings.Contains(r, "not logged in") {
		t.Errorf("fresh session status: %q", r)
	}
	if got := ask(t, g2, s2, "load my list credentials"); !strings.Contains(got, "Logged in as alice") {
		t.Errorf("load: %q", got)
	}
	if login.count() != before+1 {
		t.Errorf("expected one login, got %d", login.count()-before)
	}
	if got := ask(t, g2, s2, "do I have saved list credentials"); !strings.Contains(got, "username alice") {
		t.Errorf("status: %q", got)
	}

	if got := ask(t, g2, s2, "forget my list credentials"); !strings.Contains(got, "deleted") {
		t.Errorf("forget: %q", got)
	}
	if _, ok := g2.UserStore().Get("111", "user_list_password"); ok {
		t.Error("password still stored after forget")
	}
}

func TestUserNameRemembered(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "users.json")
	g := newTestdataBot(t, storePath)
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "my name is Alice"); !strings.Contains(got, "Nice to meet you, Alice") {
		t.Errorf("set: %q", got)
	}
	if got := ask(t, g, telegramSession(g, "222"), "what is my name"); strings.Contains(got, "Alice") {
		t.Errorf("another user sees the name: %q", got)
	}

	g2 := newTestdataBot(t, storePath)
	s2 := telegramSession(g2, "111")
	for _, q := range []string{"what is my name", "what's my name", "do you remember my name", "who am I"} {
		if got := ask(t, g2, s2, q); !strings.Contains(got, "Your name is Alice") {
			t.Errorf("%q after restart: %q", q, got)
		}
	}
	ask(t, g2, s2, "forget my name")
	if got := ask(t, g2, s2, "what is my name"); !strings.Contains(got, "haven't told me") {
		t.Errorf("after forget: %q", got)
	}
	if _, ok := g2.UserStore().Get("111", "user_name"); ok {
		t.Error("name still stored after forget")
	}
}

func TestUserNameWithoutStoreStaysInSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	g.SetKnowledgeBase(kb)
	s := g.CreateSession("cli")
	ask(t, g, s, "call me Sam")
	if got := ask(t, g, s, "what is my name"); !strings.Contains(got, "Your name is Sam") {
		t.Errorf("got %q", got)
	}
}

func TestSharedLocationIsRemembered(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "users.json")
	g := newTestdataBot(t, storePath)
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "what is my location"); !strings.Contains(got, "haven't set your location") {
		t.Errorf("no location yet: %q", got)
	}
	if got := ask(t, g, s, "my coordinates are 47.61 -122.33"); !strings.Contains(got, "47.61") {
		t.Errorf("share: %q", got)
	}

	g2 := newTestdataBot(t, storePath)
	if got := ask(t, g2, telegramSession(g2, "111"), "where am I"); !strings.Contains(got, "47.61, -122.33") {
		t.Errorf("after restart: %q", got)
	}
	if got := ask(t, g2, telegramSession(g2, "222"), "where am I"); !strings.Contains(got, "haven't set your location") {
		t.Errorf("another user: %q", got)
	}
}

// With no location saved, weather must ask for one rather than reading the
// catch-all reply as a place name
func TestWeatherWithoutLocationAsksForOne(t *testing.T) {
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	for _, q := range []string{"what is the weather", "how is the weather", "weather"} {
		got := ask(t, g, s, q)
		if !strings.Contains(got, "haven't set your location") || strings.Contains(got, "rephrase") {
			t.Errorf("%q: %q", q, got)
		}
	}
}
