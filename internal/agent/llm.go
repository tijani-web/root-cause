package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const Model = "claude-haiku-4-5"

// LLMMessage is a single turn (user or assistant only)
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type anthropicRequest struct {
	Model      string               `json:"model"`
	MaxTokens  int                  `json:"max_tokens"`
	System     string               `json:"system,omitempty"`
	Messages   []LLMMessage         `json:"messages"`
	Tools      []anthropicTool      `json:"tools,omitempty"`
	ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Type  string                 `json:"type"`
		Text  string                 `json:"text,omitempty"`
		Input map[string]interface{} `json:"input,omitempty"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// loadEnv manually parses a .env file to avoid external dependencies
func loadEnv() {
	b, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

// Call sends messages to Anthropic using Tool Use to guarantee structured JSON output.
func Call(messages []LLMMessage) (string, error) {
	loadEnv()
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY env var not set")
	}

	var systemPrompt string
	var userMessages []LLMMessage
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			userMessages = append(userMessages, m)
		}
	}

	// Define the strict diagnostic tool schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"proposed_cause": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"n_plus_one_query", "lock_contention", "gc_pause",
					"connection_pool_exhaustion", "slow_downstream", "stale_cache",
					"thread_starvation", "disk_io_saturation", "memory_pressure",
					"network_retry_storm", "pagination_bug", "none",
				},
			},
			"confidence": map[string]interface{}{
				"type":        "number",
				"description": "Confidence score between 0.0 and 1.0",
			},
			"evidence": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"metric":         map[string]interface{}{"type": "string"},
						"baseline_value": map[string]interface{}{"type": "number"},
						"incident_value": map[string]interface{}{"type": "number"},
						"delta":          map[string]interface{}{"type": "string"},
					},
					"required": []string{"metric", "baseline_value", "incident_value", "delta"},
				},
			},
			"ruled_out": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"cause":        map[string]interface{}{"type": "string"},
						"why_rejected": map[string]interface{}{"type": "string"},
					},
					"required": []string{"cause", "why_rejected"},
				},
			},
			"reasoning": map[string]interface{}{"type": "string"},
		},
		"required": []string{"proposed_cause", "confidence", "evidence", "ruled_out", "reasoning"},
	}

	reqBody := anthropicRequest{
		Model:     Model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages:  userMessages,
		Tools: []anthropicTool{
			{
				Name:        "submit_diagnosis",
				Description: "Submit the structured diagnostic report based on metric deltas.",
				InputSchema: schema,
			},
		},
		ToolChoice: &anthropicToolChoice{
			Type: "tool",
			Name: "submit_diagnosis",
		},
	}

	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	
	workspaceID := os.Getenv("ANTHROPIC_WORKSPACE_ID")
	if workspaceID != "" {
		req.Header.Set("anthropic-workspace-id", workspaceID)
	}

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

	// Find the tool_use content block
	for _, c := range result.Content {
		if c.Type == "tool_use" {
			// Return the raw JSON arguments of the tool call
			out, _ := json.Marshal(c.Input)
			return string(out), nil
		}
	}

	return "", fmt.Errorf("no tool_use block found in response: %s", string(raw))
}
