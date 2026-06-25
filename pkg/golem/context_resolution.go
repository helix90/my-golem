package golem

import (
	"math"
	"regexp"
	"strings"
)

// FuzzyContextMatcher provides fuzzy matching capabilities for context resolution
type FuzzyContextMatcher struct {
	EditDistanceThreshold int
	PhoneticMatching      bool
	Stemming              bool
	MinSimilarity         float64
}

// NewFuzzyContextMatcher creates a new fuzzy context matcher with default settings
func NewFuzzyContextMatcher() *FuzzyContextMatcher {
	return &FuzzyContextMatcher{
		EditDistanceThreshold: 2,
		PhoneticMatching:      true,
		Stemming:              true,
		MinSimilarity:         0.7,
	}
}

// MatchWithFuzzy performs fuzzy matching between context and pattern
func (f *FuzzyContextMatcher) MatchWithFuzzy(context, pattern string) (bool, float64) {
	// Normalize both strings
	normalizedContext := f.normalizeString(context)
	normalizedPattern := f.normalizeString(pattern)

	// Try exact match first
	if normalizedContext == normalizedPattern {
		return true, 1.0
	}

	// Calculate similarity score
	similarity := f.calculateSimilarity(normalizedContext, normalizedPattern)

	return similarity >= f.MinSimilarity, similarity
}

// normalizeString normalizes a string for fuzzy matching
func (f *FuzzyContextMatcher) normalizeString(s string) string {
	// Convert to uppercase
	s = strings.ToUpper(s)

	// Remove extra whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	// Remove punctuation
	s = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(s, "")

	// Apply stemming if enabled
	if f.Stemming {
		s = f.stemString(s)
	}

	return s
}

// calculateSimilarity calculates the similarity between two strings
func (f *FuzzyContextMatcher) calculateSimilarity(s1, s2 string) float64 {
	// Calculate multiple similarity metrics
	editDistance := f.levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	// Edit distance similarity (0-1, higher is better)
	editSimilarity := 1.0 - (float64(editDistance) / maxLen)

	// Word overlap similarity
	wordSimilarity := f.calculateWordOverlap(s1, s2)

	// Phonetic similarity (if enabled)
	phoneticSimilarity := 0.0
	if f.PhoneticMatching {
		phoneticSimilarity = f.calculatePhoneticSimilarity(s1, s2)
	}

	// Combine similarities with weights
	similarity := (editSimilarity * 0.4) + (wordSimilarity * 0.4) + (phoneticSimilarity * 0.2)

	return similarity
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (f *FuzzyContextMatcher) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1, // deletion
				minInt(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// calculateWordOverlap calculates word-level overlap similarity
func (f *FuzzyContextMatcher) calculateWordOverlap(s1, s2 string) float64 {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create word sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	// Calculate intersection and union
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	return float64(intersection) / float64(union)
}

// calculatePhoneticSimilarity calculates phonetic similarity using Soundex
func (f *FuzzyContextMatcher) calculatePhoneticSimilarity(s1, s2 string) float64 {
	soundex1 := f.soundex(s1)
	soundex2 := f.soundex(s2)

	if soundex1 == soundex2 {
		return 1.0
	}

	// Calculate partial phonetic similarity
	return f.calculatePartialPhoneticSimilarity(soundex1, soundex2)
}

// soundex implements the Soundex algorithm
func (f *FuzzyContextMatcher) soundex(s string) string {
	if len(s) == 0 {
		return ""
	}

	// Convert to uppercase
	s = strings.ToUpper(s)

	// Keep first letter
	result := string(s[0])

	// Remove vowels and H, W, Y
	consonants := ""
	for _, char := range s[1:] {
		if !strings.ContainsRune("AEIOUHWY", char) {
			consonants += string(char)
		}
	}

	// Replace consonants with numbers
	for _, char := range consonants {
		switch char {
		case 'B', 'F', 'P', 'V':
			result += "1"
		case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
			result += "2"
		case 'D', 'T':
			result += "3"
		case 'L':
			result += "4"
		case 'M', 'N':
			result += "5"
		case 'R':
			result += "6"
		}
	}

	// Remove duplicates
	result = f.removeDuplicateDigits(result)

	// Pad with zeros
	for len(result) < 4 {
		result += "0"
	}

	return result[:4]
}

// removeDuplicateDigits removes consecutive duplicate digits
func (f *FuzzyContextMatcher) removeDuplicateDigits(s string) string {
	if len(s) <= 1 {
		return s
	}

	result := string(s[0])
	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1] {
			result += string(s[i])
		}
	}

	return result
}

