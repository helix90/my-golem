package golem

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// UserStore keeps predicates for each user, such as a Telegram user, across
// sessions and restarts. It is the place for what the bot remembers about a
// person: their name, location, saved credentials and settings.
//
// Unlike categories learned with <learnf>, nothing in the store can be
// reached by matching input, so one user cannot read another's data.
//
// A session uses the store when its UserID is set: before each input, the
// user's persistent predicates are copied into the session, and afterwards
// any that changed are saved. Which predicates persist is set by glob
// patterns (default "user_*"); which are secret by another set (default
// "*password*", "*secret*", "*token*"). Secret values are encrypted in the
// file when the store has a key, and should be left out of anything shown
// to users or written to logs.
type UserStore struct {
	mu      sync.Mutex
	path    string
	aead    cipher.AEAD // nil when the store has no key
	users   map[string]map[string]string
	persist []string
	secret  []string
}

// DefaultPersistPatterns name the predicates a UserStore keeps by default
var DefaultPersistPatterns = []string{"user_*"}

// DefaultSecretPatterns name the predicates a UserStore treats as secret by
// default
var DefaultSecretPatterns = []string{"*password*", "*secret*", "*token*"}

// userStoreFile is the format of the store's file
type userStoreFile struct {
	Version int                          `json:"version"`
	Users   map[string]map[string]string `json:"users"`
}

const (
	userStoreVersion  = 1
	encryptedPrefix   = "enc:v1:"
	userStoreFileMode = 0o600
)

// OpenUserStore opens the store kept in the file at path, creating it when
// it does not exist yet. With a non-empty key, secret values are encrypted
// with AES-256-GCM under a key derived from it; without one they are saved
// as they are, so keep the file private.
func OpenUserStore(path string, key string) (*UserStore, error) {
	store := &UserStore{
		path:    path,
		users:   make(map[string]map[string]string),
		persist: append([]string(nil), DefaultPersistPatterns...),
		secret:  append([]string(nil), DefaultSecretPatterns...),
	}
	if key != "" {
		sum := sha256.Sum256([]byte(key))
		block, err := aes.NewCipher(sum[:])
		if err != nil {
			return nil, err
		}
		if store.aead, err = cipher.NewGCM(block); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read user store: %v", err)
	}
	var file userStoreFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse user store %s: %v", path, err)
	}
	for user, predicates := range file.Users {
		values := make(map[string]string, len(predicates))
		for name, value := range predicates {
			if strings.HasPrefix(value, encryptedPrefix) {
				if value, err = store.decrypt(value); err != nil {
					return nil, fmt.Errorf("failed to decrypt %s for user %s: %v", name, user, err)
				}
			}
			values[name] = value
		}
		store.users[user] = values
	}
	return store, nil
}

// Encrypted reports whether the store encrypts secret values
func (s *UserStore) Encrypted() bool {
	return s.aead != nil
}

// SetPersistPatterns sets the glob patterns (as in path.Match, e.g.
// "user_*") of the predicates the store keeps
func (s *UserStore) SetPersistPatterns(patterns []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persist = append([]string(nil), patterns...)
}

// SetSecretPatterns sets the glob patterns of the predicates whose values
// are secret
func (s *UserStore) SetSecretPatterns(patterns []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secret = append([]string(nil), patterns...)
}

// IsPersistent reports whether the store keeps the predicate with this name
func (s *UserStore) IsPersistent(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return matchesAny(s.persist, name)
}

// IsSecret reports whether the predicate with this name holds a secret
func (s *UserStore) IsSecret(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return matchesAny(s.secret, name)
}

// matchesAny reports whether name matches one of the glob patterns, ignoring
// case
func matchesAny(patterns []string, name string) bool {
	name = strings.ToLower(name)
	for _, pattern := range patterns {
		if ok, _ := path.Match(strings.ToLower(pattern), name); ok {
			return true
		}
	}
	return false
}

// Get returns a user's predicate
func (s *UserStore) Get(user, name string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.users[user][name]
	return value, ok
}

// All returns a copy of a user's predicates
func (s *UserStore) All(user string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := make(map[string]string, len(s.users[user]))
	for name, value := range s.users[user] {
		values[name] = value
	}
	return values
}

// Users returns the users with at least one predicate, sorted
func (s *UserStore) Users() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]string, 0, len(s.users))
	for user := range s.users {
		users = append(users, user)
	}
	sort.Strings(users)
	return users
}

// Set saves a user's predicate. An empty value deletes it.
func (s *UserStore) Set(user, name, value string) error {
	return s.Update(user, map[string]string{name: value})
}

// Delete removes a user's predicate
func (s *UserStore) Delete(user, name string) error {
	return s.Update(user, map[string]string{name: ""})
}

