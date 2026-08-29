package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const Model = "claude-haiku-4-5"

// LLMMessage is a single turn (user or assistant only — system is top-level in Anthropic API)
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []LLMMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Call sends messages to the Anthropic Messages API.
// The first message with role "system" is extracted and sent as the top-level system field.
func Call(messages []LLMMessage) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY env var not set")
	}

	// Extract system message (Anthropic puts it top-level, not as a message role)
	var systemPrompt string
	var userMessages []LLMMessage
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			userMessages = append(userMessages, m)
		}
	}

	reqBody := anthropicRequest{
		Model:     Model,
		MaxTokens: 512,
		System:    systemPrompt,
		Messages:  userMessages,
	}

	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result anthropicResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("unmarshal error: %w\nraw: %s", err, string(raw))
	}
	if result.Error.Message != "" {
		return "", fmt.Errorf("api error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in response: %s", string(raw))
	}
	return result.Content[0].Text, nil
}