// calculatePartialPhoneticSimilarity calculates partial phonetic similarity
func (f *FuzzyContextMatcher) calculatePartialPhoneticSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Calculate character-level similarity
	matches := 0
	minLen := minInt(len(s1), len(s2))

	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			matches++
		}
	}

	return float64(matches) / float64(maxInt(len(s1), len(s2)))
}

// stemString performs basic stemming
func (f *FuzzyContextMatcher) stemString(s string) string {
	words := strings.Fields(s)
	stemmed := make([]string, len(words))

	for i, word := range words {
		stemmed[i] = f.stemWord(word)
	}

	return strings.Join(stemmed, " ")
}

// stemWord performs basic Porter stemming
func (f *FuzzyContextMatcher) stemWord(word string) string {
	if len(word) <= 2 {
		return word
	}

	// Convert to lowercase for stemming
	word = strings.ToLower(word)

	// Remove common suffixes
	suffixes := []string{"ing", "ed", "er", "est", "ly", "tion", "sion", "ness", "ment"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word) > len(suffix)+2 {
			word = word[:len(word)-len(suffix)]
			break
		}
	}

	return word
}

// SemanticContextMatcher provides semantic similarity matching
type SemanticContextMatcher struct {
	Synonyms       map[string][]string
	Antonyms       map[string][]string
	WordWeights    map[string]float64
	MinSimilarity  float64
	DomainMappings map[string][]string // Domain-specific word mappings (e.g., "COLORS" -> ["RED", "BLUE", "GREEN", "ORANGE"])
}

// NewSemanticContextMatcher creates a new semantic context matcher
func NewSemanticContextMatcher() *SemanticContextMatcher {
	return &SemanticContextMatcher{
		Synonyms:       make(map[string][]string),
		Antonyms:       make(map[string][]string),
		WordWeights:    make(map[string]float64),
		MinSimilarity:  0.4,
		DomainMappings: make(map[string][]string),
	}
}

