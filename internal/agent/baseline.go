package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/micro1-hackathon/rootcause/internal/models"
)

const baselineSystemPrompt = `You are a performance diagnosis expert. You will be given raw service metrics from a production incident. Your job is to identify the root cause of the performance bottleneck.

Respond ONLY with a JSON object in this exact format:
{
  "root_cause": "<one of: n_plus_one_query, lock_contention, gc_pause, connection_pool_exhaustion, slow_downstream, stale_cache, thread_starvation, disk_io_saturation, memory_pressure, network_retry_storm, pagination_bug, none>",
  "reasoning": "<one sentence explanation>"
}

Do not include any text outside the JSON.`

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

	// Parse the JSON reply
	raw = strings.TrimSpace(raw)
	// strip markdown code fences if present
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result struct {
		RootCause string `json:"root_cause"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return "", raw, fmt.Errorf("failed to parse LLM JSON reply: %w\nraw: %s", err, raw)
	}

	return result.RootCause, result.Reasoning, nil
}
