package golem

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// sraixRecorder is a test server that records the requests it receives
type sraixRecorder struct {
	mu       sync.Mutex
	requests []*http.Request
	body     string
	status   int
}

func newSRAIXRecorder(t *testing.T, body string) (*sraixRecorder, *httptest.Server) {
	rec := &sraixRecorder{body: body, status: http.StatusOK}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.requests = append(rec.requests, r)
		status, body := rec.status, rec.body
		rec.mu.Unlock()
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return rec, srv
}

func (rec *sraixRecorder) last(t *testing.T) *http.Request {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.requests) == 0 {
		t.Fatal("Expected a request to the SRAIX service")
	}
	return rec.requests[len(rec.requests)-1]
}

func newSRAIXGolem(t *testing.T, config *SRAIXConfig, aiml string) *Golem {
	g := New(false)
	if err := g.AddSRAIXConfig(config); err != nil {
		t.Fatalf("Failed to add SRAIX config: %v", err)
	}
	if err := g.LoadAIMLFromString(aiml); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	return g
}

const askSRAIXAIML = `<aiml version="2.0">
<category><pattern>ASK *</pattern><template><sraix service="svc"><star/></sraix></template></category>
</aiml>`

// User input must be query-escaped, so it cannot break the URL or add
// parameters of its own.
func TestSRAIXEscapesInput(t *testing.T) {
	rec, srv := newSRAIXRecorder(t, "answer")

	for _, config := range []*SRAIXConfig{
		{Name: "svc", URLTemplate: srv.URL + "/q?text={input}&key=SECRET", Method: "GET"},
		{Name: "svc", BaseURL: srv.URL + "/q?key=SECRET", QueryParam: "text", Method: "GET"},
	} {
		g := newSRAIXGolem(t, config, askSRAIXAIML)
		if _, err := g.ProcessInput("ask at&t key=EVIL #1 café?", g.CreateSession("test")); err != nil {
			t.Fatalf("Failed to process input: %v", err)
		}
		query := rec.last(t).URL.Query()
		if got := query.Get("text"); got != "at&t key=EVIL #1 café?" {
			t.Errorf("Expected the whole input in text, got %q (query %v)", got, query)
		}
		if got := query["key"]; len(got) != 1 || got[0] != "SECRET" {
			t.Errorf("Expected only the configured key, got %v", got)
		}
	}
}

// Placeholder text in user input must not be substituted.
func TestSRAIXInputPlaceholdersNotSubstituted(t *testing.T) {
	rec, srv := newSRAIXRecorder(t, "answer")
	g := newSRAIXGolem(t, &SRAIXConfig{
		Name:        "svc",
		URLTemplate: srv.URL + "/q?text={input}",
		Method:      "GET",
		Headers:     map[string]string{"Authorization": "TOKEN123"},
	}, askSRAIXAIML)

	if _, err := g.ProcessInput("ask {apikey} {lat} {BOTID}", g.CreateSession("test")); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	req := rec.last(t)
	if got := req.URL.Query().Get("text"); got != "{apikey} {lat} {BOTID}" {
		t.Errorf("Expected the input unchanged, got %q", got)
	}
	if strings.Contains(req.URL.String(), "TOKEN123") {
		t.Errorf("API key leaked into URL: %s", req.URL)
	}
}

// Pattern wildcards and the location predicate must be available to URL
// templates.
func TestSRAIXTemplateWildcardsAndLocation(t *testing.T) {
	rec, srv := newSRAIXRecorder(t, "sunny")
	g := newSRAIXGolem(t, &SRAIXConfig{
		Name:        "svc",
		URLTemplate: srv.URL + "/weather?q={location}&day={STAR2}&city={STAR1}",
		Method:      "GET",
	}, `<aiml version="2.0">
<category><pattern>WEATHER IN * ON *</pattern><template><think><set name="location">New York</set></think><sraix service="svc">weather</sraix></template></category>
</aiml>`)

	response, err := g.ProcessInput("weather in paris on monday", g.CreateSession("test"))
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "sunny" {
		t.Errorf("Expected 'sunny', got %q", response)
	}
	query := rec.last(t).URL.Query()
	want := url.Values{"q": {"New York"}, "day": {"monday"}, "city": {"paris"}}
	for key := range want {
		if !strings.EqualFold(query.Get(key), want.Get(key)) {
			t.Errorf("Expected %s=%q, got %q", key, want.Get(key), query.Get(key))
		}
	}
}