// InitializeSynonyms initializes the synonym dictionary with a minimal set
func (s *SemanticContextMatcher) InitializeSynonyms() {
	// Basic synonym mappings - minimal set to avoid duplicates
	synonymMap := map[string][]string{
		"happy":       {"glad", "joyful", "cheerful", "pleased", "content"},
		"sad":         {"unhappy", "depressed", "gloomy", "melancholy", "sorrowful"},
		"angry":       {"mad", "furious", "irritated", "annoyed", "upset"},
		"good":        {"great", "excellent", "wonderful", "fantastic", "awesome"},
		"bad":         {"terrible", "awful", "horrible", "dreadful", "poor"},
		"big":         {"large", "huge", "enormous", "giant", "massive"},
		"small":       {"tiny", "little", "miniature", "petite", "mini"},
		"fast":        {"quick", "rapid", "swift", "speedy", "brisk"},
		"slow":        {"sluggish", "leisurely", "gradual", "delayed", "tardy"},
		"beautiful":   {"pretty", "lovely", "gorgeous", "attractive", "stunning"},
		"ugly":        {"unattractive", "hideous", "repulsive", "disgusting", "gross"},
		"smart":       {"intelligent", "clever", "bright", "wise", "brilliant"},
		"stupid":      {"dumb", "foolish", "silly", "idiotic", "unintelligent"},
		"love":        {"adore", "cherish", "treasure", "enjoy", "like"},
		"hate":        {"despise", "loathe", "detest", "abhor", "dislike"},
		"want":        {"desire", "wish", "need", "require", "crave"},
		"have":        {"possess", "own", "hold", "contain", "include"},
		"get":         {"obtain", "acquire", "receive", "gain", "earn"},
		"make":        {"create", "build", "construct", "produce", "generate"},
		"see":         {"look", "view", "observe", "watch", "notice"},
		"hear":        {"listen", "audit", "perceive", "detect", "sense"},
		"know":        {"understand", "comprehend", "realize", "recognize", "grasp"},
		"think":       {"believe", "consider", "suppose", "imagine", "contemplate"},
		"say":         {"speak", "tell", "express", "communicate", "utter"},
		"go":          {"move", "travel", "proceed", "advance", "journey"},
		"come":        {"arrive", "approach", "reach", "enter", "appear"},
		"give":        {"provide", "offer", "supply", "deliver", "present"},
		"take":        {"grab", "seize", "capture", "obtain", "acquire"},
		"find":        {"discover", "locate", "detect", "uncover", "spot"},
		"lose":        {"misplace", "forfeit", "surrender", "abandon", "drop"},
		"help":        {"assist", "aid", "support", "serve", "facilitate"},
		"work":        {"labor", "toil", "function", "operate", "perform"},
		"play":        {"game", "sport", "entertainment", "recreation", "fun"},
		"eat":         {"consume", "devour", "ingest", "dine", "feed"},
		"drink":       {"sip", "gulp", "swallow", "imbibe", "quaff"},
		"sleep":       {"rest", "slumber", "nap", "doze", "repose"},
		"walk":        {"stroll", "march", "hike", "pace", "stride"},
		"run":         {"sprint", "dash", "race", "jog", "rush"},
		"jump":        {"leap", "hop", "bounce", "spring", "vault"},
		"sit":         {"rest", "settle", "perch", "occupy", "position"},
		"stand":       {"rise", "upright", "erect", "position", "place"},
		"lie":         {"recline", "rest", "stretch", "position", "place"},
		"open":        {"unlock", "unseal", "reveal", "expose", "uncover"},
		"close":       {"shut", "seal", "lock", "cover", "conceal"},
		"start":       {"begin", "commence", "initiate", "launch", "open"},
		"stop":        {"end", "finish", "halt", "cease", "terminate"},
		"continue":    {"proceed", "persist", "maintain", "keep", "carry"},
		"change":      {"alter", "modify", "transform", "convert", "shift"},
		"keep":        {"maintain", "preserve", "retain", "hold", "save"},
		"put":         {"place", "set", "position", "lay", "deposit"},
		"move":        {"shift", "relocate", "transfer", "displace", "transport"},
		"turn":        {"rotate", "spin", "twist", "pivot", "revolve"},
		"push":        {"press", "shove", "thrust", "force", "drive"},
		"pull":        {"tug", "drag", "draw", "yank", "haul"},
		"throw":       {"toss", "hurl", "fling", "cast", "pitch"},
		"catch":       {"grab", "seize", "capture", "snatch", "intercept"},
		"hit":         {"strike", "punch", "slap", "beat", "smack"},
		"kick":        {"strike", "boot", "punt", "hit", "smack"},
		"touch":       {"feel", "contact", "handle", "stroke", "pat"},
		"feel":        {"sense", "perceive", "experience", "touch", "detect"},
		"smell":       {"scent", "odor", "aroma", "fragrance", "stench"},
		"taste":       {"flavor", "savor", "sample", "try", "experience"},
		"hot":         {"warm", "heated", "burning", "scorching", "blazing"},
		"cold":        {"cool", "chilly", "freezing", "frigid", "icy"},
		"warm":        {"mild", "temperate", "cozy", "comfortable", "pleasant"},
		"cool":        {"chilly", "refreshing", "brisk", "crisp", "fresh"},
		"bright":      {"luminous", "radiant", "brilliant", "shining", "glowing"},
		"dark":        {"dim", "gloomy", "shadowy", "murky", "obscure"},
		"loud":        {"noisy", "booming", "thunderous", "deafening", "ear-splitting"},
		"quiet":       {"silent", "hushed", "peaceful", "calm", "still"},
		"high":        {"tall", "elevated", "lofty", "towering", "soaring"},
		"low":         {"short", "small", "shallow", "depressed", "sunken"},
		"long":        {"lengthy", "extended", "prolonged", "stretched", "drawn"},
		"short":       {"brief", "concise", "abbreviated", "condensed", "compact"},
		"wide":        {"broad", "extensive", "spacious", "expansive", "roomy"},
		"narrow":      {"thin", "slim", "slender", "tight", "constricted"},
		"thick":       {"dense", "heavy", "solid", "substantial", "bulky"},
		"thin":        {"slim", "slender", "lean", "narrow", "delicate"},
		"heavy":       {"weighty", "burdensome", "massive", "substantial", "dense"},
		"light":       {"weightless", "feathery", "airy", "buoyant", "floating"},
		"strong":      {"powerful", "mighty", "robust", "sturdy", "tough"},
		"weak":        {"feeble", "frail", "delicate", "fragile", "powerless"},
		"hard":        {"difficult", "tough", "challenging", "rigorous", "demanding"},
		"easy":        {"simple", "effortless", "straightforward", "uncomplicated", "basic"},
		"new":         {"fresh", "recent", "modern", "contemporary", "latest"},
		"old":         {"ancient", "aged", "vintage", "antique", "elderly"},
		"young":       {"youthful", "juvenile", "adolescent", "immature", "fresh"},
		"clean":       {"pure", "spotless", "pristine", "immaculate", "sanitary"},
		"dirty":       {"filthy", "soiled", "stained", "contaminated", "unclean"},
		"full":        {"complete", "entire", "whole", "packed", "crammed"},
		"empty":       {"vacant", "hollow", "void", "bare", "unoccupied"},
		"rich":        {"wealthy", "affluent", "prosperous", "opulent", "luxurious"},
		"poor":        {"impoverished", "destitute", "needy", "broke", "penniless"},
		"expensive":   {"costly", "pricey", "dear", "valuable", "premium"},
		"cheap":       {"inexpensive", "affordable", "budget", "economical", "low-cost"},
		"free":        {"gratis", "complimentary", "no-cost", "unpaid", "voluntary"},
		"busy":        {"occupied", "engaged", "active", "working", "involved"},
		"available":   {"free", "unoccupied", "idle", "leisurely", "unengaged"},
		"safe":        {"secure", "protected", "harmless", "reliable", "trustworthy"},
		"dangerous":   {"risky", "hazardous", "perilous", "unsafe", "threatening"},
		"important":   {"significant", "crucial", "vital", "essential", "critical"},
		"unimportant": {"trivial", "minor", "insignificant", "irrelevant", "meaningless"},
		"possible":    {"feasible", "achievable", "practical", "viable", "doable"},
		"impossible":  {"unfeasible", "unachievable", "impractical", "unviable", "undoable"},
		"true":        {"correct", "accurate", "valid", "genuine", "authentic"},
		"false":       {"incorrect", "wrong", "invalid", "fake", "artificial"},
		"real":        {"actual", "genuine", "authentic", "true", "legitimate"},
		"fake":        {"artificial", "false", "imitation", "synthetic", "counterfeit"},
		"right":       {"correct", "proper", "appropriate", "suitable", "fitting"},
		"wrong":       {"incorrect", "improper", "inappropriate", "unsuitable", "unfitting"},
		"yes":         {"yeah", "yep", "sure", "absolutely", "definitely"},
		"no":          {"nope", "nah", "never", "not", "negative"},
		"maybe":       {"perhaps", "possibly", "might", "could", "potentially"},
		"always":      {"forever", "constantly", "continuously", "perpetually", "eternally"},
		"never":       {"not", "not at all", "absolutely not", "definitely not", "no way"},
		"sometimes":   {"occasionally", "periodically", "intermittently", "sporadically", "now and then"},
		"often":       {"frequently", "regularly", "commonly", "usually", "typically"},
		"rarely":      {"seldom", "infrequently", "hardly ever", "scarcely", "barely"},
		"here":        {"this place", "present", "current location", "where I am", "this spot"},
		"there":       {"that place", "over there", "yonder", "that location", "that spot"},
		"now":         {"currently", "at present", "at this moment", "right now", "immediately"},
		"then":        {"at that time", "previously", "earlier", "before", "in the past"},
		"today":       {"this day", "now", "present day", "current day", "this moment"},
		"tomorrow":    {"next day", "the day after", "future day", "coming day", "next"},
		"yesterday":   {"previous day", "the day before", "past day", "earlier day", "last day"},
		"morning":     {"dawn", "early", "sunrise", "break of day", "first light"},
		"afternoon":   {"midday", "noon", "after noon", "daytime", "sunny time"},
		"evening":     {"dusk", "twilight", "sunset", "end of day", "nightfall"},
		"night":       {"darkness", "midnight", "late", "after dark", "nocturnal"},
		"day":         {"daylight", "daytime", "sunny period", "bright time", "light hours"},
		"week":        {"seven days", "period", "time span", "duration", "interval"},
		"month":       {"thirty days", "period", "time span", "duration", "interval"},
		"year":        {"twelve months", "period", "time span", "duration", "interval"},
		"time":        {"moment", "instant", "period", "duration", "interval"},
		"place":       {"location", "position", "spot", "site", "area"},
		"thing":       {"object", "item", "entity", "matter", "substance"},
		"person":      {"individual", "human", "being", "someone", "somebody"},
		"people":      {"individuals", "humans", "beings", "persons", "folks"},
		"man":         {"male", "guy", "fellow", "gentleman", "dude"},
		"woman":       {"female", "lady", "gal", "girl", "dame"},
		"child":       {"kid", "youngster", "youth", "minor", "little one"},
		"baby":        {"infant", "newborn", "toddler", "little one", "bundle"},
		"friend":      {"buddy", "pal", "companion", "acquaintance", "mate"},
		"family":      {"relatives", "kin", "clan", "household", "loved ones"},
		"home":        {"house", "residence", "dwelling", "abode", "place"},
		"house":       {"home", "residence", "dwelling", "abode", "building"},
		"car":         {"automobile", "vehicle", "auto", "machine", "wheels"},
		"food":        {"meal", "nourishment", "sustenance", "cuisine", "fare"},
		"water":       {"liquid", "H2O", "aqua", "fluid", "drink"},
		"money":       {"cash", "currency", "funds", "dollars", "bucks"},
		"book":        {"novel", "tome", "publication", "volume", "text"},
		"movie":       {"film", "picture", "cinema", "flick", "show"},
		"music":       {"song", "melody", "tune", "harmony", "sound"},
		"game":        {"play", "sport", "entertainment", "recreation", "fun"},
		"job":         {"work", "labor", "toil", "function", "operate", "perform"},
		"school":      {"education", "learning", "academy", "institution", "college"},
		"hospital":    {"medical center", "clinic", "healthcare", "infirmary", "medical"},
		"store":       {"shop", "market", "retail", "outlet", "business"},
		"restaurant":  {"eatery", "diner", "cafe", "bistro", "food place"},
		"hotel":       {"inn", "lodging", "accommodation", "resort", "hostel"},
		"airport":     {"terminal", "aviation", "flight", "travel", "departure"},
		"station":     {"depot", "terminal", "stop", "platform", "hub"},
		"street":      {"road", "avenue", "boulevard", "path", "way"},
		"city":        {"town", "metropolis", "urban", "municipality", "community"},
		"country":     {"nation", "state", "land", "territory", "republic"},
		"world":       {"earth", "globe", "planet", "universe", "cosmos"},
		"life":        {"existence", "being", "living", "survival", "reality"},
		"death":       {"passing", "demise", "end", "expiration", "departure"},
		"birth":       {"beginning", "start", "creation", "genesis", "origin"},
	}

	// Build bidirectional synonym map
	for word, synonyms := range synonymMap {
		s.Synonyms[word] = synonyms
		for _, synonym := range synonyms {
			if s.Synonyms[synonym] == nil {
				s.Synonyms[synonym] = []string{}
			}
			s.Synonyms[synonym] = append(s.Synonyms[synonym], word)
		}
	}
}

