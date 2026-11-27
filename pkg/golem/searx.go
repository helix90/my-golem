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
	"strings"
	"time"
)

// The searx SRAIX handler asks a SearXNG (or searx) instance, through its
// JSON API: the service's URL template is the instance's search page, e.g.
// http://searx:8080/search, and the instance must allow the json format
// (search: formats: [html, json] in its settings.yml).
//
// Operations:
//
//	answer - for a sum or a conversion ("2 + 2", "7 times 6", "15% of 80",
//	         "5 miles in km", "100 usd in eur"): the instant answer of
//	         SearXNG's calculator, unit converter or currency engines.
//	         Anything else isn't sent, so questions such as "what is a
//	         quasar" don't go to the search engines. Sets searx_status to
//	         "found", "none" (no answer) or "skip" (not a sum), and
//	         searx_answer to the reply, e.g. "2 + 2 is 4."
//	search - a web search: the first results, setting searx_status,
//	         searx_answer (the top result's title and text) and
//	         searx_links (a <button> linking to each of the top three)
//
// A failed request is an error, so <sraix> gives the service's fallback.
var searxOperations = map[string]bool{"answer": true, "search": true}

var (
	searxNumberRe     = regexp.MustCompile(`\d`)
	searxPercentOfRe  = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:%|percent|per cent)\s+of\s+`)
	searxConversionRe = regexp.MustCompile(`(?i)^\s*-?\d[\d.,]*\s*[a-z°$€£¥]+[a-z0-9 ]*\s+(?:in|to|into)\s+[a-z°$€£¥]+[a-z0-9 ]*\s*$`)
	searxSumRe        = regexp.MustCompile(`^[0-9\s+\-*/^().%]+$`)
	searxOperatorRe   = regexp.MustCompile(`[+\-*/^%]`)
	searxThousandsRe  = regexp.MustCompile(`(\d),(\d{3})`)
	searxWords        = []struct {
		re   *regexp.Regexp
		with string
	}{
		{regexp.MustCompile(`(?i)\bmultiplied by\b`), "*"},
		{regexp.MustCompile(`(?i)\bdivided by\b`), "/"},
		{regexp.MustCompile(`(?i)\bto the power of\b`), "^"},
		{regexp.MustCompile(`(?i)\bplus\b`), "+"},
		{regexp.MustCompile(`(?i)\bminus\b`), "-"},
		{regexp.MustCompile(`(?i)\btimes\b`), "*"},
		{regexp.MustCompile(`(?i)(\d)\s*x\s*(\d)`), "$1 * $2"},
		{regexp.MustCompile(`(?i)\bover\b`), "/"},
		{regexp.MustCompile(`(?i)\bsquared\b`), "^2"},
		{regexp.MustCompile(`(?i)\bcubed\b`), "^3"},
		{regexp.MustCompile(`(?i)\bpercent\b|\bper cent\b`), "%"},
	}
)

// searxQuery returns what to ask SearXNG for a question, and whether it is
// a sum or a conversion at all: words become operators ("7 times 6" is
// "7 * 6") and "15% of 80" becomes "15/100 * 80"
func searxQuery(question string) (string, bool) {
	q := strings.TrimSpace(strings.Trim(strings.TrimSpace(question), "?!. "))
	q = strings.TrimSpace(regexp.MustCompile(`(?i)^(?:what is|what's|how much is|calculate|compute|convert)\s+`).ReplaceAllString(q, ""))
	if !searxNumberRe.MatchString(q) {
		return "", false
	}
	if searxConversionRe.MatchString(q) && !searxSumRe.MatchString(q) {
		return q, true
	}
	sum := searxThousandsRe.ReplaceAllString(q, "$1$2")
	sum = searxPercentOfRe.ReplaceAllString(sum, "$1/100 * ")
	for _, w := range searxWords {
		sum = w.re.ReplaceAllString(sum, w.with)
	}
	sum = strings.Join(strings.Fields(sum), " ")
	if searxSumRe.MatchString(sum) && searxOperatorRe.MatchString(sum) {
		return sum, true
	}
	return "", false
}