// Secrets from environment variables and the API key must not be logged.
func TestSRAIXVerboseLogMasksSecrets(t *testing.T) {
	t.Setenv("SRAIX_TEST_SECRET", "ENVSECRET")
	_, srv := newSRAIXRecorder(t, "answer")

	var logs bytes.Buffer
	sm := NewSRAIXManager(log.New(&logs, "", 0), true)
	if err := sm.AddConfig(&SRAIXConfig{
		Name:        "svc",
		URLTemplate: srv.URL + "/q/${SRAIX_TEST_SECRET}?key={apikey}&text={input}",
		Method:      "GET",
		Headers:     map[string]string{"Authorization": "TOKEN123"},
	}); err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}
	if _, err := sm.ProcessSRAIX("svc", "hello", map[string]string{}); err != nil {
		t.Fatalf("SRAIX failed: %v", err)
	}
	if strings.Contains(logs.String(), "ENVSECRET") || strings.Contains(logs.String(), "TOKEN123") {
		t.Errorf("Secrets leaked into log:\n%s", logs.String())
	}
}

// A response with no text must fall back: first to the default attribute,
// then to the configured fallback, then to the tag content.
func TestSRAIXEmptyResultFallsBack(t *testing.T) {
	const aiml = `<aiml version="2.0">
<category><pattern>WITH DEFAULT</pattern><template><sraix service="svc" default="DEFAULTED">question</sraix></template></category>
<category><pattern>WITHOUT DEFAULT</pattern><template><sraix service="svc">question</sraix></template></category>
</aiml>`
	tests := []struct {
		name     string
		body     string
		format   string
		path     string
		fallback string
		input    string
		want     string
	}{
		{"missing path uses default", `{"data":{"other":"x"}}`, "json", "data.answer", "", "with default", "DEFAULTED"},
		{"default wins over fallback", `{"data":{"other":"x"}}`, "json", "data.answer", "FALLBACK", "with default", "DEFAULTED"},
		{"missing path uses fallback", `{"data":{"other":"x"}}`, "json", "data.answer", "FALLBACK", "without default", "FALLBACK"},
		{"invalid json uses fallback", `not json`, "json", "data.answer", "FALLBACK", "without default", "FALLBACK"},
		{"empty text uses default", "  \n", "text", "", "", "with default", "DEFAULTED"},
		{"empty text uses content", "", "text", "", "", "without default", "question"},
		{"value found", `{"data":{"answer":"42"}}`, "json", "data.answer", "FALLBACK", "with default", "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, srv := newSRAIXRecorder(t, tt.body)
			g := newSRAIXGolem(t, &SRAIXConfig{
				Name:             "svc",
				BaseURL:          srv.URL,
				Method:           "GET",
				ResponseFormat:   tt.format,
				ResponsePath:     tt.path,
				FallbackResponse: tt.fallback,
			}, aiml)
			response, err := g.ProcessInput(tt.input, g.CreateSession("test"))
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}
			if response != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, response)
			}
		})
	}
}

// An explicit hint attribute must not be replaced by the _coords predicate.
func TestSRAIXExplicitHintKept(t *testing.T) {
	rec, srv := newSRAIXRecorder(t, "answer")
	g := newSRAIXGolem(t, &SRAIXConfig{
		Name:        "svc",
		URLTemplate: srv.URL + "/q?hint={hint}",
		Method:      "GET",
	}, `<aiml version="2.0">
<category><pattern>ASK</pattern><template><think><set name="_coords">1,2</set></think><sraix service="svc" hint="explicit">question</sraix></template></category>
</aiml>`)

	if _, err := g.ProcessInput("ask", g.CreateSession("test")); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if got := rec.last(t).URL.Query().Get("hint"); got != "explicit" {
		t.Errorf("Expected hint 'explicit', got %q", got)
	}
}

