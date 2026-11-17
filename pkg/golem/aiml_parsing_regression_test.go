package golem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

const thatCollisionAIML = `<aiml version="2.0">
<category><pattern>DO YOU LIKE IT</pattern><template>Do you like cats?</template></category>
<category><pattern>YES</pattern><that>DO YOU LIKE CATS</that><template>cats yes</template></category>
<category><pattern>YES</pattern><template>generic yes</template></category>
</aiml>`

// Categories sharing a pattern but differing by <that> must all be indexed,
// regardless of which loader is used.
func TestLoadersKeepCategoriesSharingPattern(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "that.aiml")
	if err := os.WriteFile(file, []byte(thatCollisionAIML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loaders := map[string]func(g *Golem) error{
		"LoadAIMLFromString": func(g *Golem) error {
			return g.LoadAIMLFromString(thatCollisionAIML)
		},
		"LoadAIML": func(g *Golem) error {
			kb, err := g.LoadAIML(file)
			if err == nil {
				g.SetKnowledgeBase(kb)
			}
			return err
		},
		"LoadAIMLFromDirectory": func(g *Golem) error {
			kb, err := g.LoadAIMLFromDirectory(dir)
			if err == nil {
				g.SetKnowledgeBase(kb)
			}
			return err
		},
	}

	for name, load := range loaders {
		t.Run(name, func(t *testing.T) {
			g := New(false)
			if err := load(g); err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}

			if got := len(g.GetKnowledgeBase().Patterns); got != 3 {
				t.Errorf("Expected 3 indexed patterns, got %d", got)
			}

			session := g.CreateSession("test")
			if _, err := g.ProcessInput("do you like it", session); err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}
			response, err := g.ProcessInput("yes", session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}
			if response != "cats yes" {
				t.Errorf("Expected 'cats yes', got '%s'", response)
			}
		})
	}
}

// Categories inside comments must not be loaded, even when the comment spans
// multiple lines.
func TestCommentedOutCategoriesNotLoaded(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<!--
<category><pattern>HIDDEN</pattern><template>multi-line comment</template></category>
-->
<!-- <category><pattern>HIDDEN TOO</pattern><template>single-line comment</template></category> -->
<category><pattern>HELLO</pattern><template>real</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	categories := g.GetKnowledgeBase().Categories
	if len(categories) != 1 {
		t.Fatalf("Expected 1 category, got %d: %+v", len(categories), categories)
	}
	if categories[0].Pattern != "HELLO" {
		t.Errorf("Expected pattern 'HELLO', got '%s'", categories[0].Pattern)
	}
}

// Malformed templates must never panic the AST parser.
func TestASTParserUnterminatedAttributeNoPanic(t *testing.T) {
	inputs := []string{
		`<get name="`,
		`<get name="x`,
		`<get name='`,
		`<get name=\"`,
		`<get name=\"x`,
		`<star index=`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parser panicked on %q: %v", input, r)
				}
			}()
			if _, err := NewASTParser(input).Parse(); err != nil {
				t.Logf("Parse returned error: %v", err)
			}
		})
	}
}

func FuzzASTParser(f *testing.F) {
	seeds := []string{
		`<get name="x"/>`,
		`<uppercase><star/> and <get name="y"/></uppercase>`,
		`<condition name="a"><li value="1">one</li><li>other</li></condition>`,
		`<!-- c --><![CDATA[<raw>]]>text`,
		`<get name=\"x\"/>`,
		`<get name="`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		root, _ := NewASTParser(input).Parse()
		_ = root.String()
	})
}

