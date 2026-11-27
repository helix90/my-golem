package golem

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// The wikipedia SRAIX handler looks topics up in Wikipedia's API (the
// service's URL template, e.g. https://en.wikipedia.org/w/api.php) and
// returns the first sentences of the article's introduction.
//
// Operations:
//
//	summary - searches for the input ("ada lovelace", "the eiffel tower")
//	          and summarizes the best match: an article whose title is the
//	          input, else the top result
//	page    - summarizes the article with the input as its exact title
//	          (redirects are followed), for the buttons summary offers
//
// Both set the predicates wiki_status ("found", "several" or "none"),
// wiki_summary (the reply), wiki_title, wiki_url and wiki_description. When the input is ambiguous
// (the match is a disambiguation page, such as "Mercury"), wiki_status is
// "several" and wiki_options holds a <button> for each of the other
// results, whose postback is "WIKIPEDIA PAGE <title>". A failed request
// (including Wikipedia's "too many requests") is an error, so <sraix>
// gives the service's fallback.
var wikipediaOperations = map[string]bool{"summary": true, "page": true}

// wikipediaMaxSummary bounds a summary's length in characters
const wikipediaMaxSummary = 420

// wikiPage is one page of a query response (formatversion=2)
type wikiPage struct {
	Title       string `json:"title"`
	Index       int    `json:"index"`
	Missing     bool   `json:"missing"`
	Extract     string `json:"extract"`
	Description string `json:"description"`
	FullURL     string `json:"fullurl"`
	PageProps   struct {
		Disambiguation *string `json:"disambiguation"`
	} `json:"pageprops"`
}

func (p wikiPage) disambiguation() bool { return p.PageProps.Disambiguation != nil }

var wikiLeadingArticleRe = regexp.MustCompile(`(?i)^(?:a|an|the)\s+`)

func (sm *SRAIXManager) sendWikipedia(config *SRAIXConfig, input string, params map[string]string) (sraixResponse, error) {
	apiURL, _, err := sm.substituteURLTemplate(config.URLTemplate, "", params, config.Headers)
	if err != nil {
		return sraixResponse{}, err
	}
	query := strings.TrimSpace(strings.Trim(strings.TrimSpace(input), "?!.\"'"))
	if query == "" {
		return wikiNone(query), nil
	}

	values := url.Values{
		"action":        {"query"},
		"format":        {"json"},
		"formatversion": {"2"},
		"prop":          {"extracts|pageprops|info|description"},
		"exintro":       {"1"},
		"explaintext":   {"1"},
		"exlimit":       {"max"},
		"inprop":        {"url"},
		"ppprop":        {"disambiguation"},
		"redirects":     {"1"},
	}
	if config.Operation == "page" {
		values.Set("titles", query)
	} else {
		search := wikiLeadingArticleRe.ReplaceAllString(query, "")
		values.Set("generator", "search")
		values.Set("gsrsearch", search)
		values.Set("gsrlimit", "6")
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	data, err := sm.wikipediaGet(ctx, config, params, apiURL+"?"+values.Encode())
	if err != nil {
		return sraixResponse{}, err
	}
	var result struct {
		Query struct {
			Pages []wikiPage `json:"pages"`
		} `json:"query"`
		Error *struct {
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return sraixResponse{}, fmt.Errorf("wikipedia replied with something other than JSON (%d bytes)", len(data))
	}
	if result.Error != nil {
		return sraixResponse{}, fmt.Errorf("wikipedia: %s", result.Error.Info)
	}
	if sm.verbose {
		sm.logger.Printf("Wikipedia %s %q: %d pages", config.Operation, query, len(result.Query.Pages))
	}

	var pages []wikiPage
	for _, page := range result.Query.Pages {
		if !page.Missing && page.Title != "" {
			pages = append(pages, page)
		}
	}
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Index < pages[j].Index })
	return chooseWikiPage(query, pages), nil
}

// wikipediaRetryWait is the longest wait before trying again when
// Wikipedia says there were too many requests
var wikipediaRetryWait = 2 * time.Second

// wikipediaGet makes a request, trying once more after a short wait when
// Wikipedia answers 429 Too Many Requests
func (sm *SRAIXManager) wikipediaGet(ctx context.Context, config *SRAIXConfig, params map[string]string, target string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		if err := sm.setHeaders(req, config, "", params); err != nil {
			return nil, err
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "golem-aiml-bot/1.0")
		}
		resp, err := sm.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("wikipedia request failed: %v", err)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxSRAIXResponseSize+1))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt == 0 {
			wait := wikipediaRetryWait
			if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && time.Duration(seconds)*time.Second < wait {
				wait = time.Duration(seconds) * time.Second
			}
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("wikipedia replied %s", resp.Status)
		}
		return data, nil
	}
}

