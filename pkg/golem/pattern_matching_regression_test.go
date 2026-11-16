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

// Wildcards must match whole words, while keeping their widths: *, ^ and #
// match zero or more words, _ matches one.
func TestWildcardsMatchWholeWords(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>HELLO *</pattern><template>hello [<star/>]</template></category>
<category><pattern>* LOVE</pattern><template>love [<star/>]</template></category>
<category><pattern>I AM * YEARS OLD</pattern><template>age [<star/>]</template></category>
<category><pattern>_ PLEASE</pattern><template>please [<star/>]</template></category>
<category><pattern>*</pattern><template>catchall</template></category>`)

	tests := map[string]string{
		"hello":             "hello []",
		"hello there":       "hello [there]",
		"hello big world":   "hello [big world]",
		"helloworld":        "catchall",
		"love":              "love []",
		"i love":            "love [i]",
		"glove":             "catchall",
		"i glove":           "catchall",
		"i am 5 years old":  "age [5]",
		"i am years old":    "age []",
		"i am5 years old":   "catchall",
		"door please":       "please [door]",
		"close door please": "catchall",
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

// Equally good matches must be chosen the same way every time: by load order.
func TestEqualPriorityMatchesAreDeterministic(t *testing.T) {
	for _, tt := range []struct {
		categories string
		want       string
	}{
		{`<category><pattern>* LOVE *</pattern><template>love</template></category>
<category><pattern>* YOU *</pattern><template>you</template></category>`, "love"},
		{`<category><pattern>* YOU *</pattern><template>you</template></category>
<category><pattern>* LOVE *</pattern><template>love</template></category>`, "you"},
	} {
		for i := 0; i < 20; i++ {
			g := loadPatternTestAIML(t, tt.categories)
			response, err := g.ProcessInput("i love you too", g.CreateSession("test"))
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}
			if response != tt.want {
				t.Fatalf("Run %d: expected the first loaded category %q, got %q", i, tt.want, response)
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