// InitializeDomainMappings initializes domain-specific word mappings
func (s *SemanticContextMatcher) InitializeDomainMappings() {
	// Color domain mappings
	s.DomainMappings["COLORS"] = []string{
		"RED", "BLUE", "GREEN", "YELLOW", "ORANGE", "PURPLE", "PINK", "BROWN", "BLACK", "WHITE",
		"GRAY", "GREY", "CYAN", "MAGENTA", "LIME", "MAROON", "NAVY", "OLIVE", "TEAL", "SILVER",
		"GOLD", "CRIMSON", "SCARLET", "VERMILION", "TURQUOISE", "INDIGO", "VIOLET", "AQUA",
	}

	// Animal domain mappings
	s.DomainMappings["ANIMALS"] = []string{
		"DOG", "CAT", "BIRD", "FISH", "HORSE", "COW", "PIG", "SHEEP", "GOAT", "CHICKEN",
		"DUCK", "GOOSE", "TURKEY", "RABBIT", "HAMSTER", "MOUSE", "RAT", "SQUIRREL", "DEER",
		"BEAR", "LION", "TIGER", "ELEPHANT", "GIRAFFE", "ZEBRA", "MONKEY", "APE", "WOLF",
		"FOX", "RACCOON", "SKUNK", "OPOSSUM", "BAT", "OWL", "EAGLE", "HAWK", "FALCON",
	}

	// Food domain mappings
	s.DomainMappings["FOOD"] = []string{
		"PIZZA", "BURGER", "SANDWICH", "SALAD", "SOUP", "STEAK", "CHICKEN", "FISH", "PASTA",
		"RICE", "BREAD", "CAKE", "COOKIE", "PIE", "ICE CREAM", "CANDY", "CHOCOLATE", "FRUIT",
		"VEGETABLE", "APPLE", "BANANA", "ORANGE", "GRAPE", "STRAWBERRY", "CARROT", "POTATO",
		"ONION", "TOMATO", "LETTUCE", "CHEESE", "MILK", "EGG", "BUTTER", "OIL", "SALT",
	}

	// Weather domain mappings
	s.DomainMappings["WEATHER"] = []string{
		"SUNNY", "CLOUDY", "RAINY", "SNOWY", "WINDY", "FOGGY", "STORMY", "HOT", "COLD",
		"WARM", "COOL", "HUMID", "DRY", "WET", "CLEAR", "OVERCAST", "THUNDERSTORM", "BLIZZARD",
		"TORNADO", "HURRICANE", "DRIZZLE", "SHOWER", "BREEZE", "GALE", "FROST", "ICE",
	}
}

