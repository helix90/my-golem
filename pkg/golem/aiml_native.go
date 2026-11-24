package golem

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Package-level compiled regexes — avoids re-compilation inside hot paths.
var (
	patternSetTagRe   = regexp.MustCompile(`<set>([^<]+)</set>`)
	patternTopicTagRe = regexp.MustCompile(`<topic>([^<]+)</topic>`)
	whitespaceNormRe  = regexp.MustCompile(`\s+`)
	// (?s) so comments spanning multiple lines are removed too
	xmlCommentRe  = regexp.MustCompile(`(?s)<!--.*?-->`)
	xmlDeclRe     = regexp.MustCompile(`<\?xml[^>]*\?>`)
	aimlVersionRe = regexp.MustCompile(`<aiml[^>]*version=["']([^"']+)["']`)
	// Patterns are matched case-insensitively, so accept any letters (including
	// non-ASCII), digits, wildcards, tag syntax, and common punctuation.
	emptySetTagRe = regexp.MustCompile(`<set>\s*</set>`)
)

// AIML represents the root AIML document
type AIML struct {
	Version    string
	Categories []Category
}

// Category represents an AIML category (pattern-template pair)
type Category struct {
	Pattern   string
	Template  string
	That      string
	ThatIndex int // Index for that context (1-based, 0 means last response)
	Topic     string
}

// SetCollection represents an ordered set (maintains insertion order while ensuring uniqueness)
type SetCollection struct {
	Items []string        // Maintains insertion order
	Index map[string]bool // For O(1) uniqueness checking
}

// NewSetCollection creates a new empty set collection
func NewSetCollection() *SetCollection {
	return &SetCollection{
		Items: make([]string, 0),
		Index: make(map[string]bool),
	}
}

// AIMLKnowledgeBase stores the parsed AIML data for efficient searching
type AIMLKnowledgeBase struct {
	Categories     []Category
	Patterns       map[string]*Category
	Sets           map[string][]string // Sets: for pattern matching (e.g., <set name="colors">)
	Topics         map[string][]string
	TopicVars      map[string]map[string]string // TopicVars: topicName -> varName -> value
	Variables      map[string]string
	Properties     map[string]string
	Maps           map[string]map[string]string // Maps: mapName -> key -> value
	Lists          map[string][]string          // Lists: listName -> []values
	Arrays         map[string][]string          // Arrays: arrayName -> []values
	SetCollections map[string]*SetCollection    // SetCollections: setName -> ordered unique values
	Substitutions  map[string]map[string]string // Substitutions: substitutionName -> pattern -> replacement

	// mu guards the data that template tags read and modify at runtime
	// (Variables, TopicVars, Properties, Maps, Lists, Arrays, SetCollections),
	// since sessions may process templates concurrently
	mu sync.RWMutex

	// patternsMu guards Categories and Patterns, which <learn> and <unlearn>
	// modify while other sessions match against them. Matching holds the read
	// lock; adding and removing categories holds the write lock. It is separate
	// from mu so that matching never waits on template data tags.
	patternsMu sync.RWMutex

	// patternOrder gives each indexed category's position in Categories, so
	// that equally good matches are chosen by load order rather than by map
	// iteration order. Guarded by patternsMu.
	patternOrder map[*Category]int

	// graph is the graphmaster tree built from Patterns, which is rebuilt
	// when Patterns changes. graphPatternsID and graphSize identify the
	// Patterns it was built from. Guarded by graphMu.
	graphMu         sync.Mutex
	graph           *gmNode
	graphPatternsID uintptr
	graphSize       int
}

// removeCategory removes the category at index i. Patterns entries point into
// Categories, so removing an element shifts later categories under their
// pointers; the index is rebuilt to keep every key on its own category.
// The caller must hold patternsMu for writing.
func (kb *AIMLKnowledgeBase) removeCategory(i int) {
	kb.Categories = append(kb.Categories[:i], kb.Categories[i+1:]...)
	kb.rebuildPatternIndex()
}

// addOrUpdateCategory adds a category, or replaces the category already
// indexed under the same key. It reports whether an existing category was
// replaced.
func (kb *AIMLKnowledgeBase) addOrUpdateCategory(category Category) bool {
	kb.patternsMu.Lock()
	defer kb.patternsMu.Unlock()

	key := categoryKey(category)
	if existing, exists := kb.Patterns[key]; exists {
		*existing = category
		return true
	}

	// When append reallocates, the existing Patterns entries still point into
	// the old array, so the whole index has to move to the new one
	oldCap := cap(kb.Categories)
	kb.Categories = append(kb.Categories, category)
	if cap(kb.Categories) != oldCap {
		kb.rebuildPatternIndex()
	} else {
		last := &kb.Categories[len(kb.Categories)-1]
		kb.Patterns[key] = last
		if kb.patternOrder == nil {
			kb.patternOrder = make(map[*Category]int)
		}
		kb.patternOrder[last] = len(kb.Categories) - 1
		kb.addToGraphmaster(last)
	}
	return false
}

// removeCategoryByKey removes the first category with this key. It reports
// whether a category was found.
func (kb *AIMLKnowledgeBase) removeCategoryByKey(key string) bool {
	kb.patternsMu.Lock()
	defer kb.patternsMu.Unlock()

	for i, cat := range kb.Categories {
		if categoryKey(cat) == key {
			kb.removeCategory(i)
			return true
		}
	}
	return false
}

// categoryCount returns the number of categories
func (kb *AIMLKnowledgeBase) categoryCount() int {
	kb.patternsMu.RLock()
	defer kb.patternsMu.RUnlock()
	return len(kb.Categories)
}

// patternCount returns the number of indexed patterns
func (kb *AIMLKnowledgeBase) patternCount() int {
	kb.patternsMu.RLock()
	defer kb.patternsMu.RUnlock()
	return len(kb.Patterns)
}

// snapshotCategories returns a copy of Categories that is safe to use while
// other sessions learn and unlearn
func (kb *AIMLKnowledgeBase) snapshotCategories() []Category {
	kb.patternsMu.RLock()
	defer kb.patternsMu.RUnlock()
	return append([]Category(nil), kb.Categories...)
}

// rebuildPatternIndex rebuilds Patterns from Categories, so that every entry
// points at its category in kb.Categories. When categories share a key, the
// later one wins.
func (kb *AIMLKnowledgeBase) rebuildPatternIndex() {
	kb.invalidateGraphmaster()
	defer kb.graphmaster()
	kb.Patterns = make(map[string]*Category, len(kb.Categories))
	kb.patternOrder = make(map[*Category]int, len(kb.Categories))
	for i := range kb.Categories {
		kb.Patterns[categoryKey(kb.Categories[i])] = &kb.Categories[i]
		kb.patternOrder[&kb.Categories[i]] = i
	}
}

// loadOrder returns the category's position in Categories, or math.MaxInt if
// it is not known (for a Patterns map built without rebuildPatternIndex)
func (kb *AIMLKnowledgeBase) loadOrder(category *Category) int {
	if order, exists := kb.patternOrder[category]; exists {
		return order
	}
	return math.MaxInt
}

// lookupVariable looks up varKey in the topic's variables, then global
// variables, then bot properties
func (kb *AIMLKnowledgeBase) lookupVariable(topic string, varKey string) (string, bool) {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	if topic != "" {
		if value, exists := kb.TopicVars[topic][varKey]; exists {
			return value, true
		}
	}
	if value, exists := kb.Variables[varKey]; exists {
		return value, true
	}
	if value, exists := kb.Properties[varKey]; exists {
		return value, true
	}
	return "", false
}

// setVariable sets a global variable
func (kb *AIMLKnowledgeBase) setVariable(varKey string, value string) {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	if kb.Variables == nil {
		kb.Variables = make(map[string]string)
	}
	kb.Variables[varKey] = value
}

// hasSetCollection reports whether a set collection with this name exists
func (kb *AIMLKnowledgeBase) hasSetCollection(name string) bool {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	_, exists := kb.SetCollections[name]
	return exists
}

// NewAIMLKnowledgeBase creates a new knowledge base
func NewAIMLKnowledgeBase() *AIMLKnowledgeBase {
	return &AIMLKnowledgeBase{
		Patterns:       make(map[string]*Category),
		Sets:           make(map[string][]string),
		Topics:         make(map[string][]string),
		TopicVars:      make(map[string]map[string]string),
		Variables:      make(map[string]string),
		Properties:     make(map[string]string),
		Maps:           make(map[string]map[string]string),
		Lists:          make(map[string][]string),
		Arrays:         make(map[string][]string),
		SetCollections: make(map[string]*SetCollection),
		Substitutions:  make(map[string]map[string]string),
	}
}

// LoadAIML parses an AIML file using native Go string manipulation
// LoadAIMLFromString loads AIML from a string and returns the parsed knowledge base
func (g *Golem) LoadAIMLFromString(content string) error {
	g.LogDebug("Loading AIML from string")

	// Parse the AIML content
	aiml, err := g.parseAIML(content)
	if err != nil {
		return err
	}

	if err := g.validateAIML(aiml); err != nil {
		return fmt.Errorf("AIML validation failed: %v", err)
	}

	// Convert AIML to AIMLKnowledgeBase
	kb := g.aimlToKnowledgeBase(aiml)

	// Merge with existing knowledge base. Holding kbMu waits for running
	// requests, so nothing they learn is lost when the merged copy replaces it.
	g.kbMu.Lock()
	defer g.kbMu.Unlock()
	if g.aimlKB != nil {
		mergedKB, err := g.mergeKnowledgeBases(g.aimlKB, kb)
		if err != nil {
			return err
		}
		kb = mergedKB
	}
	g.aimlKB = kb

	g.LogDebug("Loaded AIML from string successfully")
	g.LogDebug("Total categories: %d", len(kb.Categories))
	g.LogDebug("Total patterns: %d", len(kb.Patterns))
	g.LogDebug("Total sets: %d", len(kb.Sets))
	g.LogDebug("Total topics: %d", len(kb.Topics))
	g.LogDebug("Total variables: %d", len(kb.Variables))
	g.LogDebug("Total properties: %d", len(kb.Properties))
	g.LogDebug("Total maps: %d", len(kb.Maps))

	return nil
}

// aimlToKnowledgeBase converts AIML to AIMLKnowledgeBase
func (g *Golem) aimlToKnowledgeBase(aiml *AIML) *AIMLKnowledgeBase {
	kb := &AIMLKnowledgeBase{
		Categories:     aiml.Categories,
		Patterns:       make(map[string]*Category),
		Sets:           make(map[string][]string),
		Topics:         make(map[string][]string),
		Variables:      make(map[string]string),
		Properties:     make(map[string]string),
		Maps:           make(map[string]map[string]string),
		Lists:          make(map[string][]string),
		Arrays:         make(map[string][]string),
		SetCollections: make(map[string]*SetCollection),
		Substitutions:  make(map[string]map[string]string),
	}

	kb.rebuildPatternIndex()

	return kb
}

// categoryKey returns the Patterns index key for a category. The key includes
// that, that index, and topic so categories sharing a pattern don't overwrite
// each other in the index.
func categoryKey(category Category) string {
	key := NormalizePattern(category.Pattern)
	if category.That != "" {
		key += "|THAT:" + NormalizePattern(category.That)
		if category.ThatIndex != 0 {
			key += fmt.Sprintf("|THATINDEX:%d", category.ThatIndex)
		}
	}
	if category.Topic != "" {
		key += "|TOPIC:" + strings.ToUpper(category.Topic)
	}
	return key
}

// mergeKnowledgeBases merges two knowledge bases
func (g *Golem) mergeKnowledgeBases(kb1, kb2 *AIMLKnowledgeBase) (*AIMLKnowledgeBase, error) {
	mergedKB := &AIMLKnowledgeBase{
		Categories:     make([]Category, 0),
		Patterns:       make(map[string]*Category),
		Sets:           make(map[string][]string),
		Topics:         make(map[string][]string),
		Variables:      make(map[string]string),
		Properties:     make(map[string]string),
		Maps:           make(map[string]map[string]string),
		Lists:          make(map[string][]string),
		Arrays:         make(map[string][]string),
		SetCollections: make(map[string]*SetCollection),
		Substitutions:  make(map[string]map[string]string),
	}

	// Copy from first knowledge base
	mergedKB.Categories = append(mergedKB.Categories, kb1.Categories...)
	for setName, members := range kb1.Sets {
		mergedKB.Sets[setName] = members
	}
	for topicName, patterns := range kb1.Topics {
		mergedKB.Topics[topicName] = patterns
	}
	for varName, value := range kb1.Variables {
		mergedKB.Variables[varName] = value
	}
	for propName, value := range kb1.Properties {
		mergedKB.Properties[propName] = value
	}
	for mapName, mapData := range kb1.Maps {
		mergedKB.Maps[mapName] = mapData
	}
	for listName, listData := range kb1.Lists {
		mergedKB.Lists[listName] = listData
	}
	for arrayName, arrayData := range kb1.Arrays {
		mergedKB.Arrays[arrayName] = arrayData
	}
	for setName, setData := range kb1.SetCollections {
		mergedKB.SetCollections[setName] = setData
	}
	for subName, subData := range kb1.Substitutions {
		mergedKB.Substitutions[subName] = subData
	}

	// Merge from second knowledge base
	mergedKB.Categories = append(mergedKB.Categories, kb2.Categories...)
	mergedKB.rebuildPatternIndex()
	for setName, members := range kb2.Sets {
		if mergedKB.Sets[setName] == nil {
			mergedKB.Sets[setName] = make([]string, 0)
		}
		mergedKB.Sets[setName] = append(mergedKB.Sets[setName], members...)
	}
	for topicName, patterns := range kb2.Topics {
		if mergedKB.Topics[topicName] == nil {
			mergedKB.Topics[topicName] = make([]string, 0)
		}
		mergedKB.Topics[topicName] = append(mergedKB.Topics[topicName], patterns...)
	}
	for varName, value := range kb2.Variables {
		mergedKB.Variables[varName] = value
	}
	for propName, value := range kb2.Properties {
		mergedKB.Properties[propName] = value
	}
	for mapName, mapData := range kb2.Maps {
		if mergedKB.Maps[mapName] == nil {
			mergedKB.Maps[mapName] = make(map[string]string)
		}
		for key, value := range mapData {
			mergedKB.Maps[mapName][key] = value
		}
	}
	for listName, listData := range kb2.Lists {
		if mergedKB.Lists[listName] == nil {
			mergedKB.Lists[listName] = make([]string, 0)
		}
		mergedKB.Lists[listName] = append(mergedKB.Lists[listName], listData...)
	}
	for arrayName, arrayData := range kb2.Arrays {
		if mergedKB.Arrays[arrayName] == nil {
			mergedKB.Arrays[arrayName] = make([]string, 0)
		}
		mergedKB.Arrays[arrayName] = append(mergedKB.Arrays[arrayName], arrayData...)
	}
	for setName, setData := range kb2.SetCollections {
		if mergedKB.SetCollections[setName] == nil {
			mergedKB.SetCollections[setName] = NewSetCollection()
		}
		// Merge items while maintaining uniqueness
		for _, item := range setData.Items {
			if !mergedKB.SetCollections[setName].Index[item] {
				mergedKB.SetCollections[setName].Items = append(mergedKB.SetCollections[setName].Items, item)
				mergedKB.SetCollections[setName].Index[item] = true
			}
		}
	}
	for subName, subData := range kb2.Substitutions {
		if mergedKB.Substitutions[subName] == nil {
			mergedKB.Substitutions[subName] = make(map[string]string)
		}
		for pattern, replacement := range subData {
			mergedKB.Substitutions[subName][pattern] = replacement
		}
	}

	return mergedKB, nil
}

func (g *Golem) LoadAIML(filename string) (*AIMLKnowledgeBase, error) {
	categories, err := g.loadAIMLCategories(filename)
	if err != nil {
		return nil, err
	}

	// Create knowledge base
	kb := NewAIMLKnowledgeBase()
	kb.Categories = categories

	// Load default properties
	err = g.loadDefaultProperties(kb)
	if err != nil {
		return nil, fmt.Errorf("failed to load default properties: %v", err)
	}

	kb.rebuildPatternIndex()

	g.LogInfo("Loaded %d AIML categories", len(kb.Categories))
	g.LogInfo("Loaded %d properties", len(kb.Properties))

	return kb, nil
}

// loadAIMLCategories reads, parses, and validates the categories in an AIML file
func (g *Golem) loadAIMLCategories(filename string) ([]Category, error) {
	g.LogInfo("Loading AIML file: %s", filename)

	content, err := g.LoadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load AIML file: %v", err)
	}

	aiml, err := g.parseAIML(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AIML: %v", err)
	}

	if err := g.validateAIML(aiml); err != nil {
		return nil, fmt.Errorf("AIML validation failed: %v", err)
	}

	return aiml.Categories, nil
}

// LoadAIMLFromDirectory loads all AIML files from a directory and merges them into a single knowledge base
func (g *Golem) LoadAIMLFromDirectory(dirPath string) (*AIMLKnowledgeBase, error) {
	g.LogInfo("Loading AIML files from directory: %s", dirPath)

	// Create a new knowledge base to merge all files into
	mergedKB := NewAIMLKnowledgeBase()

	// Load default properties first
	err := g.loadDefaultProperties(mergedKB)
	if err != nil {
		return nil, fmt.Errorf("failed to load default properties: %v", err)
	}

	// Walk through the directory to find all .aiml files
	var aimlFiles []string
	err = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .aiml file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".aiml") {
			aimlFiles = append(aimlFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(aimlFiles) == 0 {
		return nil, fmt.Errorf("no AIML files found in directory: %s", dirPath)
	}

	g.LogInfo("Found %d AIML files in directory", len(aimlFiles))

	// Load the categories from each AIML file
	for _, aimlFile := range aimlFiles {
		categories, err := g.loadAIMLCategories(aimlFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogWarn("Failed to load %s: %v", aimlFile, err)
			continue
		}
		mergedKB.Categories = append(mergedKB.Categories, categories...)
	}
	mergedKB.rebuildPatternIndex()

	// Load map files from the same directory
	maps, err := g.LoadMapsFromDirectory(dirPath)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load maps from directory: %v", err)
	} else {
		// Merge maps into the knowledge base
		for mapName, mapData := range maps {
			mergedKB.Maps[mapName] = mapData
		}
	}

	// Load set files from the same directory
	sets, err := g.LoadSetsFromDirectory(dirPath)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load sets from directory: %v", err)
	} else {
		// Merge sets into the knowledge base
		for setName, setMembers := range sets {
			mergedKB.AddSetMembers(setName, setMembers)
		}
	}

	// Load substitution files from the same directory
	substitutions, err := g.LoadSubstitutionsFromDirectory(dirPath)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load substitutions from directory: %v", err)
	} else {
		// Merge substitutions into the knowledge base
		for subName, subData := range substitutions {
			mergedKB.Substitutions[subName] = subData
		}
	}

	// Load properties files from the same directory
	properties, err := g.loadPropertiesFiles(dirPath)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load properties from directory: %v", err)
	} else {
		mergePropertiesFiles(mergedKB.Properties, properties)
	}

	// Load pdefaults files from the same directory
	pdefaults, err := g.LoadPDefaultsFromDirectory(dirPath)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load pdefaults from directory: %v", err)
	} else {
		// Merge pdefaults into the knowledge base (as default user properties)
		for pdefaultName, pdefaultData := range pdefaults {
			for key, value := range pdefaultData {
				// Store pdefaults as a special type of property with prefix
				mergedKB.Properties["pdefault."+pdefaultName+"."+key] = value
			}
		}
	}

	g.LogInfo("Merged %d AIML files into knowledge base", len(aimlFiles))
	g.LogInfo("Total categories: %d", len(mergedKB.Categories))
	g.LogInfo("Total patterns: %d", len(mergedKB.Patterns))
	g.LogInfo("Total sets: %d", len(mergedKB.Sets))
	g.LogInfo("Total topics: %d", len(mergedKB.Topics))
	g.LogInfo("Total variables: %d", len(mergedKB.Variables))
	g.LogInfo("Total properties: %d", len(mergedKB.Properties))
	g.LogInfo("Total maps: %d", len(mergedKB.Maps))
	g.LogInfo("Total substitutions: %d", len(mergedKB.Substitutions))

	return mergedKB, nil
}

// LoadMapFromFile loads a .map file containing JSON array of key-value pairs
func (g *Golem) LoadMapFromFile(filename string) (map[string]string, error) {
	g.LogInfo("Loading map file: %s", filename)

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read map file %s: %v", filename, err)
	}

	// Parse JSON array
	var mapEntries []map[string]string
	err = json.Unmarshal(content, &mapEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in map file %s: %v", filename, err)
	}

	// Convert array to map
	result := make(map[string]string)
	for _, entry := range mapEntries {
		key, hasKey := entry["key"]
		value, hasValue := entry["value"]

		if !hasKey || !hasValue {
			g.LogInfo("Warning: skipping entry missing key or value: %v", entry)
			continue
		}

		result[key] = value
	}

	g.LogInfo("Loaded %d map entries from %s", len(result), filename)

	return result, nil
}

// LoadMapsFromDirectory loads all .map files from a directory
func (g *Golem) LoadMapsFromDirectory(dirPath string) (map[string]map[string]string, error) {
	g.LogInfo("Loading map files from directory: %s", dirPath)

	// Create a map to store all maps
	allMaps := make(map[string]map[string]string)

	// Walk through the directory to find all .map files
	var mapFiles []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .map file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".map") {
			mapFiles = append(mapFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(mapFiles) == 0 {
		g.LogInfo("No map files found in directory: %s", dirPath)
		return allMaps, nil
	}

	g.LogInfo("Found %d map files in directory", len(mapFiles))

	// Load each map file
	for _, mapFile := range mapFiles {
		g.LogInfo("Loading map file: %s", mapFile)

		// Load the individual map file
		mapData, err := g.LoadMapFromFile(mapFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogInfo("Warning: failed to load %s: %v", mapFile, err)
			continue
		}

		// Use the filename (without extension) as the map name
		mapName := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))
		allMaps[mapName] = mapData
	}

	g.LogInfo("Loaded %d map files", len(allMaps))

	return allMaps, nil
}

// LoadSetFromFile loads a .set file containing JSON array of set members
func (g *Golem) LoadSetFromFile(filename string) ([]string, error) {
	g.LogInfo("Loading set file: %s", filename)

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read set file %s: %v", filename, err)
	}

	// Parse JSON array
	var setMembers []string
	err = json.Unmarshal(content, &setMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in set file %s: %v", filename, err)
	}

	g.LogInfo("Loaded %d set members from %s", len(setMembers), filename)

	return setMembers, nil
}

// LoadSetsFromDirectory loads all .set files from a directory
func (g *Golem) LoadSetsFromDirectory(dirPath string) (map[string][]string, error) {
	g.LogInfo("Loading set files from directory: %s", dirPath)

	// Create a map to store all sets
	allSets := make(map[string][]string)

	// Walk through the directory to find all .set files
	var setFiles []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .set file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".set") {
			setFiles = append(setFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(setFiles) == 0 {
		g.LogInfo("No set files found in directory: %s", dirPath)
		return allSets, nil
	}

	g.LogInfo("Found %d set files in directory", len(setFiles))

	// Load each set file
	for _, setFile := range setFiles {
		g.LogInfo("Loading set file: %s", setFile)

		// Load the individual set file
		setMembers, err := g.LoadSetFromFile(setFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogInfo("Warning: failed to load %s: %v", setFile, err)
			continue
		}

		// Use the filename (without extension) as the set name
		setName := strings.TrimSuffix(filepath.Base(setFile), filepath.Ext(setFile))
		allSets[setName] = setMembers
	}

	g.LogInfo("Loaded %d set files", len(allSets))

	return allSets, nil
}

// LoadSubstitutionFromFile loads a .substitution file containing JSON array of [pattern, replacement] pairs
func (g *Golem) LoadSubstitutionFromFile(filename string) (map[string]string, error) {
	g.LogInfo("Loading substitution file: %s", filename)

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read substitution file %s: %v", filename, err)
	}

	// Parse JSON array of [pattern, replacement] pairs
	var substitutionPairs [][]string
	err = json.Unmarshal(content, &substitutionPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in substitution file %s: %v", filename, err)
	}

	// Convert array to map
	result := make(map[string]string)
	for _, pair := range substitutionPairs {
		if len(pair) != 2 {
			g.LogInfo("Warning: skipping invalid substitution pair: %v", pair)
			continue
		}

		pattern := pair[0]
		replacement := pair[1]

		if pattern == "" {
			g.LogInfo("Warning: skipping empty pattern in substitution: %v", pair)
			continue
		}

		result[pattern] = replacement
	}

	g.LogInfo("Loaded %d substitution rules from %s", len(result), filename)

	return result, nil
}

