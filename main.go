package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

const gomentoURL = "http://localhost:4000/api"

func main() {
	// 1️⃣ Initialize The "Memory Bank" (The Space)
	spaceID, err := createSpace("agent-learning-space")
	if err != nil {
		log.Fatalf("❌ failed to create space: %v", err)
	}
	fmt.Printf("✅ Connected to Space (Long-Term Memory): %s\n", spaceID)

	// 2️⃣ Initialize The Session (The Short-Term Context)
	sessionID, err := createSession(spaceID)
	if err != nil {
		log.Fatalf("❌ failed to create session: %v", err)
	}
	fmt.Printf("✅ Started Session (Short-Term Memory): %s\n", sessionID)

	// 3️⃣ Simulate Conversation
	inputs := []string{
		"User: I want to build a Go agent that uses pgvector.",
		"Agent: That sounds great. You should use the pgvector-go library.",
		"User: How do I optimize the search query?",
		"Agent: You should use an HNSW index for performance.",
	}

	for _, text := range inputs {
		role := "user"
		if len(text) > 5 && text[:5] == "Agent" {
			role = "assistant"
		}
		if err := sendMessage(sessionID, role, text); err != nil {
			log.Printf("⚠️ failed to add memory: %v", err)
		}
	}
	fmt.Println("✅ Populated conversation history.")

	// ⏳ Wait for worker
	fmt.Println("⏳ Indexing memories (waiting for async workers)...")
	time.Sleep(2 * time.Second)

	// 4️⃣ Retrieve Context & Build Prompt
	userQuery := "What index did you recommend for vector search?"

	prompt, err := retrieveContextAndBuildPrompt(spaceID, sessionID, userQuery)
	if err != nil {
		log.Fatalf("❌ failed to build prompt: %v", err)
	}

	fmt.Println("\n----- Final LLM Prompt -----")
	fmt.Println(prompt)
	fmt.Println("----------------------------")
}

func retrieveContextAndBuildPrompt(spaceID, sessionID, query string) (string, error) {
	// A. Short-Term Memory (Recent Messages)
	shortTerm, err := getHistory(sessionID, 5)
	if err != nil {
		return "", fmt.Errorf("short-term error: %w", err)
	}

	// B. Long-Term Memory (Vector Search)
	longTerm, err := searchSpace(spaceID, query)
	if err != nil {
		return "", fmt.Errorf("long-term error: %w", err)
	}

	// C. Construct the Prompt
	var sb bytes.Buffer
	sb.WriteString("System: You are a helpful assistant.\n")

	if len(longTerm) > 0 {
		sb.WriteString("\nRelevant Knowledge (Long-Term Memory):\n")
		for _, msg := range longTerm {
			if len(msg.Parts) > 0 {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Parts[0].Text))
			}
		}
	}

	sb.WriteString("\nConversation History (Short-Term Memory):\n")
	for i := len(shortTerm) - 1; i >= 0; i-- {
		msg := shortTerm[i]
		if len(msg.Parts) > 0 {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Parts[0].Text))
		}
	}

	sb.WriteString(fmt.Sprintf("\nUser: %s", query))
	return sb.String(), nil
}

func createSpace(name string) (string, error) {
	payload := map[string]string{"name": name}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(gomentoURL+"/v1/spaces", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("status: %s", resp.Status)
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}

func createSession(spaceID string) (string, error) {
	payload := map[string]string{"space_id": spaceID}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(gomentoURL+"/v1/sessions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %s %s", resp.Status, string(b))
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}
func sendMessage(sessionID, role, text string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 1. Add "role" field
	if err := writer.WriteField("role", role); err != nil {
		return fmt.Errorf("failed to write role: %w", err)
	}

	// 2. Add "parts" field
	parts := []map[string]string{
		{"type": "text", "text": text},
	}
	partsJson, err := json.Marshal(parts)
	if err != nil {
		return fmt.Errorf("failed to marshal parts: %w", err)
	}

	if err := writer.WriteField("parts", string(partsJson)); err != nil {
		return fmt.Errorf("failed to write parts: %w", err)
	}

	// 3. Close the writer to seal the multipart boundary
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	// 4. Send Request
	url := fmt.Sprintf("%s/v1/sessions/%s/messages", gomentoURL, sessionID)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 5. Check for issues
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status: %s body: %s", resp.Status, string(b))
	}

	return nil
}

func getHistory(sessionID string, limit int) ([]Message, error) {
	url := fmt.Sprintf("%s/v1/sessions/%s/messages?limit=%d", gomentoURL, sessionID, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status: %s", resp.Status)
	}

	var res struct {
		Items []Message `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Items, nil
}

func searchSpace(spaceID, query string) ([]Message, error) {
	baseURL := fmt.Sprintf("%s/v1/spaces/%s/messages", gomentoURL, spaceID)
	params := url.Values{}
	params.Add("q", query)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s %s", resp.Status, string(b))
	}

	var items []Message
	bodyBytes, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(bodyBytes, &items); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return items, nil
}

type Message struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}
