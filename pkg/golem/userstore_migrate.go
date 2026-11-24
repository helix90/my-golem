package golem

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// legacyUserLookups are the categories the earlier persistence AIML learned
// with <learnf> to remember things about a user, such as
// "GET LIST PASSWORD FOR 123456789", and the predicates that hold them now.
// Any user could match those patterns, so they are moved to the UserStore.
var legacyUserLookups = []struct {
	prefix    string
	predicate string
}{
	{"GET LIST USERNAME FOR ", "user_list_username"},
	{"GET LIST PASSWORD FOR ", "user_list_password"},
	{"GET NAME FOR ", "user_name"},
	{"GET LOCATION FOR ", "user_location"},
	{"GET LATITUDE FOR ", "user_latitude"},
	{"GET LONGITUDE FOR ", "user_longitude"},
}

// legacyUserLookup reports the user and predicate a learned category stored,
// if it is one of legacyUserLookups
func legacyUserLookup(category Category) (user, predicate string, ok bool) {
	if category.That != "" || category.Topic != "" {
		return "", "", false
	}
	pattern := NormalizePattern(category.Pattern)
	for _, lookup := range legacyUserLookups {
		if !strings.HasPrefix(pattern, lookup.prefix) {
			continue
		}
		user = strings.TrimPrefix(pattern, lookup.prefix)
		if user == "" || strings.ContainsAny(user, " *_^#$<>") {
			return "", "", false
		}
		return user, lookup.predicate, true
	}
	return "", "", false
}

// MigrateLegacyUserData moves what the earlier persistence AIML remembered
// about users with <learnf> into the UserStore, and deletes those categories
// from the knowledge base, the learned categories file, and its backups. A
// value already in the store is kept. It returns how many categories it
// moved. It is safe to run at every start.
func (g *Golem) MigrateLegacyUserData() (int, error) {
	g.kbMu.Lock()
	defer g.kbMu.Unlock()

	store := g.userStore
	if store == nil {
		return 0, fmt.Errorf("no user store set")
	}
	if g.persistentLearning == nil || g.persistentLearning.StoragePath == "" {
		return 0, nil
	}

	mainFile := filepath.Join(g.persistentLearning.StoragePath, "learned_categories.json")
	data, err := g.persistentLearning.loadFromFile(mainFile)
	if os.IsNotExist(err) || (err != nil && !fileExists(mainFile)) {
		data = PersistentLearningData{}
	} else if err != nil {
		return 0, err
	}

	moved := 0
	kept := []PersistentCategory{}
	for _, pc := range data.Categories {
		user, predicate, ok := legacyUserLookup(pc.Category)
		if !ok {
			kept = append(kept, pc)
			continue
		}
		if _, exists := store.Get(user, predicate); !exists {
			value := strings.TrimSpace(html.UnescapeString(pc.Category.Template))
			if err := store.Set(user, predicate, value); err != nil {
				return moved, err
			}
		}
		if g.aimlKB != nil {
			g.aimlKB.removeCategoryByKey(categoryKey(pc.Category))
		}
		moved++
	}
	if moved > 0 {
		data.Categories = kept
		data.TotalLearned = len(kept)
		if err := g.persistentLearning.saveToFile(mainFile, data); err != nil {
			return moved, err
		}
	}

	// Learned categories that were never saved to the file, and stale
	// copies in the backups, hold the same data
	if g.aimlKB != nil {
		for _, category := range g.aimlKB.snapshotCategories() {
			if _, _, ok := legacyUserLookup(category); ok {
				g.aimlKB.removeCategoryByKey(categoryKey(category))
			}
		}
	}
	if err := g.scrubLegacyUserDataFromBackups(); err != nil {
		return moved, err
	}
	return moved, nil
}

// scrubLegacyUserDataFromBackups removes legacyUserLookups categories from
// every backup of the learned categories file
func (g *Golem) scrubLegacyUserDataFromBackups() error {
	plm := g.persistentLearning
	if plm.BackupPath == "" {
		return nil
	}
	files, err := filepath.Glob(filepath.Join(plm.BackupPath, "learned_categories_*.json"))
	if err != nil {
		return err
	}
	for _, file := range files {
		data, err := plm.loadFromFile(file)
		if err != nil {
			return fmt.Errorf("failed to read backup %s: %v", file, err)
		}
		kept := []PersistentCategory{}
		for _, pc := range data.Categories {
			if _, _, ok := legacyUserLookup(pc.Category); !ok {
				kept = append(kept, pc)
			}
		}
		if len(kept) == len(data.Categories) {
			continue
		}
		data.Categories = kept
		data.TotalLearned = len(kept)
		if err := plm.saveToFile(file, data); err != nil {
			return fmt.Errorf("failed to clean backup %s: %v", file, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
