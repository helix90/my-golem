package golem

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// The graphmaster matches input as AIML 2.0 specifies. Every category is a
// path of words through a tree: its pattern, then its <that> pattern, then
// its <topic> pattern. Input is
// matched word by word, and at each word the branches are tried in priority
// order, so the first complete path found is the best match:
//
//	$WORD  a word with priority over wildcards
//	#      zero or more words
//	_      one or more words
//	WORD   an exact word, or <bot name="..."/>
//	<set>  one of the set's phrases
//	^      zero or more words
//	*      one or more words
//
// Wildcards match as few words as they can. A <set> matches its longest
// phrase first. A category without a <that> or <topic> matches any <that>
// or <topic> last of all, without capturing it.

// Sections of a graphmaster path
const (
	gmInput = iota
	gmThat
	gmTopic
)

// gmNode is a node of the graphmaster tree
type gmNode struct {
	dollar     map[string]*gmNode
	hash       *gmNode
	underscore *gmNode
	words      map[string]*gmNode
	bots       []gmNamedNode // <bot name="..."/>, sorted by name
	sets       []gmNamedNode // <set>name</set>, sorted by name
	caret      *gmNode
	star       *gmNode
	// any matches the whole section, for a category without <that> or <topic>
	any *gmNode

	// next continues the path into the next section (<that>, then <topic>)
	next *gmNode
	// categories end here, at the end of the topic section, in load order
	categories []*Category
}

type gmNamedNode struct {
	name string
	node *gmNode
}

// gmTokenRe splits a normalized pattern into words, keeping <set> and <bot>
// elements whole
var gmTokenRe = regexp.MustCompile(`(?i)<set>\s*([^<]*?)\s*</set>|<bot\s+name\s*=\s*["']([^"']*)["']\s*/>|\S+`)

// graphmaster returns the tree for the current Patterns, building it again
// when categories have been added or removed. The caller must hold
// patternsMu.
func (kb *AIMLKnowledgeBase) graphmaster() *gmNode {
	kb.graphMu.Lock()
	defer kb.graphMu.Unlock()

	patternsID := reflect.ValueOf(kb.Patterns).Pointer()
	if kb.graph != nil && kb.graphPatternsID == patternsID && kb.graphSize == len(kb.Patterns) {
		return kb.graph
	}

	keys := make([]string, 0, len(kb.Patterns))
	for key := range kb.Patterns {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := kb.loadOrder(kb.Patterns[keys[i]]), kb.loadOrder(kb.Patterns[keys[j]])
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})

	root := &gmNode{}
	for _, key := range keys {
		root.add(kb.Patterns[key])
	}

	kb.graph = root
	kb.graphPatternsID = patternsID
	kb.graphSize = len(kb.Patterns)
	return root
}

// invalidateGraphmaster makes the next match rebuild the tree
func (kb *AIMLKnowledgeBase) invalidateGraphmaster() {
	kb.graphMu.Lock()
	kb.graph = nil
	kb.graphMu.Unlock()
}

// addToGraphmaster adds a newly indexed category to the tree, if the tree is
// up to date apart from it. The caller must hold patternsMu for writing.
func (kb *AIMLKnowledgeBase) addToGraphmaster(category *Category) {
	kb.graphMu.Lock()
	defer kb.graphMu.Unlock()
	if kb.graph != nil && kb.graphPatternsID == reflect.ValueOf(kb.Patterns).Pointer() && kb.graphSize == len(kb.Patterns)-1 {
		kb.graph.add(category)
		kb.graphSize++
	} else {
		kb.graph = nil
	}
}

// add adds a category's path to the tree
func (n *gmNode) add(category *Category) {
	sections := [3]string{category.Pattern, category.That, category.Topic}
	node := n
	for s, section := range sections {
		if s > gmInput {
			if node.next == nil {
				node.next = &gmNode{}
			}
			node = node.next
		}
		tokens := gmTokenRe.FindAllStringSubmatch(NormalizePattern(section), -1)
		if len(tokens) == 0 && s > gmInput {
			node = wildcardChild(&node.any)
			continue
		}
		for _, token := range tokens {
			node = node.child(token)
		}
	}
	node.categories = append(node.categories, category)
}

// child returns the node for a pattern token, adding it if needed
func (n *gmNode) child(token []string) *gmNode {
	word := token[0]
	switch {
	case strings.HasPrefix(strings.ToLower(word), "<set>"):
		return namedChild(&n.sets, strings.ToUpper(token[1]))
	case strings.HasPrefix(strings.ToLower(word), "<bot"):
		return namedChild(&n.bots, strings.ToLower(token[2]))
	case word == "#":
		return wildcardChild(&n.hash)
	case word == "_":
		return wildcardChild(&n.underscore)
	case word == "^":
		return wildcardChild(&n.caret)
	case word == "*":
		return wildcardChild(&n.star)
	case len(word) > 1 && word[0] == '$':
		return wordChild(&n.dollar, word[1:])
	}
	return wordChild(&n.words, word)
}