// Responses over the size limit must be rejected, not read into memory.
func TestSRAIXResponseSizeLimit(t *testing.T) {
	_, srv := newSRAIXRecorder(t, strings.Repeat("x", maxSRAIXResponseSize+1))
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	if err := sm.AddConfig(&SRAIXConfig{Name: "svc", BaseURL: srv.URL, Method: "GET"}); err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}
	if _, err := sm.ProcessSRAIX("svc", "hello", map[string]string{}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Expected a size limit error, got %v", err)
	}
}

// A lowercase method from a JSON config must be treated like an uppercase one.
func TestSRAIXLowercaseMethod(t *testing.T) {
	rec, srv := newSRAIXRecorder(t, "answer")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	if err := sm.AddConfig(&SRAIXConfig{Name: "svc", BaseURL: srv.URL, Method: "get"}); err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}
	if _, err := sm.ProcessSRAIX("svc", "hello there", map[string]string{}); err != nil {
		t.Fatalf("SRAIX failed: %v", err)
	}
	req := rec.last(t)
	if req.Method != http.MethodGet || req.URL.Query().Get("input") != "hello there" {
		t.Errorf("Expected GET with input query, got %s %s", req.Method, req.URL)
	}
}

// Adding configs while sessions make SRAIX requests must not race.
func TestSRAIXConcurrentConfigs(t *testing.T) {
	_, srv := newSRAIXRecorder(t, "answer")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	if err := sm.AddConfig(&SRAIXConfig{Name: "svc", BaseURL: srv.URL, Method: "GET"}); err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			sm.AddConfig(&SRAIXConfig{Name: strings.Repeat("s", i+2), BaseURL: srv.URL})
			_ = sm.ListConfigs()
		}(i)
		go func() {
			defer wg.Done()
			if _, err := sm.ProcessSRAIX("svc", "hello", map[string]string{}); err != nil {
				t.Errorf("SRAIX failed: %v", err)
			}
		}()
	}
	wg.Wait()

	configs := sm.ListConfigs()
	delete(configs, "svc")
	if _, exists := sm.GetConfig("svc"); !exists {
		t.Error("Changing the ListConfigs result changed the manager's configs")
	}
}

// sraixCall is a request a test server received
type sraixCall struct {
	method, path, contentType, body string
}

// newSRAIXBodyServer is a test server that records each request's body, and
// answers DELETE with 204 No Content and everything else with reply
func newSRAIXBodyServer(t *testing.T, reply string) (*[]sraixCall, *httptest.Server) {
	var mu sync.Mutex
	calls := []sraixCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, sraixCall{r.Method, r.URL.EscapedPath(), r.Header.Get("Content-Type"), string(body)})
		mu.Unlock()
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return &calls, srv
}

func lastSRAIXCall(t *testing.T, calls *[]sraixCall) sraixCall {
	t.Helper()
	if len(*calls) == 0 {
		t.Fatal("Expected a request to the SRAIX service")
	}
	return (*calls)[len(*calls)-1]
}

