package main

import (
	"fmt"
	"log"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance
	g := golem.New(true)

	// Load AIML content with list and array examples
	aimlContent := `
<aiml version="2.0">
    <!-- Shopping List Management -->
    <category>
        <pattern>ADD * TO SHOPPING LIST</pattern>
        <template>
            <list name="shopping" operation="add"><star/></list>
            I've added <star/> to your shopping list.
        </template>
    </category>
    
    <category>
        <pattern>SHOW SHOPPING LIST</pattern>
        <template>
            <list name="shopping" operation="size"></list> items in your shopping list:
            <list name="shopping"></list>
        </template>
    </category>
    
    <category>
        <pattern>REMOVE * FROM SHOPPING LIST</pattern>
        <template>
            <list name="shopping" operation="remove"><star/></list>
            I've removed <star/> from your shopping list.
        </template>
    </category>
    
    <category>
        <pattern>CLEAR SHOPPING LIST</pattern>
        <template>
            <list name="shopping" operation="clear"></list>
            Your shopping list has been cleared.
        </template>
    </category>
    
    <!-- Task Management with Arrays -->
    <category>
        <pattern>SET TASK * AT POSITION *</pattern>
        <template>
            <array name="tasks" index="<star2/>" operation="set"><star/></array>
            Task "<star/>" set at position <star2/>.
        </template>
    </category>
    
    <category>
        <pattern>SHOW TASK AT POSITION *</pattern>
        <template>
            Task at position <star/>: <array name="tasks" index="<star/>"></array>
        </template>
    </category>
    
    <category>
        <pattern>SHOW ALL TASKS</pattern>
        <template>
            All tasks: <array name="tasks"></array>
        </template>
    </category>
    
    <category>
        <pattern>HOW MANY TASKS</pattern>
        <template>
            You have <array name="tasks" operation="size"></array> tasks.
        </template>
    </category>
    
    <!-- User Preferences with Lists -->
    <category>
        <pattern>I LIKE *</pattern>
        <template>
            <list name="preferences" operation="add"><star/></list>
            I've noted that you like <star/>.
        </template>
    </category>
    
    <category>
        <pattern>WHAT DO I LIKE</pattern>
        <template>
            You like: <list name="preferences"></list>
        </template>
    </category>
    
    <!-- Score Tracking with Arrays -->
    <category>
        <pattern>MY SCORE IS *</pattern>
        <template>
            <set name="current_score"><star/></set>
            <array name="scores" index="0" operation="set"><get name="current_score"/></array>
            Your score <get name="current_score"/> has been recorded.
        </template>
    </category>
    
    <category>
        <pattern>SHOW MY SCORE</pattern>
        <template>
            Your current score: <array name="scores" index="0"></array>
        </template>
    </category>
    
    <!-- Memory Management -->
    <category>
        <pattern>REMEMBER *</pattern>
        <template>
            <list name="memories" operation="add"><star/></list>
            I'll remember that: <star/>
        </template>
    </category>
    
    <category>
        <pattern>WHAT DO YOU REMEMBER</pattern>
        <template>
            I remember: <list name="memories"></list>
        </template>
    </category>
    
    <!-- Fallback -->
    <category>
        <pattern>*</pattern>
        <template>
            I can help you manage lists and arrays. Try:
            - "Add milk to shopping list"
            - "Show shopping list"
            - "Set task buy groceries at position 1"
            - "Show task at position 1"
            - "I like pizza"
            - "What do I like"
            - "My score is 95"
            - "Remember I have a meeting tomorrow"
        </template>
    </category>
</aiml>`

	// Load the AIML content
	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		log.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a new chat session
	session := g.CreateSession("demo-session")

	fmt.Println("=== Golem List and Array Demo ===")
	fmt.Println("Type 'quit' to exit")
	fmt.Println()

	// Interactive demo
	inputs := []string{
		"Add milk to shopping list",
		"Add bread to shopping list",
		"Add eggs to shopping list",
		"Show shopping list",
		"Set task buy groceries at position 1",
		"Set task call mom at position 2",
		"Show task at position 1",
		"Show all tasks",
		"How many tasks",
		"I like pizza",
		"I like ice cream",
		"What do I like",
		"My score is 95",
		"Show my score",
		"Remember I have a meeting tomorrow",
		"Remember I need to call the dentist",
		"What do you remember",
		"Remove bread from shopping list",
		"Show shopping list",
		"Clear shopping list",
		"Show shopping list",
	}

	for _, input := range inputs {
		fmt.Printf("User: %s\n", input)
		response, err := g.ProcessInput(input, session)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Bot: %s\n\n", response)
	}
}
