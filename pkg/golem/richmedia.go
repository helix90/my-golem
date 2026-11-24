package golem

import (
	"html"
	"regexp"
	"strings"
)

// Rich media: a template can offer the user choices with the rich media
// elements of AIML 2.1 (as Pandorabots defines them). The engine passes
// them through in its response; a client such as the Telegram bot turns
// them into buttons with ParseRichResponse.
//
//	<button><text>Yes</text><postback>CONFIRM DELETE EVENT DENTIST</postback></button>
//	<button><text>Open map</text><url>https://example.com/map</url></button>
//	<reply><text>Today</text><postback>WHAT IS ON MY CALENDAR TODAY</postback></reply>
//	<reply><text>Share my location</text><location/></reply>
//
// A <button> is attached to the message (an inline button); a <reply> is a
// suggested reply the user can tap instead of typing. Tapping either sends
// its postback (or, without one, its text) as the user's next input.
// <location/> in a reply asks the client for the user's location instead.
// <that> sees only the response's text.

// RichButton is one <button> or <reply>
type RichButton struct {
	Text     string
	Postback string
	URL      string
	// RequestLocation asks the client to share the user's location
	RequestLocation bool
}

// RichResponse is a response split into its text and its rich media
type RichResponse struct {
	Text    string
	Buttons []RichButton // inline buttons, from <button>
	Replies []RichButton // suggested replies, from <reply>
}

// HasMedia reports whether the response has buttons or replies
func (r RichResponse) HasMedia() bool {
	return len(r.Buttons) > 0 || len(r.Replies) > 0
}

var (
	richElementRe  = regexp.MustCompile(`(?is)<(button|reply)>(.*?)</(?:button|reply)>`)
	richTextRe     = regexp.MustCompile(`(?is)<text>(.*?)</text>`)
	richPostbackRe = regexp.MustCompile(`(?is)<postback>(.*?)</postback>`)
	richURLRe      = regexp.MustCompile(`(?is)<url>(.*?)</url>`)
	richLocationRe = regexp.MustCompile(`(?is)<location\s*/>|<location>\s*</location>`)
	richSpaceRe    = regexp.MustCompile(`[ \t]+`)
	richBlankRe    = regexp.MustCompile(`\n[ \t]*\n[\s]*`)
)

// ParseRichResponse splits a response into its text and its buttons and
// replies, in order. An element with no text is dropped.
func ParseRichResponse(response string) RichResponse {
	var result RichResponse
	text := richElementRe.ReplaceAllStringFunc(response, func(element string) string {
		m := richElementRe.FindStringSubmatch(element)
		inner := m[2]
		button := RichButton{
			Text:            richPart(richTextRe, inner),
			Postback:        richPart(richPostbackRe, inner),
			URL:             richPart(richURLRe, inner),
			RequestLocation: richLocationRe.MatchString(inner),
		}
		if button.Text == "" {
			// <button>Yes</button>, with no <text>, is its own text
			button.Text = strings.TrimSpace(html.UnescapeString(richLocationRe.ReplaceAllString(inner, "")))
			if strings.Contains(button.Text, "<") {
				return ""
			}
		}
		if button.Text == "" {
			return ""
		}
		if button.Postback == "" && button.URL == "" && !button.RequestLocation {
			button.Postback = button.Text
		}
		if strings.EqualFold(m[1], "reply") {
			result.Replies = append(result.Replies, button)
		} else {
			result.Buttons = append(result.Buttons, button)
		}
		return " "
	})
	text = richSpaceRe.ReplaceAllString(text, " ")
	text = richBlankRe.ReplaceAllString(text, "\n\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	result.Text = strings.TrimSpace(strings.Join(lines, "\n"))
	return result
}

// RichText returns a response's text without its rich media
func RichText(response string) string {
	if !strings.Contains(response, "<") {
		return response
	}
	return ParseRichResponse(response).Text
}

func richPart(re *regexp.Regexp, inner string) string {
	m := re.FindStringSubmatch(inner)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
}
