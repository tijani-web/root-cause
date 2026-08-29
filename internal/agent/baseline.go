package agent

import (
	"encoding/json"
	"fmt"

	"github.com/micro1-hackathon/rootcause/internal/models"
)

const baselineSystemPrompt = `You are a performance diagnosis expert. You will be given raw service metrics from a production incident. Your job is to identify the root cause of the performance bottleneck.

You must explicitly extract the metric deltas as evidence, and explicitly list at least one alternative cause you considered and why you ruled it out.`

// BaselineDiagnose sends the raw incident snapshot to the LLM with no context, no verification.
// This is Phase 3 — it establishes the unassisted baseline score.
func BaselineDiagnose(s models.ScenarioData) (string, string, error) {
	b, _ := json.MarshalIndent(s.Incident, "", "  ")
	userMsg := fmt.Sprintf("Incident snapshot:\n%s", string(b))

	messages := []LLMMessage{
		{Role: "system", Content: baselineSystemPrompt},
		{Role: "user", Content: userMsg},
	}

	raw, err := Call(messages)
	if err != nil {
		return "", "", err
	}

	// Parse the JSON reply (guaranteed by Tool Use)
	var result struct {
		ProposedCause string `json:"proposed_cause"`
		Reasoning     string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return "", raw, fmt.Errorf("failed to parse LLM JSON reply: %w\nraw: %s", err, raw)
	}

	return result.ProposedCause, result.Reasoning, nil
}
