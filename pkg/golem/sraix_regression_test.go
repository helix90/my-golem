package golem

import (
	"bytes"
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

// A response with no text must fall back: first to the configured fallback,
// then to the default attribute, then to the tag content.
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
		{"missing path uses fallback", `{"data":{"other":"x"}}`, "json", "data.answer", "FALLBACK", "with default", "FALLBACK"},
		{"invalid json uses fallback", `not json`, "json", "data.answer", "FALLBACK", "with default", "FALLBACK"},
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