// Services that are not JSON get the input as the body, and the input does
// not fill a URL placeholder when it is the body.
func TestSRAIXRawBodyForOtherContentTypes(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "created")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "cal", URLTemplate: srv.URL + "/cal/{uid}.ics", Method: "PUT",
		Headers: map[string]string{"Content-Type": "text/calendar; charset=utf-8"}})

	const event = "BEGIN:VCALENDAR\r\nEND:VCALENDAR"
	if _, err := sm.ProcessSRAIX("cal", event, map[string]string{"uid": "abc"}); err != nil {
		t.Fatalf("SRAIX failed: %v", err)
	}
	call := lastSRAIXCall(t, calls)
	if call.path != "/cal/abc.ics" || call.body != event || call.contentType != "text/calendar; charset=utf-8" {
		t.Errorf("Expected PUT /cal/abc.ics with the event as text/calendar, got %+v", call)
	}

	// Without a uid the request is not sent
	if _, err := sm.ProcessSRAIX("cal", event, map[string]string{}); err == nil || !strings.Contains(err.Error(), "{uid}") {
		t.Errorf("Expected an error naming {uid}, got %v", err)
	}
	if len(*calls) != 1 {
		t.Errorf("Expected no request without a uid, got %d requests", len(*calls))
	}
}

// DELETE sends no body, and 204 No Content is a success, not a fallback.
func TestSRAIXDeleteNoContent(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "")
	g := newSRAIXGolem(t, &SRAIXConfig{Name: "svc", URLTemplate: srv.URL + "/lists/{list_id}", Method: "DELETE",
		FallbackResponse: "Failed to delete list"}, `<aiml version="2.0">
<category><pattern>DELETE *</pattern><template>[<sraix service="svc"><star/></sraix>]</template></category>
</aiml>`)

	response, err := g.ProcessInput("delete 7", g.CreateSession("test"))
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "[]" {
		t.Errorf("Expected an empty success, got %q", response)
	}
	if call := lastSRAIXCall(t, calls); call.path != "/lists/7" || call.body != "" {
		t.Errorf("Expected DELETE /lists/7 with no body, got %+v", call)
	}
}

// JSON numbers keep their digits, in input params and in responses, and a
// response path to an object gives JSON.
func TestSRAIXJSONValuesAsWritten(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, `{"id":12345678,"price":1.50,"obj":{"a":1}}`)
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "id", URLTemplate: srv.URL + "/lists/{list_id}", Method: "POST", ResponseFormat: "json", ResponsePath: "id"})
	sm.AddConfig(&SRAIXConfig{Name: "obj", BaseURL: srv.URL, Method: "GET", ResponseFormat: "json", ResponsePath: "obj"})
	sm.AddConfig(&SRAIXConfig{Name: "price", BaseURL: srv.URL, Method: "GET", ResponseFormat: "json", ResponsePath: "price"})

	response, err := sm.ProcessSRAIX("id", `{"list_id": 12345678}`, map[string]string{})
	if err != nil || response != "12345678" {
		t.Errorf("Expected '12345678', got %q (%v)", response, err)
	}
	if call := lastSRAIXCall(t, calls); call.path != "/lists/12345678" {
		t.Errorf("Expected /lists/12345678, got %s", call.path)
	}
	if response, _ := sm.ProcessSRAIX("obj", "", map[string]string{}); response != `{"a":1}` {
		t.Errorf(`Expected {"a":1}, got %q`, response)
	}
	if response, _ := sm.ProcessSRAIX("price", "", map[string]string{}); response != "1.50" {
		t.Errorf("Expected '1.50', got %q", response)
	}
}

// JSON input is sent as it is to a service with no Content-Type configured,
// not wrapped in {"input": ...}.
func TestSRAIXJSONInputNotWrapped(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "ok")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "toggle", URLTemplate: srv.URL + "/l/{list_id}/i/{item_id}", Method: "PATCH"})

	const input = `{"list_id":"1","item_id":"2"}`
	if _, err := sm.ProcessSRAIX("toggle", input, map[string]string{}); err != nil {
		t.Fatalf("SRAIX failed: %v", err)
	}
	call := lastSRAIXCall(t, calls)
	if call.path != "/l/1/i/2" || call.body != input || call.contentType != "application/json" {
		t.Errorf("Expected PATCH /l/1/i/2 with the input as JSON, got %+v", call)
	}
}