// LoadSubstitutionsFromDirectory loads all .substitution files from a directory
func (g *Golem) LoadSubstitutionsFromDirectory(dirPath string) (map[string]map[string]string, error) {
	g.LogInfo("Loading substitution files from directory: %s", dirPath)

	// Create a map to store all substitutions
	allSubstitutions := make(map[string]map[string]string)

	// Walk through the directory to find all .substitution files
	var substitutionFiles []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .substitution file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".substitution") {
			substitutionFiles = append(substitutionFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(substitutionFiles) == 0 {
		g.LogInfo("No substitution files found in directory: %s", dirPath)
		return allSubstitutions, nil
	}

	g.LogInfo("Found %d substitution files in directory", len(substitutionFiles))

	// Load each substitution file
	for _, substitutionFile := range substitutionFiles {
		g.LogInfo("Loading substitution file: %s", substitutionFile)

		// Load the individual substitution file
		substitutionData, err := g.LoadSubstitutionFromFile(substitutionFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogInfo("Warning: failed to load %s: %v", substitutionFile, err)
			continue
		}

		// Use the filename (without extension) as the substitution name
		substitutionName := strings.TrimSuffix(filepath.Base(substitutionFile), filepath.Ext(substitutionFile))
		allSubstitutions[substitutionName] = substitutionData
	}

	g.LogInfo("Loaded %d substitution files", len(allSubstitutions))

	return allSubstitutions, nil
}

// LoadPropertiesFromFile loads a .properties file containing JSON array of [key, value] pairs
func (g *Golem) LoadPropertiesFromFile(filename string) (map[string]string, error) {
	g.LogInfo("Loading properties file: %s", filename)

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read properties file %s: %v", filename, err)
	}

	// Parse JSON array of [key, value] pairs
	var propertyPairs [][]string
	err = json.Unmarshal(content, &propertyPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in properties file %s: %v", filename, err)
	}

	// Convert array to map
	result := make(map[string]string)
	for _, pair := range propertyPairs {
		if len(pair) != 2 {
			g.LogInfo("Warning: skipping invalid property pair: %v", pair)
			continue
		}

		key := pair[0]
		value := pair[1]

		if key == "" {
			g.LogInfo("Warning: skipping empty key in properties: %v", pair)
			continue
		}

		result[key] = value
	}

	g.LogInfo("Loaded %d properties from %s", len(result), filename)

	return result, nil
}

// LoadPropertiesFromDirectory loads all .properties files from a directory,
// keyed by file name without its extension
func (g *Golem) LoadPropertiesFromDirectory(dirPath string) (map[string]map[string]string, error) {
	files, err := g.loadPropertiesFiles(dirPath)
	if err != nil {
		return nil, err
	}
	allProperties := make(map[string]map[string]string)
	for _, file := range files {
		allProperties[file.name] = file.properties
	}
	return allProperties, nil
}

// loadedProperties is one loaded .properties file
type loadedProperties struct {
	name       string // file name without its extension
	properties map[string]string
}

// mergePropertiesFiles copies the properties of files into dst in order, so a
// key set in several files takes its value from the last, as a category does
func mergePropertiesFiles(dst map[string]string, files []loadedProperties) {
	for _, file := range files {
		for key, value := range file.properties {
			dst[key] = value
		}
	}
}

// loadPropertiesFiles loads all .properties files from a directory in walk
// (lexical path) order
func (g *Golem) loadPropertiesFiles(dirPath string) ([]loadedProperties, error) {
	g.LogInfo("Loading properties files from directory: %s", dirPath)

	var files []loadedProperties

	// Walk through the directory to find all .properties files
	var propertiesFiles []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .properties file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".properties") {
			propertiesFiles = append(propertiesFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(propertiesFiles) == 0 {
		g.LogInfo("No properties files found in directory: %s", dirPath)
		return nil, nil
	}

	g.LogInfo("Found %d properties files in directory", len(propertiesFiles))

	// Load each properties file
	for _, propertiesFile := range propertiesFiles {
		g.LogInfo("Loading properties file: %s", propertiesFile)

		// Load the individual properties file
		propertiesData, err := g.LoadPropertiesFromFile(propertiesFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogInfo("Warning: failed to load %s: %v", propertiesFile, err)
			continue
		}

		// Use the filename (without extension) as the properties name
		propertiesName := strings.TrimSuffix(filepath.Base(propertiesFile), filepath.Ext(propertiesFile))
		files = append(files, loadedProperties{name: propertiesName, properties: propertiesData})
	}

	g.LogInfo("Loaded %d properties files", len(files))

	return files, nil
}

// LoadPDefaultsFromFile loads a .pdefaults file containing JSON array of [key, value] pairs
func (g *Golem) LoadPDefaultsFromFile(filename string) (map[string]string, error) {
	g.LogInfo("Loading pdefaults file: %s", filename)

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdefaults file %s: %v", filename, err)
	}

	// Parse JSON array of [key, value] pairs
	var pdefaultsPairs [][]string
	err = json.Unmarshal(content, &pdefaultsPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in pdefaults file %s: %v", filename, err)
	}

	// Convert array to map
	result := make(map[string]string)
	for _, pair := range pdefaultsPairs {
		if len(pair) != 2 {
			g.LogInfo("Warning: skipping invalid pdefaults pair: %v", pair)
			continue
		}

		key := pair[0]
		value := pair[1]

		if key == "" {
			g.LogInfo("Warning: skipping empty key in pdefaults: %v", pair)
			continue
		}

		result[key] = value
	}

	g.LogInfo("Loaded %d pdefaults from %s", len(result), filename)

	return result, nil
}

// LoadPDefaultsFromDirectory loads all .pdefaults files from a directory
func (g *Golem) LoadPDefaultsFromDirectory(dirPath string) (map[string]map[string]string, error) {
	g.LogInfo("Loading pdefaults files from directory: %s", dirPath)

	// Create a map to store all pdefaults
	allPDefaults := make(map[string]map[string]string)

	// Walk through the directory to find all .pdefaults files
	var pdefaultsFiles []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a .pdefaults file
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".pdefaults") {
			pdefaultsFiles = append(pdefaultsFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", dirPath, err)
	}

	if len(pdefaultsFiles) == 0 {
		g.LogInfo("No pdefaults files found in directory: %s", dirPath)
		return allPDefaults, nil
	}

	g.LogInfo("Found %d pdefaults files in directory", len(pdefaultsFiles))

	// Load each pdefaults file
	for _, pdefaultsFile := range pdefaultsFiles {
		g.LogInfo("Loading pdefaults file: %s", pdefaultsFile)

		// Load the individual pdefaults file
		pdefaultsData, err := g.LoadPDefaultsFromFile(pdefaultsFile)
		if err != nil {
			// Log the error but continue with other files
			g.LogInfo("Warning: failed to load %s: %v", pdefaultsFile, err)
			continue
		}

		// Use the filename (without extension) as the pdefaults name
		pdefaultsName := strings.TrimSuffix(filepath.Base(pdefaultsFile), filepath.Ext(pdefaultsFile))
		allPDefaults[pdefaultsName] = pdefaultsData
	}

	g.LogInfo("Loaded %d pdefaults files", len(allPDefaults))

	return allPDefaults, nil
}

// parseAIML parses AIML content using native Go string manipulation
//
// DO NOT replace this with encoding/xml (xml.Unmarshal, xml.Decoder, or its
// tokenizer). That approach has been tried and fails badly on AIML-specific
// structures. Fix bugs within this hand-written parser instead.
func (g *Golem) parseAIML(content string) (*AIML, error) {
	aiml := &AIML{
		Categories: []Category{},
	}

	// Remove XML declaration and comments
	content = g.removeComments(content)
	content = g.removeXMLDeclaration(content)

	// Extract version
	versionMatch := aimlVersionRe.FindStringSubmatch(content)
	if len(versionMatch) > 1 {
		aiml.Version = versionMatch[1]
	} else {
		aiml.Version = "2.0" // Default version
	}

	if err := g.parseCategories(content, "", aiml); err != nil {
		return nil, err
	}

	return aiml, nil
}

// parseCategories appends the categories in content to aiml, descending into
// <topic name="..."> blocks. Categories inside a topic block get its name as
// their topic unless they set their own.
func (g *Golem) parseCategories(content string, topic string, aiml *AIML) error {
	// Position of the next <topic tag, cached so that files without topic
	// blocks aren't rescanned for every category
	nextTopic := indexOpenTag(content, "topic", 0)

	for pos := 0; ; {
		if nextTopic != -1 && nextTopic < pos {
			nextTopic = indexOpenTag(content, "topic", pos)
		}
		category, hasCategory := findTag(content, "category", pos)

		if nextTopic != -1 && (!hasCategory || nextTopic < category.start) {
			if block, ok := findTag(content, "topic", nextTopic); ok && !block.selfClosing {
				blockTopic := strings.TrimSpace(block.attributes["name"])
				if err := g.parseCategories(content[block.contentStart:block.contentEnd], blockTopic, aiml); err != nil {
					return err
				}
				pos = block.end
			} else {
				nextTopic = indexOpenTag(content, "topic", nextTopic+1)
			}
			continue
		}

		if !hasCategory {
			return nil
		}

		parsed, err := g.parseCategory(content[category.contentStart:category.contentEnd])
		if err != nil {
			return fmt.Errorf("failed to parse category: %v", err)
		}
		if parsed.Topic == "" {
			parsed.Topic = topic
		}
		aiml.Categories = append(aiml.Categories, parsed)
		pos = category.end
	}
}

// parseCategory parses a single category using tag-aware parsing
func (g *Golem) parseCategory(content string) (Category, error) {
	category := Category{}

	// Extract template using tag-aware parsing (handles nested <template> tags).
	// Pattern, that, and topic are only looked for outside the template, so
	// tags inside it (e.g. a category inside <learn>) aren't mistaken for this
	// category's own.
	header := content
	if template, found := findTag(content, "template", 0); found {
		category.Template = strings.TrimSpace(content[template.contentStart:template.contentEnd])
		header = content[:template.start] + content[template.end:]
	}

	if pattern, found := findTag(header, "pattern", 0); found {
		category.Pattern = html.UnescapeString(strings.TrimSpace(header[pattern.contentStart:pattern.contentEnd]))
	}

	// Extract that (optional) with index support
	if that, found := findTag(header, "that", 0); found {
		category.That = html.UnescapeString(strings.TrimSpace(header[that.contentStart:that.contentEnd]))

		// Parse index attribute if provided
		if indexStr, hasIndex := that.attributes["index"]; hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil {
				// Validate index range (1-10 for reasonable history depth)
				if index < 1 || index > 10 {
					return Category{}, fmt.Errorf("that index must be between 1 and 10, got %d", index)
				}
				category.ThatIndex = index
			} else {
				return Category{}, fmt.Errorf("invalid that index: %s", indexStr)
			}
		} else {
			category.ThatIndex = 0 // Default to last response when no index specified
		}

		// Validate that pattern
		if err := validateThatPattern(category.That); err != nil {
			return Category{}, fmt.Errorf("invalid that pattern: %v", err)
		}
	}

	// Extract topic (optional)
	if topic, found := findTag(header, "topic", 0); found {
		category.Topic = html.UnescapeString(strings.TrimSpace(header[topic.contentStart:topic.contentEnd]))
	}

	return category, nil
}

// tagSpan is the location of an element found by findTag. For a self-closing
// tag, contentStart, contentEnd, and end are all the index after "/>".
type tagSpan struct {
	start        int // index of the opening '<'
	contentStart int // index just after the opening tag's '>'
	contentEnd   int // index of the closing tag's '<'
	end          int // index just after the closing tag's '>'
	attributes   map[string]string
	selfClosing  bool
}

// findTag finds the first <tagName> element at or after from, matching its
// closing tag by nesting depth. Tag names must match exactly (<that> does not
// match <thatstar/>), and nested self-closing tags such as <topic/> don't
// affect the depth.
func findTag(input string, tagName string, from int) (tagSpan, bool) {
	start := indexOpenTag(input, tagName, from)
	if start == -1 {
		return tagSpan{}, false
	}

	attributes, contentStart, selfClosing, ok := parseOpenTag(input, start+1+len(tagName))
	if !ok {
		return tagSpan{}, false
	}
	span := tagSpan{
		start:        start,
		contentStart: contentStart,
		attributes:   attributes,
		selfClosing:  selfClosing,
	}
	if selfClosing {
		span.contentEnd = contentStart
		span.end = contentStart
		return span, true
	}

	closeTag := "</" + tagName
	depth := 1
	for i := contentStart; i < len(input); {
		lt := strings.IndexByte(input[i:], '<')
		if lt == -1 {
			break
		}
		i += lt

		if strings.HasPrefix(input[i:], closeTag) && isTagNameEnd(input, i+len(closeTag)) {
			gt := strings.IndexByte(input[i:], '>')
			if gt == -1 {
				break
			}
			depth--
			if depth == 0 {
				span.contentEnd = i
				span.end = i + gt + 1
				return span, true
			}
			i += gt + 1
			continue
		}

		if isOpenTagAt(input, i, tagName) {
			_, next, nestedSelfClosing, ok := parseOpenTag(input, i+1+len(tagName))
			if !ok {
				break
			}
			if !nestedSelfClosing {
				depth++
			}
			i = next
			continue
		}

		i++
	}

	// No matching closing tag
	return tagSpan{}, false
}

// indexOpenTag returns the index of the first <tagName opening tag at or after
// from, or -1. The name must be followed by whitespace, '>' or '/'.
func indexOpenTag(input string, tagName string, from int) int {
	open := "<" + tagName
	for i := from; i < len(input); {
		idx := strings.Index(input[i:], open)
		if idx == -1 {
			return -1
		}
		idx += i
		if isTagNameEnd(input, idx+len(open)) {
			return idx
		}
		i = idx + 1
	}
	return -1
}

// isOpenTagAt reports whether a <tagName opening tag starts at index i
func isOpenTagAt(input string, i int, tagName string) bool {
	return strings.HasPrefix(input[i:], "<"+tagName) && isTagNameEnd(input, i+1+len(tagName))
}

// isTagNameEnd reports whether a tag name ending at index i is complete,
// i.e. followed by whitespace, '>' or '/'
func isTagNameEnd(input string, i int) bool {
	if i >= len(input) {
		return false
	}
	switch input[i] {
	case '>', '/', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// parseOpenTag parses an opening tag's attributes, starting just after the tag
// name, through its closing '>'. Attribute values may be single- or
// double-quoted, and entities in them are decoded. It returns the attributes,
// the index after the tag, whether it is self-closing, and false if the tag
// is unterminated.
func parseOpenTag(input string, i int) (map[string]string, int, bool, bool) {
	attributes := make(map[string]string)
	for i < len(input) {
		c := input[i]
		switch {
		case c == '>':
			return attributes, i + 1, false, true
		case c == '/' && i+1 < len(input) && input[i+1] == '>':
			return attributes, i + 2, true, true
		case c == '/' || isXMLSpace(c):
			i++
		default:
			nameStart := i
			for i < len(input) && input[i] != '=' && input[i] != '>' && input[i] != '/' && !isXMLSpace(input[i]) {
				i++
			}
			name := input[nameStart:i]
			for i < len(input) && isXMLSpace(input[i]) {
				i++
			}
			if i >= len(input) || input[i] != '=' {
				attributes[name] = ""
				continue
			}
			i++ // skip '='
			for i < len(input) && isXMLSpace(input[i]) {
				i++
			}
			if i < len(input) && (input[i] == '"' || input[i] == '\'') {
				quote := input[i]
				i++
				valueStart := i
				for i < len(input) && input[i] != quote {
					i++
				}
				attributes[name] = html.UnescapeString(input[valueStart:i])
				if i < len(input) {
					i++ // skip closing quote
				}
			} else {
				valueStart := i
				for i < len(input) && input[i] != '>' && !isXMLSpace(input[i]) && !strings.HasPrefix(input[i:], "/>") {
					i++
				}
				attributes[name] = input[valueStart:i]
			}
		}
	}
	return nil, i, false, false
}

func isXMLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// removeComments removes XML comments from content
func (g *Golem) removeComments(content string) string {
	return xmlCommentRe.ReplaceAllString(content, "")
}

// removeXMLDeclaration removes XML declaration
func (g *Golem) removeXMLDeclaration(content string) string {
	return xmlDeclRe.ReplaceAllString(content, "")
}

// validateAIML validates the AIML structure. Categories with invalid patterns
// are dropped with a warning so that one bad category doesn't discard the rest
// of the file; it's an error only if no valid categories remain.
func (g *Golem) validateAIML(aiml *AIML) error {
	if aiml.Version == "" {
		return fmt.Errorf("AIML version is required")
	}

	if len(aiml.Categories) == 0 {
		return fmt.Errorf("AIML must contain at least one category")
	}

	valid := make([]Category, 0, len(aiml.Categories))
	for i, category := range aiml.Categories {
		if err := g.validatePattern(category.Pattern); err != nil {
			g.LogWarn("Skipping category %d: invalid pattern '%s': %v", i, category.Pattern, err)
			continue
		}
		valid = append(valid, category)
	}
	aiml.Categories = valid

	if len(aiml.Categories) == 0 {
		return fmt.Errorf("AIML must contain at least one valid category")
	}

	return nil
}

// validatePattern validates AIML pattern syntax
func (g *Golem) validatePattern(pattern string) error {
	// Basic pattern validation
	pattern = strings.TrimSpace(pattern)

	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if emptySetTagRe.MatchString(pattern) {
		return fmt.Errorf("set name cannot be empty")
	}

	// Any characters may appear in a pattern; normalization removes the
	// punctuation that input loses too

	// Check for balanced wildcards (count all wildcard types)
	wildcardCounts := CountWildcardsByType(pattern)
	totalWildcards := wildcardCounts["star"] + wildcardCounts["underscore"] + wildcardCounts["caret"] + wildcardCounts["hash"]

	if totalWildcards > 9 {
		return fmt.Errorf("pattern contains too many wildcards (max 9)")
	}

	// Check for valid set references
	matches := patternSetTagRe.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		if len(match) > 1 && strings.TrimSpace(match[1]) == "" {
			return fmt.Errorf("set name cannot be empty")
		}
	}

	return nil
}

// MatchPattern attempts to match user input against AIML patterns with highest priority matching
func (kb *AIMLKnowledgeBase) MatchPattern(input string) (*Category, map[string]string, error) {
	return kb.MatchPatternWithTopicAndThat(input, "", "")
}

// MatchPatternWithTopicAndThat attempts to match user input against AIML patterns with topic and that filtering
func (kb *AIMLKnowledgeBase) MatchPatternWithTopicAndThat(input string, topic string, that string) (*Category, map[string]string, error) {
	return kb.MatchPatternWithTopicAndThatIndex(input, topic, that, 0)
}

// MatchPatternWithTopicAndThatIndex attempts to match user input against AIML patterns with topic and that filtering with index support
func (kb *AIMLKnowledgeBase) MatchPatternWithTopicAndThatIndex(input string, topic string, that string, thatIndex int) (*Category, map[string]string, error) {
	// Normalize the input for pattern matching
	normalizedInput := NormalizePattern(input)
	// Use original input for case-preserving wildcard extraction
	return kb.MatchPatternWithTopicAndThatIndexOriginal(normalizedInput, input, topic, that, thatIndex)
}

func (kb *AIMLKnowledgeBase) MatchPatternWithTopicAndThatIndexOriginal(normalizedInput string, originalInput string, topic string, that string, thatIndex int) (*Category, map[string]string, error) {
	return kb.MatchPatternWithTopicAndThatIndexOriginalCached(nil, normalizedInput, originalInput, topic, that, thatIndex)
}

// MatchPatternWithTopicAndThatIndexOriginalCached attempts to match user input against AIML patterns with caching support.
// The returned category is a copy, so it stays valid while other sessions learn and unlearn.
func (kb *AIMLKnowledgeBase) MatchPatternWithTopicAndThatIndexOriginalCached(g *Golem, normalizedInput string, originalInput string, topic string, that string, thatIndex int) (*Category, map[string]string, error) {
	kb.patternsMu.RLock()
	defer kb.patternsMu.RUnlock()

	category, wildcards, err := kb.matchPattern(g, normalizedInput, originalInput, topic, that, thatIndex)
	if category != nil {
		matched := *category
		category = &matched
	}
	return category, wildcards, err
}

// matchPattern does the matching for MatchPatternWithTopicAndThatIndexOriginalCached.
// The caller must hold patternsMu.
func (kb *AIMLKnowledgeBase) matchPattern(g *Golem, normalizedInput string, originalInput string, topic string, that string, thatIndex int) (*Category, map[string]string, error) {
	normalizedThat := ""
	if that != "" {
		normalizedThat = NormalizeThatPattern(that)
	}
	inputWords := strings.Fields(normalizedInput)
	normalize := NormalizePattern
	if g != nil {
		normalize = g.CachedNormalizePattern
	}

	m := &gmMatcher{
		kb:        kb,
		sections:  [3][]string{inputWords, strings.Fields(normalizedThat), strings.Fields(NormalizePattern(topic))},
		thatIndex: thatIndex,
		original:  newOriginalText(originalInput, inputWords, normalize),
	}
	if category := m.match(kb.graphmaster(), gmInput, 0); category != nil {
		return category, m.wildcards(), nil
	}

	// A DEFAULT category, as in AIML 1.0, answers what nothing else matches
	if category, exists := kb.Patterns["DEFAULT"]; exists {
		if (topic == "" || category.Topic == "" || strings.EqualFold(category.Topic, topic)) &&
			(normalizedThat == "" || category.That == "" || category.That == normalizedThat) {
			return category, make(map[string]string), nil
		}
	}

	return nil, nil, fmt.Errorf("no matching pattern found")
}

// MatchPatternWithTopic attempts to match user input against AIML patterns with topic filtering
func (kb *AIMLKnowledgeBase) MatchPatternWithTopic(input string, topic string) (*Category, map[string]string, error) {
	return kb.MatchPatternWithTopicAndThat(input, topic, "")
}

// ProcessTemplate processes an AIML template and returns the response
func (g *Golem) ProcessTemplate(template string, wildcards map[string]string) string {
	g.kbMu.RLock()
	defer g.kbMu.RUnlock()

	// Create variable context for template processing
	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       nil, // No session context for ProcessTemplate
		Topic:         "",  // Topic tracking will be implemented in future version
		KnowledgeBase: g.aimlKB,
	}

	if g.aimlKB != nil {
		g.LogInfo("ProcessTemplate: KB pointer=%p, KB variables=%v", g.aimlKB, g.aimlKB.Variables)
		g.LogInfo("ProcessTemplate: Context KB pointer=%p, Context KB variables=%v", ctx.KnowledgeBase, ctx.KnowledgeBase.Variables)
	} else {
		g.LogInfo("ProcessTemplate: No knowledge base set")
	}

	result := g.processTemplateWithContext(template, wildcards, ctx)

	if g.aimlKB != nil {
		g.LogInfo("ProcessTemplate: After processing, KB pointer=%p, KB variables=%v", g.aimlKB, g.aimlKB.Variables)
		g.LogInfo("ProcessTemplate: After processing, Context KB pointer=%p, Context KB variables=%v", ctx.KnowledgeBase, ctx.KnowledgeBase.Variables)
	}

	return result
}

