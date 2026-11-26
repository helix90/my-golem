package golem

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONFormatEmptyArrays(t *testing.T) {
	g := NewForTesting(t, false)
	for template, want := range map[string]string{
		`<jsonformat type="lists">[]</jsonformat>`:                                                                                                     "You have no lists yet.",
		`<jsonformat type="items">[]</jsonformat>`:                                                                                                     "No items in this list.",
		`<jsonformat>[]</jsonformat>`:                                                                                                                  "There's nothing here yet.",
		`<jsonformat>[{"id":1,"name":"groceries"}]</jsonformat>`:                                                                                       "You have one list: groceries.",
		`<jsonformat type="lists">[{"id":1,"name":"a"}]</jsonformat>`:                                                                                  "You have one list: a.",
		`<jsonformat>[{"id":1,"name":"a"},{"id":2,"name":"b"},{"id":3,"name":"c"}]</jsonformat>`:                                                       "You have 3 lists: a, b and c.",
		`<jsonformat>[{"id":1,"name":"a","description":"x"},{"id":2,"name":"b"}]</jsonformat>`:                                                         "You have 2 lists:\n• a – x\n• b",
		`<jsonformat button="LIST SHOW {list}">[{"id":1,"name":"a"},{"id":2,"name":"b &amp; c"}]</jsonformat>`:                                         "You have 2 lists: a and b & c.<button><text>a</text><postback>LIST SHOW a</postback></button><button><text>b &amp; c</text><postback>LIST SHOW b &amp; c</postback></button>",
		`<jsonformat type="lists" button="LIST SHOW {list}" only="buttons">[{"id":1,"name":"a"}]</jsonformat>`:                                         "<button><text>a</text><postback>LIST SHOW a</postback></button>",
		`<jsonformat type="lists" only="buttons">[]</jsonformat>`:                                                                                      "",
		`<jsonformat>{"id":1,"name":"groceries","items":[]}</jsonformat>`:                                                                              "Your groceries list is empty.",
		`<jsonformat>{"id":1,"name":"packing list","items":[{"id":5,"content":"socks"}]}</jsonformat>`:                                                 "Your packing list has 1 item:\n☐ socks",
		`<jsonformat>{"id":1,"name":"g","description":"weekly","items":[{"id":5,"content":"a","is_completed":1},{"id":6,"content":"b"}]}</jsonformat>`: "Your g list (weekly) has 2 items:\n☑ a\n☐ b",
	} {
		if got := g.ProcessTemplate(template, nil); got != want {
			t.Errorf("%s = %q, want %q", template, got, want)
		}
	}
}

// A new user with no lists, through the list commands
func TestShowListsWhenThereAreNone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/auth/login":
			fmt.Fprint(w, `{"access_token":"tok","token_type":"bearer"}`)
		default: // GET /users/me/lists and GET /users/me/lists/{name}/items
			fmt.Fprint(w, `[]`)
		}
	}))
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)

	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	ask(t, g, s, "list login helix pw")
	for input, want := range map[string]string{
		"show my lists":        "You have no lists yet.",
		"what lists do I have": "You have no lists yet.",
		"list groceries items": "No items in this list.",
	} {
		if got := ask(t, g, s, input); !strings.Contains(got, want) || strings.Contains(got, "[]") {
			t.Errorf("%q: %q, want %q", input, got, want)
		}
	}
}