// MatchWithSemanticSimilarity performs semantic similarity matching
func (s *SemanticContextMatcher) MatchWithSemanticSimilarity(context, pattern string) (bool, float64) {
	// Normalize both strings
	normalizedContext := s.normalizeString(context)
	normalizedPattern := s.normalizeString(pattern)

	// Try exact match first
	if normalizedContext == normalizedPattern {
		return true, 1.0
	}

	// Calculate semantic similarity
	similarity := s.calculateSemanticSimilarity(normalizedContext, normalizedPattern)

	return similarity >= s.MinSimilarity, similarity
}

// normalizeString normalizes a string for semantic matching
func (s *SemanticContextMatcher) normalizeString(str string) string {
	// Convert to uppercase
	str = strings.ToUpper(str)

	// Remove extra whitespace
	str = regexp.MustCompile(`\s+`).ReplaceAllString(str, " ")
	str = strings.TrimSpace(str)

	// Remove punctuation
	str = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(str, "")

	return str
}

// calculateSemanticSimilarity calculates semantic similarity between two strings
func (s *SemanticContextMatcher) calculateSemanticSimilarity(s1, s2 string) float64 {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Calculate word-level semantic similarity
	wordSimilarities := make([]float64, 0)

	for _, word1 := range words1 {
		maxSimilarity := 0.0
		for _, word2 := range words2 {
			similarity := s.calculateWordSemanticSimilarity(word1, word2)
			if similarity > maxSimilarity {
				maxSimilarity = similarity
			}
		}
		wordSimilarities = append(wordSimilarities, maxSimilarity)
	}

	// Calculate average similarity
	totalSimilarity := 0.0
	for _, similarity := range wordSimilarities {
		totalSimilarity += similarity
	}

	return totalSimilarity / float64(len(wordSimilarities))
}