// searxAnswerText reads the answers of a SearXNG response, which are
// strings in older versions and objects with an "answer" in newer ones
func searxAnswerText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var obj map[string]interface{}
	if json.Unmarshal(raw, &obj) == nil {
		for _, key := range []string{"answer", "content", "title"} {
			if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

type searxResponse struct {
	Answers   []json.RawMessage `json:"answers"`
	Results   []searxResult     `json:"results"`
	Infoboxes []struct {
		Infobox string `json:"infobox"`
		Content string `json:"content"`
	} `json:"infoboxes"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func (sm *SRAIXManager) sendSearx(config *SRAIXConfig, input string, params map[string]string) (sraixResponse, error) {
	question := strings.TrimSpace(input)
	query := question
	if config.Operation == "answer" {
		var ok bool
		if query, ok = searxQuery(question); !ok {
			return sraixResponse{text: "SKIP", set: map[string]string{"searx_status": "skip", "searx_answer": ""}}, nil
		}
	}
	if query == "" {
		return sraixResponse{text: "NONE", set: map[string]string{"searx_status": "none", "searx_answer": "", "searx_links": ""}}, nil
	}

	searchURL, _, err := sm.substituteURLTemplate(config.URLTemplate, "", params, config.Headers)
	if err != nil {
		return sraixResponse{}, err
	}
	values := url.Values{"q": {query}, "format": {"json"}}
	if strings.Contains(searchURL, "?") {
		searchURL += "&" + values.Encode()
	} else {
		searchURL += "?" + values.Encode()
	}
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return sraixResponse{}, err
	}
	if err := sm.setHeaders(req, config, "", params); err != nil {
		return sraixResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := sm.client.Do(req)
	if err != nil {
		return sraixResponse{}, fmt.Errorf("searx request failed: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSRAIXResponseSize+1))
	if err != nil {
		return sraixResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		// 403 usually means the instance doesn't allow format=json
		return sraixResponse{}, fmt.Errorf("searx replied %s", resp.Status)
	}
	var result searxResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return sraixResponse{}, fmt.Errorf("searx replied with something other than JSON (%d bytes); is the json format enabled?", len(data))
	}
	if sm.verbose {
		sm.logger.Printf("Searx %s %q: %d answers, %d results", config.Operation, query, len(result.Answers), len(result.Results))
	}

	if config.Operation == "answer" {
		for _, raw := range result.Answers {
			if answer := searxAnswerText(raw); answer != "" {
				reply := searxAnswerReply(question, answer)
				return sraixResponse{text: reply, set: map[string]string{"searx_status": "found", "searx_answer": reply}}, nil
			}
		}
		return sraixResponse{text: "NONE", set: map[string]string{"searx_status": "none", "searx_answer": ""}}, nil
	}

	// search
	var top string
	if len(result.Infoboxes) > 0 && strings.TrimSpace(result.Infoboxes[0].Content) != "" {
		top = summarizeWikiExtract(result.Infoboxes[0].Content)
	}
	var links strings.Builder
	shown := 0
	for _, r := range result.Results {
		if r.URL == "" || shown == 3 {
			continue
		}
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = r.URL
		}
		if top == "" {
			top = title
			if content := strings.TrimSpace(r.Content); content != "" {
				top += ": " + summarizeWikiExtract(content)
			}
		}
		if len([]rune(title)) > 60 {
			title = string([]rune(title)[:59]) + "…"
		}
		fmt.Fprintf(&links, "<button><text>%s</text><url>%s</url></button>", html.EscapeString(title), html.EscapeString(r.URL))
		shown++
	}
	if top == "" {
		return sraixResponse{text: "NONE", set: map[string]string{"searx_status": "none", "searx_answer": "", "searx_links": ""}}, nil
	}
	return sraixResponse{text: top, set: map[string]string{"searx_status": "found", "searx_answer": top, "searx_links": links.String()}}, nil
}

// searxAnswerReply phrases an instant answer: SearXNG's calculator answers
// "2+2 = 4", which reads "2 + 2 is 4." with the question as asked
func searxAnswerReply(question, answer string) string {
	asked := strings.TrimSpace(strings.Trim(strings.TrimSpace(question), "?!. "))
	asked = strings.TrimSpace(regexp.MustCompile(`(?i)^(?:what is|what's|how much is|calculate|compute|convert)\s+`).ReplaceAllString(asked, ""))
	if i := strings.LastIndex(answer, " = "); i >= 0 {
		answer = strings.TrimSpace(answer[i+3:])
	}
	answer = strings.TrimSuffix(answer, ".")
	if asked == "" {
		return answer + "."
	}
	return asked + " is " + answer + "."
}
