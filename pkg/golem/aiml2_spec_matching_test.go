package golem

import (
	"testing"
)

type specCase struct {
	inputs []string
	want   string
}

func runSpecCases(t *testing.T, g *Golem, cases []specCase) {
	t.Helper()
	for _, tt := range cases {
		session := g.CreateSession("test")
		var response string
		for _, input := range tt.inputs {
			var err error
			if response, err = g.ProcessInput(input, session); err != nil {
				response = "error: " + err.Error()
			}
		}
		if response != tt.want {
			t.Errorf("%q: expected %q, got %q", tt.inputs, tt.want, response)
		}
	}
}

// * and _ match one or more words; ^ and # match zero or more.
func TestSpecWildcardWidths(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>STAR *</pattern><template>star [<star/>]</template></category>
<category><pattern>UNDER _</pattern><template>under [<star/>]</template></category>
<category><pattern>CARET ^</pattern><template>caret [<star/>]</template></category>
<category><pattern>HASH #</pattern><template>hash [<star/>]</template></category>
<category><pattern>^ WORDS</pattern><template>words [<star/>]</template></category>`)

	runSpecCases(t, g, []specCase{
		{[]string{"star"}, "error: no matching pattern found"},
		{[]string{"star a b"}, "star [a b]"},
		{[]string{"under"}, "error: no matching pattern found"},
		{[]string{"under a b"}, "under [a b]"},
		{[]string{"caret"}, "caret []"},
		{[]string{"caret a b"}, "caret [a b]"},
		{[]string{"hash"}, "hash []"},
		{[]string{"hash a b"}, "hash [a b]"},
		{[]string{"words"}, "words []"},
		{[]string{"many words"}, "words [many]"},
	})
}

// At each word the branches are tried in the order $WORD, #, _, WORD,
// <set>, ^, *, and the earlier word decides.
func TestSpecMatchPriority(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>GO *</pattern><template>star</template></category>
<category><pattern>GO ^</pattern><template>caret</template></category>
<category><pattern>GO <set>places</set></pattern><template>set</template></category>
<category><pattern>GO HOME</pattern><template>word</template></category>
<category><pattern>GO _</pattern><template>underscore</template></category>
<category><pattern>GO #</pattern><template>hash</template></category>
<category><pattern>GO $HOME</pattern><template>dollar</template></category>
<category><pattern>COME *</pattern><template>star</template></category>
<category><pattern>COME ^</pattern><template>caret</template></category>
<category><pattern>COME <set>places</set></pattern><template>set</template></category>
<category><pattern>COME HOME</pattern><template>word</template></category>
<category><pattern>LEAVE *</pattern><template>star</template></category>
<category><pattern>LEAVE ^</pattern><template>caret</template></category>
<category><pattern>LEAVE <set>places</set></pattern><template>set</template></category>
<category><pattern>STAY *</pattern><template>star</template></category>
<category><pattern>STAY ^</pattern><template>caret</template></category>
<category><pattern>HELLO _</pattern><template>hello underscore</template></category>
<category><pattern>_ WORLD</pattern><template>underscore world</template></category>
<category><pattern>TEACH * MEANS *</pattern><template>teach means</template></category>
<category><pattern>TEACH *</pattern><template>teach</template></category>
<category><pattern>*</pattern><template>catchall</template></category>`)
	g.aimlKB.AddSetMembers("places", []string{"home", "new york"})

	runSpecCases(t, g, []specCase{
		{[]string{"go home"}, "dollar"},
		{[]string{"come home"}, "word"},
		{[]string{"leave home"}, "set"},
		{[]string{"leave new york"}, "set"},
		{[]string{"stay home"}, "caret"},
		{[]string{"hello world"}, "underscore world"},
		{[]string{"teach x means y"}, "teach means"},
		{[]string{"teach x"}, "teach"},
	})
}

// The input pattern counts before <that> and <topic>, and a category
// without them comes after one with them.
func TestSpecThatAndTopicInPath(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>ASK</pattern><template>Do you like cats?</template></category>
<category><pattern>YES</pattern><template>plain yes</template></category>
<category><pattern>*</pattern><that>DO YOU LIKE *</that><template>any [<thatstar/>]</template></category>
<category><pattern>NO</pattern><that>do you like *</that><template>no [<thatstar/>]</template></category>
<category><pattern>NO</pattern><template>plain no</template></category>
<category><pattern>PLAY</pattern><template><think><set name="topic">board games</set></think>ok</template></category>
<category><pattern>MOVE</pattern><topic>BOARD *</topic><template>move [<topicstar/>]</template></category>
<category><pattern>MOVE</pattern><template>plain move</template></category>
<category><pattern>OFFER *</pattern><template>Would you like a <star/>?</template></category>
<category><pattern>SURE</pattern><that>WOULD YOU LIKE A <set>fruits</set></that><template>fruit [<thatstar/>]</template></category>
<category><pattern>SURE</pattern><template>plain sure</template></category>`)
	g.aimlKB.AddSetMembers("fruits", []string{"apple", "passion fruit"})

	runSpecCases(t, g, []specCase{
		{[]string{"offer apple", "sure"}, "fruit [APPLE]"},
		{[]string{"offer passion fruit", "sure"}, "fruit [PASSION FRUIT]"},
		{[]string{"offer rock", "sure"}, "plain sure"},
		{[]string{"ask", "yes"}, "plain yes"},
		{[]string{"ask", "maybe"}, "any [CATS]"},
		{[]string{"ask", "no"}, "no [CATS]"},
		{[]string{"no"}, "plain no"},
		{[]string{"move"}, "plain move"},
		{[]string{"play", "move"}, "move [GAMES]"},
	})
}

// Patterns may refer to bot properties and contain any characters.
func TestSpecPatternContent(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>MY NAME IS <bot name="name"/></pattern><template>same name</template></category>
<category><pattern>MY NAME IS *</pattern><template>other name</template></category>
<category><pattern>R&amp;D</pattern><template>research</template></category>
<category><pattern>MAIL ME @ HOME</pattern><template>mail</template></category>`)
	g.aimlKB.Properties["name"] = "Golem Bot"

	runSpecCases(t, g, []specCase{
		{[]string{"my name is golem bot"}, "same name"},
		{[]string{"my name is alice"}, "other name"},
		{[]string{"r&d"}, "research"},
		{[]string{"mail me @ home"}, "mail"},
	})
}

// Input is split into sentences, each answered in turn, and each sentence's
// response is the <that> for the next.
func TestSpecInputSentences(t *testing.T) {
	g := loadPatternTestAIML(t, `
<category><pattern>HI</pattern><template>Hello! Do you like cats?</template></category>
<category><pattern>YES</pattern><that>DO YOU LIKE CATS</that><template>Me too.</template></category>
<category><pattern>BYE</pattern><template>Goodbye.</template></category>`)

	runSpecCases(t, g, []specCase{
		{[]string{"hi. yes!"}, "Hello! Do you like cats? Me too."},
		{[]string{"hi? what? bye"}, "Hello! Do you like cats? Goodbye."},
		{[]string{"what?"}, "error: no matching pattern found"},
	})
}