// calculateWordSemanticSimilarity calculates semantic similarity between two words
func (s *SemanticContextMatcher) calculateWordSemanticSimilarity(word1, word2 string) float64 {
	// Exact match
	if word1 == word2 {
		return 1.0
	}

	// Check if words are in the same domain (e.g., both colors)
	if s.areInSameDomain(word1, word2) {
		return 0.95 // High similarity for domain matches
	}

	// Check synonyms
	if s.areSynonyms(word1, word2) {
		return 0.9
	}

	// Check if one word is a synonym of the other
	if s.hasSynonym(word1, word2) {
		return 0.8
	}

	// Check antonyms (opposite meaning)
	if s.areAntonyms(word1, word2) {
		return 0.1
	}

	// Check partial matches (substring)
	if strings.Contains(word1, word2) || strings.Contains(word2, word1) {
		return 0.6
	}

	// Check phonetic similarity
	phoneticSimilarity := s.calculatePhoneticSimilarity(word1, word2)
	if phoneticSimilarity > 0.7 {
		return phoneticSimilarity * 0.5
	}

	// Check edit distance
	editDistance := s.levenshteinDistance(word1, word2)
	maxLen := math.Max(float64(len(word1)), float64(len(word2)))
	if maxLen > 0 {
		editSimilarity := 1.0 - (float64(editDistance) / maxLen)
		if editSimilarity > 0.5 {
			return editSimilarity * 0.3
		}
	}

	return 0.0
}