func wildcardChild(slot **gmNode) *gmNode {
	if *slot == nil {
		*slot = &gmNode{}
	}
	return *slot
}

func wordChild(children *map[string]*gmNode, word string) *gmNode {
	if *children == nil {
		*children = make(map[string]*gmNode)
	}
	child, exists := (*children)[word]
	if !exists {
		child = &gmNode{}
		(*children)[word] = child
	}
	return child
}

func namedChild(children *[]gmNamedNode, name string) *gmNode {
	i := sort.Search(len(*children), func(i int) bool { return (*children)[i].name >= name })
	if i < len(*children) && (*children)[i].name == name {
		return (*children)[i].node
	}
	child := &gmNode{}
	*children = append(*children, gmNamedNode{})
	copy((*children)[i+1:], (*children)[i:])
	(*children)[i] = gmNamedNode{name: name, node: child}
	return child
}

// gmCapture is the words a wildcard or <set> matched
type gmCapture struct {
	section    int
	start, end int
	kind       string // star, underscore, caret, hash, or set
}

// gmMatcher matches one input against the tree
type gmMatcher struct {
	kb        *AIMLKnowledgeBase
	sections  [3][]string
	thatIndex int
	captures  []gmCapture
	// original maps the input words to the user's own text, if possible
	original *originalText
	// setPhrases caches each set's phrases for this match
	setPhrases map[string]map[string]bool
	setMaxLen  map[string]int
}

// match returns the first category whose path matches the rest of the
// input from position pos of section s
func (m *gmMatcher) match(n *gmNode, s, pos int) *Category {
	words := m.sections[s]
	rest := len(words) - pos

	if rest > 0 && n.dollar != nil {
		if child, exists := n.dollar[words[pos]]; exists {
			if c := m.match(child, s, pos+1); c != nil {
				return c
			}
		}
	}
	if c := m.wildcard(n.hash, s, pos, 0, "hash"); c != nil {
		return c
	}
	if c := m.wildcard(n.underscore, s, pos, 1, "underscore"); c != nil {
		return c
	}
	if rest > 0 && n.words != nil {
		if child, exists := n.words[words[pos]]; exists {
			if c := m.match(child, s, pos+1); c != nil {
				return c
			}
		}
	}
	for _, bot := range n.bots {
		value := strings.Fields(NormalizePattern(m.kb.botProperty(bot.name)))
		if len(value) > 0 && len(value) <= rest && equalWords(value, words[pos:pos+len(value)]) {
			if c := m.match(bot.node, s, pos+len(value)); c != nil {
				return c
			}
		}
	}
	if rest == 0 {
		if s == gmTopic {
			if c := m.pick(n); c != nil {
				return c
			}
		} else if n.next != nil {
			if c := m.match(n.next, s+1, 0); c != nil {
				return c
			}
		}
	}
	for _, set := range n.sets {
		if c := m.matchSet(set, s, pos); c != nil {
			return c
		}
	}
	if c := m.wildcard(n.caret, s, pos, 0, "caret"); c != nil {
		return c
	}
	if c := m.wildcard(n.star, s, pos, 1, "star"); c != nil {
		return c
	}
	if n.any != nil {
		return m.match(n.any, s, len(words))
	}
	return nil
}

// wildcard matches a wildcard child against as few words as it can, but at
// least min. A missing <that> or <topic> is matched by any wildcard. In the
// input, it first tries the lengths that end on a boundary of the user's own
// words, so that "* *" splits "user@example.com secret" (normalized to
// "USER AT EXAMPLECOM SECRET") as the user did.
func (m *gmMatcher) wildcard(child *gmNode, s, pos, min int, kind string) *Category {
	if child == nil {
		return nil
	}
	words := m.sections[s]
	if len(words) == 0 && s > gmInput {
		min = 0
	}
	if s != gmInput || m.original == nil {
		return m.wildcardEnds(child, s, pos, min, kind, func(int) bool { return true })
	}
	aligned := func(end int) bool { return end == pos || m.original.last[end-1] }
	if c := m.wildcardEnds(child, s, pos, min, kind, aligned); c != nil {
		return c
	}
	return m.wildcardEnds(child, s, pos, min, kind, func(end int) bool { return !aligned(end) })
}

// wildcardEnds is wildcard for the ends that try accepts, shortest first
func (m *gmMatcher) wildcardEnds(child *gmNode, s, pos, min int, kind string, try func(end int) bool) *Category {
	for end := pos + min; end <= len(m.sections[s]); end++ {
		if !try(end) {
			continue
		}
		m.captures = append(m.captures, gmCapture{section: s, start: pos, end: end, kind: kind})
		if c := m.match(child, s, end); c != nil {
			return c
		}
		m.captures = m.captures[:len(m.captures)-1]
	}
	return nil
}

