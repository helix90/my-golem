package golem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// fakeSearx answers like SearXNG's JSON API: its calculator plugin answers
// "<query> = <result>", in newer versions as objects and in older ones as
// strings
type fakeSearx struct {
	mu       sync.Mutex
	queries  []string
	oldStyle bool
	noJSON   bool
}

func (f *fakeSearx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	f.mu.Lock()
	f.queries = append(f.queries, q)
	oldStyle, noJSON := f.oldStyle, f.noJSON
	f.mu.Unlock()
	if r.URL.Path != "/search" || r.URL.Query().Get("format") != "json" || noJSON {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	answers := map[string]string{
		"2 + 2":         "2 + 2 = 4",
		"7 * 6":         "7 * 6 = 42",
		"15/100 * 80":   "15/100 * 80 = 12",
		"1234 * 5678":   "1234 * 5678 = 7006652",
		"2^10":          "2^10 = 1024",
		"5 miles in km": "5 mi = 8.04672 km",
	}
	out := map[string]interface{}{"query": q, "results": []interface{}{}, "answers": []interface{}{}, "infoboxes": []interface{}{}}
	if a, ok := answers[q]; ok {
		if oldStyle {
			out["answers"] = []interface{}{a}
		} else {
			out["answers"] = []interface{}{map[string]string{"answer": a}}
		}
	}
	if q == "best pizza in lisbon" {
		out["results"] = []interface{}{
			map[string]string{"title": "The 10 best pizza places in Lisbon", "url": "https://example.com/pizza", "content": "Our pick of the best pizza in Lisbon. Updated for 2026."},
			map[string]string{"title": "Pizza – Lisbon forum", "url": "https://example.org/forum"},
			map[string]string{"title": "Lisbon food guide", "url": "https://example.net/food"},
			map[string]string{"title": "A fourth result", "url": "https://example.net/4"},
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func newSearxBot(t *testing.T) (*Golem, *ChatSession, *fakeSearx, *fakeWikipedia) {
	t.Helper()
	f := &fakeSearx{}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	t.Setenv("SEARX_URL", server.URL)
	wiki := &fakeWikipedia{}
	wikiServer := httptest.NewServer(wiki)
	t.Cleanup(wikiServer.Close)
	t.Setenv("WIKIPEDIA_URL", wikiServer.URL+"/w/api.php")
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	return g, g.CreateSession("searx"), f, wiki
}

func TestSearxQuery(t *testing.T) {
	for question, want := range map[string]string{
		"2 + 2":                    "2 + 2",
		"what is 2+2?":             "2+2",
		"7 times 6":                "7 * 6",
		"7 x 6":                    "7 * 6",
		"10 divided by 4":          "10 / 4",
		"9 minus 3 plus 1":         "9 - 3 + 1",
		"15% of 80":                "15/100 * 80",
		"20 percent of 45":         "20/100 * 45",
		"1,234 multiplied by 2":    "1234 * 2",
		"3 squared":                "3 ^2",
		"2 to the power of 10":     "2 ^ 10",
		"(3 + 4) * 2":              "(3 + 4) * 2",
		"5 miles in km":            "5 miles in km",
		"100 usd in eur":           "100 usd in eur",
		"72 fahrenheit to celsius": "72 fahrenheit to celsius",
	} {
		if got, ok := searxQuery(question); !ok || got != want {
			t.Errorf("%q = %q, %v; want %q", question, got, ok, want)
		}
	}
	for _, question := range []string{"a quasar", "the meaning of life", "2001: A Space Odyssey", "Apollo 11", "42", "route 66 in arizona", "the 1990s"} {
		if got, ok := searxQuery(question); ok {
			t.Errorf("%q should not be a sum, got %q", question, got)
		}
	}
}

func TestSearxAnswers(t *testing.T) {
	g, s, f, wiki := newSearxBot(t)
	for input, want := range map[string]string{
		"calculate 2 + 2":       "2 + 2 is 4.",
		"calculate 7 times 6":   "7 times 6 is 42.",
		"how much is 15% of 80": "15% of 80 is 12.",
		"compute 1,234 * 5,678": "1,234 * 5,678 is 7006652.",
		"convert 5 miles in km": "5 miles in km is 8.04672 km.",
		"calculate a quasar":    `That doesn't look like a sum I can work out. Try something like "calculate 15% of 80".`,
		"calculate 9 + 9 + 9":   "I couldn't work that out.",
	} {
		if got := ask(t, g, s, input); got != want {
			t.Errorf("%q: %q, want %q", input, got, want)
		}
	}
	f.mu.Lock()
	f.oldStyle = true
	f.mu.Unlock()
	if got := ask(t, g, s, "calculate 2^10"); got != "2^10 is 1024." {
		t.Errorf("old-style answer: %q", got)
	}

	// LOOK UP ANSWER: a sum to SearXNG, anything else to Wikipedia
	if got := ask(t, g, s, "look up answer 2 + 2"); got != "2 + 2 is 4." {
		t.Errorf("sum: %q", got)
	}
	before := len(f.queries)
	r := ParseRichResponse(ask(t, g, s, "look up answer ada lovelace"))
	if !strings.HasPrefix(r.Text, "Augusta Ada King") {
		t.Errorf("wikipedia: %q", r.Text)
	}
	if len(f.queries) != before {
		t.Errorf("a question that isn't a sum went to SearXNG: %v", f.queries[before:])
	}
	if len(wiki.requests) == 0 {
		t.Error("Wikipedia wasn't asked")
	}

	// SearXNG refusing JSON (format not enabled); a sum isn't then looked up
	// in Wikipedia
	f.mu.Lock()
	f.noJSON = true
	f.mu.Unlock()
	if got := ask(t, g, s, "calculate 2 + 2"); got != "I can't reach the calculator right now. Is SEARX_URL set?" {
		t.Errorf("403: %q", got)
	}
	if got := ask(t, g, s, "look up answer 2 + 2"); got != "I can't reach the calculator right now. Is SEARX_URL set?" {
		t.Errorf("403, what is: %q", got)
	}
}

func TestSearxSearch(t *testing.T) {
	g, s, _, _ := newSearxBot(t)
	r := ParseRichResponse(ask(t, g, s, "search for best pizza in lisbon"))
	if r.Text != "The 10 best pizza places in Lisbon: Our pick of the best pizza in Lisbon. Updated for 2026." {
		t.Errorf("text: %q", r.Text)
	}
	want := []RichButton{
		{Text: "The 10 best pizza places in Lisbon", URL: "https://example.com/pizza"},
		{Text: "Pizza – Lisbon forum", URL: "https://example.org/forum"},
		{Text: "Lisbon food guide", URL: "https://example.net/food"},
	}
	if !reflect.DeepEqual(r.Buttons, want) {
		t.Errorf("buttons: %+v", r.Buttons)
	}
	if got := ask(t, g, s, "google nothing at all here"); got != `I didn't find anything for "nothing at all here".` {
		t.Errorf("none: %q", got)
	}
}

func TestSearxNotConfigured(t *testing.T) {
	t.Setenv("SEARX_URL", "")
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := g.CreateSession("x")
	if got := ask(t, g, s, "search for pizza"); got != "I can't search the web right now. Is SEARX_URL set?" {
		t.Errorf("got %q", got)
	}
	if got := ask(t, g, s, "look up answer 2 + 2"); got != "I can't reach the calculator right now. Is SEARX_URL set?" {
		t.Errorf("sum: %q", got)
	}
}