// rlockKnowledgeBase takes kbMu for reading, first creating an empty
// knowledge base if none is loaded. The caller must call kbMu.RUnlock.
func (g *Golem) rlockKnowledgeBase() {
	for {
		g.kbMu.RLock()
		if g.aimlKB != nil {
			return
		}
		g.kbMu.RUnlock()

		g.kbMu.Lock()
		if g.aimlKB == nil {
			g.aimlKB = NewAIMLKnowledgeBase()
		}
		g.kbMu.Unlock()
	}
}

// ProcessTemplateWithContext processes an AIML template with full context support
func (g *Golem) ProcessTemplateWithContext(template string, wildcards map[string]string, session *ChatSession) string {
	// Ensure knowledge base is initialized for variable/collection operations
	g.rlockKnowledgeBase()
	defer g.kbMu.RUnlock()
	return g.processTemplateForSession(template, wildcards, session)
}

// processTemplateForSession processes a template for a session. The caller
// must hold kbMu (ProcessInput already does, and RWMutex read locks must not
// be taken twice).
func (g *Golem) processTemplateForSession(template string, wildcards map[string]string, session *ChatSession) string {
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          session.GetSessionTopic(),
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	return g.processTemplateWithContext(template, wildcards, ctx)
}

// processTemplateWithContext parses a template into an AST and evaluates it
// with the tree processor, which handles each tag where it appears, innermost
// content first. The caller must hold kbMu for reading, once.
func (g *Golem) processTemplateWithContext(template string, wildcards map[string]string, ctx *VariableContext) string {
	if g.treeProcessor == nil {
		g.treeProcessor = NewTreeProcessor(g)
	}
	// TreeProcessor holds per-call state (ctx, starCounter), so each call gets
	// its own; sharing one across concurrent sessions mixed up their contexts.
	// Only the metrics registry is shared.
	tp := &TreeProcessor{golem: g, metrics: g.treeProcessor.metrics, kbReadLocked: true}
	response, err := tp.ProcessTemplate(template, wildcards, ctx)
	if err != nil {
		g.LogError("Error in tree-based template processing: %v", err)
		// NEVER return templates with XML tags - return error message instead
		return "[Error processing template]"
	}
	return response
}

// SubstitutePronouns performs pronoun substitution for person tags
func (g *Golem) SubstitutePronouns(text string) string {
	// Comprehensive pronoun mapping for first/second person substitution
	pronounMap := map[string]string{
		// First person to second person
		"I": "you", "i": "you",
		"me": "you",
		"my": "your", "My": "Your",
		"mine": "yours", "Mine": "Yours",
		"we": "you", "We": "you",
		"us": "you", "Us": "you",
		"our": "your", "Our": "your",
		"ours": "yours", "Ours": "yours",
		"myself": "yourself", "Myself": "yourself",
		"ourselves": "yourselves", "Ourselves": "yourselves",

		// Second person to first person
		"you": "I", "You": "I",
		"your": "my", "Your": "my",
		"yours": "mine", "Yours": "mine",
		"yourself": "myself", "Yourself": "myself",
		"yourselves": "ourselves", "Yourselves": "ourselves",

		// Contractions - first person to second person
		"I'm": "you're", "i'm": "you're", "I'M": "you're",
		"I've": "you've", "i've": "you've", "I'VE": "you've",
		"I'll": "you'll", "i'll": "you'll", "I'LL": "you'll",
		"I'd": "you'd", "i'd": "you'd", "I'D": "you'd",

		// Contractions - second person to first person
		"you're": "I'm", "You're": "I'm", "YOU'RE": "I'm",
		"you've": "I've", "You've": "I've", "YOU'VE": "I've",
		"you'll": "I'll", "You'll": "I'll", "YOU'LL": "I'll",
		"you'd": "I'd", "You'd": "I'd", "YOU'D": "I'd",
	}

	// Split text into words while preserving whitespace
	words := strings.Fields(text)
	substitutedWords := make([]string, len(words))

	for i, word := range words {
		// Check for exact match first
		if substitution, exists := pronounMap[word]; exists {
			substitutedWords[i] = substitution
			continue
		}

		// Handle contractions and possessives more carefully
		substituted := word

		// Check for contractions (apostrophe)
		if strings.Contains(word, "'") {
			// Split on apostrophe and check each part
			parts := strings.Split(word, "'")
			if len(parts) == 2 {
				firstPart := parts[0]
				secondPart := parts[1]

				// Check if first part needs substitution
				if sub, exists := pronounMap[firstPart]; exists {
					substituted = sub + "'" + secondPart
				} else if sub, exists := pronounMap[firstPart+"'"]; exists {
					substituted = sub + secondPart
				}
			}
		}

		// Check for possessive forms (ending with 's or s')
		if strings.HasSuffix(word, "'s") || strings.HasSuffix(word, "s'") {
			base := strings.TrimSuffix(strings.TrimSuffix(word, "'s"), "s'")
			if sub, exists := pronounMap[base]; exists {
				if strings.HasSuffix(word, "'s") {
					substituted = sub + "'s"
				} else {
					substituted = sub + "s'"
				}
			}
		}

		// Check for words ending with common suffixes that might be pronouns
		if strings.HasSuffix(word, "ing") || strings.HasSuffix(word, "ed") || strings.HasSuffix(word, "er") || strings.HasSuffix(word, "est") {
			// Don't substitute if it's a verb form
			substituted = word
		}

		substitutedWords[i] = substituted
	}

	result := strings.Join(substitutedWords, " ")

	// Handle verb agreement after pronoun substitution
	result = g.fixVerbAgreement(result)

	g.LogInfo("Person substitution: '%s' -> '%s'", text, result)

	return result
}

// fixVerbAgreement fixes verb agreement after pronoun substitution
func (g *Golem) fixVerbAgreement(text string) string {
	// Common verb agreement fixes
	verbFixes := map[string]string{
		"you am":  "you are",
		"You Am":  "You Are",
		"you Am":  "you Are",
		"I are":   "I am",
		"you is":  "you are",
		"I is":    "I am",
		"you was": "you were",
		"I were":  "I was",
		"you has": "you have",
		"I have":  "I have", // Keep as is
	}

	result := text
	for wrong, correct := range verbFixes {
		result = strings.ReplaceAll(result, wrong, correct)
	}

	return result
}

// SubstitutePronouns2 performs first-to-third person pronoun substitution for person2 tags
func (g *Golem) SubstitutePronouns2(text string) string {
	// Comprehensive pronoun mapping for first-to-third person substitution
	pronounMap := map[string]string{
		// First person to third person (neutral/they)
		"I": "they", "i": "they",
		"me": "them",
		"my": "their", "My": "Their",
		"mine": "theirs", "Mine": "Theirs",
		"we": "they", "We": "They",
		"us": "them", "Us": "Them",
		"our": "their", "Our": "Their",
		"ours": "theirs", "Ours": "Theirs",
		"myself": "themselves", "Myself": "Themselves",
		"ourselves": "themselves", "Ourselves": "Themselves",

		// Contractions - first person to third person
		"I'm": "they're", "i'm": "they're", "I'M": "they're",
		"I've": "they've", "i've": "they've", "I'VE": "they've",
		"I'll": "they'll", "i'll": "they'll", "I'LL": "they'll",
		"I'd": "they'd", "i'd": "they'd", "I'D": "they'd",
		"we're": "they're", "We're": "They're", "WE'RE": "they're",
		"we've": "they've", "We've": "They've", "WE'VE": "they've",
		"we'll": "they'll", "We'll": "They'll", "WE'LL": "they'll",
		"we'd": "they'd", "We'd": "They'd", "WE'D": "they'd",
	}

	// Split text into words while preserving whitespace
	words := strings.Fields(text)
	substitutedWords := make([]string, len(words))

	for i, word := range words {
		// Check for exact match first
		if substitution, exists := pronounMap[word]; exists {
			substitutedWords[i] = substitution
			continue
		}

		// Handle contractions and possessives more carefully
		substituted := word

		// Check for contractions (apostrophe) - only if the word starts with the contraction
		if strings.Contains(word, "'") {
			for contraction, replacement := range pronounMap {
				if strings.HasPrefix(word, contraction) && strings.Contains(word, "'") {
					substituted = strings.ReplaceAll(word, contraction, replacement)
					break
				}
			}
		}

		// Check for possessive forms (only for pronouns)
		if strings.HasSuffix(word, "'s") || strings.HasSuffix(word, "s'") {
			base := strings.TrimSuffix(strings.TrimSuffix(word, "'s"), "s'")
			// Only process if the base word is a pronoun
			if replacement, exists := pronounMap[base]; exists {
				if strings.HasSuffix(word, "'s") {
					substituted = replacement + "'s"
				} else {
					substituted = replacement + "s'"
				}
			}
		}

		substitutedWords[i] = substituted
	}

	result := strings.Join(substitutedWords, " ")

	// Handle verb agreement after pronoun substitution
	result = g.fixVerbAgreement2(result)

	g.LogInfo("Person2 substitution: '%s' -> '%s'", text, result)

	return result
}

// fixVerbAgreement2 fixes verb agreement after person2 pronoun substitution
func (g *Golem) fixVerbAgreement2(text string) string {
	// Common verb agreement fixes for third person
	verbFixes := map[string]string{
		"they am":      "they are",
		"they is":      "they are",
		"they was":     "they were",
		"they has":     "they have",
		"they does":    "they do",
		"they doesn't": "they don't",
		"they isn't":   "they aren't",
		"they wasn't":  "they weren't",
		"they hasn't":  "they haven't",
	}

	result := text
	for wrong, correct := range verbFixes {
		result = strings.ReplaceAll(result, wrong, correct)
	}

	return result
}

// SubstituteGenderPronouns performs gender-based pronoun substitution for gender tags
func (g *Golem) SubstituteGenderPronouns(text string) string {
	// Split text into words for more precise substitution
	words := strings.Fields(text)
	result := make([]string, len(words))

	for i, word := range words {
		// Clean word for matching (remove punctuation)
		cleanWord := strings.Trim(word, ".,!?;:\"'()[]{}")
		lowerWord := strings.ToLower(cleanWord)

		// Gender pronoun mapping (masculine to feminine and vice versa)
		genderMap := map[string]string{
			// Masculine to feminine
			"he": "she", "him": "her", "his": "her", "himself": "herself",
			"he's": "she's", "he'll": "she'll", "he'd": "she'd",

			// Feminine to masculine
			"she": "he", "her": "his", "hers": "his", "herself": "himself",
			"she's": "he's", "she'll": "he'll", "she'd": "he'd",
		}

		// Check if we need to substitute
		if substitute, exists := genderMap[lowerWord]; exists {
			// Preserve original case
			if strings.ToUpper(cleanWord) == cleanWord {
				// All caps
				result[i] = strings.ToUpper(substitute)
			} else if len(cleanWord) > 0 && cleanWord[0] >= 'A' && cleanWord[0] <= 'Z' {
				// Title case (first letter capitalized)
				if len(substitute) > 0 {
					result[i] = strings.ToUpper(string(substitute[0])) + strings.ToLower(substitute[1:])
				} else {
					result[i] = substitute
				}
			} else {
				// Lower case
				result[i] = substitute
			}

			// Add back any punctuation that was removed
			if len(cleanWord) < len(word) {
				suffix := word[len(cleanWord):]
				result[i] += suffix
			}
		} else {
			// No substitution needed
			result[i] = word
		}
	}

	// Join words back together
	finalResult := strings.Join(result, " ")

	// Fix verb agreement after gender substitution
	finalResult = g.fixGenderVerbAgreement(finalResult)

	g.LogInfo("Gender substitution: '%s' -> '%s'", text, finalResult)

	return finalResult
}

// fixGenderVerbAgreement fixes verb agreement after gender pronoun substitution
func (g *Golem) fixGenderVerbAgreement(text string) string {
	// Common verb agreement fixes for gender pronouns
	verbFixes := map[string]string{
		"she am":   "she is",
		"he am":    "he is",
		"she are":  "she is",
		"he are":   "he is",
		"she was":  "she was", // Keep as is
		"he was":   "he was",  // Keep as is
		"she were": "she was",
		"he were":  "he was",
		"she has":  "she has", // Keep as is
		"he has":   "he has",  // Keep as is
		"she have": "she has",
		"he have":  "he has",
	}

	result := text
	for wrong, correct := range verbFixes {
		result = strings.ReplaceAll(result, wrong, correct)
	}

	return result
}

// extractSubstring extracts a substring from text based on start and end positions
func (g *Golem) extractSubstring(input, startStr, endStr string) string {
	// Convert to rune slice to handle Unicode properly
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}

	// Parse start position
	start := 0
	if startStr != "" {
		if startInt, err := strconv.Atoi(startStr); err == nil {
			start = startInt
		}
	}

	// Parse end position
	end := len(runes)
	if endStr != "" {
		if endInt, err := strconv.Atoi(endStr); err == nil {
			end = endInt
		}
	}

	// Validate bounds
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}

	// Extract substring
	return string(runes[start:end])
}

// pluralizeText converts singular words to their plural forms
func (g *Golem) pluralizeText(input string) string {
	// Split into words
	words := strings.Fields(input)
	if len(words) == 0 {
		return ""
	}

	// Pluralize each word
	pluralizedWords := make([]string, len(words))
	for i, word := range words {
		pluralizedWords[i] = g.pluralizeWord(word)
	}

	return strings.Join(pluralizedWords, " ")
}

// pluralizeWord converts a single word to its plural form
func (g *Golem) pluralizeWord(word string) string {
	if len(word) == 0 {
		return word
	}

	// Convert to lowercase for processing
	lowerWord := strings.ToLower(word)

	// Check if word is already plural
	if g.isAlreadyPlural(lowerWord) {
		return word // Return original word with preserved case
	}

	// Handle irregular plurals
	irregularPlurals := map[string]string{
		"child":       "children",
		"person":      "people",
		"man":         "men",
		"woman":       "women",
		"foot":        "feet",
		"tooth":       "teeth",
		"mouse":       "mice",
		"goose":       "geese",
		"ox":          "oxen",
		"sheep":       "sheep",
		"deer":        "deer",
		"fish":        "fish",
		"moose":       "moose",
		"series":      "series",
		"species":     "species",
		"crisis":      "crises",
		"thesis":      "theses",
		"analysis":    "analyses",
		"basis":       "bases",
		"diagnosis":   "diagnoses",
		"oasis":       "oases",
		"parenthesis": "parentheses",
		"synopsis":    "synopses",
		"cactus":      "cacti",
		"fungus":      "fungi",
		"nucleus":     "nuclei",
		"stimulus":    "stimuli",
		"syllabus":    "syllabi",
		"alumnus":     "alumni",
		"radius":      "radii",
		"focus":       "foci",
		"appendix":    "appendices",
		"index":       "indices",
		"matrix":      "matrices",
		"vertex":      "vertices",
		"vortex":      "vortices",
		"corpus":      "corpora",
		"genus":       "genera",
		"opus":        "opera",
		"stratum":     "strata",
		"datum":       "data",
		"medium":      "media",
		"memorandum":  "memoranda",
		"referendum":  "referenda",
		"agenda":      "agenda",
		"curriculum":  "curricula",
		"maximum":     "maxima",
		"minimum":     "minima",
		"optimum":     "optima",
		"quantum":     "quanta",
		"spectrum":    "spectra",
		"forum":       "fora",
		"stadium":     "stadia",
		"aquarium":    "aquaria",
		"planetarium": "planetaria",
		"sanitarium":  "sanitaria",
		"solarium":    "solaria",
		"terrarium":   "terraria",
		"vivarium":    "vivaria",
		"atrium":      "atria",
		"auditorium":  "auditoria",
		"gymnasium":   "gymnasia",
		"emporium":    "emporia",
		"crematorium": "crematoria",
		"laboratory":  "laboratories",
		"library":     "libraries",
		"factory":     "factories",
		"story":       "stories",
		"country":     "countries",
		"city":        "cities",
		"baby":        "babies",
		"lady":        "ladies",
		"party":       "parties",
		"company":     "companies",
		"family":      "families",
		"army":        "armies",
		"enemy":       "enemies",
		"monkey":      "monkeys",
		"key":         "keys",
		"toy":         "toys",
		"boy":         "boys",
		"day":         "days",
		"way":         "ways",
		"play":        "plays",
		"stay":        "stays",
		"say":         "says",
		"buy":         "buys",
		"guy":         "guys",
		"cry":         "cries",
		"fly":         "flies",
		"try":         "tries",
		"spy":         "spies",
		"sky":         "skies",
		"dry":         "dries",
		"shy":         "shies",
		"worry":       "worries",
		"hurry":       "hurries",
		"carry":       "carries",
		"marry":       "marries",
		"study":       "studies",
		"apply":       "applies",
		"reply":       "replies",
		"supply":      "supplies",
		"multiply":    "multiplies",
		"identify":    "identifies",
		"classify":    "classifies",
		"justify":     "justifies",
		"purify":      "purifies",
		"amplify":     "amplifies",
		"simplify":    "simplifies",
		"beautify":    "beautifies",
		"diversify":   "diversifies",
		"intensify":   "intensifies",
		"magnify":     "magnifies",
		"modify":      "modifies",
		"notify":      "notifies",
		"qualify":     "qualifies",
		"ratify":      "ratifies",
		"rectify":     "rectifies",
		"satisfy":     "satisfies",
		"specify":     "specifies",
		"testify":     "testifies",
		"verify":      "verifies",
	}

	// Check for irregular plurals
	if plural, exists := irregularPlurals[lowerWord]; exists {
		// Preserve original case
		if word == strings.ToUpper(word) {
			return strings.ToUpper(plural)
		} else if word[0] == strings.ToUpper(string(word[0]))[0] {
			return strings.ToUpper(string(plural[0])) + plural[1:]
		}
		return plural
	}

	// Regular pluralization rules
	runes := []rune(word)
	if len(runes) == 0 {
		return word
	}

	lastChar := runes[len(runes)-1]
	lastTwoChars := ""
	if len(runes) >= 2 {
		lastTwoChars = string(runes[len(runes)-2:])
	}

	// Words ending in -s, -ss, -sh, -ch, -x, -z
	if lastChar == 's' || lastChar == 'x' || lastChar == 'z' ||
		lastTwoChars == "sh" || lastTwoChars == "ch" {
		return word + "es"
	}

	// Words ending in -f or -fe
	if lastChar == 'f' {
		return string(runes[:len(runes)-1]) + "ves"
	}
	if lastTwoChars == "fe" {
		return string(runes[:len(runes)-2]) + "ves"
	}

	// Words ending in -y preceded by a consonant
	if lastChar == 'y' && len(runes) > 1 {
		secondLastChar := runes[len(runes)-2]
		if !isVowel(secondLastChar) {
			return string(runes[:len(runes)-1]) + "ies"
		}
	}

	// Words ending in -o preceded by a consonant
	if lastChar == 'o' && len(runes) > 1 {
		secondLastChar := runes[len(runes)-2]
		if !isVowel(secondLastChar) {
			return word + "es"
		}
	}

	// Default: add -s
	return word + "s"
}

// isVowel checks if a rune is a vowel
func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	}
	return false
}

// isAlreadyPlural checks if a word is already in plural form
func (g *Golem) isAlreadyPlural(word string) bool {
	if len(word) < 2 {
		return false
	}

	// Check for common plural endings
	runes := []rune(word)
	lastChar := runes[len(runes)-1]
	lastTwoChars := ""
	lastThreeChars := ""

	if len(runes) >= 2 {
		lastTwoChars = string(runes[len(runes)-2:])
	}
	if len(runes) >= 3 {
		lastThreeChars = string(runes[len(runes)-3:])
	}

	// Words ending in -es (but not -ses, -ches, -shes, -xes, -zes which are singular)
	if lastTwoChars == "es" && lastThreeChars != "ses" && lastThreeChars != "ches" &&
		lastThreeChars != "shes" && lastThreeChars != "xes" && lastThreeChars != "zes" {
		return true
	}

	// Words ending in -ies (but not -sies, -chies, -shies, -xies, -zies which are singular)
	if lastThreeChars == "ies" && len(runes) > 3 {
		// Check if the character before "ies" is a consonant
		beforeIes := runes[len(runes)-4]
		if !isVowel(beforeIes) {
			return true
		}
	}

	// Words ending in -ves (but not -fves, -ffes which are singular)
	if lastThreeChars == "ves" && len(runes) > 3 {
		// Check if the character before "ves" is not 'f'
		beforeVes := runes[len(runes)-4]
		if beforeVes != 'f' {
			return true
		}
	}

	// Words ending in -s (but not -ss, -sh, -ch, -x, -z which are singular)
	// Only consider it plural if it's not a word that would normally get -es
	if lastChar == 's' && lastTwoChars != "ss" && lastTwoChars != "sh" && lastTwoChars != "ch" {
		// Check if it's not a word that would normally get -es (like bus -> buses)
		if lastChar == 's' && lastTwoChars != "us" && lastTwoChars != "is" && lastTwoChars != "as" {
			return true
		}
	}

	return false
}

// shuffleText randomly shuffles the order of words in the text
func (g *Golem) shuffleText(input string) string {
	// Split into words
	words := strings.Fields(input)
	if len(words) <= 1 {
		return input
	}

	// Create a copy of the words slice to avoid modifying the original
	shuffledWords := make([]string, len(words))
	copy(shuffledWords, words)

	// Shuffle the words using Fisher-Yates algorithm
	for i := len(shuffledWords) - 1; i > 0; i-- {
		// Generate a random index between 0 and i (inclusive)
		j := g.randomInt(i + 1)
		// Swap words at positions i and j
		shuffledWords[i], shuffledWords[j] = shuffledWords[j], shuffledWords[i]
	}

	return strings.Join(shuffledWords, " ")
}

// calculateLength calculates the length of text based on the specified type
func (g *Golem) calculateLength(content, lengthType string) string {
	// Use the utility function for most cases
	switch strings.ToLower(lengthType) {
	case "words", "sentences", "characters", "chars", "letters", "words_no_punctuation":
		return CalculateLength(content, lengthType)
	case "digits":
		// Count only digits
		digitCount := 0
		for _, r := range content {
			if r >= '0' && r <= '9' {
				digitCount++
			}
		}
		return strconv.Itoa(digitCount)
	case "lines":
		// Count lines
		lines := strings.Split(content, "\n")
		return strconv.Itoa(len(lines))
	default:
		// Default: count characters (including spaces)
		return strconv.Itoa(len(content))
	}
}

// splitText splits text by delimiter with optional limit
func (g *Golem) splitText(content, delimiter, limitStr string) string {
	if content == "" {
		return ""
	}

	// Parse limit if provided
	var limit int
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Split the text
	var parts []string
	if limit > 0 {
		parts = strings.SplitN(content, delimiter, limit)
	} else {
		parts = strings.Split(content, delimiter)
	}

	// Join parts with spaces for output
	return strings.Join(parts, " ")
}

// joinText joins words with delimiter
func (g *Golem) joinText(content, delimiter string) string {
	if content == "" {
		return ""
	}

	// Split content into words and join with delimiter
	words := strings.Fields(content)
	return strings.Join(words, delimiter)
}