// areInSameDomain checks if two words belong to the same domain
func (s *SemanticContextMatcher) areInSameDomain(word1, word2 string) bool {
	word1 = strings.ToUpper(word1)
	word2 = strings.ToUpper(word2)

	// Check each domain to see if both words are in it
	for _, domainWords := range s.DomainMappings {
		word1InDomain := false
		word2InDomain := false

		for _, domainWord := range domainWords {
			if domainWord == word1 {
				word1InDomain = true
			}
			if domainWord == word2 {
				word2InDomain = true
			}
		}

		// If both words are in the same domain, they're similar
		if word1InDomain && word2InDomain {
			return true
		}
	}

	return false
}

// areSynonyms checks if two words are synonyms
func (s *SemanticContextMatcher) areSynonyms(word1, word2 string) bool {
	if synonyms, exists := s.Synonyms[word1]; exists {
		for _, synonym := range synonyms {
			if synonym == word2 {
				return true
			}
		}
	}
	return false
}

// hasSynonym checks if word1 has word2 as a synonym
func (s *SemanticContextMatcher) hasSynonym(word1, word2 string) bool {
	return s.areSynonyms(word1, word2)
}

// areAntonyms checks if two words are antonyms
func (s *SemanticContextMatcher) areAntonyms(word1, word2 string) bool {
	if antonyms, exists := s.Antonyms[word1]; exists {
		for _, antonym := range antonyms {
			if antonym == word2 {
				return true
			}
		}
	}
	return false
}

// calculatePhoneticSimilarity calculates phonetic similarity using Soundex
func (s *SemanticContextMatcher) calculatePhoneticSimilarity(word1, word2 string) float64 {
	soundex1 := s.soundex(word1)
	soundex2 := s.soundex(word2)

	if soundex1 == soundex2 {
		return 1.0
	}

	// Calculate partial phonetic similarity
	return s.calculatePartialPhoneticSimilarity(soundex1, soundex2)
}

// soundex implements the Soundex algorithm
func (s *SemanticContextMatcher) soundex(word string) string {
	if len(word) == 0 {
		return ""
	}

	// Convert to uppercase
	word = strings.ToUpper(word)

	// Keep first letter
	result := string(word[0])

	// Remove vowels and H, W, Y
	consonants := ""
	for _, char := range word[1:] {
		if !strings.ContainsRune("AEIOUHWY", char) {
			consonants += string(char)
		}
	}

	// Replace consonants with numbers
	for _, char := range consonants {
		switch char {
		case 'B', 'F', 'P', 'V':
			result += "1"
		case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
			result += "2"
		case 'D', 'T':
			result += "3"
		case 'L':
			result += "4"
		case 'M', 'N':
			result += "5"
		case 'R':
			result += "6"
		}
	}

	// Remove duplicates
	result = s.removeDuplicateDigits(result)

	// Pad with zeros
	for len(result) < 4 {
		result += "0"
	}

	return result[:4]
}

// removeDuplicateDigits removes consecutive duplicate digits
func (s *SemanticContextMatcher) removeDuplicateDigits(str string) string {
	if len(str) <= 1 {
		return str
	}

	result := string(str[0])
	for i := 1; i < len(str); i++ {
		if str[i] != str[i-1] {
			result += string(str[i])
		}
	}

	return result
}

// calculatePartialPhoneticSimilarity calculates partial phonetic similarity
func (s *SemanticContextMatcher) calculatePartialPhoneticSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Calculate character-level similarity
	matches := 0
	minLen := minInt(len(s1), len(s2))

	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			matches++
		}
	}

	return float64(matches) / float64(maxInt(len(s1), len(s2)))
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (s *SemanticContextMatcher) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1, // deletion
				minInt(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// Helper functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
