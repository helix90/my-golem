package golem

import (
	"testing"
)

const sraiContextAIML = `<aiml version="2.0">
<category><pattern>START GAMES</pattern><template><think><set name="topic">GAMES</set></think>ok</template></category>
<topic name="GAMES"><category><pattern>INNER</pattern><template>games inner</template></category></topic>
<category><pattern>INNER</pattern><template>generic inner</template></category>
<category><pattern>VIA SRAI</pattern><template><srai>INNER</srai></template></category>
<category><pattern>VIA SR *</pattern><template><sr/></template></category>
<category><pattern>SET AND SRAI</pattern><template><think><set name="topic">GAMES</set></think><srai>INNER</srai></template></category>
<category><pattern>ASK</pattern><template>Do you like cats?</template></category>
<category><pattern>YES</pattern><that>DO YOU LIKE CATS</that><template>cats yes</template></category>
<category><pattern>YES</pattern><template>generic yes</template></category>
<category><pattern>YEAH</pattern><template><srai>YES</srai></template></category>
</aiml>`

// <srai> and <sr/> must match against the current topic and <that>, like
// ordinary input does.
func TestSRAIUsesTopicAndThat(t *testing.T) {
	g := New(false)
	if err := g.LoadAIMLFromString(sraiContextAIML); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		name   string
		inputs []string
		want   string
	}{
		{"no topic", []string{"via srai"}, "generic inner"},
		{"srai in topic", []string{"start games", "via srai"}, "games inner"},
		{"sr in topic", []string{"start games", "via sr inner"}, "games inner"},
		{"topic set earlier in same template", []string{"set and srai"}, "games inner"},
		{"srai with that", []string{"ask", "yeah"}, "cats yes"},
		{"srai without that", []string{"yeah"}, "generic yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession(tt.name)
			var response string
			for _, input := range tt.inputs {
				var err error
				if response, err = g.ProcessInput(input, session); err != nil {
					t.Fatalf("Failed to process %q: %v", input, err)
				}
			}
			if response != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, response)
			}
		})
	}
}

// <sr/> must stop at the srai recursion limit, and must not disturb
// predicates whose names start with "star".
func TestSRSharesSRAIHandling(t *testing.T) {
	g := New(false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>LOOP *</pattern><template><srai>LOOP <sr/></srai></template></category>
<category><pattern>SIGN *</pattern><template><sr/></template></category>
<category><pattern>ARIES</pattern><template><think><set name="starsign">aries</set></think>[<star/>]</template></category>
</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test")
	if _, err := g.ProcessInput("loop forever", session); err != nil {
		t.Fatalf("Failed to process looping input: %v", err)
	}

	session.Variables["starsign"] = "unknown"
	response, err := g.ProcessInput("sign aries", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "[]" {
		t.Errorf("Expected the inner <star/> to be empty, got %q", response)
	}
	if got := session.Variables["starsign"]; got != "aries" {
		t.Errorf("Expected starsign 'aries', got %q", got)
	}
	if _, exists := session.Variables["star1"]; exists {
		t.Errorf("Expected no star1 left in the session, got %q", session.Variables["star1"])
	}
}
