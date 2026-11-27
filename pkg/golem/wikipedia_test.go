package golem

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeWikipedia answers searches from the responses in
// testdata/wikipedia (recorded from en.wikipedia.org), named by the search
type fakeWikipedia struct {
	mu       sync.Mutex
	status   int
	requests []*http.Request
}

func (f *fakeWikipedia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r)
	status := f.status
	f.mu.Unlock()
	if status != 0 {
		w.WriteHeader(status)
		w.Write([]byte("You are making too many requests to the API."))
		return
	}
	q := r.URL.Query()
	name := q.Get("gsrsearch")
	if titles := q.Get("titles"); titles != "" {
		name = map[string]string{"Mercury (planet)": "mercury_planet"}[titles]
	}
	data, err := os.ReadFile(filepath.Join("testdata", "wikipedia", strings.ReplaceAll(strings.ToLower(name), " ", "_")+".json"))
	if err != nil {
		w.Write([]byte(`{"batchcomplete":true}`)) // no results
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (f *fakeWikipedia) last() *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

func newWikipediaBot(t *testing.T) (*Golem, *ChatSession, *fakeWikipedia) {
	t.Helper()
	f := &fakeWikipedia{}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	t.Setenv("WIKIPEDIA_URL", server.URL+"/w/api.php")
	t.Setenv("WIKIPEDIA_USER_AGENT", "test-assistant/1.0 (test@example.com)")
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	return g, g.CreateSession("wiki"), f
}

func TestWikipediaLookups(t *testing.T) {
	g, s, f := newWikipediaBot(t)

	r := ParseRichResponse(ask(t, g, s, "wikipedia ada lovelace"))
	if !strings.HasPrefix(r.Text, "Augusta Ada King, Countess of Lovelace (née Byron; 10 December 1815 – 27 November 1852), also known as Ada Lovelace, was an English mathematician") ||
		!strings.HasSuffix(r.Text, "She was the first to recognise the machine had applications beyond pure calculation.") {
		t.Errorf("summary: %q", r.Text)
	}
	if len(r.Buttons) != 1 || r.Buttons[0].URL != "https://en.wikipedia.org/wiki/Ada_Lovelace" {
		t.Errorf("buttons: %+v", r.Buttons)
	}
	req := f.last()
	if got := req.Header.Get("User-Agent"); got != "test-assistant/1.0 (test@example.com)" {
		t.Errorf("User-Agent %q", got)
	}
	if q := req.URL.Query(); q.Get("gsrsearch") != "ada lovelace" || q.Get("generator") != "search" || q.Get("explaintext") != "1" {
		t.Errorf("query %v", q)
	}

	// The pronunciation is left out, and a leading "the" isn't searched for
	r = ParseRichResponse(ask(t, g, s, "look up the Eiffel Tower"))
	if r.Text != "The Eiffel Tower is a lattice tower on the Champ de Mars in Paris, France. It is named after the engineer Gustave Eiffel, whose company designed and built the tower from 1887 to 1889." {
		t.Errorf("eiffel: %q", r.Text)
	}
	if got := f.last().URL.Query().Get("gsrsearch"); got != "Eiffel Tower" {
		t.Errorf("searched %q", got)
	}

	// Ambiguous: a button for each likely article
	r = ParseRichResponse(ask(t, g, s, "wikipedia mercury"))
	if r.Text != `"mercury" could mean a few things. Which one?` {
		t.Errorf("mercury: %q", r.Text)
	}
	want := []RichButton{
		{Text: "Freddie Mercury – British rock singer and songwriter (1946–1991)", Postback: "WIKIPEDIA PAGE Freddie Mercury"},
		{Text: "Mercury (element)", Postback: "WIKIPEDIA PAGE Mercury (element)"},
		{Text: "Mercury (planet)", Postback: "WIKIPEDIA PAGE Mercury (planet)"},
		{Text: "Mercury Records – American record label", Postback: "WIKIPEDIA PAGE Mercury Records"},
	}
	if !reflect.DeepEqual(r.Buttons, want) {
		t.Errorf("mercury buttons: %+v", r.Buttons)
	}
	r = ParseRichResponse(ask(t, g, s, "WIKIPEDIA PAGE Mercury (planet)"))
	if !strings.HasPrefix(r.Text, "Mercury is the first planet from the Sun") {
		t.Errorf("planet: %q", r.Text)
	}
	if got := f.last().URL.Query().Get("titles"); got != "Mercury (planet)" {
		t.Errorf("page titles %q", got)
	}

	if got := ask(t, g, s, "wikipedia asdfqwer zxcv"); got != `I couldn't find anything on Wikipedia about "asdfqwer zxcv".` {
		t.Errorf("none: %q", got)
	}

	wikipediaRetryWait = 10 * time.Millisecond
	t.Cleanup(func() { wikipediaRetryWait = 2 * time.Second })
	f.mu.Lock()
	f.status = http.StatusTooManyRequests
	before := len(f.requests)
	f.mu.Unlock()
	if got := ask(t, g, s, "wikipedia ada lovelace"); got != "I can't reach Wikipedia right now. Please try again in a minute." {
		t.Errorf("429: %q", got)
	}
	f.mu.Lock()
	if n := len(f.requests) - before; n != 2 {
		t.Errorf("want one retry after 429, got %d requests", n)
	}
	f.status = 0
	f.mu.Unlock()

}

func TestSummarizeWikiExtract(t *testing.T) {
	for extract, want := range map[string]string{
		"Mount St. Helens is an active volcano. It is in Washington. It erupted in 1980.":                                               "Mount St. Helens is an active volcano. It is in Washington.",
		"J. R. R. Tolkien was an English writer. He wrote The Hobbit.\nSecond paragraph.":                                               "J. R. R. Tolkien was an English writer. He wrote The Hobbit.",
		"Paris (French pronunciation: [paʁi] ) is the capital of France.":                                                               "Paris is the capital of France.",
		"A quasar ( KWAY-zar) is an extremely luminous active galactic nucleus (AGN). It is sometimes known as a quasi-stellar object.": "A quasar is an extremely luminous active galactic nucleus (AGN). It is sometimes known as a quasi-stellar object.",
		"Gif (  JIF; ) is a format.": "Gif is a format.",
		"Python is a language, e.g. for scripts. It was made in 1991. Guido made it.": "Python is a language, e.g. for scripts. It was made in 1991.",
		"":                                    "",
		strings.Repeat("word ", 120) + "end.": strings.TrimSpace(strings.Repeat("word ", 84)) + "…",
	} {
		if got := summarizeWikiExtract(extract); got != want {
			t.Errorf("%.40q:\n got  %q\n want %q", extract, got, want)
		}
	}
}

func TestChooseWikiPagePrefersTheExactTitle(t *testing.T) {
	pages := []wikiPage{
		{Title: "Lisbon Treaty", Index: 1, Extract: "The treaty."},
		{Title: "Lisbon", Index: 2, Extract: "Lisbon is the capital of Portugal."},
	}
	if got := chooseWikiPage("lisbon", pages); got.text != "Lisbon is the capital of Portugal." || got.set["wiki_title"] != "Lisbon" {
		t.Errorf("got %+v", got)
	}
	if got := chooseWikiPage("treaty of lisbon", pages); got.text != "The treaty." {
		t.Errorf("top result: %+v", got)
	}
}