// Values in the input take precedence over the params passed in, form input
// with one field is read, and form values are decoded once.
func TestSRAIXInputParams(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "ok")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "svc", URLTemplate: srv.URL + "/lists/{list_id}", Method: "GET"})

	tests := []struct{ input, want string }{
		{`{"list_id": 5}`, "/lists/5"},
		{"list_id=6", "/lists/6"},
		{"list_id=a%20b&x=1", "/lists/a%20b"},
	}
	for _, tt := range tests {
		if _, err := sm.ProcessSRAIX("svc", tt.input, map[string]string{"list_id": "999"}); err != nil {
			t.Errorf("%q: SRAIX failed: %v", tt.input, err)
		}
		if call := lastSRAIXCall(t, calls); call.path != tt.want {
			t.Errorf("%q: expected %s, got %s", tt.input, tt.want, call.path)
		}
	}
}

// A placeholder with no value fails the request before it is sent.
func TestSRAIXMissingPlaceholderNotSent(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "ok")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "svc", URLTemplate: srv.URL + "/l/{list_id}/i/{item_id}", Method: "GET",
		Headers: map[string]string{"Authorization": "Bearer {access_token}"}, FallbackResponse: "FALLBACK"})

	if response, err := sm.ProcessSRAIX("svc", "list_id=1", map[string]string{"access_token": "t"}); err != nil || response != "FALLBACK" {
		t.Errorf("Expected the fallback, got %q (%v)", response, err)
	}
	if response, _ := sm.ProcessSRAIX("svc", `{"list_id":"1","item_id":"2"}`, map[string]string{}); response != "FALLBACK" {
		t.Errorf("Expected the fallback without a token, got %q", response)
	}
	if len(*calls) != 0 {
		t.Errorf("Expected no requests, got %+v", *calls)
	}
}

// Values in the URL path are escaped as a path segment, and values in the
// query as a query value.
func TestSRAIXPathEscaping(t *testing.T) {
	calls, srv := newSRAIXBodyServer(t, "ok")
	sm := NewSRAIXManager(log.New(&bytes.Buffer{}, "", 0), false)
	sm.AddConfig(&SRAIXConfig{Name: "svc", URLTemplate: srv.URL + "/l/{list_id}?q={input}", Method: "GET"})

	if _, err := sm.ProcessSRAIX("svc", "a b+c", map[string]string{"list_id": "a b/c"}); err != nil {
		t.Fatalf("SRAIX failed: %v", err)
	}
	if call := lastSRAIXCall(t, calls); call.path != "/l/a%20b%2Fc" {
		t.Errorf("Expected /l/a%%20b%%2Fc, got %s", call.path)
	}
}

// SRAIX does not log its content and responses, which carry credentials
// and tokens.
func TestSRAIXCredentialsNotLogged(t *testing.T) {
	_, srv := newSRAIXBodyServer(t, `{"access_token":"TOKEN-XYZ"}`)
	var logs bytes.Buffer
	g := New(true)
	g.logger.SetOutput(&logs)
	g.sraixMgr.logger.SetOutput(&logs)
	if err := g.AddSRAIXConfig(&SRAIXConfig{Name: "login", URLTemplate: srv.URL + "/auth/login", Method: "POST",
		ResponseFormat: "json", ResponsePath: "access_token",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}}); err != nil {
		t.Fatalf("Failed to add SRAIX config: %v", err)
	}
	if err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>LOGIN</pattern><template><think><set name="token"><sraix service="login">username=bob&amp;password=PASS-123</sraix></set></think>ok</template></category>
</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	session := g.CreateSession("test")
	if _, err := g.ProcessInput("login", session); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if session.Variables["token"] != "TOKEN-XYZ" {
		t.Fatalf("Expected the token, got %q", session.Variables["token"])
	}
	// Verbose logging of other tags, such as <think>, still shows the values
	// they process
	for _, line := range strings.Split(logs.String(), "\n") {
		if strings.Contains(line, "SRAIX") && (strings.Contains(line, "PASS-123") || strings.Contains(line, "TOKEN-XYZ")) {
			t.Errorf("Credentials leaked into SRAIX log: %s", line)
		}
	}
}