// matchSet matches a <set> child against the longest phrase of the set that
// the input continues with
func (m *gmMatcher) matchSet(set gmNamedNode, s, pos int) *Category {
	phrases, maxLen := m.setLookup(set.name)
	words := m.sections[s]
	for end := min(len(words), pos+maxLen); end > pos; end-- {
		if !phrases[strings.Join(words[pos:end], " ")] {
			continue
		}
		m.captures = append(m.captures, gmCapture{section: s, start: pos, end: end, kind: "set"})
		if c := m.match(set.node, s, end); c != nil {
			return c
		}
		m.captures = m.captures[:len(m.captures)-1]
	}
	return nil
}

// setLookup returns a set's phrases, normalized like input, and the most
// words in any of them
func (m *gmMatcher) setLookup(name string) (map[string]bool, int) {
	if phrases, exists := m.setPhrases[name]; exists {
		return phrases, m.setMaxLen[name]
	}
	if m.setPhrases == nil {
		m.setPhrases = make(map[string]map[string]bool)
		m.setMaxLen = make(map[string]int)
	}
	phrases := make(map[string]bool)
	maxLen := 0
	for _, member := range m.kb.setMembers(name) {
		words := strings.Fields(NormalizePattern(member))
		if len(words) == 0 {
			continue
		}
		phrases[strings.Join(words, " ")] = true
		maxLen = max(maxLen, len(words))
	}
	m.setPhrases[name] = phrases
	m.setMaxLen[name] = maxLen
	return phrases, maxLen
}

// pick returns the first category at the end of a path that applies to the
// requested <that> index. Index 0 (the last response) is matched only by
// categories without a <that> index; another index only by categories with
// a <that> for that index or for any index.
func (m *gmMatcher) pick(n *gmNode) *Category {
	for _, category := range n.categories {
		if m.thatIndex == 0 && category.ThatIndex == 0 {
			return category
		}
		if m.thatIndex != 0 && category.That != "" && (category.ThatIndex == 0 || category.ThatIndex == m.thatIndex) {
			return category
		}
	}
	return nil
}

// wildcards returns the captured words, keyed as the template tags read
// them: star1... for the input, that_star1... for <that> and topic_star1...
// for <topic>. <that> captures are also keyed by wildcard type
// (that_underscore1...). Input captures are taken from the user's own text
// where possible (see originalText).
func (m *gmMatcher) wildcards() map[string]string {
	wildcards := make(map[string]string)
	counts := [3]int{}
	for _, capture := range m.captures {
		counts[capture.section]++
		index := counts[capture.section]
		value := strings.Join(m.sections[capture.section][capture.start:capture.end], " ")
		if capture.section == gmInput {
			if text, ok := m.original.span(capture.start, capture.end); ok {
				value = text
			}
		}
		switch capture.section {
		case gmInput:
			wildcards[fmt.Sprintf("star%d", index)] = value
		case gmThat:
			wildcards[fmt.Sprintf("that_star%d", index)] = value
			if capture.kind != "star" && capture.kind != "set" {
				wildcards[fmt.Sprintf("that_%s%d", capture.kind, index)] = value
			}
		case gmTopic:
			wildcards[fmt.Sprintf("topic_star%d", index)] = value
		}
	}
	return wildcards
}

// originalText maps the matched input words back to the user's own text,
// so that wildcards keep its case and punctuation. Each whitespace-separated
// token of the text is normalized on its own; when together they give the
// matched words, a capture made of whole tokens is taken from the text.
type originalText struct {
	tokens []string
	// token, first and last give, for each matched word, its token and
	// whether it is the first and last word of that token
	token       []int
	first, last []bool
}

// newOriginalText returns the mapping of words to the tokens of text, or nil
// if normalizing the tokens one by one doesn't give the same words (for
// example after a substitution that spans tokens). normalize must be the
// normalization that gave words, such as Golem.CachedNormalizePattern, which
// applies substitutions.
func newOriginalText(text string, words []string, normalize func(string) string) *originalText {
	o := &originalText{tokens: strings.Fields(text)}
	for i, token := range o.tokens {
		tokenWords := strings.Fields(normalize(token))
		for j := range tokenWords {
			o.token = append(o.token, i)
			o.first = append(o.first, j == 0)
			o.last = append(o.last, j == len(tokenWords)-1)
		}
		if len(o.token) > len(words) || !equalWords(tokenWords, words[len(o.token)-len(tokenWords):len(o.token)]) {
			return nil
		}
	}
	if len(o.token) != len(words) {
		return nil
	}
	return o
}

// span returns the text of matched words start to end, if they are whole
// tokens
func (o *originalText) span(start, end int) (string, bool) {
	if o == nil || start >= end || !o.first[start] || !o.last[end-1] {
		return "", false
	}
	return strings.Join(o.tokens[o.token[start]:o.token[end-1]+1], " "), true
}

func equalWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// botProperty returns a bot property for pattern matching
func (kb *AIMLKnowledgeBase) botProperty(name string) string {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	for key, value := range kb.Properties {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

// setMembers returns the members of a set for pattern matching
func (kb *AIMLKnowledgeBase) setMembers(name string) []string {
	if members, exists := kb.Sets[name]; exists {
		return members
	}
	for key, members := range kb.Sets {
		if strings.EqualFold(key, name) {
			return members
		}
	}
	return nil
}