// Update saves several of a user's predicates at once; an empty value
// deletes that predicate. The file is written once.
func (s *UserStore) Update(user string, changes map[string]string) error {
	if user == "" {
		return fmt.Errorf("no user given")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	for name, value := range changes {
		current, exists := s.users[user][name]
		switch {
		case value == "" && exists:
			delete(s.users[user], name)
			changed = true
		case value != "" && (!exists || current != value):
			if s.users[user] == nil {
				s.users[user] = make(map[string]string)
			}
			s.users[user][name] = value
			changed = true
		}
	}
	if len(s.users[user]) == 0 {
		delete(s.users, user)
	}
	if !changed {
		return nil
	}
	return s.save()
}

// DeleteUser removes everything the store keeps about a user
func (s *UserStore) DeleteUser(user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[user]; !exists {
		return nil
	}
	delete(s.users, user)
	return s.save()
}

// save writes the store to its file, replacing it atomically. The caller
// must hold s.mu.
func (s *UserStore) save() error {
	file := userStoreFile{Version: userStoreVersion, Users: make(map[string]map[string]string, len(s.users))}
	for user, predicates := range s.users {
		values := make(map[string]string, len(predicates))
		for name, value := range predicates {
			if s.aead != nil && matchesAny(s.secret, name) {
				encrypted, err := s.encrypt(value)
				if err != nil {
					return err
				}
				value = encrypted
			}
			values[name] = value
		}
		file.Users[user] = values
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create user store directory: %v", err)
	}
	tmp, err := os.CreateTemp(dir, ".users-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to write user store: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(userStoreFileMode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write user store: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("failed to write user store: %v", err)
	}
	return nil
}

func (s *UserStore) encrypt(value string) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := s.aead.Seal(nonce, nonce, []byte(value), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *UserStore) decrypt(value string) (string, error) {
	if s.aead == nil {
		return "", fmt.Errorf("the value is encrypted but the store has no key (set USER_STORE_KEY)")
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encryptedPrefix))
	if err != nil {
		return "", err
	}
	if len(sealed) < s.aead.NonceSize() {
		return "", fmt.Errorf("encrypted value is too short")
	}
	plain, err := s.aead.Open(nil, sealed[:s.aead.NonceSize()], sealed[s.aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("wrong key or corrupted value")
	}
	return string(plain), nil
}

// RedactedVariables returns a copy of variables with every secret value
// replaced, for showing to users or writing to logs
func (s *UserStore) RedactedVariables(variables map[string]string) map[string]string {
	redacted := make(map[string]string, len(variables))
	for name, value := range variables {
		if value != "" && s.IsSecret(name) {
			value = "(hidden)"
		}
		redacted[name] = value
	}
	return redacted
}

// SetUserStore makes sessions whose UserID is set keep their persistent
// predicates in store. Bot properties "userstore.persist" and
// "userstore.secret" (comma-separated glob patterns), when present, replace
// the store's patterns.
func (g *Golem) SetUserStore(store *UserStore) {
	g.kbMu.Lock()
	defer g.kbMu.Unlock()
	g.userStore = store
	g.applyUserStoreProperties()
}

// UserStore returns the store set with SetUserStore, or nil
func (g *Golem) UserStore() *UserStore {
	g.kbMu.RLock()
	defer g.kbMu.RUnlock()
	return g.userStore
}

// applyUserStoreProperties applies the userstore.* bot properties. The
// caller must hold kbMu.
func (g *Golem) applyUserStoreProperties() {
	if g.userStore == nil || g.aimlKB == nil {
		return
	}
	if patterns := splitPatterns(g.aimlKB.Properties["userstore.persist"]); len(patterns) > 0 {
		g.userStore.SetPersistPatterns(patterns)
	}
	if patterns := splitPatterns(g.aimlKB.Properties["userstore.secret"]); len(patterns) > 0 {
		g.userStore.SetSecretPatterns(patterns)
	}
}

func splitPatterns(value string) []string {
	var patterns []string
	for _, pattern := range strings.Split(value, ",") {
		if pattern = strings.TrimSpace(pattern); pattern != "" {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

// loadUserPredicates replaces the session's persistent predicates with the
// stored ones of the session's user, and returns them, so that
// saveUserPredicates can tell what changed. Persistent predicates already in
// the session (from another user, in a group chat) are removed first. The
// caller must hold the session's lock.
func (g *Golem) loadUserPredicates(session *ChatSession) map[string]string {
	store := g.userStore
	if store == nil || session.UserID == "" {
		return nil
	}
	if session.Variables == nil {
		session.Variables = make(map[string]string)
	}
	for name := range session.Variables {
		if store.IsPersistent(name) {
			delete(session.Variables, name)
		}
	}
	stored := store.All(session.UserID)
	for name, value := range stored {
		if store.IsPersistent(name) {
			session.Variables[name] = value
		}
	}
	return stored
}

// saveUserPredicates saves the session's persistent predicates that differ
// from before, the ones loadUserPredicates returned. A predicate set to ""
// is deleted. The caller must hold the session's lock.
func (g *Golem) saveUserPredicates(session *ChatSession, before map[string]string) {
	store := g.userStore
	if store == nil || session.UserID == "" {
		return
	}
	changes := make(map[string]string)
	for name, value := range session.Variables {
		if store.IsPersistent(name) && before[name] != value {
			changes[name] = value
		}
	}
	for name := range before {
		if _, exists := session.Variables[name]; !exists && store.IsPersistent(name) {
			changes[name] = ""
		}
	}
	if len(changes) == 0 {
		return
	}
	if err := store.Update(session.UserID, changes); err != nil {
		g.LogWarn("Failed to save predicates of user %s: %v", session.UserID, err)
	}
}
