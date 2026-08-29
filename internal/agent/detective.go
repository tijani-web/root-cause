package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/micro1-hackathon/rootcause/internal/models"
	"github.com/micro1-hackathon/rootcause/internal/signatures"
)

const detectiveSystemPrompt = `You are RootCause Detective — a performance diagnosis agent.

You will be given TWO snapshots of service metrics:
1. HEALTHY BASELINE — what the service looks like under normal load
2. INCIDENT SNAPSHOT — what the service looks like right now during a performance problem

Your job is to identify the root cause by comparing the two. Look for signals that have changed significantly from the baseline, not just high absolute values.

You MUST respond with ONLY a JSON object in this exact format:
{
  "proposed_cause": "<one of: n_plus_one_query, lock_contention, gc_pause, connection_pool_exhaustion, slow_downstream, stale_cache, thread_starvation, disk_io_saturation, memory_pressure, network_retry_storm, pagination_bug, none>",
  "reasoning": "<explanation referencing specific delta changes from baseline>"
}

Do not include any text outside the JSON.`

// DetectiveResult is the parsed response from the Detective LLM
type DetectiveResult struct {
	ProposedCause string `json:"proposed_cause"`
	Reasoning     string `json:"reasoning"`
}

// AgentDiagnosis is the final structured report after verification passes
type AgentDiagnosis struct {
	ScenarioID        string                 `json:"scenario_id"`
	FinalCause        string                 `json:"final_cause"`
	Confidence        string                 `json:"confidence"`
	VerifiedBy        string                 `json:"verified_by"`
	Evidence          map[string]interface{} `json:"evidence"`
	RejectedHypotheses []RejectedHypothesis  `json:"rejected_hypotheses"`
	Reasoning         string                 `json:"reasoning"`
	Attempts          int                    `json:"attempts"`
	Trajectory        []LLMMessage           `json:"-"`
}

// RejectedHypothesis records a failed attempt for the report
type RejectedHypothesis struct {
	Cause           string `json:"cause"`
	FailedCondition string `json:"failed_condition"`
}

// AgentDiagnose runs the Detective/Verifier loop for one scenario
func AgentDiagnose(s models.ScenarioData) (*AgentDiagnosis, error) {
	const maxAttempts = 4

	var rejected []RejectedHypothesis
	var conversation []LLMMessage

	// Initial Detective prompt with both snapshots
	hb, _ := json.MarshalIndent(s.Healthy, "", "  ")
	ib, _ := json.MarshalIndent(s.Incident, "", "  ")
	userMsg := fmt.Sprintf("HEALTHY BASELINE:\n%s\n\nINCIDENT SNAPSHOT:\n%s", string(hb), string(ib))

	conversation = append(conversation,
		LLMMessage{Role: "system", Content: detectiveSystemPrompt},
		LLMMessage{Role: "user", Content: userMsg},
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		raw, err := Call(conversation)
		if err != nil {
			return nil, fmt.Errorf("llm call failed on attempt %d: %w", attempt, err)
		}

		// Parse Detective reply
		parsed := parseDetective(raw)
		if parsed == nil {
			// Unparseable reply — try once more with correction
			conversation = append(conversation,
				LLMMessage{Role: "assistant", Content: raw},
				LLMMessage{Role: "user", Content: "Your response was not valid JSON. Respond ONLY with the JSON object, no other text."},
			)
			continue
		}

		conversation = append(conversation, LLMMessage{Role: "assistant", Content: raw})

		// Special case: agent says "none" — run clean check
		if parsed.ProposedCause == "none" {
			// Check whether any signature actually passes
			anyPass := false
			for _, checkFn := range signatures.Registry {
				if checkFn(s.Healthy, s.Incident).Passed {
					anyPass = true
					break
				}
			}
			if !anyPass {
				return &AgentDiagnosis{
					ScenarioID: s.ScenarioID,
					FinalCause: "none",
					Confidence: "high",
					VerifiedBy: "no_signature_matched",
					Evidence:   map[string]interface{}{},
					RejectedHypotheses: rejected,
					Reasoning:  parsed.Reasoning,
					Attempts:   attempt,
					Trajectory: conversation,
				}, nil
			}
			// Some signature passes — Detective was wrong, push back
			conversation = append(conversation, LLMMessage{
				Role:    "user",
				Content: "VERIFIER REJECTION: You proposed 'none' but at least one signature condition is failing. Reconsider which signature matches the deltas between healthy and incident.",
			})
			rejected = append(rejected, RejectedHypothesis{Cause: "none", FailedCondition: "signature exists but agent said none"})
			continue
		}

		// Look up the signature in the registry
		checkFn, exists := signatures.Registry[parsed.ProposedCause]
		if !exists {
			conversation = append(conversation, LLMMessage{
				Role:    "user",
				Content: fmt.Sprintf("VERIFIER REJECTION: '%s' is not a known cause. Choose from: n_plus_one_query, lock_contention, gc_pause, connection_pool_exhaustion, slow_downstream, stale_cache, thread_starvation, disk_io_saturation, memory_pressure, network_retry_storm, pagination_bug, none.", parsed.ProposedCause),
			})
			rejected = append(rejected, RejectedHypothesis{Cause: parsed.ProposedCause, FailedCondition: "unknown cause"})
			continue
		}

		// Run the Verifier
		result := checkFn(s.Healthy, s.Incident)
		if result.Passed {
			// ✅ Verified — build the final report
			return &AgentDiagnosis{
				ScenarioID: s.ScenarioID,
				FinalCause: parsed.ProposedCause,
				Confidence: "high",
				VerifiedBy: "signature_check_v1",
				Evidence:   result.MatchedFields,
				RejectedHypotheses: rejected,
				Reasoning:  parsed.Reasoning,
				Attempts:   attempt,
				Trajectory: conversation,
			}, nil
		}

		// ❌ Rejected — feed the exact failing condition back to the Detective
		rejectionMsg := fmt.Sprintf(
			"VERIFIER REJECTION: Your hypothesis '%s' failed signature check.\nFailing condition: %s\n\nRevise your hypothesis based on the delta between healthy and incident.",
			parsed.ProposedCause, result.FailedCondition,
		)
		conversation = append(conversation, LLMMessage{Role: "user", Content: rejectionMsg})
		rejected = append(rejected, RejectedHypothesis{
			Cause:           parsed.ProposedCause,
			FailedCondition: result.FailedCondition,
		})
	}

	// Exhausted max attempts — report unconfirmed
	return &AgentDiagnosis{
		ScenarioID:        s.ScenarioID,
		FinalCause:        "unconfirmed",
		Confidence:        "low",
		VerifiedBy:        "exhausted_max_attempts",
		Evidence:          map[string]interface{}{},
		RejectedHypotheses: rejected,
		Reasoning:         "Agent could not find a passing signature within max attempts",
		Attempts:          maxAttempts,
		Trajectory:        conversation,
	}, nil
}

func parseDetective(raw string) *DetectiveResult {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var r DetectiveResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil
	}
	return &r
}