// indentText adds indentation to each line of text
func (g *Golem) indentText(content string, level int, char string) string {
	if content == "" {
		return ""
	}

	// Convert literal \n to actual newlines
	content = strings.ReplaceAll(content, "\\n", "\n")

	// Create the indentation string
	indent := strings.Repeat(char, level)

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Add indentation to each line
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// Keep empty lines as-is
			result = append(result, line)
		} else {
			// Add indentation to non-empty lines
			result = append(result, indent+line)
		}
	}

	return strings.Join(result, "\n")
}

// dedentText removes indentation from each line of text
func (g *Golem) dedentText(content string, level int, char string) string {
	if content == "" {
		return ""
	}

	// Convert literal \n to actual newlines
	content = strings.ReplaceAll(content, "\\n", "\n")
	// Convert literal \t to actual tabs
	content = strings.ReplaceAll(content, "\\t", "\t")

	// Create the dedentation string
	dedent := strings.Repeat(char, level)

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Remove indentation from each line
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// Keep empty lines as-is
			result = append(result, line)
		} else if strings.HasPrefix(line, dedent) {
			// Remove the specified indentation if present
			result = append(result, line[len(dedent):])
		} else {
			// Line doesn't have the expected indentation, keep as-is
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// uniqueText removes duplicate elements from text using the specified delimiter
func (g *Golem) uniqueText(content string, delimiter string) string {
	if content == "" {
		return ""
	}

	// Split content by delimiter
	elements := strings.Split(content, delimiter)

	// Use a map to track seen elements and maintain order
	seen := make(map[string]bool)
	var uniqueElements []string

	for _, element := range elements {
		// Trim whitespace for comparison but preserve original spacing
		trimmed := strings.TrimSpace(element)
		if !seen[trimmed] {
			seen[trimmed] = true
			uniqueElements = append(uniqueElements, element)
		}
	}

	return strings.Join(uniqueElements, delimiter)
}

// randomInt generates a random integer between 0 and max (exclusive)
func (g *Golem) randomInt(max int) int {
	// Use a simple linear congruential generator for deterministic randomness
	// This ensures the same input always produces the same output for caching
	if g.randomSeed == 0 {
		g.randomSeed = 1
	}
	g.randomSeed = (g.randomSeed*1103515245 + 12345) & 0x7fffffff
	return int(g.randomSeed) % max
}

// normalizeTextForOutput normalizes text for output (similar to pattern matching but for display)
func (g *Golem) normalizeTextForOutput(input string) string {
	text := strings.TrimSpace(input)

	// Convert to uppercase
	text = strings.ToUpper(text)

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Replace special characters with spaces first
	text = strings.ReplaceAll(text, "@", " ")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.ReplaceAll(text, "_", " ")
	text = strings.ReplaceAll(text, ".", " ")
	text = strings.ReplaceAll(text, ":", " ")

	// Remove other punctuation for normalization
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, "#", "")
	text = strings.ReplaceAll(text, "$", "")
	text = strings.ReplaceAll(text, "%", "")
	text = strings.ReplaceAll(text, "^", "")
	text = strings.ReplaceAll(text, "&", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "(", "")
	text = strings.ReplaceAll(text, ")", "")

	// Expand contractions for better normalization
	text = expandContractions(text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// denormalizeText reverses normalization to restore more natural text
func (g *Golem) denormalizeText(input string) string {
	text := strings.TrimSpace(input)

	// Convert to lowercase for more natural text
	text = strings.ToLower(text)

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Capitalize first letter of each sentence
	text = g.capitalizeSentences(text)

	// Add basic punctuation where appropriate
	// This is a simplified approach - more sophisticated denormalization could be added
	if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") && !strings.HasSuffix(text, "?") && text != "" {
		text += "."
	}

	return text
}

// learnCategories parses the AIML categories in a <learn> (session-specific)
// or <learnf> (persistent) tag's content and adds them
func (g *Golem) learnCategories(content string, ctx *VariableContext, persistent bool) {
	if g.aimlKB == nil {
		return
	}
	content = strings.TrimSpace(content)
	g.LogInfo("Processing learn (persistent=%v): '%s'", persistent, content)

	categories, err := g.parseLearnContent(content)
	if err != nil {
		g.LogError("Failed to parse learn content: %v", err)
		return
	}

	for _, category := range categories {
		if persistent {
			err = g.addPersistentCategory(category)
		} else {
			err = g.addSessionCategory(category, ctx)
		}
		if err != nil {
			g.LogInfo("Failed to add learned category: %v", err)
		}
	}
}

// unlearnSessionCategories parses the AIML categories in an <unlearn> tag's
// content and removes them from the session's learned categories
func (g *Golem) unlearnSessionCategories(content string, ctx *VariableContext) {
	if g.aimlKB == nil {
		return
	}
	content = strings.TrimSpace(content)
	g.LogInfo("Processing unlearn: '%s'", content)

	categories, err := g.parseLearnContent(content)
	if err != nil {
		g.LogInfo("Failed to parse unlearn content: %v", err)
		return
	}

	for _, category := range categories {
		if err := g.removeSessionCategory(category, ctx); err != nil {
			g.LogInfo("Failed to remove session category: %v", err)
		}
	}
}

// VariableScope represents different scopes for variable resolution
type VariableScope int

const (
	ScopeLocal      VariableScope = iota // Local scope (within current template)
	ScopeSession                         // Session scope (within current chat session)
	ScopeTopic                           // Topic scope (within current topic)
	ScopeGlobal                          // Global scope (knowledge base wide)
	ScopeProperties                      // Properties scope (bot properties, read-only)
)

const (
	MaxSRAIRecursionDepth = 9 // Maximum recursion depth for SRAI processing
)

// VariableContext holds the context for variable resolution
type VariableContext struct {
	LocalVars      map[string]string  // Local variables (highest priority)
	Session        *ChatSession       // Session context
	Topic          string             // Current topic
	KnowledgeBase  *AIMLKnowledgeBase // Knowledge base context
	RecursionDepth int                // Current recursion depth for SRAI processing
	Wildcards      map[string]string  // Wildcard values from pattern matching
}

// resolveVariable resolves a variable using proper scope hierarchy
func (g *Golem) resolveVariable(varName string, ctx *VariableContext) string {
	g.LogInfo("Resolving variable '%s'", varName)

	// Check cache first
	if g.variableResolutionCache != nil {
		if value, found := g.variableResolutionCache.GetResolvedVariable(varName, ctx); found {
			g.LogInfo("Found variable '%s' in cache: '%s'", varName, value)
			return value
		}
	}

	// 1. Check local scope (highest priority)
	if ctx.LocalVars != nil {
		if value, exists := ctx.LocalVars[varName]; exists {
			g.LogInfo("Found variable '%s' in local scope: '%s'", varName, value)
			// Cache the result
			if g.variableResolutionCache != nil {
				g.variableResolutionCache.SetResolvedVariable(varName, value, ctx)
			}
			return value
		}
	}

	// 2. Check session scope
	if ctx.Session != nil && ctx.Session.Variables != nil {
		if value, exists := ctx.Session.Variables[varName]; exists {
			g.LogInfo("Found variable '%s' in session scope: '%s'", varName, value)
			// Cache the result
			if g.variableResolutionCache != nil {
				g.variableResolutionCache.SetResolvedVariable(varName, value, ctx)
			}
			return value
		}
	}

	// 3. Check topic scope
	if ctx.Session != nil && ctx.KnowledgeBase != nil && ctx.KnowledgeBase.TopicVars != nil {
		currentTopic := ctx.Session.GetSessionTopic()
		if currentTopic == "" {
			currentTopic = "default"
		}
		if topicVars, exists := ctx.KnowledgeBase.TopicVars[currentTopic]; exists {
			if value, exists := topicVars[varName]; exists {
				g.LogInfo("Found variable '%s' in topic scope '%s': '%s'", varName, currentTopic, value)
				// Cache the result
				if g.variableResolutionCache != nil {
					g.variableResolutionCache.SetResolvedVariable(varName, value, ctx)
				}
				return value
			}
		}
	}

	// 4. Check global scope (knowledge base variables)
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Variables != nil {
		g.LogInfo("Checking knowledge base variables: %v", ctx.KnowledgeBase.Variables)
		g.LogInfo("Knowledge base pointer: %p", ctx.KnowledgeBase)
		g.LogInfo("Knowledge base Variables pointer: %p", ctx.KnowledgeBase.Variables)
		if value, exists := ctx.KnowledgeBase.Variables[varName]; exists {
			g.LogInfo("Found variable '%s' in knowledge base: '%s'", varName, value)
			// Cache the result
			if g.variableResolutionCache != nil {
				g.variableResolutionCache.SetResolvedVariable(varName, value, ctx)
			}
			return value
		}
	} else {
		g.LogInfo("Knowledge base is nil or Variables is nil: KB=%v, Variables=%v", ctx.KnowledgeBase != nil, ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Variables != nil)
		if ctx.KnowledgeBase != nil {
			g.LogInfo("Knowledge base exists but Variables is nil - THIS IS THE BUG!")
		}
	}

	// 5. Check properties scope (read-only)
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Properties != nil {
		if value, exists := ctx.KnowledgeBase.Properties[varName]; exists {
			g.LogInfo("Found variable '%s' in properties: '%s'", varName, value)
			// Cache the result
			if g.variableResolutionCache != nil {
				g.variableResolutionCache.SetResolvedVariable(varName, value, ctx)
			}
			return value
		}
	}

	g.LogInfo("Variable '%s' not found", varName)
	return "" // Variable not found
}

// setVariable sets a variable in the appropriate scope
func (g *Golem) setVariable(varName, varValue string, scope VariableScope, ctx *VariableContext) {
	g.LogInfo("setVariable called: varName='%s', varValue='%s', scope=%v", varName, varValue, scope)

	switch scope {
	case ScopeLocal:
		if ctx.LocalVars == nil {
			ctx.LocalVars = make(map[string]string)
		}
		ctx.LocalVars[varName] = varValue
	case ScopeSession:
		if ctx.Session != nil {
			if ctx.Session.Variables == nil {
				ctx.Session.Variables = make(map[string]string)
			}
			ctx.Session.Variables[varName] = varValue
			g.noteSecret(varName, varValue)

			// Special case: if setting the topic variable, also set the session topic
			if varName == "topic" {
				ctx.Session.SetSessionTopic(varValue)
			}
		}
	case ScopeTopic:
		// Topic variables are stored in the knowledge base
		if ctx.KnowledgeBase != nil {
			if ctx.KnowledgeBase.TopicVars == nil {
				ctx.KnowledgeBase.TopicVars = make(map[string]map[string]string)
			}
			currentTopic := ""
			if ctx.Session != nil {
				currentTopic = ctx.Session.GetSessionTopic()
			}
			if currentTopic == "" {
				currentTopic = "default"
			}
			if ctx.KnowledgeBase.TopicVars[currentTopic] == nil {
				ctx.KnowledgeBase.TopicVars[currentTopic] = make(map[string]string)
			}
			ctx.KnowledgeBase.TopicVars[currentTopic][varName] = varValue
			g.LogDebug("Set topic variable '%s' to '%s' in topic '%s'", varName, varValue, currentTopic)
		}
	case ScopeGlobal:
		g.LogInfo("Setting global variable '%s' to '%s'", varName, varValue)
		g.LogInfo("Before: KB Variables=%v", ctx.KnowledgeBase.Variables)
		if ctx.KnowledgeBase != nil {
			if ctx.KnowledgeBase.Variables == nil {
				g.LogInfo("Creating new Variables map - THIS IS THE BUG!")
				ctx.KnowledgeBase.Variables = make(map[string]string)
			}
			ctx.KnowledgeBase.Variables[varName] = varValue
		}
		g.LogInfo("After: KB Variables=%v", ctx.KnowledgeBase.Variables)
	case ScopeProperties:
		// Properties are read-only, cannot be set
		g.LogInfo("Warning: Cannot set property '%s' - properties are read-only", varName)
	}
}

// convertToGoTimeFormat converts C-style time format strings to Go time format strings
func (g *Golem) convertToGoTimeFormat(format string) string {
	// Predefined format names (these take precedence)
	predefinedFormats := map[string]string{
		// Date formats
		"short":       "01/02/06",        // MM/DD/YY
		"long":        "January 2, 2006", // Full month name, day, year
		"iso":         "2006-01-02",      // YYYY-MM-DD
		"us":          "01/02/2006",      // MM/DD/YYYY
		"european":    "02/01/2006",      // DD/MM/YYYY
		"day":         "02",              // Day of month (01-31)
		"month":       "01",              // Month (01-12)
		"year":        "2006",            // 4-digit year
		"dayofyear":   "002",             // Day of year (001-366)
		"weekday":     "0",               // Weekday (0-6, Sunday=0)
		"week":        "01",              // Week number (00-53)
		"quarter":     "Q1",              // Quarter (Q1-Q4) - special handling needed
		"leapyear":    "yes",             // Leap year (yes/no) - special handling needed
		"daysinmonth": "31",              // Days in month (28-31) - special handling needed
		"daysinyear":  "365",             // Days in year (365/366) - special handling needed

		// Time formats
		"12":          "3:04 PM",                   // 12-hour format with AM/PM
		"24":          "15:04",                     // 24-hour format
		"time_iso":    "15:04:05",                  // ISO time format
		"hour":        "15",                        // Hour (00-23)
		"minute":      "04",                        // Minute (00-59)
		"second":      "05",                        // Second (00-59)
		"millisecond": "000",                       // Millisecond (000-999)
		"timezone":    "MST",                       // Timezone abbreviation
		"offset":      "-0700",                     // Timezone offset
		"unix":        "1136239445",                // Unix timestamp - special handling needed
		"unixmilli":   "1136239445000",             // Unix timestamp in milliseconds - special handling needed
		"unixnano":    "1136239445000000000",       // Unix timestamp in nanoseconds - special handling needed
		"rfc3339":     "2006-01-02T15:04:05Z07:00", // RFC3339 format
		"rfc822":      "02 Jan 06 15:04 MST",       // RFC822 format
		"kitchen":     "3:04PM",                    // Kitchen time format
		"stamp":       "Jan _2 15:04:05",           // Timestamp format
		"stampmilli":  "Jan _2 15:04:05.000",       // Timestamp with milliseconds
		"stampmicro":  "Jan _2 15:04:05.000000",    // Timestamp with microseconds
		"stampnano":   "Jan _2 15:04:05.000000000", // Timestamp with nanoseconds
	}

	// Check for predefined formats first
	if goFormat, exists := predefinedFormats[format]; exists {
		return goFormat
	}

	// Common C-style to Go time format conversions
	conversions := map[string]string{
		// Hours
		"%H": "15", // 24-hour format (00-23)
		"%I": "03", // 12-hour format (01-12)
		"%h": "3",  // 12-hour format (1-12)

		// Minutes and seconds
		"%M": "04",     // Minutes (00-59)
		"%S": "05",     // Seconds (00-59)
		"%f": "000000", // Microseconds (000000-999999)

		// Date
		"%Y": "2006", // 4-digit year
		"%y": "06",   // 2-digit year
		"%m": "01",   // Month (01-12)
		"%d": "02",   // Day (01-31)
		"%j": "002",  // Day of year (001-366)

		// Weekday
		"%w": "0",      // Weekday (0-6, Sunday=0)
		"%u": "1",      // Weekday (1-7, Monday=1)
		"%A": "Monday", // Full weekday name
		"%a": "Mon",    // Abbreviated weekday name
		"%W": "01",     // Week number (00-53)

		// Month
		"%B": "January", // Full month name
		"%b": "Jan",     // Abbreviated month name

		// Timezone
		"%Z": "MST",   // Timezone abbreviation
		"%z": "-0700", // Timezone offset

		// AM/PM
		"%p": "PM", // AM/PM indicator

		// Common patterns
		"HH":   "15",   // 24-hour format
		"MM":   "04",   // Minutes
		"SS":   "05",   // Seconds
		"YYYY": "2006", // 4-digit year
		"YY":   "06",   // 2-digit year
		"DD":   "02",   // Day
		"hh":   "03",   // 12-hour format
		"mm":   "04",   // Minutes
		"ss":   "05",   // Seconds
	}

	result := format

	// Apply conversions
	for cStyle, goStyle := range conversions {
		result = strings.ReplaceAll(result, cStyle, goStyle)
	}

	// If no conversions were made and it looks like a Go format, return as-is
	if result == format && g.looksLikeGoTimeFormat(format) {
		return format
	}

	return result
}

// looksLikeGoTimeFormat checks if the format string looks like a Go time format
func (g *Golem) looksLikeGoTimeFormat(format string) bool {
	goTimeVerbs := []string{
		"2006", "01", "02", "15", "04", "05", "Mon", "Monday", "Jan", "January",
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
		"PM", "pm", "AM", "am", "MST", "UTC", "Z07:00", "-07:00",
	}

	for _, verb := range goTimeVerbs {
		if strings.Contains(format, verb) {
			return true
		}
	}

	return false
}

// loadDefaultProperties loads default bot properties
func (g *Golem) loadDefaultProperties(kb *AIMLKnowledgeBase) error {
	// Set default properties
	defaultProps := map[string]string{
		"name":              "Golem",
		"version":           GetVersion(),
		"master":            "User",
		"birthplace":        "Go",
		"birthday":          "2025-09-23",
		"gender":            "neutral",
		"species":           "AI",
		"job":               "Assistant",
		"personality":       "friendly",
		"mood":              "helpful",
		"attitude":          "positive",
		"language":          "English",
		"location":          "Virtual",
		"timezone":          "UTC",
		"max_loops":         "10",
		"timeout":           "30000",
		"jokemode":          "true",
		"learnmode":         "true",
		"default_response":  "I'm not sure I understand. Could you rephrase that?",
		"error_response":    "Sorry, I encountered an error processing your request.",
		"thinking_response": "Let me think about that...",
		"memory_size":       "1000",
		"forget_time":       "3600",
		"learning_enabled":  "true",
		"pattern_limit":     "1000",
		"response_limit":    "5000",
	}

	// Copy default properties to knowledge base
	for key, value := range defaultProps {
		kb.Properties[key] = value
	}

	// Try to load properties from file
	propertiesFile := "testdata/bot.properties"
	content, err := g.LoadFile(propertiesFile)
	if err == nil {
		// Parse properties file
		fileProps, err := g.parsePropertiesFile(content)
		if err == nil {
			// Override defaults with file properties
			for key, value := range fileProps {
				kb.Properties[key] = value
			}
			g.LogInfo("Loaded properties from file: %s", propertiesFile)
		} else {
			g.LogInfo("Could not parse properties file: %v", err)
		}
	} else {
		g.LogInfo("Could not load properties file: %v", err)
	}

	return nil
}

// parsePropertiesFile parses a properties file
func (g *Golem) parsePropertiesFile(content string) (map[string]string, error) {
	properties := make(map[string]string)
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid property format at line %d: %s", lineNum+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty property key at line %d", lineNum+1)
		}

		properties[key] = value
	}

	return properties, nil
}

// GetProperty retrieves a property value
func (kb *AIMLKnowledgeBase) GetProperty(key string) string {
	// Check properties first
	if value, exists := kb.Properties[key]; exists {
		return value
	}
	// Check variables as fallback
	if value, exists := kb.Variables[key]; exists {
		return value
	}
	return ""
}

// SetProperty sets a property value
func (kb *AIMLKnowledgeBase) SetProperty(key, value string) {
	kb.Properties[key] = value
}

// AddSetMember adds a member to a set
func (kb *AIMLKnowledgeBase) AddSetMember(setName, member string) {
	setName = strings.ToUpper(setName)
	if kb.Sets[setName] == nil {
		kb.Sets[setName] = make([]string, 0)
	}
	// Check if member already exists
	for _, existing := range kb.Sets[setName] {
		if existing == strings.ToUpper(member) {
			return // Already exists
		}
	}
	kb.Sets[setName] = append(kb.Sets[setName], strings.ToUpper(member))
}

// AddSetMembers adds multiple members to a set
func (kb *AIMLKnowledgeBase) AddSetMembers(setName string, members []string) {
	for _, member := range members {
		kb.AddSetMember(setName, member)
	}
}

// GetSetMembers returns all members of a set
func (kb *AIMLKnowledgeBase) GetSetMembers(setName string) []string {
	setName = strings.ToUpper(setName)
	if kb.Sets[setName] == nil {
		return []string{}
	}
	return kb.Sets[setName]
}

// IsSetMember checks if a word is a member of a set
func (kb *AIMLKnowledgeBase) IsSetMember(setName, word string) bool {
	setName = strings.ToUpper(setName)
	if kb.Sets[setName] == nil {
		return false
	}
	upperWord := strings.ToUpper(word)
	for _, member := range kb.Sets[setName] {
		if member == upperWord {
			return true
		}
	}
	return false
}

// SetTopic sets the current topic for a category
func (kb *AIMLKnowledgeBase) SetTopic(pattern, topic string) {
	kb.patternsMu.Lock()
	defer kb.patternsMu.Unlock()
	if category, exists := kb.Patterns[pattern]; exists {
		category.Topic = strings.ToUpper(topic)
	}
}

// GetTopic returns the topic for a pattern
func (kb *AIMLKnowledgeBase) GetTopic(pattern string) string {
	kb.patternsMu.RLock()
	defer kb.patternsMu.RUnlock()
	if category, exists := kb.Patterns[pattern]; exists {
		return category.Topic
	}
	return ""
}

// SetSessionTopic sets the current topic for a session
func (session *ChatSession) SetSessionTopic(topic string) {
	session.Topic = topic
}

// GetSessionTopic returns the current topic for a session
func (session *ChatSession) GetSessionTopic() string {
	return session.Topic
}

// AddToThatHistory adds a bot response to the that history with enhanced management
func (session *ChatSession) AddToThatHistory(response string) {
	// <that> sees the text the user read, not rich media markup
	response = RichText(response)

	// Use enhanced context management if available
	if session.ContextConfig != nil && session.ContextConfig.EnableCompression {
		session.AddToThatHistoryEnhanced(response, []string{}, make(map[string]interface{}))
		return
	}

	// Fallback to basic management
	maxDepth := 10
	if session.ContextConfig != nil {
		maxDepth = session.ContextConfig.MaxThatDepth
	}

	// Keep only the last N responses to prevent memory bloat
	if len(session.ThatHistory) >= maxDepth {
		session.ThatHistory = session.ThatHistory[1:]
	}
	session.ThatHistory = append(session.ThatHistory, response)
}

// GetLastThat returns the last bot response for that matching
func (session *ChatSession) GetLastThat() string {
	if len(session.ThatHistory) == 0 {
		return ""
	}
	return session.ThatHistory[len(session.ThatHistory)-1]
}

// GetThatHistory returns the that history
func (session *ChatSession) GetThatHistory() []string {
	return session.ThatHistory
}

// AddToRequestHistory adds a user request to the request history
func (session *ChatSession) AddToRequestHistory(request string) {
	// Keep only the last 10 requests to prevent memory bloat
	if len(session.RequestHistory) >= 10 {
		session.RequestHistory = session.RequestHistory[1:]
	}
	session.RequestHistory = append(session.RequestHistory, request)
}

// GetRequestHistory returns the request history
func (session *ChatSession) GetRequestHistory() []string {
	return session.RequestHistory
}

// GetRequestByIndex returns a request by index (1-based, where 1 is most recent)
func (session *ChatSession) GetRequestByIndex(index int) string {
	if index < 1 || index > len(session.RequestHistory) {
		return ""
	}
	// Convert to 0-based index (most recent is at the end)
	actualIndex := len(session.RequestHistory) - index
	return session.RequestHistory[actualIndex]
}

