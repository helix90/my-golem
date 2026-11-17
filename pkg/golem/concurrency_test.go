package golem

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// These tests are most useful under the race detector: go test -race

const concurrencyTestAIML = `<aiml version="2.0">
<category><pattern>HELLO</pattern><template>Hi <srai>NAME</srai></template></category>
<category><pattern>NAME</pattern><template><get name="name"/></template></category>
<category><pattern>MY NAME IS *</pattern><template><think><set name="name"><star/></set></think>ok <star/></template></category>
<category><pattern>ADD *</pattern><template><list name="items" operation="add"><star/></list><list name="items" operation="size"/></template></category>
<category><pattern>TEACH *</pattern><template><think><learn><category><pattern>TAUGHT <eval><star/></eval></pattern><template>yes</template></category></learn></think>taught</template></category>
<category><pattern>TOPIC *</pattern><template><think><set name="topic"><star/></set></think>t</template></category>
<category><pattern>*</pattern><template><request/> <response/></template></category>
</aiml>`

func concurrencyTestInputs(i int) []string {
	return []string{"hello", fmt.Sprintf("my name is u%d", i), fmt.Sprintf("add item%d", i),
		fmt.Sprintf("teach w%d", i), fmt.Sprintf("taught w%d", i), "topic cars", "anything goes"}
}

func newConcurrencyTestGolem(t *testing.T) *Golem {
	g := NewForTesting(t, false)
	if err := g.LoadAIMLFromString(concurrencyTestAIML); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	return g
}

// Sessions created concurrently without an ID must get distinct IDs.
func TestConcurrentCreateSessionIDs(t *testing.T) {
	g := newConcurrencyTestGolem(t)
	var wg sync.WaitGroup
	sessions := make([]*ChatSession, 20)
	for i := range sessions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessions[i] = g.CreateSession("")
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool)
	for _, s := range sessions {
		if seen[s.ID] {
			t.Errorf("Duplicate session ID %q", s.ID)
		}
		seen[s.ID] = true
	}
}

// Concurrent requests on one session must run one at a time, so none of
// them is lost from the session's history (and the process doesn't crash
// with concurrent map writes).
func TestConcurrentProcessInputSameSession(t *testing.T) {
	g := newConcurrencyTestGolem(t)
	session := g.CreateSession("shared")

	const goroutines, rounds = 4, 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < rounds; n++ {
				for _, input := range concurrencyTestInputs(i) {
					if _, err := g.ProcessInput(input, session); err != nil {
						t.Errorf("ProcessInput(%q) failed: %v", input, err)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	if want := goroutines * rounds * len(concurrencyTestInputs(0)); len(session.History) != want {
		t.Errorf("Expected %d history entries, got %d", want, len(session.History))
	}
}

// Loading AIML while sessions are chatting must not lose what they learn:
// the merged knowledge base must not be built from a copy taken mid-request.
func TestConcurrentLoadWhileLearning(t *testing.T) {
	g := newConcurrencyTestGolem(t)

	const learners, loads = 4, 20
	var wg sync.WaitGroup
	for i := 0; i < learners; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := g.CreateSession("")
			for n := 0; n < loads; n++ {
				g.ProcessInput(fmt.Sprintf("teach l%dn%d", i, n), session)
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < loads; n++ {
			aiml := fmt.Sprintf(`<aiml version="2.0"><category><pattern>EXTRA %d</pattern><template>x</template></category></aiml>`, n)
			if err := g.LoadAIMLFromString(aiml); err != nil {
				t.Errorf("LoadAIMLFromString failed: %v", err)
			}
		}
	}()
	wg.Wait()

	session := g.CreateSession("check")
	for i := 0; i < learners; i++ {
		for n := 0; n < loads; n++ {
			input := fmt.Sprintf("taught l%dn%d", i, n)
			if response, _ := g.ProcessInput(input, session); !strings.Contains(response, "yes") {
				t.Errorf("Learned category for %q was lost, got %q", input, response)
			}
		}
	}
	for n := 0; n < loads; n++ {
		if response, _ := g.ProcessInput(fmt.Sprintf("extra %d", n), session); response != "x" {
			t.Errorf("Loaded category EXTRA %d missing, got %q", n, response)
		}
	}
}

// WithLock lets other code read and reset a session while requests are being
// processed on it
func TestConcurrentWithLockAndProcessInput(t *testing.T) {
	g := newConcurrencyTestGolem(t)
	session := g.CreateSession("locked")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for n := 0; n < 50; n++ {
			for _, input := range concurrencyTestInputs(n) {
				if _, err := g.ProcessInput(input, session); err != nil {
					t.Errorf("ProcessInput(%q) failed: %v", input, err)
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for n := 0; n < 50; n++ {
			session.WithLock(func() {
				_ = len(session.History) + len(session.Variables)
				session.History = []string{}
				session.ThatHistory = []string{}
				session.Variables = make(map[string]string)
				session.Topic = ""
			})
		}
	}()
	wg.Wait()
}

// The exported ProcessTemplate* methods hold the knowledge base lock, so they
// can run while another goroutine replaces the knowledge base.
func TestConcurrentProcessTemplateAndSetKnowledgeBase(t *testing.T) {
	g := newConcurrencyTestGolem(t)
	session := g.CreateSession("templates")
	kb := g.GetKnowledgeBase()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for n := 0; n < 50; n++ {
			g.ProcessTemplateWithContext(`<size/> <bot name="name"/>`, nil, session)
			g.ProcessTemplateWithSession(`<size/>`, nil, session)
			g.ProcessTemplate(`<size/>`, nil)
		}
	}()
	go func() {
		defer wg.Done()
		for n := 0; n < 50; n++ {
			g.SetKnowledgeBase(kb)
		}
	}()
	wg.Wait()
}
