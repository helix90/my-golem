package golem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUserStoreRoundTripAndEncryption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "users.json")
	store, err := OpenUserStore(path, "test key")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("111", map[string]string{
		"user_name":          "Alice",
		"user_list_password": "s3cret",
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "s3cret") {
		t.Errorf("secret saved in plain text: %s", raw)
	}
	if !strings.Contains(string(raw), "Alice") {
		t.Errorf("non-secret value missing from file: %s", raw)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("store file mode = %v, %v; want 0600", info.Mode().Perm(), err)
	}

	reopened, err := OpenUserStore(path, "test key")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := reopened.Get("111", "user_list_password"); got != "s3cret" {
		t.Errorf("reopened password = %q, want s3cret", got)
	}
	if _, err := OpenUserStore(path, "wrong key"); err == nil {
		t.Error("opening with the wrong key should fail")
	}
	if _, err := OpenUserStore(path, ""); err == nil {
		t.Error("opening an encrypted store without a key should fail")
	}

	if err := reopened.Set("111", "user_name", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Get("111", "user_name"); ok {
		t.Error("setting an empty value should delete the predicate")
	}
}

func TestUserStorePatterns(t *testing.T) {
	store, err := OpenUserStore(filepath.Join(t.TempDir(), "users.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{"user_name": true, "USER_LOCATION": true, "list_access_token": false, "mood": false} {
		if got := store.IsPersistent(name); got != want {
			t.Errorf("IsPersistent(%q) = %v, want %v", name, got, want)
		}
	}
	for name, want := range map[string]bool{"user_list_password": true, "list_access_token": true, "api_secret": true, "user_name": false} {
		if got := store.IsSecret(name); got != want {
			t.Errorf("IsSecret(%q) = %v, want %v", name, got, want)
		}
	}
	redacted := store.RedactedVariables(map[string]string{"user_list_password": "x", "user_name": "Alice"})
	if redacted["user_list_password"] != "(hidden)" || redacted["user_name"] != "Alice" {
		t.Errorf("RedactedVariables = %v", redacted)
	}
}

const userStoreTestAIML = `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category><pattern>MY NAME IS *</pattern><template><think><set name="user_name"><star/></set><set name="mood">set</set></think>Hi <star/></template></category>
<category><pattern>WHAT IS MY NAME</pattern><template>NAME=<get name="user_name"/></template></category>
<category><pattern>FORGET MY NAME</pattern><template><think><set name="user_name"></set></think>forgotten</template></category>
</aiml>`

func newUserStoreTestBot(t *testing.T, storePath string) *Golem {
	t.Helper()
	g := NewForTesting(t, false)
	if err := g.LoadAIMLFromString(userStoreTestAIML); err != nil {
		t.Fatal(err)
	}
	store, err := OpenUserStore(storePath, "")
	if err != nil {
		t.Fatal(err)
	}
	g.SetUserStore(store)
	return g
}

func ask(t *testing.T, g *Golem, s *ChatSession, input string) string {
	t.Helper()
	r, err := g.ProcessInput(input, s)
	if err != nil {
		t.Fatalf("%q: %v", input, err)
	}
	return strings.TrimSpace(r)
}

func TestUserPredicatesPersistPerUser(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "users.json")
	g := newUserStoreTestBot(t, storePath)

	alice := g.CreateSession("chat_1")
	alice.UserID = "111"
	ask(t, g, alice, "my name is Alice")

	bob := g.CreateSession("chat_2")
	bob.UserID = "222"
	if got := ask(t, g, bob, "what is my name"); got != "NAME=" {
		t.Errorf("bob sees %q, want no name", got)
	}

	// A restart: new engine, new session, same store file
	g2 := newUserStoreTestBot(t, storePath)
	again := g2.CreateSession("chat_1")
	again.UserID = "111"
	if got := ask(t, g2, again, "what is my name"); got != "NAME=Alice" {
		t.Errorf("after restart got %q, want NAME=Alice", got)
	}
	// Only persistent predicates are stored
	if _, ok := g2.UserStore().Get("111", "mood"); ok {
		t.Error("non-persistent predicate was stored")
	}

	ask(t, g2, again, "forget my name")
	if _, ok := g2.UserStore().Get("111", "user_name"); ok {
		t.Error("clearing a predicate should delete it from the store")
	}
}

func TestUserPredicatesInGroupChat(t *testing.T) {
	g := newUserStoreTestBot(t, filepath.Join(t.TempDir(), "users.json"))
	group := g.CreateSession("group")

	group.UserID = "111"
	ask(t, g, group, "my name is Alice")
	group.UserID = "222"
	ask(t, g, group, "my name is Bob")

	group.UserID = "111"
	if got := ask(t, g, group, "what is my name"); got != "NAME=Alice" {
		t.Errorf("alice in group got %q", got)
	}
	group.UserID = "333"
	if got := ask(t, g, group, "what is my name"); got != "NAME=" {
		t.Errorf("a new member sees %q, want no name", got)
	}
}

func TestMigrateLegacyUserData(t *testing.T) {
	learnedDir := t.TempDir()
	legacy := []Category{
		{Pattern: "GET LIST PASSWORD FOR 111", Template: "s3cret"},
		{Pattern: "GET LIST USERNAME FOR 111", Template: "alice"},
		{Pattern: "GET LOCATION FOR 111", Template: "Paris &amp; around"},
		{Pattern: "FAVORITE COLOR", Template: "blue"},
	}
	plm := NewPersistentLearningManager(learnedDir)
	if err := plm.SavePersistentCategories(legacy, "learnf"); err != nil {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(filepath.Join(learnedDir, "backups", "*.json"))
	if len(backups) == 0 {
		t.Fatal("expected a backup to exist")
	}
	time.Sleep(10 * time.Millisecond)

	g := NewForTestingWithDir(t, false, learnedDir)
	if err := g.LoadAIMLFromString(userStoreTestAIML); err != nil {
		t.Fatal(err)
	}
	if err := g.LoadPersistentCategories(); err != nil {
		t.Fatal(err)
	}
	if got := ask(t, g, g.CreateSession("x"), "get list password for 111"); got != "s3cret" {
		t.Fatalf("setup: legacy category not loaded, got %q", got)
	}

	store, err := OpenUserStore(filepath.Join(t.TempDir(), "users.json"), "k")
	if err != nil {
		t.Fatal(err)
	}
	g.SetUserStore(store)
	moved, err := g.MigrateLegacyUserData()
	if err != nil {
		t.Fatal(err)
	}
	if moved != 3 {
		t.Errorf("moved %d categories, want 3", moved)
	}
	for name, want := range map[string]string{"user_list_password": "s3cret", "user_list_username": "alice", "user_location": "Paris & around"} {
		if got, _ := store.Get("111", name); got != want {
			t.Errorf("store %s = %q, want %q", name, got, want)
		}
	}

	// Gone from the knowledge base...
	r, _ := g.ProcessInput("get list password for 111", g.CreateSession("y"))
	if strings.Contains(r, "s3cret") {
		t.Errorf("password still reachable after migration: %q", r)
	}
	// ...and from the file and every backup, while other learning stays
	files, _ := filepath.Glob(filepath.Join(learnedDir, "backups", "*.json"))
	files = append(files, filepath.Join(learnedDir, "learned_categories.json"))
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "s3cret") || strings.Contains(string(raw), "GET LIST") {
			t.Errorf("%s still holds legacy user data", file)
		}
		var data PersistentLearningData
		if err := json.Unmarshal(raw, &data); err != nil {
			t.Fatal(err)
		}
		if len(data.Categories) != 1 || data.Categories[0].Category.Pattern != "FAVORITE COLOR" {
			t.Errorf("%s: other learned categories should be kept, got %+v", file, data.Categories)
		}
	}

	// Running it again changes nothing
	if moved, err := g.MigrateLegacyUserData(); err != nil || moved != 0 {
		t.Errorf("second migration moved %d, err %v", moved, err)
	}
}