// AddToResponseHistory adds a bot response to the response history
func (session *ChatSession) AddToResponseHistory(response string) {
	// Keep only the last 10 responses to prevent memory bloat
	if len(session.ResponseHistory) >= 10 {
		session.ResponseHistory = session.ResponseHistory[1:]
	}
	session.ResponseHistory = append(session.ResponseHistory, response)
}

// GetResponseHistory returns the response history
func (session *ChatSession) GetResponseHistory() []string {
	return session.ResponseHistory
}

// GetResponseByIndex returns a response by index (1-based, where 1 is most recent)
func (session *ChatSession) GetResponseByIndex(index int) string {
	if index < 1 || index > len(session.ResponseHistory) {
		return ""
	}
	// Convert to 0-based index (most recent is at the end)
	actualIndex := len(session.ResponseHistory) - index
	return session.ResponseHistory[actualIndex]
}

// GetThatByIndex returns a that context by index (1-based, where 1 is most recent, 0 means last)
func (session *ChatSession) GetThatByIndex(index int) string {
	if index == 0 {
		// 0 means last response (most recent)
		return session.GetLastThat()
	}
	if index < 1 || index > len(session.ThatHistory) {
		return ""
	}
	// Convert to 0-based index (most recent is at the end)
	// Index 1 = most recent (last item in array)
	// Index 2 = second most recent (second to last item in array)
	actualIndex := len(session.ThatHistory) - index
	return session.ThatHistory[actualIndex]
}

// GetThatHistoryStats returns statistics about the that history
func (session *ChatSession) GetThatHistoryStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_items":    len(session.ThatHistory),
		"max_depth":      10,
		"memory_usage":   session.calculateThatHistoryMemoryUsage(),
		"oldest_item":    "",
		"newest_item":    "",
		"average_length": 0.0,
	}

	if session.ContextConfig != nil {
		stats["max_depth"] = session.ContextConfig.MaxThatDepth
	}

	if len(session.ThatHistory) > 0 {
		stats["oldest_item"] = session.ThatHistory[0]
		stats["newest_item"] = session.ThatHistory[len(session.ThatHistory)-1]

		// Calculate average length
		totalLength := 0
		for _, item := range session.ThatHistory {
			totalLength += len(item)
		}
		stats["average_length"] = float64(totalLength) / float64(len(session.ThatHistory))
	}

	return stats
}

// calculateThatHistoryMemoryUsage estimates memory usage of that history
func (session *ChatSession) calculateThatHistoryMemoryUsage() int {
	return CalculateMemoryUsage(session.ThatHistory)
}

// CompressThatHistory compresses the that history using smart compression
func (session *ChatSession) CompressThatHistory() {
	if session.ContextConfig == nil || !session.ContextConfig.EnableCompression {
		return
	}

	// If we're under the compression threshold, no need to compress
	if len(session.ThatHistory) < session.ContextConfig.CompressionThreshold {
		return
	}

	// Keep the most recent items and compress older ones
	keepCount := session.ContextConfig.MaxThatDepth / 2
	if keepCount < 5 {
		keepCount = 5
	}

	// Keep the most recent items
	if len(session.ThatHistory) > keepCount {
		// Remove older items (keep the last keepCount items)
		itemsToRemove := len(session.ThatHistory) - keepCount
		session.ThatHistory = session.ThatHistory[itemsToRemove:]
	}
}

// ValidateThatHistory validates the that history for consistency
func (session *ChatSession) ValidateThatHistory() []string {
	var errors []string

	// Check for empty items
	for i, item := range session.ThatHistory {
		if strings.TrimSpace(item) == "" {
			errors = append(errors, fmt.Sprintf("Empty that history item at index %d", i))
		}
	}

	// Check for duplicate consecutive items
	for i := 1; i < len(session.ThatHistory); i++ {
		if session.ThatHistory[i] == session.ThatHistory[i-1] {
			errors = append(errors, fmt.Sprintf("Duplicate consecutive that history items at indices %d and %d", i-1, i))
		}
	}

	// Check memory usage
	memoryUsage := session.calculateThatHistoryMemoryUsage()
	if memoryUsage > 100*1024 { // 100KB
		errors = append(errors, fmt.Sprintf("That history memory usage too high: %d bytes", memoryUsage))
	}

	return errors
}

// ClearThatHistory clears the that history
func (session *ChatSession) ClearThatHistory() {
	session.ThatHistory = make([]string, 0)
}

// GetThatHistoryDebugInfo returns detailed debug information about that history
func (session *ChatSession) GetThatHistoryDebugInfo() map[string]interface{} {
	debugInfo := map[string]interface{}{
		"history":           session.ThatHistory,
		"length":            len(session.ThatHistory),
		"memory_usage":      session.calculateThatHistoryMemoryUsage(),
		"validation_errors": session.ValidateThatHistory(),
		"config":            session.ContextConfig,
	}

	// Add pattern matching debug info
	if len(session.ThatHistory) > 0 {
		debugInfo["last_that"] = session.GetLastThat()
		debugInfo["normalized_last_that"] = NormalizeThatPattern(session.GetLastThat())
	}

	return debugInfo
}

// InitializeContextConfig initializes the context configuration with default values
func (session *ChatSession) InitializeContextConfig() {
	if session.ContextConfig == nil {
		session.ContextConfig = &ContextConfig{
			MaxThatDepth:         20,
			MaxRequestDepth:      20,
			MaxResponseDepth:     20,
			MaxTotalContext:      100,
			CompressionThreshold: 50,
			WeightDecay:          0.9,
			EnableCompression:    true,
			EnableAnalytics:      true,
			EnablePruning:        true,
		}
	}

	// Initialize maps if they don't exist
	if session.ContextWeights == nil {
		session.ContextWeights = make(map[string]float64)
	}
	if session.ContextUsage == nil {
		session.ContextUsage = make(map[string]int)
	}
	if session.ContextTags == nil {
		session.ContextTags = make(map[string][]string)
	}
	if session.ContextMetadata == nil {
		session.ContextMetadata = make(map[string]interface{})
	}
}

// AddToThatHistoryEnhanced adds a bot response to the that history with enhanced context management
func (session *ChatSession) AddToThatHistoryEnhanced(response string, tags []string, metadata map[string]interface{}) {
	response = RichText(response)
	session.InitializeContextConfig()

	// Apply depth limit
	if len(session.ThatHistory) >= session.ContextConfig.MaxThatDepth {
		session.ThatHistory = session.ThatHistory[1:]
	}

	session.ThatHistory = append(session.ThatHistory, response)

	// Update context analytics if enabled
	if session.ContextConfig.EnableAnalytics {
		session.updateContextAnalytics("that", response, tags, metadata)
	}

	// Apply smart pruning if enabled
	if session.ContextConfig.EnablePruning {
		session.pruneContextIfNeeded()
	}
}

// AddToRequestHistoryEnhanced adds a user request to the request history with enhanced context management
func (session *ChatSession) AddToRequestHistoryEnhanced(request string, tags []string, metadata map[string]interface{}) {
	session.InitializeContextConfig()

	// Apply depth limit
	if len(session.RequestHistory) >= session.ContextConfig.MaxRequestDepth {
		session.RequestHistory = session.RequestHistory[1:]
	}

	session.RequestHistory = append(session.RequestHistory, request)

	// Update context analytics if enabled
	if session.ContextConfig.EnableAnalytics {
		session.updateContextAnalytics("request", request, tags, metadata)
	}

	// Apply smart pruning if enabled
	if session.ContextConfig.EnablePruning {
		session.pruneContextIfNeeded()
	}
}

// AddToResponseHistoryEnhanced adds a bot response to the response history with enhanced context management
func (session *ChatSession) AddToResponseHistoryEnhanced(response string, tags []string, metadata map[string]interface{}) {
	session.InitializeContextConfig()

	// Apply depth limit
	if len(session.ResponseHistory) >= session.ContextConfig.MaxResponseDepth {
		session.ResponseHistory = session.ResponseHistory[1:]
	}

	session.ResponseHistory = append(session.ResponseHistory, response)

	// Update context analytics if enabled
	if session.ContextConfig.EnableAnalytics {
		session.updateContextAnalytics("response", response, tags, metadata)
	}

	// Apply smart pruning if enabled
	if session.ContextConfig.EnablePruning {
		session.pruneContextIfNeeded()
	}
}

// updateContextAnalytics updates context analytics data
func (session *ChatSession) updateContextAnalytics(_, content string, tags []string, metadata map[string]interface{}) {
	// Update usage count
	session.ContextUsage[content]++

	// Update tags
	if len(tags) > 0 {
		session.ContextTags[content] = tags
		for _, tag := range tags {
			if session.ContextMetadata["tag_distribution"] == nil {
				session.ContextMetadata["tag_distribution"] = make(map[string]int)
			}
			if tagDist, ok := session.ContextMetadata["tag_distribution"].(map[string]int); ok {
				tagDist[tag]++
			}
		}
	}

	// Update metadata
	if metadata != nil {
		session.ContextMetadata[content] = metadata
	}

	// Update weights based on usage
	session.updateContextWeights()
}

// updateContextWeights updates context weights based on usage and age
func (session *ChatSession) updateContextWeights() {

	// Update weights for that history
	for i, content := range session.ThatHistory {
		age := len(session.ThatHistory) - i - 1
		usageCount := session.ContextUsage[content]
		weight := float64(usageCount) * math.Pow(session.ContextConfig.WeightDecay, float64(age))
		session.ContextWeights[fmt.Sprintf("that_%d", i)] = weight
	}

	// Update weights for request history
	for i, content := range session.RequestHistory {
		age := len(session.RequestHistory) - i - 1
		usageCount := session.ContextUsage[content]
		weight := float64(usageCount) * math.Pow(session.ContextConfig.WeightDecay, float64(age))
		session.ContextWeights[fmt.Sprintf("request_%d", i)] = weight
	}

	// Update weights for response history
	for i, content := range session.ResponseHistory {
		age := len(session.ResponseHistory) - i - 1
		usageCount := session.ContextUsage[content]
		weight := float64(usageCount) * math.Pow(session.ContextConfig.WeightDecay, float64(age))
		session.ContextWeights[fmt.Sprintf("response_%d", i)] = weight
	}
}

// pruneContextIfNeeded applies smart pruning when context limits are exceeded
func (session *ChatSession) pruneContextIfNeeded() {
	totalContext := len(session.ThatHistory) + len(session.RequestHistory) + len(session.ResponseHistory)

	if totalContext > session.ContextConfig.MaxTotalContext {
		// Calculate items to remove
		itemsToRemove := totalContext - session.ContextConfig.MaxTotalContext

		// Find least weighted items to remove
		itemsToPrune := session.findLeastWeightedItems(itemsToRemove)

		// Remove items
		for _, item := range itemsToPrune {
			session.removeContextItem(item)
		}

		// Update pruning count
		if session.ContextMetadata["pruning_count"] == nil {
			session.ContextMetadata["pruning_count"] = 0
		}
		session.ContextMetadata["pruning_count"] = session.ContextMetadata["pruning_count"].(int) + 1
		session.ContextMetadata["last_pruned"] = time.Now().Format(time.RFC3339)
	}
}

// findLeastWeightedItems finds the least weighted context items for pruning
func (session *ChatSession) findLeastWeightedItems(count int) []string {
	type weightedItem struct {
		key    string
		weight float64
	}

	var items []weightedItem

	// Collect all context items with their weights
	for i, content := range session.ThatHistory {
		key := fmt.Sprintf("that_%d", i)
		weight := session.ContextWeights[key]
		items = append(items, weightedItem{key: content, weight: weight})
	}

	for i, content := range session.RequestHistory {
		key := fmt.Sprintf("request_%d", i)
		weight := session.ContextWeights[key]
		items = append(items, weightedItem{key: content, weight: weight})
	}

	for i, content := range session.ResponseHistory {
		key := fmt.Sprintf("response_%d", i)
		weight := session.ContextWeights[key]
		items = append(items, weightedItem{key: content, weight: weight})
	}

	// Sort by weight (ascending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].weight < items[j].weight
	})

	// Return the least weighted items
	var result []string
	for i := 0; i < count && i < len(items); i++ {
		result = append(result, items[i].key)
	}

	return result
}

// removeContextItem removes a context item from all histories
func (session *ChatSession) removeContextItem(content string) {
	// Remove from that history
	for i, item := range session.ThatHistory {
		if item == content {
			session.ThatHistory = append(session.ThatHistory[:i], session.ThatHistory[i+1:]...)
			break
		}
	}

	// Remove from request history
	for i, item := range session.RequestHistory {
		if item == content {
			session.RequestHistory = append(session.RequestHistory[:i], session.RequestHistory[i+1:]...)
			break
		}
	}

	// Remove from response history
	for i, item := range session.ResponseHistory {
		if item == content {
			session.ResponseHistory = append(session.ResponseHistory[:i], session.ResponseHistory[i+1:]...)
			break
		}
	}

	// Clean up associated data
	delete(session.ContextUsage, content)
	delete(session.ContextTags, content)
	delete(session.ContextMetadata, content)
}

// GetContextAnalytics returns current context analytics
func (session *ChatSession) GetContextAnalytics() *ContextAnalytics {
	session.InitializeContextConfig()

	analytics := &ContextAnalytics{
		TotalItems:      len(session.ThatHistory) + len(session.RequestHistory) + len(session.ResponseHistory),
		ThatItems:       len(session.ThatHistory),
		RequestItems:    len(session.RequestHistory),
		ResponseItems:   len(session.ResponseHistory),
		TagDistribution: make(map[string]int),
	}

	// Calculate average weight
	totalWeight := 0.0
	weightCount := 0
	for _, weight := range session.ContextWeights {
		totalWeight += weight
		weightCount++
	}
	if weightCount > 0 {
		analytics.AverageWeight = totalWeight / float64(weightCount)
	}

	// Find most and least used items
	var mostUsed, leastUsed []string
	maxUsage := 0
	minUsage := int(^uint(0) >> 1) // Max int

	for content, usage := range session.ContextUsage {
		if usage > maxUsage {
			maxUsage = usage
			mostUsed = []string{content}
		} else if usage == maxUsage {
			mostUsed = append(mostUsed, content)
		}

		if usage < minUsage {
			minUsage = usage
			leastUsed = []string{content}
		} else if usage == minUsage {
			leastUsed = append(leastUsed, content)
		}
	}

	analytics.MostUsedItems = mostUsed
	analytics.LeastUsedItems = leastUsed

	// Calculate memory usage (rough estimate)
	analytics.MemoryUsage = len(session.ThatHistory)*50 + len(session.RequestHistory)*50 + len(session.ResponseHistory)*50

	// Get tag distribution
	if tagDist, ok := session.ContextMetadata["tag_distribution"].(map[string]int); ok {
		analytics.TagDistribution = tagDist
	}

	// Get pruning info
	if pruningCount, ok := session.ContextMetadata["pruning_count"].(int); ok {
		analytics.PruningCount = pruningCount
	}
	if lastPruned, ok := session.ContextMetadata["last_pruned"].(string); ok {
		analytics.LastPruned = lastPruned
	}

	return analytics
}

