package golem

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Log redaction: with verbose logging, golem logs inputs, predicate values
// and template results, which can carry passwords and tokens. Every log line
// is filtered first:
//
//   - Inputs that carry a secret are matched by redaction patterns; the last
//     capture group of each match (or the whole match, without groups) is
//     masked. The defaults cover the bundled list-credential commands;
//     AddLogRedaction adds more.
//   - The value of every secret predicate (one the user store treats as
//     secret, by default *password*, *secret* and *token*) is masked wherever
//     it appears once it has been set or loaded.

// logMask replaces a secret in a log line
const logMask = "********"

// minLoggedSecret is the shortest value masked wherever it appears; shorter
// values would mask ordinary words
const minLoggedSecret = 3

// DefaultLogRedactions match inputs of the bundled AIML that carry a
// password: the last group is the password
var DefaultLogRedactions = []string{
	`(?i)\b(?:save|store|remember)\s+(?:my\s+)?list\s+credentials\s+\S+\s+(\S+)`,
	`(?i)\blist\s+login\s+\S+\s+(\S+)`,
	`(?i)\bpassword=([^&\s]+)`,
}

// logRedactor holds the patterns and secret values masked in log lines
type logRedactor struct {
	mu       sync.RWMutex
	patterns []*regexp.Regexp
	values   map[string]bool
	valuesRe *regexp.Regexp // the values, longest first; nil when none
}

func newLogRedactor() *logRedactor {
	r := &logRedactor{values: make(map[string]bool)}
	for _, pattern := range DefaultLogRedactions {
		r.patterns = append(r.patterns, regexp.MustCompile(pattern))
	}
	return r
}

// AddLogRedaction masks, in every log line, the last capture group of each
// match of pattern (or the whole match if it has no groups)
func (g *Golem) AddLogRedaction(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	g.redactor.mu.Lock()
	defer g.redactor.mu.Unlock()
	g.redactor.patterns = append(g.redactor.patterns, re)
	return nil
}

// noteSecret masks value in later log lines if name is a secret predicate
func (g *Golem) noteSecret(name, value string) {
	value = strings.TrimSpace(value)
	if g.redactor == nil || len(value) < minLoggedSecret || !g.isSecretPredicate(name) {
		return
	}
	r := g.redactor
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values[value] {
		return
	}
	r.values[value] = true
	quoted := make([]string, 0, len(r.values))
	for v := range r.values {
		quoted = append(quoted, regexp.QuoteMeta(v))
	}
	// Longest first, so that a value containing another is masked whole
	sortByLengthDesc(quoted)
	r.valuesRe = regexp.MustCompile(`(?i)` + strings.Join(quoted, "|"))
}

// isSecretPredicate reports whether a predicate holds a secret, by the user
// store's patterns or else the default ones
func (g *Golem) isSecretPredicate(name string) bool {
	if store := g.userStore; store != nil {
		return store.IsSecret(name)
	}
	return matchesAny(DefaultSecretPatterns, name)
}

// redact masks the secrets in a log line
func (r *logRedactor) redact(line string) string {
	if r == nil {
		return line
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, re := range r.patterns {
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			groups := re.FindStringSubmatchIndex(match)
			if len(groups) < 4 || groups[len(groups)-2] < 0 {
				return logMask
			}
			start, end := groups[len(groups)-2], groups[len(groups)-1]
			return match[:start] + logMask + match[end:]
		})
	}
	if r.valuesRe != nil {
		line = r.valuesRe.ReplaceAllString(line, logMask)
	}
	return line
}

// logf writes a log line with its secrets masked
func (g *Golem) logf(prefix, format string, args ...interface{}) {
	g.logger.Print(g.redactor.redact(prefix + fmt.Sprintf(format, args...)))
}

func sortByLengthDesc(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && len(values[j]) > len(values[j-1]); j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
