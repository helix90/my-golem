package golem

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeListAPI records list handler requests and answers them
type fakeListAPI struct {
	mu       sync.Mutex
	requests []fakeListRequest
}

type fakeListRequest struct {
	method, path string
	body         map[string]interface{}
	rawBody      string
}

func (f *fakeListAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	if r.URL.Path == "/auth/login" {
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
		return
	}
	req := fakeListRequest{method: r.Method, path: r.URL.EscapedPath(), rawBody: string(raw)}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req.body); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"detail":"invalid JSON"}`))
			f.record(req)
			return
		}
	}
	f.record(req)
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/users/me/lists":
		fmt.Fprintf(w, `{"id":3,"name":%q}`, req.body["name"])
	case r.Method == http.MethodPost:
		fmt.Fprintf(w, `{"id":7,"content":%q,"is_completed":0}`, req.body["content"])
	case r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet:
		_, _ = w.Write([]byte(`{"id":3,"name":"groceries","items":[{"id":7,"content":"x","is_completed":1}]}`))
	default:
		_, _ = w.Write([]byte(`{"id":7,"content":"x","is_completed":1}`))
	}
}

func (f *fakeListAPI) record(r fakeListRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r)
}

func (f *fakeListAPI) last() fakeListRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

func TestListRequestBodiesWithSpecialCharacters(t *testing.T) {
	fake := &fakeListAPI{}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("LIST_HANDLER_URL", server.URL)

	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")
	ask(t, g, s, "list login alice pw")
	s.Variables["list_current"] = "groceries" // so adding makes no lookup after the request

	steps := []struct {
		input, reply, method, path string
		body                       map[string]interface{}
	}{
		{`add 12" pizza to list groceries`, `Added "12" pizza" to your groceries list.`, "POST", "/users/me/lists/groceries/items",
			map[string]interface{}{"content": `12" pizza`}},
		{`list groceries add C:\temp\notes`, `Added "C:\temp\notes" to your groceries list.`, "POST", "/users/me/lists/groceries/items",
			map[string]interface{}{"content": `C:\temp\notes`}},
		{`list create Kid's stuff description toys, books & "games"`, `Created a list called "Kid's stuff" (toys, books & "games").`, "POST", "/users/me/lists",
			map[string]interface{}{"name": "Kid's stuff", "description": `toys, books & "games"`}},
		// A plain create must not reuse the description above
		{`create list holiday`, `Created a list called "holiday".`, "POST", "/users/me/lists",
			map[string]interface{}{"name": "holiday", "description": ""}},
		{`list groceries update item 7 to 2% milk "organic"`, "", "PUT", "/users/me/lists/groceries/items/7",
			map[string]interface{}{"content": `2% milk "organic"`}},
		{`list groceries delete item 7`, "", "DELETE", "/users/me/lists/groceries/items/7", nil},
	}
	for _, step := range steps {
		got := ask(t, g, s, step.input)
		if step.reply != "" && !strings.Contains(got, step.reply) {
			t.Errorf("%q: reply %q, want it to contain %q", step.input, got, step.reply)
		}
		req := fake.last()
		if req.method != step.method || req.path != step.path {
			t.Errorf("%q: request %s %s, want %s %s", step.input, req.method, req.path, step.method, step.path)
		}
		for key, want := range step.body {
			if req.body[key] != want {
				t.Errorf("%q: body %s: %s = %q, want %q", step.input, req.rawBody, key, req.body[key], want)
			}
		}
	}

	// Ticking an item off sends a PATCH and then shows the list again
	ask(t, g, s, "list groceries toggle item 7")
	fake.mu.Lock()
	n := len(fake.requests)
	patch, show := fake.requests[n-2], fake.requests[n-1]
	fake.mu.Unlock()
	if patch.method != "PATCH" || patch.path != "/users/me/lists/groceries/items/7" || show.method != "GET" {
		t.Errorf("toggle sent %s %s then %s %s", patch.method, patch.path, show.method, show.path)
	}
}