// chooseWikiPage picks the page a query means: one whose title is the query
// (ignoring case and a leading "the"), else the top result. A
// disambiguation page offers the other results instead.
func chooseWikiPage(query string, pages []wikiPage) sraixResponse {
	if len(pages) == 0 {
		return wikiNone(query)
	}
	normal := func(s string) string {
		return strings.ToLower(wikiLeadingArticleRe.ReplaceAllString(strings.TrimSpace(s), ""))
	}
	chosen := pages[0]
	for _, page := range pages {
		if !page.disambiguation() && normal(page.Title) == normal(query) {
			chosen = page
			break
		}
	}
	if chosen.disambiguation() {
		var buttons strings.Builder
		count := 0
		for _, page := range pages {
			if page.disambiguation() || count == 4 {
				continue
			}
			label := page.Title
			if page.Description != "" && !strings.Contains(page.Title, "(") {
				label += " – " + page.Description
			}
			fmt.Fprintf(&buttons, "<button><text>%s</text><postback>%s</postback></button>",
				html.EscapeString(label), html.EscapeString("WIKIPEDIA PAGE "+page.Title))
			count++
		}
		if count > 0 {
			return sraixResponse{text: "SEVERAL", set: map[string]string{
				"wiki_status": "several", "wiki_title": chosen.Title, "wiki_url": chosen.FullURL,
				"wiki_description": chosen.Description, "wiki_options": buttons.String(),
			}}
		}
	}
	summary := summarizeWikiExtract(chosen.Extract)
	if summary == "" {
		return wikiNone(query)
	}
	return sraixResponse{text: summary, set: map[string]string{
		"wiki_status": "found", "wiki_title": chosen.Title, "wiki_url": chosen.FullURL,
		"wiki_description": chosen.Description, "wiki_options": "", "wiki_summary": summary,
	}}
}

func wikiNone(query string) sraixResponse {
	return sraixResponse{text: "NONE", set: map[string]string{
		"wiki_status": "none", "wiki_title": "", "wiki_url": "", "wiki_description": "", "wiki_options": "",
	}}
}

var (
	// A parenthesis holding a pronunciation: what is left of a template,
	// which starts with a space ("( KWAY-zar)", "(  EYE-fəl; French: ...)"),
	// or one with IPA in brackets or slashes
	wikiPronunciationRe = regexp.MustCompile(`\s*\((?:\s|[^()]*[\[/][^()]*)[^()]*\)`)
	wikiSpacesRe        = regexp.MustCompile(`\s+`)
	wikiSpaceBeforeRe   = regexp.MustCompile(`\s+([,.;:)])`)
	// Words a period follows without ending a sentence
	wikiAbbreviations = map[string]bool{"st": true, "dr": true, "mr": true, "mrs": true, "ms": true, "jr": true, "sr": true,
		"vs": true, "no": true, "ca": true, "c": true, "e.g": true, "i.e": true, "etc": true, "mt": true, "ft": true, "approx": true, "inc": true, "ltd": true, "co": true}
)

// summarizeWikiExtract returns the first sentences of an introduction, up
// to two and about wikipediaMaxSummary characters, without pronunciations
func summarizeWikiExtract(extract string) string {
	paragraph := strings.TrimSpace(strings.SplitN(strings.TrimSpace(extract), "\n", 2)[0])
	paragraph = wikiPronunciationRe.ReplaceAllString(paragraph, "")
	paragraph = wikiSpacesRe.ReplaceAllString(paragraph, " ")
	paragraph = strings.TrimSpace(wikiSpaceBeforeRe.ReplaceAllString(paragraph, "$1"))
	if paragraph == "" {
		return ""
	}
	sentences := splitWikiSentences(paragraph)
	summary := sentences[0]
	if len(sentences) > 1 && len(summary)+1+len(sentences[1]) <= wikipediaMaxSummary {
		summary += " " + sentences[1]
	}
	if len([]rune(summary)) > wikipediaMaxSummary {
		runes := []rune(summary)[:wikipediaMaxSummary]
		cut := strings.LastIndex(string(runes), " ")
		if cut < wikipediaMaxSummary/2 {
			cut = len(string(runes))
		}
		summary = strings.TrimRight(string(runes)[:cut], " ,;:") + "…"
	}
	return summary
}

// splitWikiSentences splits text after a period, question or exclamation
// mark that is followed by a space and a capital letter or digit, and that
// doesn't end an abbreviation or initial ("St.", "J. R. R.")
func splitWikiSentences(text string) []string {
	var sentences []string
	runes := []rune(text)
	start := 0
	for i := 0; i < len(runes)-2; i++ {
		r := runes[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if runes[i+1] != ' ' || !(unicode.IsUpper(runes[i+2]) || unicode.IsDigit(runes[i+2])) {
			continue
		}
		if r == '.' {
			wordStart := i
			for wordStart > start && runes[wordStart-1] != ' ' && runes[wordStart-1] != '(' {
				wordStart--
			}
			word := strings.ToLower(string(runes[wordStart:i]))
			if wikiAbbreviations[word] || (len([]rune(word)) == 1 && unicode.IsLetter(runes[wordStart])) {
				continue
			}
		}
		sentences = append(sentences, strings.TrimSpace(string(runes[start:i+1])))
		start = i + 2
	}
	sentences = append(sentences, strings.TrimSpace(string(runes[start:])))
	return sentences
}
