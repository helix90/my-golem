package golem

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// fakeListsAPI holds lists of items by name, found ignoring case as the
// list handler finds them (it also ignores plurals; that is tested there)
type fakeListsAPI struct {
	mu       sync.Mutex
	order    []string
	items    map[string][]string
	requests []string
}

func newFakeListsAPI(lists ...string) *fakeListsAPI {
	f := &fakeListsAPI{items: map[string][]string{}}
	for _, name := range lists {
		f.order = append(f.order, name)
		f.items[name] = nil
	}
	return f
}

func (f *fakeListsAPI) find(name string) (string, bool) {
	for _, list := range f.order {
		if strings.EqualFold(list, name) {
			return list, true
		}
	}
	return "", false
}

func (f *fakeListsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := r.URL.EscapedPath()
	if path == "/auth/login" {
		fmt.Fprint(w, `{"access_token":"tok","token_type":"bearer"}`)
		return
	}
	f.requests = append(f.requests, r.Method+" "+path)
	parts := strings.Split(strings.TrimPrefix(path, "/users/me/lists"), "/")
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		var lists []map[string]interface{}
		for i, name := range f.order {
			lists = append(lists, map[string]interface{}{"id": i + 1, "name": name, "description": ""})
		}
		if lists == nil {
			lists = []map[string]interface{}{}
		}
		_ = json.NewEncoder(w).Encode(lists)
	case len(parts) == 1 && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.order = append(f.order, body["name"])
		f.items[body["name"]] = nil
		fmt.Fprintf(w, `{"id":%d,"name":%q}`, len(f.order), body["name"])
	default:
		typed, _ := url.PathUnescape(parts[1])
		name, ok := f.find(typed)
		if !ok {
			http.Error(w, `{"detail":"List not found"}`, http.StatusNotFound)
			return
		}
		switch {
		case len(parts) == 2 && r.Method == http.MethodGet:
			items := []map[string]interface{}{}
			for i, content := range f.items[name] {
				items = append(items, map[string]interface{}{"id": i + 1, "content": content, "is_completed": 0})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "name": name, "description": "", "items": items})
		case len(parts) == 3 && parts[2] == "items" && r.Method == http.MethodPost:
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.items[name] = append(f.items[name], body["content"])
			fmt.Fprintf(w, `{"id":%d,"content":%q}`, len(f.items[name]), body["content"])
		case len(parts) == 4 && parts[3] == "search":
			fmt.Fprint(w, `[]`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (f *fakeListsAPI) snapshot() ([]string, map[string][]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := map[string][]string{}
	for name, list := range f.items {
		items[name] = append([]string(nil), list...)
	}
	return append([]string(nil), f.order...), items
}

func newListConversationBot(t *testing.T, f *fakeListsAPI) (*Golem, *ChatSession) {
	t.Helper()
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	t.Setenv("LIST_HANDLER_URL", server.URL)
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "list login helix pw"); !strings.Contains(got, "Logged in") {
		t.Fatalf("login: %q", got)
	}
	return g, s
}

func TestListOffersToCreateAMissingList(t *testing.T) {
	f := newFakeListsAPI("hardware")
	g, s := newListConversationBot(t, f)

	got := ParseRichResponse(ask(t, g, s, "list groceries add milk"))
	if got.Text != `You don't have a list called "groceries" yet. Should I create it and add "milk" to it?` {
		t.Errorf("offer: %q", got.Text)
	}
	wantButtons := []RichButton{{Text: "Yes, create it", Postback: "LIST CONFIRM CREATE"}, {Text: "No", Postback: "LIST CANCEL CREATE"}}
	if !reflect.DeepEqual(got.Buttons, wantButtons) {
		t.Errorf("buttons: %+v", got.Buttons)
	}
	if reply := ask(t, g, s, "yes please"); reply != `Created a list called "groceries". Added "milk" to your groceries list.` {
		t.Errorf("yes: %q", reply)
	}
	lists, items := f.snapshot()
	if !reflect.DeepEqual(lists, []string{"hardware", "groceries"}) || !reflect.DeepEqual(items["groceries"], []string{"milk"}) {
		t.Errorf("lists %v, items %v", lists, items)
	}
	if s.Variables["list_current"] != "groceries" {
		t.Errorf("list_current = %q", s.Variables["list_current"])
	}

	// Showing a missing list offers to create it; the button works later on
	if reply := ask(t, g, s, "show list packing"); !strings.HasPrefix(ParseRichResponse(reply).Text, `You don't have a list called "packing". Should I create it?`) {
		t.Errorf("show: %q", reply)
	}
	ask(t, g, s, "what time is it")
	if reply := ask(t, g, s, "LIST CONFIRM CREATE"); reply != `Created a list called "packing".` {
		t.Errorf("button: %q", reply)
	}
	if reply := ask(t, g, s, "LIST CONFIRM CREATE"); reply != "There's no list waiting to be created." {
		t.Errorf("again: %q", reply)
	}

	// "no" creates nothing
	ask(t, g, s, "list books add dune")
	if reply := ask(t, g, s, "no"); reply != "Okay, I won't create it." {
		t.Errorf("no: %q", reply)
	}
	if lists, _ := f.snapshot(); len(lists) != 3 {
		t.Errorf("lists after no: %v", lists)
	}
}

// The reply and list_current use the list's real name, which is looked up
// only when the name typed isn't the list in use
func TestListChangesUseTheRealName(t *testing.T) {
	f := newFakeListsAPI("groceries")
	g, s := newListConversationBot(t, f)
	if reply := ask(t, g, s, "list GROCERIES add milk"); reply != `Added "milk" to your groceries list.` {
		t.Errorf("add: %q", reply)
	}
	if got := f.requests; !reflect.DeepEqual(got, []string{"POST /users/me/lists/GROCERIES/items", "GET /users/me/lists/GROCERIES"}) {
		t.Errorf("requests: %v", got)
	}
	if s.Variables["list_current"] != "groceries" {
		t.Errorf("list_current = %q", s.Variables["list_current"])
	}
	f.requests = nil
	if reply := ask(t, g, s, "list groceries add eggs"); reply != `Added "eggs" to your groceries list.` {
		t.Errorf("add again: %q", reply)
	}
	if got := f.requests; !reflect.DeepEqual(got, []string{"POST /users/me/lists/groceries/items"}) {
		t.Errorf("requests for the list in use: %v", got)
	}
}

func TestListShowAndLookup(t *testing.T) {
	f := newFakeListsAPI("groceries", "hardware")
	g, s := newListConversationBot(t, f)

	// Showing a list makes it the list in use, under its real name
	reply := ParseRichResponse(ask(t, g, s, "list show GROCERIES"))
	if reply.Text != "Your groceries list is empty." {
		t.Errorf("show: %q", reply.Text)
	}
	if s.Variables["list_current"] != "groceries" {
		t.Errorf("list_current = %q", s.Variables["list_current"])
	}

	// Every list is a button
	reply = ParseRichResponse(ask(t, g, s, "show my lists"))
	if reply.Text != "You have 2 lists: groceries and hardware." {
		t.Errorf("lists: %q", reply.Text)
	}
	if want := []RichButton{{Text: "groceries", Postback: "LIST SHOW groceries"}, {Text: "hardware", Postback: "LIST SHOW hardware"}}; !reflect.DeepEqual(reply.Buttons, want) {
		t.Errorf("list buttons: %+v", reply.Buttons)
	}

	for name, want := range map[string]string{"Hardware": "found hardware", "unicorns": "missing "} {
		ask(t, g, s, "list lookup "+name)
		if got := s.Variables["list_lookup_status"] + " " + s.Variables["found_list_name"]; got != want {
			t.Errorf("lookup %s: %q, want %q", name, got, want)
		}
	}

	// An item on a list that isn't there
	reply = ParseRichResponse(ask(t, g, s, "list unicorns check off milk"))
	if reply.Text != `You don't have a list called "unicorns". Here are the lists you have:` {
		t.Errorf("not found: %q", reply.Text)
	}
	want := []RichButton{
		{Text: "groceries", Postback: "LIST SHOW groceries"},
		{Text: "hardware", Postback: "LIST SHOW hardware"},
		{Text: `➕ Create "unicorns"`, Postback: "LIST CREATE unicorns"},
	}
	if !reflect.DeepEqual(reply.Buttons, want) {
		t.Errorf("not found buttons: %+v", reply.Buttons)
	}
}

func TestListLookupWithoutLogin(t *testing.T) {
	f := newFakeListsAPI("groceries")
	server := httptest.NewServer(f)
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	if got := ask(t, g, s, "list lookup groceries"); got != "" || s.Variables["list_lookup_status"] != "login" {
		t.Errorf("got %q, status %q", got, s.Variables["list_lookup_status"])
	}
	if len(f.requests) != 0 {
		t.Errorf("requests without a login: %v", f.requests)
	}
}

func TestListChoiceButtons(t *testing.T) {
	f := newFakeListsAPI("groceries", "hardware")
	g, s := newListConversationBot(t, f)
	s.Variables["pending_add_item"] = "milk"
	buttons := ParseRichResponse(ask(t, g, s, "list choice buttons")).Buttons
	if want := []RichButton{{Text: "groceries", Postback: "LIST ADD PENDING TO groceries"}, {Text: "hardware", Postback: "LIST ADD PENDING TO hardware"}}; !reflect.DeepEqual(buttons, want) {
		t.Errorf("buttons: %+v", buttons)
	}
	if reply := ask(t, g, s, "LIST ADD PENDING TO hardware"); reply != `Added "milk" to your hardware list.` {
		t.Errorf("add: %q", reply)
	}
	if reply := ask(t, g, s, "LIST ADD PENDING TO hardware"); reply != "What should I add to your hardware list?" {
		t.Errorf("again: %q", reply)
	}
}
