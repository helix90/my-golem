package golem

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSRAIXBodyTemplateEscaping(t *testing.T) {
	sm := NewSRAIXManager(log.New(io.Discard, "", 0), false)
	params := map[string]string{"user": "alice", "password": `p&ss+w=rd "x" \ é`}

	form := &SRAIXConfig{Name: "f", BodyTemplate: "username={user}&password={password}",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}}
	body, contentType, err := sm.requestBody(form, "ignored", nil, params)
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(string(body))
	if err != nil || values.Get("password") != params["password"] || values.Get("username") != "alice" || contentType != "" {
		t.Errorf("form body %q parsed to %v (%v), content type %q", body, values, err, contentType)
	}

	jsonConfig := &SRAIXConfig{Name: "j", BodyTemplate: `{"name":"{user}","secret":"{password}","note":"{input}"}`}
	body, contentType, err = sm.requestBody(jsonConfig, `say "hi"`, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"alice","secret":"p&ss+w=rd \"x\" \\ é","note":"say \"hi\""}`
	if string(body) != want || contentType != "application/json" {
		t.Errorf("json body = %s (%s), want %s", body, contentType, want)
	}

	if _, _, err := sm.requestBody(&SRAIXConfig{Name: "m", BodyTemplate: "{missing}"}, "", nil, nil); err == nil {
		t.Error("a placeholder with no value should fail the request")
	}
}

// fakeStrictLogin accepts exactly one username and password
type fakeStrictLogin struct {
	mu       sync.Mutex
	password string
	got      []string
}

func (f *fakeStrictLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	form, err := url.ParseQuery(string(body))
	f.mu.Lock()
	f.got = append(f.got, form.Get("password"))
	f.mu.Unlock()
	if err != nil || form.Get("username") != "alice" || form.Get("password") != f.password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, _ = w.Write([]byte(`{"access_token":"tok-abc123xyz","token_type":"bearer"}`))
}

func TestListLoginWithSpecialCharactersInPassword(t *testing.T) {
	const password = "P&ss+w=rd%25#1"
	fake := &fakeStrictLogin{password: password}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)

	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "save my list credentials alice "+password); !strings.Contains(got, "logged you in as alice") {
		t.Errorf("save: %q (server got %q)", got, fake.got)
	}
	if got := ask(t, g, s, "list login alice "+password); !strings.Contains(got, "Logged in successfully") {
		t.Errorf("list login: %q (server got %q)", got, fake.got)
	}
}

func TestVerboseLogsHideSecrets(t *testing.T) {
	const password = "Hunter2xyz"
	fake := &fakeStrictLogin{password: password}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)

	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	var logs bytes.Buffer
	g.logger = log.New(&logs, "", 0)
	g.SetLogLevel(LogLevelTrace)

	s := telegramSession(g, "111")
	ask(t, g, s, "save my list credentials alice "+password)
	ask(t, g, s, "list login alice "+password)
	ask(t, g, s, "list status")
	ask(t, g, s, "show list groceries")

	text := strings.ToLower(logs.String())
	for _, secret := range []string{password, "tok-abc123xyz"} {
		if n := strings.Count(text, strings.ToLower(secret)); n > 0 {
			for _, line := range strings.Split(logs.String(), "\n") {
				if strings.Contains(strings.ToLower(line), strings.ToLower(secret)) {
					t.Errorf("%q appears %d times in the log, e.g. %.160s", secret, n, line)
					break
				}
			}
		}
	}
	if !strings.Contains(logs.String(), "save my list credentials alice ********") {
		t.Error("expected the input to be logged with the password masked")
	}
	if !strings.Contains(logs.String(), "show list groceries") {
		t.Error("ordinary input should be logged as it is")
	}
}

func TestLogRedactorMasksOnlySecrets(t *testing.T) {
	g := NewForTesting(t, false)
	if err := g.AddLogRedaction(`(?i)pin is (\d+)`); err != nil {
		t.Fatal(err)
	}
	g.noteSecret("api_token", "s3cr3t-value")
	g.noteSecret("user_name", "Alice")  // not a secret predicate
	g.noteSecret("user_password", "ab") // too short to mask everywhere
	r := g.redactor
	for in, want := range map[string]string{
		"my pin is 1234":                         "my pin is ********",
		"token S3CR3T-VALUE sent":                "token ******** sent",
		"Alice said ab":                          "Alice said ab",
		"LIST LOGIN bob pa55word":                "LIST LOGIN bob ********",
		"username=bob&password=x%26y&grant=pass": "username=bob&password=********&grant=pass",
	} {
		if got := r.redact(in); got != want {
			t.Errorf("redact(%q) = %q, want %q", in, got, want)
		}
	}
}
