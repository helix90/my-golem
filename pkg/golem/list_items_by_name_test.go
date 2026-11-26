package golem

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeItemsAPI holds one list's items and implements the list handler's
// item search, update and delete
type fakeItemsAPI struct {
	mu       sync.Mutex
	items    map[int]string // id -> content
	done     map[int]int
	order    []int
	noSearch bool // an older list handler, before items/search
	changes  []string
}

func newFakeItemsAPI(contents ...string) *fakeItemsAPI {
	f := &fakeItemsAPI{items: map[int]string{}, done: map[int]int{}}
	for i, content := range contents {
		id := 10 + i
		f.items[id] = content
		f.order = append(f.order, id)
	}
	return f
}

func (f *fakeItemsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const prefix = "/users/me/lists/groceries/items"
	path := r.URL.Path
	switch {
	case path == "/auth/login":
		fmt.Fprint(w, `{"access_token":"tok","token_type":"bearer"}`)
	case path == "/users/me/lists/groceries" && r.Method == http.MethodGet:
		fmt.Fprint(w, `{"id":1,"name":"groceries","items":[]}`)
	case path == prefix+"/search" && r.Method == http.MethodGet && !f.noSearch:
		q := strings.ToLower(r.URL.Query().Get("q"))
		var exact, starts, contains []map[string]interface{}
		for _, id := range f.order {
			content, ok := f.items[id]
			if !ok {
				continue
			}
			item := map[string]interface{}{"id": id, "content": content, "is_completed": f.done[id]}
			text := strings.ToLower(content)
			switch {
			case text == q:
				exact = append(exact, item)
			case strings.HasPrefix(text, q):
				starts = append(starts, item)
			case strings.Contains(text, q):
				contains = append(contains, item)
			}
		}
		all := append(append(append([]map[string]interface{}{}, exact...), starts...), contains...)
		_ = json.NewEncoder(w).Encode(all)
	case strings.HasPrefix(path, prefix+"/"):
		id, err := strconv.Atoi(strings.TrimPrefix(path, prefix+"/"))
		if err != nil {
			w.WriteHeader(http.StatusMethodNotAllowed) // as FastAPI answers GET .../items/search without the route
			return
		}
		if _, ok := f.items[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			var update map[string]interface{}
			if err := json.Unmarshal(body, &update); err != nil {
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			done, ok := update["is_completed"].(float64)
			if !ok {
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			f.done[id] = int(done)
			f.changes = append(f.changes, fmt.Sprintf("PUT %d %s", id, body))
			fmt.Fprintf(w, `{"id":%d,"content":%q,"is_completed":%d}`, id, f.items[id], f.done[id])
		case http.MethodDelete:
			delete(f.items, id)
			f.changes = append(f.changes, fmt.Sprintf("DELETE %d", id))
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeItemsAPI) lastChange() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.changes) == 0 {
		return ""
	}
	return f.changes[len(f.changes)-1]
}

func newItemsByNameBot(t *testing.T, f *fakeItemsAPI) (*Golem, *ChatSession) {
	t.Helper()
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	t.Setenv("LIST_HANDLER_URL", server.URL)
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	ask(t, g, s, "list login helix pw")
	return g, s
}

func TestListItemsByName(t *testing.T) {
	f := newFakeItemsAPI("Oat milk", "milk", "Eggs", "Bread")
	g, s := newItemsByNameBot(t, f)

	steps := []struct{ input, reply, change string }{
		// The exact match wins over "Oat milk"
		{"list groceries check off milk", `Ticked off "milk" on your groceries list.`, `PUT 11 {"is_completed":1}`},
		{"check off OAT on list groceries", `Ticked off "Oat milk" on your groceries list.`, `PUT 10 {"is_completed":1}`},
		{"list groceries cross off milk", `Ticked off "milk"`, `PUT 11 {"is_completed":1}`},
		{"list groceries uncheck milk", `Unticked "milk" on your groceries list.`, `PUT 11 {"is_completed":0}`},
		{"remove eggs from list groceries", `Removed "Eggs" from your groceries list.`, `DELETE 12`},
	}
	for _, step := range steps {
		if got := ask(t, g, s, step.input); got != step.reply && !strings.HasPrefix(got, step.reply) {
			t.Errorf("%q: %q, want %q", step.input, got, step.reply)
		}
		if got := f.lastChange(); got != step.change {
			t.Errorf("%q: change %q, want %q", step.input, got, step.change)
		}
	}

	before := f.lastChange()
	if got := ask(t, g, s, "list groceries check off cheese"); got != `I couldn't find "cheese" on your groceries list.` {
		t.Errorf("not found: %q", got)
	}
	// Eggs was removed above
	if got := ask(t, g, s, "list groceries remove eggs"); got != `I couldn't find "eggs" on your groceries list.` {
		t.Errorf("removed item: %q", got)
	}
	if f.lastChange() != before {
		t.Errorf("nothing should change when no item matches, got %q", f.lastChange())
	}
}

// With a list handler from before items/search, nothing is changed: the
// search fails rather than matching the wrong item
func TestListItemsByNameWithoutSearch(t *testing.T) {
	f := newFakeItemsAPI("Oat milk", "milk")
	f.noSearch = true
	g, s := newItemsByNameBot(t, f)
	if got := ask(t, g, s, "list groceries check off milk"); got != `I couldn't find "milk" on your groceries list.` {
		t.Errorf("got %q", got)
	}
	if f.lastChange() != "" {
		t.Errorf("an item was changed: %q", f.lastChange())
	}
}
