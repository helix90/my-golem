package golem

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseRichResponse(t *testing.T) {
	got := ParseRichResponse(`Should I delete "Dentist"?
		<button><text>Yes, delete it</text><postback>CONFIRM DELETE EVENT Dentist</postback></button>
		<button><text>Map</text><url>https://example.com/?a=1&amp;b=2</url></button>
		<button>Maybe</button>
		<reply><text>📍 Share my location</text><location/></reply>
		<reply><text>Today</text></reply>`)
	want := RichResponse{
		Text: `Should I delete "Dentist"?`,
		Buttons: []RichButton{
			{Text: "Yes, delete it", Postback: "CONFIRM DELETE EVENT Dentist"},
			{Text: "Map", URL: "https://example.com/?a=1&b=2"},
			{Text: "Maybe", Postback: "Maybe"},
		},
		Replies: []RichButton{
			{Text: "📍 Share my location", RequestLocation: true},
			{Text: "Today", Postback: "Today"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	if plain := ParseRichResponse("No media here."); plain.HasMedia() || plain.Text != "No media here." {
		t.Errorf("plain text: %+v", plain)
	}
	if RichText("A <button><text>B</text></button> C") != "A C" {
		t.Errorf("RichText = %q", RichText("A <button><text>B</text></button> C"))
	}
}

func TestDeleteEventAsksFirst(t *testing.T) {
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	s := telegramSession(g, "111")

	r := ParseRichResponse(ask(t, g, s, "delete event Dentist"))
	if r.Text != `Should I delete the event "Dentist"?` || len(r.Buttons) != 2 || r.Buttons[0].Postback != "CONFIRM DELETE EVENT Dentist" {
		t.Fatalf("question: %+v", r)
	}
	// Typing "yes" answers the question: with no calendar server here, the
	// delete is attempted and fails
	if got := ask(t, g, s, "yes"); !strings.Contains(got, `couldn't delete that event`) || !strings.Contains(got, "Dentist") {
		t.Errorf("yes: %q", got)
	}

	ask(t, g, s, "delete event Dentist")
	if got := ask(t, g, s, "no"); !strings.Contains(got, "leave it on your calendar") {
		t.Errorf("no: %q", got)
	}
	// The button's postback works as input on its own
	if got := ask(t, g, s, r.Buttons[0].Postback); !strings.Contains(got, `couldn't delete that event`) {
		t.Errorf("postback: %q", got)
	}
}

func TestListShowsTickButtons(t *testing.T) {
	g := NewForTesting(t, false)
	out := g.ProcessTemplate(`<jsonformat button="LIST {list} TOGGLE ITEM {id}">{"id":3,"name":"Groceries","description":"","items":[{"id":12,"content":"Milk & eggs","is_completed":0},{"id":13,"content":"Bread","is_completed":1}]}</jsonformat>`, nil)
	r := ParseRichResponse(out)
	if r.Text != "Your Groceries list has 2 items:\n☐ Milk & eggs\n☑ Bread" {
		t.Errorf("text: %q", r.Text)
	}
	want := []RichButton{
		{Text: "☐ Milk & eggs", Postback: "LIST Groceries TOGGLE ITEM 12"},
		{Text: "☑ Bread", Postback: "LIST Groceries TOGGLE ITEM 13"},
	}
	if !reflect.DeepEqual(r.Buttons, want) {
		t.Errorf("buttons: %+v", r.Buttons)
	}
	// Without the attribute the output has no buttons
	if plain := ParseRichResponse(g.ProcessTemplate(`<jsonformat>{"id":3,"name":"G","items":[{"id":1,"content":"x","is_completed":1}]}</jsonformat>`, nil)); plain.HasMedia() || !strings.Contains(plain.Text, "☑ x") {
		t.Errorf("plain list: %+v", plain)
	}
}

func TestWeatherOffersLocationButton(t *testing.T) {
	g := newTestdataBot(t, filepath.Join(t.TempDir(), "users.json"))
	r := ParseRichResponse(ask(t, g, telegramSession(g, "111"), "what is the weather"))
	if len(r.Replies) != 1 || !r.Replies[0].RequestLocation {
		t.Errorf("replies: %+v", r.Replies)
	}
}