// SearchContext searches through context history
func (session *ChatSession) SearchContext(query string, contextTypes []string) []ContextItem {
	session.InitializeContextConfig()

	var results []ContextItem

	query = strings.ToLower(query)

	// Search that history
	if len(contextTypes) == 0 || containsString(contextTypes, "that") {
		for i, content := range session.ThatHistory {
			if strings.Contains(strings.ToLower(content), query) {
				weight := session.ContextWeights[fmt.Sprintf("that_%d", i)]
				usageCount := session.ContextUsage[content]
				tags := session.ContextTags[content]
				metadata := session.ContextMetadata[content]

				var metadataMap map[string]interface{}
				if metadata != nil {
					if m, ok := metadata.(map[string]interface{}); ok {
						metadataMap = m
					}
				}

				item := ContextItem{
					Content:    content,
					Type:       "that",
					Index:      i,
					Weight:     weight,
					Tags:       tags,
					Metadata:   metadataMap,
					UsageCount: usageCount,
				}
				results = append(results, item)
			}
		}
	}

	// Search request history
	if len(contextTypes) == 0 || containsString(contextTypes, "request") {
		for i, content := range session.RequestHistory {
			if strings.Contains(strings.ToLower(content), query) {
				weight := session.ContextWeights[fmt.Sprintf("request_%d", i)]
				usageCount := session.ContextUsage[content]
				tags := session.ContextTags[content]
				metadata := session.ContextMetadata[content]

				var metadataMap map[string]interface{}
				if metadata != nil {
					if m, ok := metadata.(map[string]interface{}); ok {
						metadataMap = m
					}
				}

				item := ContextItem{
					Content:    content,
					Type:       "request",
					Index:      i,
					Weight:     weight,
					Tags:       tags,
					Metadata:   metadataMap,
					UsageCount: usageCount,
				}
				results = append(results, item)
			}
		}
	}

	// Search response history
	if len(contextTypes) == 0 || containsString(contextTypes, "response") {
		for i, content := range session.ResponseHistory {
			if strings.Contains(strings.ToLower(content), query) {
				weight := session.ContextWeights[fmt.Sprintf("response_%d", i)]
				usageCount := session.ContextUsage[content]
				tags := session.ContextTags[content]
				metadata := session.ContextMetadata[content]

				var metadataMap map[string]interface{}
				if metadata != nil {
					if m, ok := metadata.(map[string]interface{}); ok {
						metadataMap = m
					}
				}

				item := ContextItem{
					Content:    content,
					Type:       "response",
					Index:      i,
					Weight:     weight,
					Tags:       tags,
					Metadata:   metadataMap,
					UsageCount: usageCount,
				}
				results = append(results, item)
			}
		}
	}

	// Sort by weight (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Weight > results[j].Weight
	})

	return results
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CompressContext compresses old context items to save memory
func (session *ChatSession) CompressContext() {
	if !session.ContextConfig.EnableCompression {
		return
	}

	session.InitializeContextConfig()

	// Only compress if we have more items than the threshold
	totalContext := len(session.ThatHistory) + len(session.RequestHistory) + len(session.ResponseHistory)
	if totalContext < session.ContextConfig.CompressionThreshold {
		return
	}

	// Compress old items by truncating them
	itemsToCompress := totalContext - session.ContextConfig.CompressionThreshold

	// Compress that history
	if len(session.ThatHistory) > 0 {
		compressCount := min(itemsToCompress, len(session.ThatHistory))
		for i := 0; i < compressCount; i++ {
			if len(session.ThatHistory[i]) > 50 {
				session.ThatHistory[i] = session.ThatHistory[i][:47] + "..."
			}
		}
	}

	// Compress request history
	if len(session.RequestHistory) > 0 {
		compressCount := min(itemsToCompress, len(session.RequestHistory))
		for i := 0; i < compressCount; i++ {
			if len(session.RequestHistory[i]) > 50 {
				session.RequestHistory[i] = session.RequestHistory[i][:47] + "..."
			}
		}
	}

	// Compress response history
	if len(session.ResponseHistory) > 0 {
		compressCount := min(itemsToCompress, len(session.ResponseHistory))
		for i := 0; i < compressCount; i++ {
			if len(session.ResponseHistory[i]) > 50 {
				session.ResponseHistory[i] = session.ResponseHistory[i][:47] + "..."
			}
		}
	}

	// Update compression ratio
	originalSize := totalContext * 50 // Rough estimate
	compressedSize := len(session.ThatHistory)*50 + len(session.RequestHistory)*50 + len(session.ResponseHistory)*50
	if originalSize > 0 {
		session.ContextMetadata["compression_ratio"] = float64(compressedSize) / float64(originalSize)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NormalizedContent represents content that has been normalized with preserved sections
type NormalizedContent struct {
	NormalizedText    string
	PreservedSections map[string]string
}

// expandContractions expands common English contractions for better pattern matching
// contractions maps contractions to their expanded forms
var contractions = map[string]string{
	// Common contractions
	"I'M": "I AM", "I'm": "I am", "i'm": "i am",
	"YOU'RE": "YOU ARE", "You're": "You are", "you're": "you are",
	"HE'S": "HE IS", "He's": "He is", "he's": "he is",
	"SHE'S": "SHE IS", "She's": "She is", "she's": "she is",
	"IT'S": "IT IS", "It's": "It is", "it's": "it is",
	"WE'RE": "WE ARE", "We're": "We are", "we're": "we are",
	"THEY'RE": "THEY ARE", "They're": "They are", "they're": "they are",

	// Negative contractions
	"DON'T": "DO NOT", "Don't": "Do not", "don't": "do not",
	"WON'T": "WILL NOT", "Won't": "Will not", "won't": "will not",
	"CAN'T": "CANNOT", "Can't": "Cannot", "can't": "cannot",
	"ISN'T": "IS NOT", "Isn't": "Is not", "isn't": "is not",
	"AREN'T": "ARE NOT", "Aren't": "Are not", "aren't": "are not",
	"WASN'T": "WAS NOT", "Wasn't": "Was not", "wasn't": "was not",
	"WEREN'T": "WERE NOT", "Weren't": "Were not", "weren't": "were not",
	"HASN'T": "HAS NOT", "Hasn't": "Has not", "hasn't": "has not",
	"HAVEN'T": "HAVE NOT", "Haven't": "Have not", "haven't": "have not",
	"HADN'T": "HAD NOT", "Hadn't": "Had not", "hadn't": "had not",
	"WOULDN'T": "WOULD NOT", "Wouldn't": "Would not", "wouldn't": "would not",
	"SHOULDN'T": "SHOULD NOT", "Shouldn't": "Should not", "shouldn't": "should not",
	"COULDN'T": "COULD NOT", "Couldn't": "Could not", "couldn't": "could not",
	"MUSTN'T": "MUST NOT", "Mustn't": "Must not", "mustn't": "must not",
	"SHAN'T": "SHALL NOT", "Shan't": "Shall not", "shan't": "shall not",

	// Future tense contractions
	"I'LL": "I WILL", "I'll": "I will", "i'll": "i will",
	"YOU'LL": "YOU WILL", "You'll": "You will", "you'll": "you will",
	"HE'LL": "HE WILL", "He'll": "He will", "he'll": "he will",
	"SHE'LL": "SHE WILL", "She'll": "She will", "she'll": "she will",
	"IT'LL": "IT WILL", "It'll": "It will", "it'll": "it will",
	"WE'LL": "WE WILL", "We'll": "We will", "we'll": "we will",
	"THEY'LL": "THEY WILL", "They'll": "They will", "they'll": "they will",

	// Perfect tense contractions
	"I'VE": "I HAVE", "I've": "I have", "i've": "i have",
	"YOU'VE": "YOU HAVE", "You've": "You have", "you've": "you have",
	"WE'VE": "WE HAVE", "We've": "We have", "we've": "we have",
	"THEY'VE": "THEY HAVE", "They've": "They have", "they've": "they have",

	// Past tense contractions (I'D can be either HAD or WOULD, context dependent)
	"I'D": "I WOULD", "I'd": "I would", "i'd": "i would",
	"YOU'D": "YOU HAD", "You'd": "You had", "you'd": "you had",
	"HE'D": "HE HAD", "He'd": "He had", "he'd": "he had",
	"SHE'D": "SHE HAD", "She'd": "She had", "she'd": "she had",
	"IT'D": "IT HAD", "It'd": "It had", "it'd": "it had",
	"WE'D": "WE HAD", "We'd": "We had", "we'd": "we had",
	"THEY'D": "THEY HAD", "They'd": "They had", "they'd": "they had",

	// Other common contractions
	"LET'S": "LET US", "Let's": "Let us", "let's": "let us",
	"THAT'S": "THAT IS", "That's": "That is", "that's": "that is",
	"THERE'S": "THERE IS", "There's": "There is", "there's": "there is",
	"HERE'S": "HERE IS", "Here's": "Here is", "here's": "here is",
	"WHAT'S": "WHAT IS", "What's": "What is", "what's": "what is",
	"WHO'S": "WHO IS", "Who's": "Who is", "who's": "who is",
	"WHERE'S": "WHERE IS", "Where's": "Where is", "where's": "where is",
	"WHEN'S": "WHEN IS", "When's": "When is", "when's": "when is",
	"WHY'S": "WHY IS", "Why's": "Why is", "why's": "why is",
	"HOW'S": "HOW IS", "How's": "How is", "how's": "how is",

	// Possessive contractions (less common but useful)
	"Y'ALL": "YOU ALL", "Y'all": "You all", "y'all": "you all",
	"MA'AM": "MADAM", "Ma'am": "Madam", "ma'am": "madam",
	"O'CLOCK": "OF THE CLOCK", "o'clock": "of the clock",

	// Complex contractions (must be processed before simple ones)
	"I'D'VE": "I WOULD HAVE", "I'd've": "I would have", "i'd've": "i would have",
	"WOULDN'T'VE": "WOULD NOT HAVE", "Wouldn't've": "Would not have", "wouldn't've": "would not have",
	"SHOULDN'T'VE": "SHOULD NOT HAVE", "Shouldn't've": "Should not have", "shouldn't've": "should not have",
	"COULDN'T'VE": "COULD NOT HAVE", "Couldn't've": "Could not have", "couldn't've": "could not have",
	"MUSTN'T'VE": "MUST NOT HAVE", "Mustn't've": "Must not have", "mustn't've": "must not have",
}

// contractionKeys lists the contractions longest first, so that longer ones
// are expanded before the shorter ones they contain
var contractionKeys = func() []string {
	keys := make([]string, 0, len(contractions))
	for k := range contractions {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}()

func expandContractions(text string) string {
	// Every contraction contains an apostrophe
	if strings.Contains(text, "'") {
		for _, contraction := range contractionKeys {
			text = strings.ReplaceAll(text, contraction, contractions[contraction])
		}
	}

	// Post-process to handle context-dependent contractions
	// "I WOULD KNOWN" should be "I HAD KNOWN" (I'd + past participle)
	text = strings.ReplaceAll(text, "I WOULD KNOWN", "I HAD KNOWN")
	text = strings.ReplaceAll(text, "I WOULD DONE", "I HAD DONE")
	text = strings.ReplaceAll(text, "I WOULD GONE", "I HAD GONE")
	text = strings.ReplaceAll(text, "I WOULD SEEN", "I HAD SEEN")
	text = strings.ReplaceAll(text, "I WOULD BEEN", "I HAD BEEN")
	text = strings.ReplaceAll(text, "I WOULD HAD", "I HAD HAD")

	return text
}

// denormalizeText restores the original content from normalized text
func denormalizeText(normalized NormalizedContent) string {
	text := normalized.NormalizedText

	// Restore preserved sections in reverse order of insertion
	// (to avoid conflicts with shorter placeholders)
	placeholders := make([]string, 0, len(normalized.PreservedSections))
	for placeholder := range normalized.PreservedSections {
		placeholders = append(placeholders, placeholder)
	}

	// Sort placeholders by length (longest first) to avoid replacement conflicts
	sort.Slice(placeholders, func(i, j int) bool {
		return len(placeholders[i]) > len(placeholders[j])
	})

	for _, placeholder := range placeholders {
		original := normalized.PreservedSections[placeholder]
		text = strings.ReplaceAll(text, placeholder, original)
	}

	return text
}

// NormalizeForMatchingCasePreserving normalizes text for pattern matching while preserving case
func NormalizeForMatchingCasePreserving(input string) string {
	// For pattern matching, we need normalization but preserve case for wildcard extraction
	// This is similar to normalizeForMatching but without case conversion

	// First, preserve set and topic tags before normalization
	tempSetTags := make(map[string]string)
	tempTopicTags := make(map[string]string)

	// Replace set tags temporarily
	setPattern := patternSetTagRe
	setMatches := setPattern.FindAllString(input, -1)
	for i, match := range setMatches {
		placeholder := fmt.Sprintf("__TEMP_SET_%d__", i)
		tempSetTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	// Replace topic tags temporarily
	topicPattern := patternTopicTagRe
	topicMatches := topicPattern.FindAllString(input, -1)
	for i, match := range topicMatches {
		placeholder := fmt.Sprintf("__TEMP_TOPIC_%d__", i)
		tempTopicTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	text := strings.TrimSpace(input)

	// Restore set and topic tags with preserved case
	for placeholder, original := range tempSetTags {
		text = strings.ReplaceAll(text, placeholder, original)
	}
	for placeholder, original := range tempTopicTags {
		text = strings.ReplaceAll(text, placeholder, original)
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Remove most punctuation for matching (but keep wildcards)
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.ReplaceAll(text, "_", " ")

	// Expand contractions for better pattern matching (before removing apostrophes)
	text = expandContractions(text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// normalizeForMatching normalizes text specifically for pattern matching
func normalizeForMatching(input string) string {
	// For pattern matching, we need more aggressive normalization
	// but still preserve set/topic tags and wildcards

	// First, preserve set and topic tags before case conversion
	tempSetTags := make(map[string]string)
	tempTopicTags := make(map[string]string)

	// Replace set tags temporarily
	setPattern := patternSetTagRe
	setMatches := setPattern.FindAllString(input, -1)
	for i, match := range setMatches {
		placeholder := fmt.Sprintf("__TEMP_SET_%d__", i)
		tempSetTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	// Replace topic tags temporarily
	topicPattern := patternTopicTagRe
	topicMatches := topicPattern.FindAllString(input, -1)
	for i, match := range topicMatches {
		placeholder := fmt.Sprintf("__TEMP_TOPIC_%d__", i)
		tempTopicTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	text := strings.ToUpper(strings.TrimSpace(input))

	// Restore set and topic tags with preserved case
	for placeholder, original := range tempSetTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}
	for placeholder, original := range tempTopicTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Remove most punctuation for matching (but keep wildcards)
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.ReplaceAll(text, "_", " ")

	// Expand contractions for better pattern matching (before removing apostrophes)
	text = expandContractions(text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// normalizeForMatchingWithSubstitutions normalizes text with loaded substitutions applied first
func (g *Golem) normalizeForMatchingWithSubstitutions(input string) string {
	// For pattern matching, we need more aggressive normalization
	// but still preserve set/topic tags and wildcards

	g.LogDebug("normalizeForMatchingWithSubstitutions: input = '%s'", input)

	// First, preserve set and topic tags before case conversion
	tempSetTags := make(map[string]string)
	tempTopicTags := make(map[string]string)

	// Replace set tags temporarily
	setPattern := patternSetTagRe
	setMatches := setPattern.FindAllString(input, -1)
	for i, match := range setMatches {
		placeholder := fmt.Sprintf("__TEMP_SET_%d__", i)
		tempSetTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	// Replace topic tags temporarily
	topicPattern := patternTopicTagRe
	topicMatches := topicPattern.FindAllString(input, -1)
	for i, match := range topicMatches {
		placeholder := fmt.Sprintf("__TEMP_TOPIC_%d__", i)
		tempTopicTags[placeholder] = match
		input = strings.Replace(input, match, placeholder, 1)
	}

	text := strings.ToUpper(strings.TrimSpace(input))
	g.LogDebug("normalizeForMatchingWithSubstitutions: after uppercase = '%s'", text)

	// Restore set and topic tags with preserved case
	for placeholder, original := range tempSetTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}
	for placeholder, original := range tempTopicTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}

	g.LogDebug("normalizeForMatchingWithSubstitutions: after tag restoration = '%s'", text)

	// Apply loaded substitutions BEFORE other normalization
	text = g.applyLoadedSubstitutions(text)
	g.LogDebug("normalizeForMatchingWithSubstitutions: after substitutions = '%s'", text)

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Remove most punctuation for matching (but keep wildcards)
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.ReplaceAll(text, "_", " ")

	g.LogDebug("normalizeForMatchingWithSubstitutions: after punctuation removal = '%s'", text)

	// Expand contractions for better pattern matching (before removing apostrophes)
	text = expandContractions(text)
	g.LogDebug("normalizeForMatchingWithSubstitutions: after contractions = '%s'", text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	g.LogDebug("normalizeForMatchingWithSubstitutions: final result = '%s'", text)

	return text
}

// applyLoadedSubstitutions applies loaded substitution rules to text
// Substitutions are applied iteratively until no more changes occur, with longer patterns taking precedence
func (g *Golem) applyLoadedSubstitutions(text string) string {
	if g.aimlKB == nil || len(g.aimlKB.Substitutions) == 0 {
		return text
	}

	originalText := text

	// Collect all substitution patterns and sort by length (longest first)
	type substitution struct {
		pattern     string
		replacement string
	}
	var allSubstitutions []substitution

	for _, substitutionMap := range g.aimlKB.Substitutions {
		for pattern, replacement := range substitutionMap {
			allSubstitutions = append(allSubstitutions, substitution{
				pattern:     strings.ToUpper(pattern),
				replacement: strings.ToUpper(replacement),
			})
		}
	}

	// Sort by pattern length (longest first)
	for i := 0; i < len(allSubstitutions)-1; i++ {
		for j := i + 1; j < len(allSubstitutions); j++ {
			if len(allSubstitutions[i].pattern) < len(allSubstitutions[j].pattern) {
				allSubstitutions[i], allSubstitutions[j] = allSubstitutions[j], allSubstitutions[i]
			}
		}
	}

	// Apply substitutions iteratively until no more changes occur
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		prevText := text
		result := strings.Builder{}
		result.Grow(len(text))

		// Apply substitutions in a single pass
		i := 0
		for i < len(text) {
			matched := false

			// Try to match the longest pattern first
			for _, sub := range allSubstitutions {
				if i+len(sub.pattern) <= len(text) && text[i:i+len(sub.pattern)] == sub.pattern {
					// Found a match, apply substitution
					result.WriteString(sub.replacement)
					i += len(sub.pattern)
					matched = true
					g.LogDebug("Applied substitution: '%s' -> '%s'", sub.pattern, sub.replacement)
					break
				}
			}

			if !matched {
				// No match, copy the character as-is
				result.WriteByte(text[i])
				i++
			}
		}

		text = result.String()

		// If no changes were made, we're done
		if text == prevText {
			break
		}
	}

	if originalText != text {
		g.LogDebug("Substitutions applied: '%s' -> '%s'", originalText, text)
	}

	return text
}

// NormalizePattern normalizes AIML patterns for matching
func NormalizePattern(pattern string) string {
	// Patterns need special handling for set and topic tags
	// First, preserve set and topic tags before case conversion
	tempSetTags := make(map[string]string)
	tempTopicTags := make(map[string]string)

	// Replace set tags temporarily
	setPattern := patternSetTagRe
	setMatches := setPattern.FindAllString(pattern, -1)
	for i, match := range setMatches {
		placeholder := fmt.Sprintf("__TEMP_SET_%d__", i)
		tempSetTags[placeholder] = match
		pattern = strings.Replace(pattern, match, placeholder, 1)
	}

	// Replace topic tags temporarily
	topicPattern := patternTopicTagRe
	topicMatches := topicPattern.FindAllString(pattern, -1)
	for i, match := range topicMatches {
		placeholder := fmt.Sprintf("__TEMP_TOPIC_%d__", i)
		tempTopicTags[placeholder] = match
		pattern = strings.Replace(pattern, match, placeholder, 1)
	}

	text := strings.ToUpper(strings.TrimSpace(pattern))

	// Restore set and topic tags with preserved case
	for placeholder, original := range tempSetTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}
	for placeholder, original := range tempTopicTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Remove punctuation that might interfere with matching
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, "-", " ")

	// Expand contractions for better pattern matching (before removing apostrophes)
	text = expandContractions(text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// SentenceSplitter handles sentence splitting with proper boundary detection
type SentenceSplitter struct {
	// Common sentence ending patterns
	sentenceEndings []string
	// Abbreviations that shouldn't end sentences
	abbreviations map[string]bool
	// Honorifics that might appear before names
	honorifics map[string]bool
}

// NewSentenceSplitter creates a new sentence splitter with default rules
func NewSentenceSplitter() *SentenceSplitter {
	return &SentenceSplitter{
		sentenceEndings: []string{".", "!", "?", "。", "！", "？"},
		abbreviations: map[string]bool{
			"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true,
			"rev": true, "gen": true, "col": true, "sgt": true, "lt": true,
			"capt": true, "cmdr": true, "adm": true, "gov": true, "sen": true,
			"rep": true, "st": true, "ave": true, "blvd": true, "rd": true,
			"inc": true, "ltd": true, "corp": true, "co": true, "etc": true,
			"vs": true, "v": true, "am": true, "pm": true,
		},
		honorifics: map[string]bool{
			"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true,
			"rev": true, "gen": true, "col": true, "sgt": true, "lt": true,
			"capt": true, "cmdr": true, "adm": true, "gov": true, "sen": true,
			"rep": true, "st": true,
		},
	}
}

// SplitSentences splits text into sentences using intelligent boundary detection
func (ss *SentenceSplitter) SplitSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Normalize whitespace first
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	var sentences []string
	var current strings.Builder

	runes := []rune(text)

	for i, r := range runes {
		current.WriteRune(r)

		// Check if this could be a sentence boundary
		if ss.isSentenceBoundary(runes, i) {
			sentence := strings.TrimSpace(current.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}

	// Add any remaining text as a sentence
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

// isSentenceBoundary determines if a position is a sentence boundary
func (ss *SentenceSplitter) isSentenceBoundary(runes []rune, pos int) bool {
	if pos >= len(runes) {
		return false
	}

	current := runes[pos]

	// Check if current character is a sentence ending
	isEnding := false
	for _, ending := range ss.sentenceEndings {
		if string(current) == ending {
			isEnding = true
			break
		}
	}

	if !isEnding {
		return false
	}

	// Look ahead to see if there's whitespace and a capital letter
	if pos+1 >= len(runes) {
		return true // End of text
	}

	// Skip whitespace
	nextPos := pos + 1
	for nextPos < len(runes) && unicode.IsSpace(runes[nextPos]) {
		nextPos++
	}

	if nextPos >= len(runes) {
		return true // End of text after whitespace
	}

	// Check if next character is uppercase (start of new sentence)
	nextChar := runes[nextPos]
	if unicode.IsUpper(nextChar) {
		// Additional check: make sure it's not an abbreviation
		return !ss.isAbbreviation(runes, pos)
	}

	return false
}

// isAbbreviation checks if the period is part of an abbreviation
func (ss *SentenceSplitter) isAbbreviation(runes []rune, pos int) bool {
	// Look backwards to find the start of the current word
	start := pos
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	// Extract the word before the period
	word := strings.ToLower(string(runes[start:pos]))

	// Check if it's a known abbreviation
	return ss.abbreviations[word]
}

// WordBoundaryDetector handles word boundary detection and tokenization
type WordBoundaryDetector struct {
	// Characters that are considered word separators
	separators map[rune]bool
	// Characters that are considered punctuation
	punctuation map[rune]bool
}

// NewWordBoundaryDetector creates a new word boundary detector
func NewWordBoundaryDetector() *WordBoundaryDetector {
	separators := make(map[rune]bool)
	punctuation := make(map[rune]bool)

	// Common word separators
	for _, r := range " \t\n\r\f\v" {
		separators[r] = true
	}

	// Common punctuation
	for _, r := range ".,!?;:\"'()[]{}<>/@#$%^&*+=|\\~`" {
		punctuation[r] = true
	}

	return &WordBoundaryDetector{
		separators:  separators,
		punctuation: punctuation,
	}
}

// SplitWords splits text into words using proper boundary detection
func (wbd *WordBoundaryDetector) SplitWords(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	var words []string
	var current strings.Builder

	for _, r := range text {
		if wbd.separators[r] {
			// End of word
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if wbd.punctuation[r] {
			// Punctuation - end current word and add punctuation as separate token
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			words = append(words, string(r))
		} else {
			// Regular character
			current.WriteRune(r)
		}
	}

	// Add any remaining word
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// capitalizeSentences capitalizes the first letter of each sentence
func (g *Golem) capitalizeSentences(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Use regex to find sentence boundaries and capitalize
	// Pattern: sentence ending followed by whitespace and any character
	sentenceRegex := regexp.MustCompile(`([.!?])\s+([a-z])`)

	// Replace lowercase letters after sentence endings with uppercase
	result := sentenceRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the parts
		parts := sentenceRegex.FindStringSubmatch(match)
		if len(parts) >= 3 {
			punctuation := parts[1]
			letter := parts[2]
			return punctuation + " " + strings.ToUpper(letter)
		}
		return match
	})

	// Also capitalize the very first letter if it's lowercase
	if len(result) > 0 && unicode.IsLower(rune(result[0])) {
		result = strings.ToUpper(string(result[0])) + result[1:]
	}

	return result
}

// capitalizeWords capitalizes the first letter of each word (title case)
func (g *Golem) capitalizeWords(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Use a simpler approach: split by spaces and capitalize each word
	words := strings.Fields(text)

	// Capitalize each word
	var capitalizedWords []string
	for _, word := range words {
		if word != "" {
			// Handle hyphenated words by capitalizing each part
			capitalized := g.capitalizeHyphenatedWord(word)
			capitalizedWords = append(capitalizedWords, capitalized)
		}
	}

	// Join words with single spaces
	return strings.Join(capitalizedWords, " ")
}

// capitalizeHyphenatedWord capitalizes a word, handling hyphens properly
func (g *Golem) capitalizeHyphenatedWord(word string) string {
	if word == "" {
		return word
	}

	// Split by hyphens and capitalize each part
	parts := strings.Split(word, "-")
	var capitalizedParts []string

	for _, part := range parts {
		if part != "" {
			capitalized := g.capitalizeFirstLetter(part)
			capitalizedParts = append(capitalizedParts, capitalized)
		}
	}

	// Join with hyphens
	return strings.Join(capitalizedParts, "-")
}

// capitalizeFirstLetter capitalizes the first letter of a word while preserving the rest
func (g *Golem) capitalizeFirstLetter(word string) string {
	if word == "" {
		return word
	}

	runes := []rune(word)
	if len(runes) == 0 {
		return word
	}

	// Capitalize first rune
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

// GetWordBoundaries returns the positions of word boundaries in text
func (wbd *WordBoundaryDetector) GetWordBoundaries(text string) []int {
	var boundaries []int

	runes := []rune(text)

	for i, r := range runes {
		if wbd.separators[r] || wbd.punctuation[r] {
			boundaries = append(boundaries, i)
		}
	}

	return boundaries
}

// IsWordBoundary checks if a position is a word boundary
func (wbd *WordBoundaryDetector) IsWordBoundary(text string, pos int) bool {
	if pos < 0 || pos >= len([]rune(text)) {
		return false
	}

	runes := []rune(text)
	r := runes[pos]

	return wbd.separators[r] || wbd.punctuation[r]
}

// NormalizeThatPattern normalizes a that pattern for matching with enhanced sentence boundary handling
func NormalizeThatPattern(pattern string) string {
	// Patterns need special handling for set and topic tags
	// First, preserve set and topic tags before case conversion
	tempSetTags := make(map[string]string)
	tempTopicTags := make(map[string]string)

	// Replace set tags temporarily
	setPattern := patternSetTagRe
	setMatches := setPattern.FindAllString(pattern, -1)
	for i, match := range setMatches {
		placeholder := fmt.Sprintf("__TEMP_SET_%d__", i)
		tempSetTags[placeholder] = match
		pattern = strings.Replace(pattern, match, placeholder, 1)
	}

	// Replace topic tags temporarily
	topicPattern := patternTopicTagRe
	topicMatches := topicPattern.FindAllString(pattern, -1)
	for i, match := range topicMatches {
		placeholder := fmt.Sprintf("__TEMP_TOPIC_%d__", i)
		tempTopicTags[placeholder] = match
		pattern = strings.Replace(pattern, match, placeholder, 1)
	}

	text := strings.ToUpper(strings.TrimSpace(pattern))

	// Restore set and topic tags with preserved case
	for placeholder, original := range tempSetTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}
	for placeholder, original := range tempTopicTags {
		text = strings.ReplaceAll(text, strings.ToUpper(placeholder), original)
	}

	// Normalize whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")

	// Handle sentence boundaries - remove trailing punctuation for better matching
	text = regexp.MustCompile(`[.!?]+$`).ReplaceAllString(text, "")

	// Remove punctuation that might interfere with matching
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "!", "")
	text = strings.ReplaceAll(text, "?", "")
	text = strings.ReplaceAll(text, ";", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, "-", " ")

	// Expand contractions for better pattern matching (before removing apostrophes)
	text = expandContractions(text)

	// Remove apostrophes after contraction expansion
	text = strings.ReplaceAll(text, "'", "")

	// Clean up whitespace
	text = whitespaceNormRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// validateThatPattern validates a that pattern for proper syntax with enhanced AIML2 wildcard support
func validateThatPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("that pattern cannot be empty")
	}

	// Check for balanced wildcards (all types)
	starCount := strings.Count(pattern, "*")
	underscoreCount := strings.Count(pattern, "_")
	caretCount := strings.Count(pattern, "^")
	hashCount := strings.Count(pattern, "#")
	dollarCount := strings.Count(pattern, "$")
	totalWildcards := starCount + underscoreCount + caretCount + hashCount + dollarCount

	if totalWildcards > 9 {
		return fmt.Errorf("that pattern contains too many wildcards (max 9), got %d", totalWildcards)
	}

	// Check for balanced set tags
	setOpenCount := strings.Count(pattern, "<set>")
	setCloseCount := strings.Count(pattern, "</set>")
	if setOpenCount != setCloseCount {
		return fmt.Errorf("unbalanced set tags in that pattern")
	}

	// Check for balanced topic tags
	topicOpenCount := strings.Count(pattern, "<topic>")
	topicCloseCount := strings.Count(pattern, "</topic>")
	if topicOpenCount != topicCloseCount {
		return fmt.Errorf("unbalanced topic tags in that pattern")
	}

	// Check for valid wildcard combinations
	if err := validateThatWildcardCombinations(pattern); err != nil {
		return err
	}

	return nil
}

// validateThatWildcardCombinations validates that wildcard combinations are valid
func validateThatWildcardCombinations(pattern string) error {
	// Check for invalid wildcard sequences
	invalidSequences := []string{
		"**", // Double star
		"__", // Double underscore
		"^^", // Double caret
		"##", // Double hash
		"$$", // Double dollar
		"*_", // Star followed by underscore
		"_*", // Underscore followed by star
		"*^", // Star followed by caret
		"^*", // Caret followed by star
		"*#", // Star followed by hash
		"#*", // Hash followed by star
		"*$", // Star followed by dollar
		"$*", // Dollar followed by star
		"_^", // Underscore followed by caret
		"^_", // Caret followed by underscore
		"_#", // Underscore followed by hash
		"#_", // Hash followed by underscore
		"_$", // Underscore followed by dollar
		"$_", // Dollar followed by underscore
		"^#", // Caret followed by hash
		"#^", // Hash followed by caret
		"^$", // Caret followed by dollar
		"$^", // Dollar followed by caret
		"#$", // Hash followed by dollar
		"$#", // Dollar followed by hash
	}

	for _, sequence := range invalidSequences {
		if strings.Contains(pattern, sequence) {
			return fmt.Errorf("invalid wildcard sequence '%s' in that pattern", sequence)
		}
	}

	// Note: Patterns can start and end with wildcards in AIML2, so we don't check for that

	return nil
}

// parseLearnContent parses AIML content within learn/learnf tags
func (g *Golem) parseLearnContent(content string) ([]Category, error) {
	// Wrap content in a minimal AIML structure for parsing
	wrappedContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
%s
</aiml>`, content)

	// Parse the wrapped content
	aiml, err := g.parseAIML(wrappedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse learn content: %v", err)
	}

	return aiml.Categories, nil
}

// ValidateLearnedCategory performs comprehensive validation on learned categories
func (g *Golem) ValidateLearnedCategory(category Category) error {
	// Basic validation
	if category.Pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if category.Template == "" {
		return fmt.Errorf("template cannot be empty")
	}

	// Security validation first (most critical)
	if err := g.validateSecurity(category); err != nil {
		return fmt.Errorf("security validation failed: %v", err)
	}

	// Content validation
	if err := g.validateContent(category); err != nil {
		return fmt.Errorf("content validation failed: %v", err)
	}

	// Pattern validation
	if err := g.validatePatternStructure(category.Pattern); err != nil {
		return fmt.Errorf("pattern validation failed: %v", err)
	}

	// Template validation
	if err := g.validateTemplate(category.Template); err != nil {
		return fmt.Errorf("template validation failed: %v", err)
	}

	return nil
}

// validatePatternStructure validates AIML patterns
func (g *Golem) validatePatternStructure(pattern string) error {
	// Check for reasonable length
	if len(pattern) > 1000 {
		return fmt.Errorf("pattern too long (max 1000 characters)")
	}

	// Check for valid wildcard usage
	starCount := strings.Count(pattern, "*")
	underscoreCount := strings.Count(pattern, "_")
	hashCount := strings.Count(pattern, "#")
	dollarCount := strings.Count(pattern, "$")
	totalWildcards := starCount + underscoreCount + hashCount + dollarCount

	// Limit total wildcards
	if totalWildcards > 10 {
		return fmt.Errorf("too many wildcards in pattern (max 10)")
	}

	// Check for invalid wildcard combinations
	if strings.Contains(pattern, "**") {
		return fmt.Errorf("consecutive wildcards not allowed")
	}

	return nil
}

// validateTemplate validates AIML templates
func (g *Golem) validateTemplate(template string) error {
	// Check for reasonable length
	if len(template) > 10000 {
		return fmt.Errorf("template too long (max 10000 characters)")
	}

	// Check for balanced tags
	if err := g.validateBalancedTags(template); err != nil {
		return fmt.Errorf("unbalanced tags in template: %v", err)
	}

	// Check for valid AIML tags
	if err := g.validateAIMLTags(template); err != nil {
		return fmt.Errorf("invalid AIML tags in template: %v", err)
	}

	// Check for reasonable nesting depth
	if err := g.validateNestingDepth(template); err != nil {
		return fmt.Errorf("excessive nesting depth in template: %v", err)
	}

	return nil
}

// validateSecurity performs security validation on learned content
func (g *Golem) validateSecurity(category Category) error {
	// Check for potential injection patterns
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"data:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onclick=",
		"eval(",
		"exec(",
		"system(",
		"shell_exec(",
	}

	content := strings.ToLower(category.Pattern + " " + category.Template)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(content, pattern) {
			return fmt.Errorf("potentially dangerous content detected: %s", pattern)
		}
	}

	// Check for excessive recursion potential
	if strings.Count(category.Template, "<srai>") > 5 {
		return fmt.Errorf("too many SRAI tags (max 5) - potential recursion")
	}

	// Check for excessive wildcard usage in template
	wildcardCount := strings.Count(category.Template, "<star")
	if wildcardCount > 10 {
		return fmt.Errorf("too many wildcard references in template (max 10)")
	}

	return nil
}

// validateContent performs content validation
func (g *Golem) validateContent(category Category) error {
	// Check for empty or whitespace-only content first
	if strings.TrimSpace(category.Pattern) == "" {
		return fmt.Errorf("pattern cannot be empty or whitespace only")
	}
	if strings.TrimSpace(category.Template) == "" {
		return fmt.Errorf("template cannot be empty or whitespace only")
	}

	// Check for minimum content length (but allow wildcards)
	trimmedPattern := strings.TrimSpace(category.Pattern)
	// Allow single wildcard patterns like "*" or "_"
	if len(trimmedPattern) < 2 && trimmedPattern != "*" && trimmedPattern != "_" {
		return fmt.Errorf("pattern too short (min 2 characters)")
	}
	if len(strings.TrimSpace(category.Template)) < 2 {
		return fmt.Errorf("template too short (min 2 characters)")
	}

	// Check for reasonable word count
	patternWords := len(strings.Fields(category.Pattern))
	if patternWords > 50 {
		return fmt.Errorf("pattern too complex (max 50 words)")
	}

	return nil
}

// validateBalancedTags validates that XML/AIML tags are balanced
func (g *Golem) validateBalancedTags(template string) error {
	// Find all opening and closing tags
	openTagRegex := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)
	closeTagRegex := regexp.MustCompile(`</([a-zA-Z][a-zA-Z0-9]*)>`)

	openTags := openTagRegex.FindAllStringSubmatch(template, -1)
	closeTags := closeTagRegex.FindAllStringSubmatch(template, -1)

	// Check for self-closing tags (like <star/>)
	selfClosingRegex := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^>]*/>`)
	selfClosingTags := selfClosingRegex.FindAllString(template, -1)

	// Count actual opening tags (excluding self-closing)
	actualOpenTags := 0
	for _, match := range openTags {
		tagName := match[1]
		// Check if this is a self-closing tag
		isSelfClosing := false
		for _, selfClosing := range selfClosingTags {
			if strings.Contains(selfClosing, tagName) {
				isSelfClosing = true
				break
			}
		}
		if !isSelfClosing {
			actualOpenTags++
		}
	}

	if actualOpenTags != len(closeTags) {
		return fmt.Errorf("unbalanced tags: %d opening tags, %d closing tags", actualOpenTags, len(closeTags))
	}

	return nil
}

// validateAIMLTags validates that only known AIML tags are used
func (g *Golem) validateAIMLTags(template string) error {
	// List of known AIML tags
	knownTags := map[string]bool{
		"aiml": true, "category": true, "pattern": true, "template": true,
		"star": true, "that": true, "sr": true, "srai": true, "sraix": true,
		"think": true, "learn": true, "learnf": true, "condition": true,
		"random": true, "li": true, "date": true, "time": true,
		"map": true, "list": true, "array": true, "set": true, "get": true,
		"bot": true, "request": true, "response": true, "person": true,
		"gender": true, "person2": true, "uppercase": true, "lowercase": true,
		"formal": true, "sentence": true, "word": true, "explode": true,
		"capitalize": true, "reverse": true, "acronym": true, "trim": true,
		"substring": true, "replace": true, "pluralize": true, "shuffle": true,
		"length": true, "count": true, "split": true, "join": true, "indent": true, "dedent": true, "unique": true, "repeat": true, "normalize": true, "denormalize": true,
		"id": true, "size": true, "version": true, "system": true, "javascript": true,
		"eval": true, "gossip": true, "loop": true, "var": true, "unlearn": true, "unlearnf": true, "topic": true,
		"uniq": true, "subj": true, "pred": true, "obj": true, // RDF operations
		"first": true, "rest": true, // List operations
		"botid": true, "host": true, "default": true, "hint": true, // SRAIX attributes
		"format": true, "jformat": true, // Date format attributes
	}

	// Find all tags
	tagRegex := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)
	matches := tagRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		tagName := strings.ToLower(match[1])
		if !knownTags[tagName] {
			return fmt.Errorf("unknown AIML tag: %s", match[1])
		}
	}

	return nil
}

// validateNestingDepth validates that nesting depth is reasonable
func (g *Golem) validateNestingDepth(template string) error {
	maxDepth := 20

	// Track nesting depth by parsing tags in order
	openTagRegex := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)
	closeTagRegex := regexp.MustCompile(`</([a-zA-Z][a-zA-Z0-9]*)>`)
	selfClosingRegex := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^>]*/>`)

	// Find all tags in order
	allTags := openTagRegex.FindAllString(template, -1)
	allCloseTags := closeTagRegex.FindAllString(template, -1)
	allSelfClosingTags := selfClosingRegex.FindAllString(template, -1)

	// Create a map of self-closing tags for quick lookup
	selfClosingMap := make(map[string]bool)
	for _, tag := range allSelfClosingTags {
		selfClosingMap[tag] = true
	}

	// Track actual nesting depth by counting opening and closing tags
	currentDepth := 0
	maxReachedDepth := 0

	// Count opening tags (excluding self-closing)
	for _, tag := range allTags {
		if !selfClosingMap[tag] {
			currentDepth++
			if currentDepth > maxReachedDepth {
				maxReachedDepth = currentDepth
			}
		}
	}

	// Subtract closing tags
	for range allCloseTags {
		currentDepth--
		if currentDepth < 0 {
			currentDepth = 0 // Shouldn't happen with balanced tags
		}
	}

	// Use the maximum depth reached during parsing
	if maxReachedDepth > maxDepth {
		return fmt.Errorf("excessive nesting depth (estimated %d, max %d)", maxReachedDepth, maxDepth)
	}

	return nil
}

// addSessionCategory adds a category to the session-specific knowledge base
func (g *Golem) addSessionCategory(category Category, ctx *VariableContext) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	// Enhanced validation of the category
	if err := g.ValidateLearnedCategory(category); err != nil {
		// Track validation errors in session stats
		if ctx.Session != nil && ctx.Session.LearningStats != nil {
			ctx.Session.LearningStats.ValidationErrors++
		}
		return fmt.Errorf("category validation failed: %v", err)
	}

	// Build the index key including that and topic
	key := categoryKey(category)

	if g.aimlKB.addOrUpdateCategory(category) {
		g.LogInfo("Updated existing session category: %s", key)
	} else {
		g.LogInfo("Added new session category: %s", key)
	}

	// Update session learning statistics
	if ctx.Session != nil && ctx.Session.LearningStats != nil {
		g.updateSessionLearningStats(ctx.Session, category, "learn")
	}

	return nil
}

// updateSessionLearningStats updates learning statistics for a session
func (g *Golem) updateSessionLearningStats(session *ChatSession, category Category, operation string) {
	if session.LearningStats == nil {
		return
	}

	now := time.Now()

	switch operation {
	case "learn":
		session.LearningStats.TotalLearned++
		session.LearningStats.LastLearned = now
		session.LearningStats.LearningSources["learn"]++

		// Track pattern type
		patternType := g.categorizePattern(category.Pattern)
		session.LearningStats.PatternTypes[patternType]++

		// Track template length
		session.LearningStats.TemplateLengths = append(session.LearningStats.TemplateLengths, len(category.Template))

		// Add to session learned categories
		session.LearnedCategories = append(session.LearnedCategories, category)

		// Calculate learning rate (categories per minute)
		if session.CreatedAt != "" {
			if created, err := time.Parse(time.RFC3339, session.CreatedAt); err == nil {
				duration := now.Sub(created)
				if duration.Minutes() > 0 {
					session.LearningStats.LearningRate = float64(session.LearningStats.TotalLearned) / duration.Minutes()
				}
			}
		}

	case "unlearn":
		session.LearningStats.TotalUnlearned++
		session.LearningStats.LastUnlearned = now
		session.LearningStats.LearningSources["unlearn"]++

		// Remove from session learned categories
		normalizedPattern := NormalizePattern(category.Pattern)
		for i, learnedCategory := range session.LearnedCategories {
			if NormalizePattern(learnedCategory.Pattern) == normalizedPattern {
				session.LearnedCategories = append(session.LearnedCategories[:i], session.LearnedCategories[i+1:]...)
				break
			}
		}
	}
}

// categorizePattern categorizes a pattern by type
func (g *Golem) categorizePattern(pattern string) string {
	if strings.Contains(pattern, "*") {
		return "wildcard"
	}
	if strings.Contains(pattern, "_") {
		return "underscore"
	}
	if strings.Contains(pattern, "#") {
		return "hash"
	}
	if strings.Contains(pattern, "$") {
		return "dollar"
	}
	if strings.Contains(pattern, "(") && strings.Contains(pattern, ")") {
		return "alternation"
	}
	return "literal"
}

// addPersistentCategory adds a category to the persistent knowledge base
func (g *Golem) addPersistentCategory(category Category) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	// Enhanced validation of the category
	if err := g.ValidateLearnedCategory(category); err != nil {
		return fmt.Errorf("category validation failed: %v", err)
	}

	// Normalize the pattern and build the proper key including that and topic
	normalizedPattern := NormalizePattern(category.Pattern)
	key := categoryKey(category)

	if g.aimlKB.addOrUpdateCategory(category) {
		g.LogInfo("Updated existing persistent category: %s", key)
	} else {
		g.LogInfo("Added new persistent category: %s", key)
	}

	// Save to persistent storage if available
	if g.persistentLearning != nil {
		if err := g.persistentLearning.AppendPersistentCategory(category, "learnf"); err != nil {
			g.LogWarn("Failed to save category to persistent storage: %v", err)
			// Don't fail the operation, just log the warning
		} else {
			g.LogInfo("Category saved to persistent storage: %s", normalizedPattern)
		}
	} else {
		g.LogWarn("Persistent learning not available - category added to memory only")
	}

	return nil
}

// removeSessionCategory removes a category from the session-specific knowledge base
func (g *Golem) removeSessionCategory(category Category, ctx *VariableContext) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	key := categoryKey(category)
	if !g.aimlKB.removeCategoryByKey(key) {
		g.LogInfo("Category not found for removal: %s", key)
		return fmt.Errorf("category not found: %s", key)
	}
	g.LogInfo("Removed session category: %s", key)

	// Update session learning statistics
	if ctx.Session != nil && ctx.Session.LearningStats != nil {
		g.updateSessionLearningStats(ctx.Session, category, "unlearn")
	}

	return nil
}

// removePersistentCategory removes a category from the persistent knowledge base
func (g *Golem) removePersistentCategory(category Category) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	key := categoryKey(category)
	if !g.aimlKB.removeCategoryByKey(key) {
		g.LogInfo("Category not found for removal: %s", key)
		return fmt.Errorf("category not found: %s", key)
	}
	g.LogInfo("Removed persistent category: %s", key)

	// Remove from persistent storage if available
	if g.persistentLearning != nil {
		if err := g.persistentLearning.RemovePersistentCategory(category); err != nil {
			g.LogWarn("Failed to remove category from persistent storage: %v", err)
			// Don't fail the operation, just log the warning
		} else {
			g.LogInfo("Category removed from persistent storage: %s", key)
		}
	}

	return nil
}

// ThatPatternValidationResult represents detailed validation results
type ThatPatternValidationResult struct {
	IsValid     bool                   `json:"is_valid"`
	Errors      []string               `json:"errors"`
	Warnings    []string               `json:"warnings"`
	Suggestions []string               `json:"suggestions"`
	Stats       map[string]interface{} `json:"stats"`
}

// ValidateThatPatternDetailed provides comprehensive validation with detailed error messages
func ValidateThatPatternDetailed(pattern string) *ThatPatternValidationResult {
	result := &ThatPatternValidationResult{
		IsValid:     true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
		Stats:       make(map[string]interface{}),
	}

	// Basic checks
	if pattern == "" {
		result.Errors = append(result.Errors, "That pattern cannot be empty")
		result.IsValid = false
		return result
	}

	// Calculate statistics
	result.Stats["length"] = len(pattern)
	result.Stats["word_count"] = len(strings.Fields(pattern))
	result.Stats["wildcard_count"] = 0
	result.Stats["wildcard_types"] = make(map[string]int)

	// Check for balanced wildcards (all types) with detailed reporting
	wildcardCounts := CountWildcardsByType(pattern)
	totalWildcards := CountWildcards(pattern)

	result.Stats["wildcard_count"] = totalWildcards
	result.Stats["wildcard_types"] = wildcardCounts

	if totalWildcards > 9 {
		result.Errors = append(result.Errors, fmt.Sprintf("Too many wildcards: %d (maximum 9 allowed). Consider simplifying the pattern.", totalWildcards))
		result.IsValid = false
	}

	// Check for valid characters with specific error reporting
	invalidChars := findInvalidCharacters(pattern)
	if len(invalidChars) > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid characters found: %s. Only A-Z, 0-9, spaces, wildcards (*_^#$), and basic punctuation are allowed.", strings.Join(invalidChars, ", ")))
		result.IsValid = false
	}

	// Check for balanced tags with specific error reporting
	tagErrors := validateBalancedTags(pattern)
	result.Errors = append(result.Errors, tagErrors...)
	if len(tagErrors) > 0 {
		result.IsValid = false
	}

	// Check for common pattern issues
	patternIssues := validatePatternStructure(pattern)
	result.Warnings = append(result.Warnings, patternIssues...)

	// Check for performance issues
	performanceIssues := validatePatternPerformance(pattern)
	result.Warnings = append(result.Warnings, performanceIssues...)

	// Generate suggestions
	result.Suggestions = generatePatternSuggestions(pattern, result.Stats)

	return result
}

// findInvalidCharacters identifies invalid characters in the pattern
func findInvalidCharacters(pattern string) []string {
	validChars := regexp.MustCompile(`^[A-Z0-9\s\*_^#$<>/'.!?,\-()]+$`)
	invalidChars := []string{}

	for i, char := range pattern {
		if !validChars.MatchString(string(char)) {
			invalidChars = append(invalidChars, fmt.Sprintf("'%c' at position %d", char, i))
		}
	}

	return invalidChars
}

// validateBalancedTags checks for balanced XML-like tags
func validateBalancedTags(pattern string) []string {
	errors := []string{}

	// Check set tags
	setOpenCount := strings.Count(pattern, "<set>")
	setCloseCount := strings.Count(pattern, "</set>")
	if setOpenCount != setCloseCount {
		errors = append(errors, fmt.Sprintf("Unbalanced set tags: %d opening, %d closing", setOpenCount, setCloseCount))
	}

	// Check topic tags
	topicOpenCount := strings.Count(pattern, "<topic>")
	topicCloseCount := strings.Count(pattern, "</topic>")
	if topicOpenCount != topicCloseCount {
		errors = append(errors, fmt.Sprintf("Unbalanced topic tags: %d opening, %d closing", topicOpenCount, topicCloseCount))
	}

	// Check alternation groups
	parenOpenCount := strings.Count(pattern, "(")
	parenCloseCount := strings.Count(pattern, ")")
	if parenOpenCount != parenCloseCount {
		errors = append(errors, fmt.Sprintf("Unbalanced alternation groups: %d opening, %d closing", parenOpenCount, parenCloseCount))
	}

	return errors
}

// validatePatternStructure checks for common structural issues
func validatePatternStructure(pattern string) []string {
	warnings := []string{}

	// Check for consecutive wildcards
	if strings.Contains(pattern, "**") || strings.Contains(pattern, "__") ||
		strings.Contains(pattern, "^^") || strings.Contains(pattern, "##") {
		warnings = append(warnings, "Consecutive wildcards detected. This may cause matching issues.")
	}

	// Check for wildcards at pattern boundaries
	if strings.HasPrefix(pattern, "*") || strings.HasPrefix(pattern, "_") ||
		strings.HasPrefix(pattern, "^") || strings.HasPrefix(pattern, "#") {
		warnings = append(warnings, "Pattern starts with wildcard. Consider if this is intentional.")
	}

	if strings.HasSuffix(pattern, "*") || strings.HasSuffix(pattern, "_") ||
		strings.HasSuffix(pattern, "^") || strings.HasSuffix(pattern, "#") {
		warnings = append(warnings, "Pattern ends with wildcard. Consider if this is intentional.")
	}

	// Check for very short patterns
	if len(strings.TrimSpace(pattern)) < 3 {
		warnings = append(warnings, "Very short pattern. Consider if this provides enough specificity.")
	}

	// Check for very long patterns
	if len(pattern) > 200 {
		warnings = append(warnings, "Very long pattern. Consider breaking into smaller, more specific patterns.")
	}

	return warnings
}

// validatePatternPerformance checks for potential performance issues
func validatePatternPerformance(pattern string) []string {
	warnings := []string{}

	// Check for complex wildcard combinations
	wildcardCount := CountWildcards(pattern)

	if wildcardCount > 5 {
		warnings = append(warnings, "High wildcard count may impact matching performance.")
	}

	// Check for nested alternation groups
	parenDepth := 0
	maxDepth := 0
	for _, char := range pattern {
		if char == '(' {
			parenDepth++
			if parenDepth > maxDepth {
				maxDepth = parenDepth
			}
		} else if char == ')' {
			parenDepth--
		}
	}

	if maxDepth > 3 {
		warnings = append(warnings, "Deeply nested alternation groups may impact performance.")
	}

	// Check for repeated subpatterns
	words := strings.Fields(pattern)
	wordCounts := make(map[string]int)
	for _, word := range words {
		wordCounts[word]++
	}

	for word, count := range wordCounts {
		if count > 3 {
			warnings = append(warnings, fmt.Sprintf("Word '%s' appears %d times. Consider if this is intentional.", word, count))
		}
	}

	return warnings
}

// generatePatternSuggestions generates helpful suggestions for pattern improvement
func generatePatternSuggestions(pattern string, stats map[string]interface{}) []string {
	suggestions := []string{}

	// Suggest based on wildcard count
	if wildcardCount, ok := stats["wildcard_count"].(int); ok {
		if wildcardCount == 0 {
			suggestions = append(suggestions, "Consider adding wildcards (*, _, ^, #) for more flexible matching.")
		} else if wildcardCount > 5 {
			suggestions = append(suggestions, "Consider reducing wildcards for more specific matching.")
		}
	}

	// Suggest based on length
	if length, ok := stats["length"].(int); ok {
		if length < 10 {
			suggestions = append(suggestions, "Short patterns may match too broadly. Consider adding more context.")
		} else if length > 100 {
			suggestions = append(suggestions, "Long patterns may be too specific. Consider using wildcards for flexibility.")
		}
	}

	// Suggest based on word count
	if wordCount, ok := stats["word_count"].(int); ok {
		if wordCount == 1 {
			suggestions = append(suggestions, "Single-word patterns are very broad. Consider adding context words.")
		}
	}

	// General suggestions
	if !strings.Contains(pattern, " ") {
		suggestions = append(suggestions, "Consider adding spaces between words for better readability.")
	}

	if strings.Contains(pattern, "  ") {
		suggestions = append(suggestions, "Multiple consecutive spaces detected. Consider normalizing whitespace.")
	}

	return suggestions
}

// ThatContextDebugger provides comprehensive debugging tools for that context
type ThatContextDebugger struct {
	Session         *ChatSession
	EnableTracing   bool
	EnableProfiling bool
	TraceLog        []ThatTraceEntry
	PerformanceLog  []ThatPerformanceEntry
}

// ThatTraceEntry represents a single trace entry for debugging
type ThatTraceEntry struct {
	Timestamp int64                  `json:"timestamp"`
	Operation string                 `json:"operation"`
	Pattern   string                 `json:"pattern"`
	Input     string                 `json:"input"`
	Result    string                 `json:"result"`
	Matched   bool                   `json:"matched"`
	Duration  int64                  `json:"duration_ns"`
	Context   map[string]interface{} `json:"context"`
	Error     string                 `json:"error,omitempty"`
}

// ThatPerformanceEntry represents performance metrics for that operations
type ThatPerformanceEntry struct {
	Timestamp    int64  `json:"timestamp"`
	Operation    string `json:"operation"`
	Duration     int64  `json:"duration_ns"`
	MemoryUsage  int64  `json:"memory_bytes"`
	PatternCount int    `json:"pattern_count"`
	HistorySize  int    `json:"history_size"`
	CacheHits    int    `json:"cache_hits"`
	CacheMisses  int    `json:"cache_misses"`
}

// NewThatContextDebugger creates a new debugger instance
func NewThatContextDebugger(session *ChatSession) *ThatContextDebugger {
	return &ThatContextDebugger{
		Session:         session,
		EnableTracing:   false,
		EnableProfiling: false,
		TraceLog:        make([]ThatTraceEntry, 0),
		PerformanceLog:  make([]ThatPerformanceEntry, 0),
	}
}

// EnableDebugging enables all debugging features
func (debugger *ThatContextDebugger) EnableDebugging() {
	debugger.EnableTracing = true
	debugger.EnableProfiling = true
}

// DisableDebugging disables all debugging features
func (debugger *ThatContextDebugger) DisableDebugging() {
	debugger.EnableTracing = false
	debugger.EnableProfiling = false
}

// TraceThatMatching traces a that pattern matching operation
func (debugger *ThatContextDebugger) TraceThatMatching(pattern, input string, matched bool, result string, duration int64, err error) {
	if !debugger.EnableTracing {
		return
	}

	entry := ThatTraceEntry{
		Timestamp: time.Now().UnixNano(),
		Operation: "that_matching",
		Pattern:   pattern,
		Input:     input,
		Result:    result,
		Matched:   matched,
		Duration:  duration,
		Context: map[string]interface{}{
			"history_size": len(debugger.Session.ThatHistory),
			"topic":        debugger.Session.Topic,
		},
	}

	if err != nil {
		entry.Error = err.Error()
	}

	debugger.TraceLog = append(debugger.TraceLog, entry)

	// Keep only last 1000 entries to prevent memory issues
	if len(debugger.TraceLog) > 1000 {
		debugger.TraceLog = debugger.TraceLog[1:]
	}
}

// TraceThatHistoryOperation traces a that history operation
func (debugger *ThatContextDebugger) TraceThatHistoryOperation(operation, input string, duration int64, err error) {
	if !debugger.EnableTracing {
		return
	}

	entry := ThatTraceEntry{
		Timestamp: time.Now().UnixNano(),
		Operation: operation,
		Pattern:   "",
		Input:     input,
		Result:    "",
		Matched:   err == nil,
		Duration:  duration,
		Context: map[string]interface{}{
			"history_size": len(debugger.Session.ThatHistory),
			"topic":        debugger.Session.Topic,
		},
	}

	if err != nil {
		entry.Error = err.Error()
	}

	debugger.TraceLog = append(debugger.TraceLog, entry)

	// Keep only last 1000 entries
	if len(debugger.TraceLog) > 1000 {
		debugger.TraceLog = debugger.TraceLog[1:]
	}
}

// RecordPerformance records performance metrics
func (debugger *ThatContextDebugger) RecordPerformance(operation string, duration, memoryUsage int64, patternCount, historySize, cacheHits, cacheMisses int) {
	if !debugger.EnableProfiling {
		return
	}

	entry := ThatPerformanceEntry{
		Timestamp:    time.Now().UnixNano(),
		Operation:    operation,
		Duration:     duration,
		MemoryUsage:  memoryUsage,
		PatternCount: patternCount,
		HistorySize:  historySize,
		CacheHits:    cacheHits,
		CacheMisses:  cacheMisses,
	}

	debugger.PerformanceLog = append(debugger.PerformanceLog, entry)

	// Keep only last 500 entries
	if len(debugger.PerformanceLog) > 500 {
		debugger.PerformanceLog = debugger.PerformanceLog[1:]
	}
}

// GetTraceSummary returns a summary of trace operations
func (debugger *ThatContextDebugger) GetTraceSummary() map[string]interface{} {
	if len(debugger.TraceLog) == 0 {
		return map[string]interface{}{
			"total_operations": 0,
			"message":          "No trace data available",
		}
	}

	operations := make(map[string]int)
	errors := 0
	totalDuration := int64(0)
	matchedCount := 0

	for _, entry := range debugger.TraceLog {
		operations[entry.Operation]++
		if entry.Error != "" {
			errors++
		}
		totalDuration += entry.Duration
		if entry.Matched {
			matchedCount++
		}
	}

	avgDuration := float64(0)
	if len(debugger.TraceLog) > 0 {
		avgDuration = float64(totalDuration) / float64(len(debugger.TraceLog))
	}

	return map[string]interface{}{
		"total_operations":  len(debugger.TraceLog),
		"operations":        operations,
		"errors":            errors,
		"matched_count":     matchedCount,
		"match_rate":        float64(matchedCount) / float64(len(debugger.TraceLog)),
		"avg_duration_ns":   avgDuration,
		"total_duration_ns": totalDuration,
	}
}

// GetPerformanceSummary returns performance analysis
func (debugger *ThatContextDebugger) GetPerformanceSummary() map[string]interface{} {
	if len(debugger.PerformanceLog) == 0 {
		return map[string]interface{}{
			"total_operations": 0,
			"message":          "No performance data available",
		}
	}

	operations := make(map[string][]int64)
	totalDuration := int64(0)
	totalMemory := int64(0)

	for _, entry := range debugger.PerformanceLog {
		operations[entry.Operation] = append(operations[entry.Operation], entry.Duration)
		totalDuration += entry.Duration
		totalMemory += entry.MemoryUsage
	}

	// Calculate averages per operation
	operationStats := make(map[string]map[string]interface{})
	for op, durations := range operations {
		if len(durations) > 0 {
			sum := int64(0)
			min := durations[0]
			max := durations[0]
			for _, d := range durations {
				sum += d
				if d < min {
					min = d
				}
				if d > max {
					max = d
				}
			}

			operationStats[op] = map[string]interface{}{
				"count":  len(durations),
				"avg_ns": float64(sum) / float64(len(durations)),
				"min_ns": min,
				"max_ns": max,
			}
		}
	}

	avgDuration := float64(0)
	avgMemory := float64(0)
	if len(debugger.PerformanceLog) > 0 {
		avgDuration = float64(totalDuration) / float64(len(debugger.PerformanceLog))
		avgMemory = float64(totalMemory) / float64(len(debugger.PerformanceLog))
	}

	return map[string]interface{}{
		"total_operations":   len(debugger.PerformanceLog),
		"operation_stats":    operationStats,
		"avg_duration_ns":    avgDuration,
		"avg_memory_bytes":   avgMemory,
		"total_duration_ns":  totalDuration,
		"total_memory_bytes": totalMemory,
	}
}

// AnalyzeThatPatterns analyzes that pattern usage and effectiveness
func (debugger *ThatContextDebugger) AnalyzeThatPatterns() map[string]interface{} {
	analysis := map[string]interface{}{
		"history_analysis":     debugger.analyzeThatHistory(),
		"pattern_analysis":     debugger.analyzePatternUsage(),
		"performance_analysis": debugger.analyzePerformance(),
		"recommendations":      debugger.generateRecommendations(),
	}

	return analysis
}

// analyzeThatHistory analyzes that history patterns
func (debugger *ThatContextDebugger) analyzeThatHistory() map[string]interface{} {
	history := debugger.Session.ThatHistory
	if len(history) == 0 {
		return map[string]interface{}{
			"message": "No that history available",
		}
	}

	// Analyze history patterns
	lengths := make([]int, len(history))
	totalLength := 0
	uniqueResponses := make(map[string]int)

	for i, response := range history {
		lengths[i] = len(response)
		totalLength += len(response)
		uniqueResponses[response]++
	}

	// Calculate statistics
	avgLength := float64(totalLength) / float64(len(history))
	minLength := lengths[0]
	maxLength := lengths[0]
	for _, l := range lengths {
		if l < minLength {
			minLength = l
		}
		if l > maxLength {
			maxLength = l
		}
	}

	// Find most common responses
	mostCommon := ""
	maxCount := 0
	for response, count := range uniqueResponses {
		if count > maxCount {
			maxCount = count
			mostCommon = response
		}
	}

	return map[string]interface{}{
		"total_responses":      len(history),
		"unique_responses":     len(uniqueResponses),
		"avg_length":           avgLength,
		"min_length":           minLength,
		"max_length":           maxLength,
		"most_common_response": mostCommon,
		"most_common_count":    maxCount,
		"repetition_rate":      float64(maxCount) / float64(len(history)),
	}
}

// analyzePatternUsage analyzes pattern matching effectiveness
func (debugger *ThatContextDebugger) analyzePatternUsage() map[string]interface{} {
	if len(debugger.TraceLog) == 0 {
		return map[string]interface{}{
			"message": "No pattern matching data available",
		}
	}

	patternStats := make(map[string]map[string]int)
	totalMatches := 0
	totalAttempts := 0

	for _, entry := range debugger.TraceLog {
		if entry.Operation == "that_matching" {
			totalAttempts++
			if entry.Matched {
				totalMatches++
			}

			if patternStats[entry.Pattern] == nil {
				patternStats[entry.Pattern] = map[string]int{
					"attempts": 0,
					"matches":  0,
				}
			}

			patternStats[entry.Pattern]["attempts"]++
			if entry.Matched {
				patternStats[entry.Pattern]["matches"]++
			}
		}
	}

	// Calculate effectiveness
	effectiveness := float64(0)
	if totalAttempts > 0 {
		effectiveness = float64(totalMatches) / float64(totalAttempts)
	}

	// Find most/least effective patterns
	mostEffective := ""
	leastEffective := ""
	maxEffectiveness := float64(0)
	minEffectiveness := float64(1)

	for pattern, stats := range patternStats {
		if stats["attempts"] > 0 {
			patternEffectiveness := float64(stats["matches"]) / float64(stats["attempts"])
			if patternEffectiveness > maxEffectiveness {
				maxEffectiveness = patternEffectiveness
				mostEffective = pattern
			}
			if patternEffectiveness < minEffectiveness {
				minEffectiveness = patternEffectiveness
				leastEffective = pattern
			}
		}
	}

	return map[string]interface{}{
		"total_patterns":          len(patternStats),
		"total_attempts":          totalAttempts,
		"total_matches":           totalMatches,
		"overall_effectiveness":   effectiveness,
		"most_effective_pattern":  mostEffective,
		"least_effective_pattern": leastEffective,
		"pattern_stats":           patternStats,
	}
}

// analyzePerformance analyzes performance characteristics
func (debugger *ThatContextDebugger) analyzePerformance() map[string]interface{} {
	if len(debugger.PerformanceLog) == 0 {
		return map[string]interface{}{
			"message": "No performance data available",
		}
	}

	// Analyze performance trends
	durations := make([]int64, len(debugger.PerformanceLog))
	memoryUsages := make([]int64, len(debugger.PerformanceLog))

	for i, entry := range debugger.PerformanceLog {
		durations[i] = entry.Duration
		memoryUsages[i] = entry.MemoryUsage
	}

	// Calculate statistics
	totalDuration := int64(0)
	totalMemory := int64(0)
	minDuration := durations[0]
	maxDuration := durations[0]
	minMemory := memoryUsages[0]
	maxMemory := memoryUsages[0]

	for i, duration := range durations {
		totalDuration += duration
		totalMemory += memoryUsages[i]

		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
		if memoryUsages[i] < minMemory {
			minMemory = memoryUsages[i]
		}
		if memoryUsages[i] > maxMemory {
			maxMemory = memoryUsages[i]
		}
	}

	avgDuration := float64(totalDuration) / float64(len(durations))
	avgMemory := float64(totalMemory) / float64(len(memoryUsages))

	return map[string]interface{}{
		"avg_duration_ns":  avgDuration,
		"min_duration_ns":  minDuration,
		"max_duration_ns":  maxDuration,
		"avg_memory_bytes": avgMemory,
		"min_memory_bytes": minMemory,
		"max_memory_bytes": maxMemory,
		"total_operations": len(debugger.PerformanceLog),
	}
}

// generateRecommendations generates optimization recommendations
func (debugger *ThatContextDebugger) generateRecommendations() []string {
	recommendations := []string{}

	// Analyze history
	historyAnalysis := debugger.analyzeThatHistory()
	if historyAnalysis["repetition_rate"] != nil {
		repetitionRate := historyAnalysis["repetition_rate"].(float64)
		if repetitionRate > 0.5 {
			recommendations = append(recommendations, "High repetition rate detected. Consider adding more variety to responses.")
		}
	}

	// Analyze patterns
	patternAnalysis := debugger.analyzePatternUsage()
	if patternAnalysis["overall_effectiveness"] != nil {
		effectiveness := patternAnalysis["overall_effectiveness"].(float64)
		if effectiveness < 0.3 {
			recommendations = append(recommendations, "Low pattern matching effectiveness. Review pattern specificity and wildcard usage.")
		}
	}

	// Analyze performance
	performanceAnalysis := debugger.analyzePerformance()
	if performanceAnalysis["avg_duration_ns"] != nil {
		avgDuration := performanceAnalysis["avg_duration_ns"].(float64)
		if avgDuration > 1000000 { // 1ms
			recommendations = append(recommendations, "High average processing time detected. Consider optimizing patterns or reducing history size.")
		}
	}

	// Check memory usage
	if len(debugger.Session.ThatHistory) > 50 {
		recommendations = append(recommendations, "Large that history detected. Consider enabling compression or reducing history depth.")
	}

	// Check cache effectiveness
	if len(debugger.PerformanceLog) > 0 {
		totalHits := 0
		totalMisses := 0
		for _, entry := range debugger.PerformanceLog {
			totalHits += entry.CacheHits
			totalMisses += entry.CacheMisses
		}

		if totalHits+totalMisses > 0 {
			hitRate := float64(totalHits) / float64(totalHits+totalMisses)
			if hitRate < 0.5 {
				recommendations = append(recommendations, "Low cache hit rate. Consider increasing cache size or optimizing pattern caching.")
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No specific recommendations at this time. System appears to be performing well.")
	}

	return recommendations
}

// ClearDebugData clears all debug data
func (debugger *ThatContextDebugger) ClearDebugData() {
	debugger.TraceLog = make([]ThatTraceEntry, 0)
	debugger.PerformanceLog = make([]ThatPerformanceEntry, 0)
}

// ExportDebugData exports debug data for analysis
func (debugger *ThatContextDebugger) ExportDebugData() map[string]interface{} {
	return map[string]interface{}{
		"trace_log":       debugger.TraceLog,
		"performance_log": debugger.PerformanceLog,
		"summary": map[string]interface{}{
			"trace_summary":       debugger.GetTraceSummary(),
			"performance_summary": debugger.GetPerformanceSummary(),
			"analysis":            debugger.AnalyzeThatPatterns(),
		},
	}
}

// ThatPatternConflict represents a conflict between patterns
type ThatPatternConflict struct {
	Type        string   `json:"type"`        // Type of conflict (overlap, ambiguity, priority)
	Pattern1    string   `json:"pattern1"`    // First conflicting pattern
	Pattern2    string   `json:"pattern2"`    // Second conflicting pattern
	Severity    string   `json:"severity"`    // Severity level (low, medium, high, critical)
	Description string   `json:"description"` // Human-readable description
	Suggestions []string `json:"suggestions"` // Suggested resolutions
	Examples    []string `json:"examples"`    // Example inputs that trigger the conflict
}

// ThatPatternConflictDetector handles pattern conflict detection
type ThatPatternConflictDetector struct {
	Patterns  []string              `json:"patterns"`  // List of patterns to analyze
	Conflicts []ThatPatternConflict `json:"conflicts"` // Detected conflicts
}

// NewThatPatternConflictDetector creates a new conflict detector
func NewThatPatternConflictDetector(patterns []string) *ThatPatternConflictDetector {
	return &ThatPatternConflictDetector{
		Patterns:  patterns,
		Conflicts: []ThatPatternConflict{},
	}
}

// DetectConflicts analyzes patterns for conflicts
func (detector *ThatPatternConflictDetector) DetectConflicts(golem *Golem) []ThatPatternConflict {
	conflictDetection := NewConflictDetection(golem)
	return conflictDetection.DetectConflicts(detector)
}

// patternsOverlap checks if two patterns have overlapping matching scope
func (detector *ThatPatternConflictDetector) patternsOverlap(pattern1, pattern2 string) bool {
	// Convert patterns to testable format
	testCases := []string{
		"HELLO WORLD",
		"HELLO",
		"WORLD",
		"HELLO THERE",
		"GOOD MORNING",
		"GOOD AFTERNOON",
		"GOOD EVENING",
		"GOOD NIGHT",
		"WHAT IS YOUR NAME",
		"WHAT DO YOU DO",
		"TELL ME ABOUT YOURSELF",
		"WHO ARE YOU",
		"WHERE ARE YOU FROM",
		"WHAT CAN YOU DO",
		"HELP ME",
		"THANK YOU",
		"GOODBYE",
		"SEE YOU LATER",
		"HAVE A NICE DAY",
		"TAKE CARE",
	}

	matches1 := 0
	matches2 := 0
	overlap := 0

	for _, testCase := range testCases {
		matched1 := detector.testPatternMatch(pattern1, testCase)
		matched2 := detector.testPatternMatch(pattern2, testCase)

		if matched1 {
			matches1++
		}
		if matched2 {
			matches2++
		}
		if matched1 && matched2 {
			overlap++
		}
	}

	// Calculate overlap percentage
	if matches1 > 0 && matches2 > 0 {
		overlapPercentage := float64(overlap) / float64(matches1+matches2-overlap)
		return overlapPercentage > 0.3 // 30% overlap threshold
	}

	return false
}

// patternsAreAmbiguous checks if two patterns create ambiguous matching
func (detector *ThatPatternConflictDetector) patternsAreAmbiguous(pattern1, pattern2 string) bool {
	// Check for patterns that could match the same input
	ambiguousCases := []string{
		"HELLO",
		"HELLO WORLD",
		"GOOD MORNING",
		"WHAT IS YOUR NAME",
		"WHO ARE YOU",
		"TELL ME ABOUT YOURSELF",
	}

	for _, testCase := range ambiguousCases {
		matched1 := detector.testPatternMatch(pattern1, testCase)
		matched2 := detector.testPatternMatch(pattern2, testCase)

		if matched1 && matched2 {
			// Check if both patterns have similar specificity
			specificity1 := detector.calculatePatternSpecificity(pattern1)
			specificity2 := detector.calculatePatternSpecificity(pattern2)

			// If specificity is similar, it's ambiguous
			if absFloat(specificity1-specificity2) < 0.2 {
				return true
			}
		}
	}

	return false
}

// Helper functions for conflict detection
func (detector *ThatPatternConflictDetector) testPatternMatch(pattern, input string) bool {
	// Simplified pattern matching for conflict detection
	// This is a basic implementation - in practice, you'd use the full pattern matching logic

	// Convert to uppercase for matching
	pattern = strings.ToUpper(pattern)
	input = strings.ToUpper(input)

	// Handle exact matches
	if pattern == input {
		return true
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Convert * to regex .*
		regexPattern := strings.ReplaceAll(pattern, "*", ".*")
		matched, _ := regexp.MatchString("^"+regexPattern+"$", input)
		return matched
	}

	// Handle underscore patterns (single word)
	if strings.Contains(pattern, "_") {
		// Convert _ to regex \w+
		regexPattern := strings.ReplaceAll(pattern, "_", "\\w+")
		matched, _ := regexp.MatchString("^"+regexPattern+"$", input)
		return matched
	}

	// Handle caret patterns (zero or more words)
	if strings.Contains(pattern, "^") {
		// Convert ^ to regex .*
		regexPattern := strings.ReplaceAll(pattern, "^", ".*")
		matched, _ := regexp.MatchString("^"+regexPattern+"$", input)
		return matched
	}

	// Handle hash patterns (zero or more words)
	if strings.Contains(pattern, "#") {
		// Convert # to regex .*
		regexPattern := strings.ReplaceAll(pattern, "#", ".*")
		matched, _ := regexp.MatchString("^"+regexPattern+"$", input)
		return matched
	}

	return false
}

func (detector *ThatPatternConflictDetector) calculatePatternSpecificity(pattern string) float64 {
	return CalculatePatternSpecificity(pattern)
}

func (detector *ThatPatternConflictDetector) countWildcards(pattern string) int {
	return CountWildcards(pattern)
}

func (detector *ThatPatternConflictDetector) calculateOverlapSeverity(pattern1, pattern2 string) string {
	overlap := detector.calculateOverlapPercentage(pattern1, pattern2)

	if overlap > 0.8 {
		return "critical"
	} else if overlap > 0.6 {
		return "high"
	} else if overlap > 0.4 {
		return "medium"
	} else {
		return "low"
	}
}

func (detector *ThatPatternConflictDetector) calculateOverlapPercentage(pattern1, pattern2 string) float64 {
	// Simplified overlap calculation
	testCases := []string{"HELLO", "HELLO WORLD", "GOOD MORNING", "WHAT IS YOUR NAME"}
	matches1 := 0
	matches2 := 0
	overlap := 0

	for _, testCase := range testCases {
		matched1 := detector.testPatternMatch(pattern1, testCase)
		matched2 := detector.testPatternMatch(pattern2, testCase)

		if matched1 {
			matches1++
		}
		if matched2 {
			matches2++
		}
		if matched1 && matched2 {
			overlap++
		}
	}

	if matches1+matches2-overlap == 0 {
		return 0
	}

	return float64(overlap) / float64(matches1+matches2-overlap)
}

// Suggestion generation functions
func (detector *ThatPatternConflictDetector) generateOverlapSuggestions(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Consider making pattern '%s' more specific", pattern1),
		fmt.Sprintf("Consider making pattern '%s' more specific", pattern2),
		"Add more specific words to differentiate the patterns",
		"Use different wildcard types to create distinct matching scopes",
	}
}

func (detector *ThatPatternConflictDetector) generateAmbiguitySuggestions(_, _ string) []string {
	return []string{
		"Reorder patterns to ensure proper priority",
		"Make one pattern more specific than the other",
		"Use different wildcard strategies for each pattern",
		"Consider combining patterns if they serve the same purpose",
	}
}

func (detector *ThatPatternConflictDetector) generatePrioritySuggestions(_, _ string) []string {
	return []string{
		"Reorder patterns to ensure more specific patterns come first",
		"Use different wildcard types to create clear priority",
		"Add more specific words to create clear differentiation",
		"Consider the intended use case for each pattern",
	}
}

func (detector *ThatPatternConflictDetector) generateWildcardSuggestions(_, _ string) []string {
	return []string{
		"Use consistent wildcard strategies across patterns",
		"Consider using different wildcard types for different purposes",
		"Ensure wildcard usage aligns with pattern intent",
		"Review wildcard placement for optimal matching",
	}
}

func (detector *ThatPatternConflictDetector) generateSpecificitySuggestions(_, _ string) []string {
	return []string{
		"Adjust pattern specificity to create clear differentiation",
		"Use more specific words in one pattern",
		"Use more general wildcards in the other pattern",
		"Consider the intended matching scope for each pattern",
	}
}

// Example generation functions
func (detector *ThatPatternConflictDetector) generateOverlapExamples(pattern1, pattern2 string) []string {
	return []string{
		"HELLO WORLD",
		"GOOD MORNING",
		"WHAT IS YOUR NAME",
	}
}

func (detector *ThatPatternConflictDetector) generateAmbiguityExamples(pattern1, pattern2 string) []string {
	return []string{
		"HELLO",
		"HELLO WORLD",
		"GOOD MORNING",
	}
}

func (detector *ThatPatternConflictDetector) generatePriorityExamples(pattern1, pattern2 string) []string {
	return []string{
		"HELLO",
		"HELLO WORLD",
		"GOOD MORNING",
	}
}

func (detector *ThatPatternConflictDetector) generateWildcardExamples(pattern1, pattern2 string) []string {
	return []string{
		"HELLO",
		"HELLO WORLD",
		"GOOD MORNING",
	}
}

func (detector *ThatPatternConflictDetector) generateSpecificityExamples(pattern1, pattern2 string) []string {
	return []string{
		"HELLO",
		"HELLO WORLD",
		"GOOD MORNING",
	}
}
