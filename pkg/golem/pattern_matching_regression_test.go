package golem

import (
	"fmt"
	"sync"
	"testing"
)

func loadPatternTestAIML(t *testing.T, categories string) *Golem {
	t.Helper()
	g := New(false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">` + categories + `</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	return g
}

// Wildcards must match whole words, with the AIML 2.0 widths: * and _
// match one or more words, ^ and # zero or more.
func TestWildcardsMatchWholeWords(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>HELLO *</pattern><template>hello [<star/>]</template></category>
<category><pattern>* LOVE</pattern><template>love [<star/>]</template></category>
<category><pattern>I AM * YEARS OLD</pattern><template>age [<star/>]</template></category>
<category><pattern>_ PLEASE</pattern><template>please [<star/>]</template></category>
<category><pattern>*</pattern><template>catchall</template></category>`)

	tests := map[string]string{
		"hello":             "catchall",
		"hello there":       "hello [there]",
		"hello big world":   "hello [big world]",
		"helloworld":        "catchall",
		"love":              "catchall",
		"i love":            "love [i]",
		"glove":             "catchall",
		"i glove":           "catchall",
		"i am 5 years old":  "age [5]",
		"i am years old":    "catchall",
		"i am5 years old":   "catchall",
		"door please":       "please [door]",
		"close door please": "please [close door]",
		"doorplease":        "catchall",
	}
	for input, want := range tests {
		response, err := g.ProcessInput(input, g.CreateSession(input))
		if err != nil {
			t.Errorf("%q: %v", input, err)
			continue
		}
		if response != want {
			t.Errorf("%q: expected %q, got %q", input, want, response)
		}
	}
}

// Matching goes word by word, so the pattern whose word matches earlier wins,
// whichever category was loaded first.
func TestEqualPriorityMatchesAreDeterministic(t *testing.T) {
	for _, tt := range []struct {
		categories string
		want       string
	}{
		{`<category><pattern>* LOVE *</pattern><template>love</template></category>
<category><pattern>* YOU *</pattern><template>you</template></category>`, "love"},
		{`<category><pattern>* YOU *</pattern><template>you</template></category>
<category><pattern>* LOVE *</pattern><template>love</template></category>`, "love"},
	} {
		for i := 0; i < 20; i++ {
			g := loadPatternTestAIML(t, tt.categories)
			response, err := g.ProcessInput("i love you too", g.CreateSession("test"))
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}
			if response != tt.want {
				t.Fatalf("Run %d: expected %q, got %q", i, tt.want, response)
			}
		}
	}
}

// Adding a set member must take effect even after the input was matched
// (and cached) before.
func TestSetChangeNotHiddenByCache(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>I LIKE <set>colors</set></pattern><template>color [<star/>]</template></category>
<category><pattern>*</pattern><template>catchall</template></category>`)
	g.aimlKB.AddSetMembers("colors", []string{"red"})
	session := g.CreateSession("test")

	if response, _ := g.ProcessInput("i like blue", session); response != "catchall" {
		t.Fatalf("Expected 'catchall' before blue is a color, got %q", response)
	}
	g.aimlKB.AddSetMember("colors", "blue")
	if response, _ := g.ProcessInput("i like blue", session); response != "color [BLUE]" && response != "color [blue]" {
		t.Errorf("Expected blue to match after adding it to the set, got %q", response)
	}
}

// Matching set patterns from concurrent sessions must not race on the
// pattern matching cache.
func TestConcurrentSetMatching(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>I LIKE <set>colors</set></pattern><template>color</template></category>
<category><pattern>*</pattern><template>catchall</template></category>`)
	g.aimlKB.AddSetMembers("colors", []string{"red", "blue", "green"})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := g.CreateSession(fmt.Sprintf("user%d", i))
			for j := 0; j < 20; j++ {
				if response, err := g.ProcessInput("i like red", session); err != nil || response != "color" {
					t.Errorf("Expected 'color', got %q, %v", response, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

// A literal word must outweigh a wildcard, so patterns with more words win
// over a catch-all and over shorter patterns.
func TestLiteralWordsOutweighWildcards(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>*</pattern><template>catchall</template></category>
<category><pattern>TEACH *</pattern><template>teach</template></category>
<category><pattern>TEACH * MEANS *</pattern><template>means [<star/>] [<star index="2"/>]</template></category>
<category><pattern>YES</pattern><that>DO *</that><template>any yes</template></category>
<category><pattern>YES</pattern><that>DO * LIKE *</that><template>like yes</template></category>
<category><pattern>ASK</pattern><template>Do you like cats</template></category>`)

	tests := map[string]string{
		"teach foo means bar": "means [foo] [bar]",
		"teach foo":           "teach",
		"hello":               "catchall",
	}
	for input, want := range tests {
		response, err := g.ProcessInput(input, g.CreateSession(input))
		if err != nil {
			t.Errorf("%q: %v", input, err)
			continue
		}
		if response != want {
			t.Errorf("%q: expected %q, got %q", input, want, response)
		}
	}

	session := g.CreateSession("that")
	g.ProcessInput("ask", session)
	if response, _ := g.ProcessInput("yes", session); response != "like yes" {
		t.Errorf("Expected <that>DO * LIKE *</that> to beat <that>DO *</that>, got %q", response)
	}
}

// <that> is matched against the last sentence of the bot's processed
// response, from its first word, and may start with a wildcard.
func TestThatMatchesLastSentenceOfResponse(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>ASK</pattern><template>Hello! <srai>QUESTION</srai></template></category>
<category><pattern>QUESTION</pattern><template>Do you like cats?</template></category>
<category><pattern>TELL</pattern><template>I really like cats.</template></category>
<category><pattern>YES</pattern><that>* LIKE CATS</that><template>leading wildcard</template></category>
<category><pattern>NO</pattern><that>LIKE CATS</that><template>unanchored</template></category>
<category><pattern>NO</pattern><template>generic no</template></category>
<category><pattern>MAYBE</pattern><that>^ LIKE CATS</that><template>caret</template></category>
<category><pattern>MAYBE</pattern><that>DO YOU LIKE CATS</that><template>exact</template></category>`)

	for _, tt := range []struct {
		inputs []string
		want   string
	}{
		{[]string{"ask", "yes"}, "leading wildcard"},
		{[]string{"tell", "yes"}, "leading wildcard"},
		{[]string{"ask", "no"}, "generic no"},
		{[]string{"tell", "no"}, "generic no"},
		{[]string{"ask", "maybe"}, "exact"},
		{[]string{"tell", "maybe"}, "caret"},
	} {
		session := g.CreateSession(tt.inputs[0] + tt.inputs[1])
		var response string
		for _, input := range tt.inputs {
			var err error
			if response, err = g.ProcessInput(input, session); err != nil {
				t.Fatalf("%q: %v", input, err)
			}
		}
		if response != tt.want {
			t.Errorf("%v: expected %q, got %q", tt.inputs, tt.want, response)
		}
	}
}
