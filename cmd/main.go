package main

import (
	"context"
	"fmt"
	"log"

	"github.com/w-h-a/agent"
)

const (
	gomentoURL = "http://localhost:4000"
	memoryPG   = "postgres://user:password@localhost:5432/memory?sslmode=disable"
)

func main() {
	ctx := context.Background()

	// 1. Create Agent
	a := agent.InitAgent(
		ctx,
		"gomento",
		gomentoURL,
		"openai",
		"",
		"gpt-3.5-turbo",
		"text-embedding-3-small",
		"You orchestrate tooling and specialists to help the user build AI agents.",
		6,
	)

	fmt.Println("--- Agent Development Kit Demo ---")

	// 2. Initialize The (Optional) Space
	spaceId, err := a.CreateSpace(ctx, "agent-learning-space")
	if err != nil {
		log.Fatalf("❌ failed to create space: %v", err)
	}
	fmt.Printf("✅ Connected to Space: %s\n", spaceId)

	// 3. Initialize The Session
	sessionId, err := a.CreateSession(ctx, spaceId)
	if err != nil {
		log.Fatalf("❌ failed to create session: %v", err)
	}
	fmt.Printf("✅ Started Session: %s\n", sessionId)

	// 4. Simulate Conversation
	userQuestions := []string{
		"I want to design an AI agent with both short term and long term memory. How should I start?",
		"tool:calculate 21 / 3",
		"tool:research Provide a concise brief on pgvector usage for AI memory.",
		"How can I wire everything together after gathering research?",
	}

	for _, msg := range userQuestions {
		rsp, err := a.Respond(ctx, spaceId, sessionId, msg)
		if err != nil {
			log.Printf("❌ Agent error: %v\n", err)
			continue
		}
		fmt.Printf("User: %s\nAgent: %s\n\n", msg, rsp)
	}
	fmt.Println("✅ Populated conversation history.")

	// 5. Consolidate Memory
	fmt.Println("⏳ Flushing to Long-Term...")
	if err := a.Flush(ctx, sessionId); err != nil {
		log.Printf("⚠️ failed to flush: %v", err)
	} else {
		log.Printf("flushed short-term memory for %s to long-term storage", sessionId)
	}
}
