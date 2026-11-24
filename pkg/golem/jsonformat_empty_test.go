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
		`<jsonformat type="lists">[]</jsonformat>`:                    "You have no lists yet.",
		`<jsonformat type="items">[]</jsonformat>`:                    "No items in this list.",
		`<jsonformat>[]</jsonformat>`:                                 "There's nothing here yet.",
		`<jsonformat>[{"id":1,"name":"groceries"}]</jsonformat>`:      "You have 1 list:\n1. groceries (ID: 1)",
		`<jsonformat type="lists">[{"id":1,"name":"a"}]</jsonformat>`: "You have 1 list:\n1. a (ID: 1)",
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