// Pattern validation is case-insensitive and accepts apostrophes, common
// punctuation, and non-ASCII letters. An invalid category is skipped without
// discarding the rest of the file.
func TestDirectoryLoadValidation(t *testing.T) {
	dir := t.TempDir()
	content := `<aiml version="2.0">
<category><pattern>hello there</pattern><template>lowercase</template></category>
<category><pattern>LET'S TALK ABOUT *</pattern><template>apostrophe</template></category>
<category><pattern>CAFÉ</pattern><template>accented</template></category>
<category><pattern>MAIL ME <set></set></pattern><template>invalid</template></category>
</aiml>`
	if err := os.WriteFile(filepath.Join(dir, "mixed.aiml"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	g := New(false)
	kb, err := g.LoadAIMLFromDirectory(dir)
	if err != nil {
		t.Fatalf("Failed to load directory: %v", err)
	}
	g.SetKnowledgeBase(kb)

	if got := len(kb.Categories); got != 3 {
		t.Fatalf("Expected 3 valid categories, got %d: %+v", got, kb.Categories)
	}

	session := g.CreateSession("test")
	for input, want := range map[string]string{
		"hello there":           "lowercase",
		"let's talk about cats": "apostrophe",
		"café":                  "accented",
	} {
		response, err := g.ProcessInput(input, session)
		if err != nil {
			t.Errorf("ProcessInput(%q) failed: %v", input, err)
			continue
		}
		if response != want {
			t.Errorf("ProcessInput(%q): expected '%s', got '%s'", input, want, response)
		}
	}
}

// LoadAIMLFromString validates categories like the file loaders do.
func TestLoadAIMLFromStringValidates(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern></pattern><template>empty pattern</template></category>
</aiml>`)
	if err == nil {
		t.Fatal("Expected error when the only category has an empty pattern")
	}

	err = g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>MAIL ME <set></set></pattern><template>invalid</template></category>
<category><pattern>HELLO</pattern><template>valid</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	categories := g.GetKnowledgeBase().Categories
	if len(categories) != 1 || categories[0].Pattern != "HELLO" {
		t.Errorf("Expected only the HELLO category, got %+v", categories)
	}
}

// A <that> or <topic> inside the template (here, in a category being learned)
// must not be attributed to the outer category.
func TestTemplateTagsNotAttributedToCategory(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>TEACH</pattern><template>ok<learn><category><pattern>X</pattern><that>FOO</that><topic>BAR</topic><template>y</template></category></learn></template></category>
<category><pattern>WHAT TOPIC</pattern><template><thatstar/> <topic/></template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	for _, category := range g.GetKnowledgeBase().Categories {
		if category.That != "" || category.Topic != "" {
			t.Errorf("Category %q: expected no that/topic, got that=%q topic=%q", category.Pattern, category.That, category.Topic)
		}
	}

	response, err := g.ProcessInput("teach", g.CreateSession("test"))
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "ok" {
		t.Errorf("Expected 'ok', got '%s'", response)
	}
}

// Categories inside <topic name="..."> blocks get that topic.
func TestTopicBlocks(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>SET *</pattern><template><think><set name="topic"><star/></set></think>ok</template></category>
<topic name="sports">
	<category><pattern>FAVORITE</pattern><template>I like basketball!</template></category>
</topic>
<topic name='music'>
	<category><pattern>FAVORITE</pattern><template>I like jazz!</template></category>
</topic>
<category><pattern>FAVORITE</pattern><template>I like many things.</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test")
	steps := []struct{ input, want string }{
		{"favorite", "I like many things."},
		{"set sports", "ok"},
		{"favorite", "I like basketball!"},
		{"set music", "ok"},
		{"favorite", "I like jazz!"},
	}
	for _, step := range steps {
		response, err := g.ProcessInput(step.input, session)
		if err != nil {
			t.Fatalf("ProcessInput(%q) failed: %v", step.input, err)
		}
		if response != step.want {
			t.Errorf("ProcessInput(%q): expected '%s', got '%s'", step.input, step.want, response)
		}
	}
}

// Attributes may be single-quoted.
func TestSingleQuotedAttributes(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version='2.0'>
<category><pattern>YES</pattern><that index='2'>QUESTION</that><template>t</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	categories := g.GetKnowledgeBase().Categories
	if len(categories) != 1 || categories[0].ThatIndex != 2 {
		t.Errorf("Expected one category with that index 2, got %+v", categories)
	}
}

func FuzzParseAIML(f *testing.F) {
	seeds := []string{
		thatCollisionAIML,
		`<aiml><topic name="a"><category><pattern>X</pattern><template><topic/></template></category></topic></aiml>`,
		`<aiml><category><pattern>X</pattern><that index='2'>Y</that><template><learn><category><pattern>Z</pattern><template>z</template></category></learn></template></category></aiml>`,
		`<category><pattern>A</pattern><template x="<">`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	g := New(false)
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = g.parseAIML(input)
	})
}

// Entities in templates are decoded in the output, including in learned
// categories and attribute values.
func TestTemplateEntitiesDecoded(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>AMP</pattern><template>A &amp; B &lt;3 &quot;hi&quot; &#233;</template></category>
<category><pattern>ATTR</pattern><template><think><set name="x">Q&amp;A</set></think><condition name="x" value="Q&amp;A">matched</condition></template></category>
<category><pattern>TEACH</pattern><template><learn><category><pattern>LEARNED</pattern><template>C &amp; D</template></category></learn>ok</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test")
	steps := []struct{ input, want string }{
		{"amp", `A & B <3 "hi" é`},
		{"attr", "matched"},
		{"teach", "ok"},
		{"learned", "C & D"},
	}
	for _, step := range steps {
		response, err := g.ProcessInput(step.input, session)
		if err != nil {
			t.Fatalf("ProcessInput(%q) failed: %v", step.input, err)
		}
		if response != step.want {
			t.Errorf("ProcessInput(%q): expected '%s', got '%s'", step.input, step.want, response)
		}
	}
}

// Re-serialized attributes are escaped and in a stable order.
func TestASTNodeStringAttributes(t *testing.T) {
	root, err := NewASTParser(`<li name="b" value="x &amp; &quot;y&quot;" a="1">z</li>`).Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	want := `<li a="1" name="b" value="x &amp; &quot;y&quot;">z</li>`
	for i := 0; i < 10; i++ {
		if got := root.String(); got != want {
			t.Fatalf("Expected %s, got %s", want, got)
		}
	}
}

// Concurrent sessions must not see each other's template context, and
// template tags that modify shared knowledge base data must not race.
func TestConcurrentSessionsIsolated(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>MY NAME IS *</pattern><template><think><set name="name"><star/></set><list name="seen" operation="add"><star/></list><map name="names" key="<star/>" operation="set"><star/></map><array name="ids" operation="add"><star/></array></think>Hi <get name="name"/></template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	const sessions = 50
	var wg sync.WaitGroup
	errs := make(chan string, sessions)
	for i := 0; i < sessions; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("user%d", i)
			session := g.CreateSession(name)
			for j := 0; j < 20; j++ {
				response, err := g.ProcessInput("my name is "+name, session)
				if err != nil {
					errs <- fmt.Sprintf("%s: %v", name, err)
					return
				}
				if response != "Hi "+name || session.Variables["name"] != name {
					errs <- fmt.Sprintf("%s: got response %q, name %q", name, response, session.Variables["name"])
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

// Removing a category must not shift other patterns onto the wrong category.
func TestUnlearnKeepsOtherCategories(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>FORGET A</pattern><template><unlearn><category><pattern>AAA</pattern><template>a</template></category></unlearn>forgot</template></category>
<category><pattern>AAA</pattern><template>a</template></category>
<category><pattern>BBB</pattern><template>b</template></category>
<category><pattern>CCC</pattern><template>c</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test")
	if _, err := g.ProcessInput("forget a", session); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response, err := g.ProcessInput("aaa", session); err == nil {
		t.Errorf("Expected AAA to be unlearned, got '%s'", response)
	}
	for input, want := range map[string]string{"bbb": "b", "ccc": "c"} {
		response, err := g.ProcessInput(input, session)
		if err != nil || response != want {
			t.Errorf("ProcessInput(%q): expected '%s', got '%s' (err=%v)", input, want, response, err)
		}
	}
}

// ClearSessionLearning removes only the categories the session learned.
func TestClearSessionLearningRemovesOnlyLearned(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>ASK</pattern><template>Do you like cats?</template></category>
<category><pattern>YES</pattern><that>DO YOU LIKE CATS</that><template>cats yes</template></category>
<category><pattern>TEACH</pattern><template><learn><category><pattern>YES</pattern><template>learned yes</template></category></learn>ok</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test")
	if _, err := g.ProcessInput("teach", session); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if err := g.ClearSessionLearning("test"); err != nil {
		t.Fatalf("ClearSessionLearning failed: %v", err)
	}

	var patterns []string
	for _, category := range g.GetKnowledgeBase().Categories {
		patterns = append(patterns, categoryKey(category))
	}
	if len(patterns) != 3 || patterns[1] != "YES|THAT:DO YOU LIKE CATS" {
		t.Errorf("Expected the original 3 categories to remain, got %v", patterns)
	}

	if response, err := g.ProcessInput("yes", session); err == nil && response == "learned yes" {
		t.Errorf("Expected learned YES to be removed, got '%s'", response)
	}
	if _, err := g.ProcessInput("ask", session); err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	response, err := g.ProcessInput("yes", session)
	if err != nil || response != "cats yes" {
		t.Errorf("Expected file category 'cats yes' to remain, got '%s' (err=%v)", response, err)
	}
}

// assertIndexPointsIntoCategories checks that every Patterns entry points at
// an element of kb.Categories, so updates through the index stay consistent.
func assertIndexPointsIntoCategories(t *testing.T, kb *AIMLKnowledgeBase) {
	t.Helper()
	inCategories := make(map[*Category]bool, len(kb.Categories))
	for i := range kb.Categories {
		inCategories[&kb.Categories[i]] = true
	}
	for key, category := range kb.Patterns {
		if !inCategories[category] {
			t.Errorf("Patterns[%q] does not point into Categories", key)
		}
	}
}

func TestPatternIndexPointsIntoCategories(t *testing.T) {
	dir := t.TempDir()
	for i, content := range []string{thatCollisionAIML, `<aiml version="2.0"><category><pattern>OTHER</pattern><template>o</template></category></aiml>`} {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.aiml", i)), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	g := New(false)
	kb, err := g.LoadAIMLFromDirectory(dir)
	if err != nil {
		t.Fatalf("Failed to load directory: %v", err)
	}
	assertIndexPointsIntoCategories(t, kb)

	g.SetKnowledgeBase(kb)
	if err := g.LoadAIMLFromString(`<aiml version="2.0"><category><pattern>MORE</pattern><template>m</template></category></aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	assertIndexPointsIntoCategories(t, g.GetKnowledgeBase())
}

// Learning and unlearning in some sessions must not race with matching in
// others, and must never make other sessions see the wrong category.
func TestConcurrentLearnAndMatch(t *testing.T) {
	g := New(false)
	err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>HELLO</pattern><template>hi</template></category>
<category><pattern>TEACH *</pattern><template><learn><category><pattern><eval><star/></eval></pattern><template>learned</template></category></learn>ok</template></category>
<category><pattern>FORGET *</pattern><template><unlearn><category><pattern><eval><star/></eval></pattern><template>learned</template></category></unlearn>ok</template></category>
</aiml>`)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	const learners, readers = 10, 10
	var wg sync.WaitGroup
	errs := make(chan string, learners+readers)
	for i := 0; i < learners; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := g.CreateSession(fmt.Sprintf("learner%d", i))
			for j := 0; j < 20; j++ {
				word := fmt.Sprintf("WORD%dX%d", i, j)
				if _, err := g.ProcessInput("teach "+word, session); err != nil {
					errs <- fmt.Sprintf("teach %s: %v", word, err)
					return
				}
				if response, err := g.ProcessInput(word, session); err != nil || response != "learned" {
					errs <- fmt.Sprintf("%s after learning: got %q, %v", word, response, err)
					return
				}
				if _, err := g.ProcessInput("forget "+word, session); err != nil {
					errs <- fmt.Sprintf("forget %s: %v", word, err)
					return
				}
			}
		}(i)
	}
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := g.CreateSession(fmt.Sprintf("reader%d", i))
			for j := 0; j < 100; j++ {
				if response, err := g.ProcessInput("hello", session); err != nil || response != "hi" {
					errs <- fmt.Sprintf("reader%d: got %q, %v", i, response, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}

	if count := g.aimlKB.categoryCount(); count != 3 {
		t.Errorf("Expected 3 categories after unlearning everything, got %d", count)
	}
}
