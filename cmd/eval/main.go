package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/micro1-hackathon/rootcause/internal/agent"
	"github.com/micro1-hackathon/rootcause/internal/models"
)

type EvalRecord struct {
	ScenarioID    string `json:"scenario_id"`
	ActualCause   string `json:"actual_cause"`
	BaselineCause string `json:"baseline_cause"`
	AgentCause    string `json:"agent_cause"`
	BaselineCorrect bool `json:"baseline_correct"`
	AgentCorrect  bool   `json:"agent_correct"`
	AgentAttempts int    `json:"agent_attempts"`
	BaselineReasoning string `json:"baseline_reasoning"`
	AgentReasoning    string `json:"agent_reasoning"`
}

func main() {
	outDir := filepath.Join("data", "scenarios")
	trajDir := filepath.Join("data", "trajectories")
	os.MkdirAll(trajDir, 0755)

	// Load ground truth
	tb, err := os.ReadFile(filepath.Join("data", "ground_truth.json"))
	if err != nil {
		fmt.Printf("Failed to read ground truth: %v\n", err)
		os.Exit(1)
	}
	var truths []models.GroundTruth
	json.Unmarshal(tb, &truths)

	truthMap := map[string]string{}
	for _, t := range truths {
		truthMap[t.ScenarioID] = t.ActualCause
	}

	var records []EvalRecord
	baselineScore := 0
	agentScore := 0

	for i := 1; i <= 13; i++ {
		scenarioID := fmt.Sprintf("%02d", i)
		fname := filepath.Join(outDir, fmt.Sprintf("scenario_%s.json", scenarioID))

		b, _ := os.ReadFile(fname)
		var s models.ScenarioData
		json.Unmarshal(b, &s)

		actualCause := truthMap[scenarioID]
		fmt.Printf("--- Scenario %s: %s (truth: %s) ---\n", scenarioID, s.Description, actualCause)

		// Phase 3: Baseline
		fmt.Printf("  [BASELINE] calling LLM...\n")
		baselineCause, baselineReasoning, err := agent.BaselineDiagnose(s)
		if err != nil {
			fmt.Printf("  [BASELINE] ERROR: %v\n", err)
			baselineCause = "error"
			baselineReasoning = err.Error()
		}
		baselineCorrect := strings.EqualFold(baselineCause, actualCause)
		if baselineCorrect {
			baselineScore++
		}
		fmt.Printf("  [BASELINE] got=%s correct=%v\n", baselineCause, baselineCorrect)

		// Phase 4: Agent (Detective/Verifier loop)
		fmt.Printf("  [AGENT]    running detective loop...\n")
		diag, err := agent.AgentDiagnose(s)
		agentCause := "error"
		agentReasoning := ""
		agentAttempts := 0
		agentCorrect := false
		if err != nil {
			fmt.Printf("  [AGENT] ERROR: %v\n", err)
			agentReasoning = err.Error()
		} else {
			agentCause = diag.FinalCause
			agentReasoning = diag.Reasoning
			agentAttempts = diag.Attempts
			agentCorrect = strings.EqualFold(agentCause, actualCause)
			if agentCorrect {
				agentScore++
			}
			if len(diag.RejectedHypotheses) > 0 {
				fmt.Printf("  [AGENT]    rejected %d hypotheses before confirming\n", len(diag.RejectedHypotheses))
			}

			// Save trajectory
			trajJSON, _ := json.MarshalIndent(diag.Trajectory, "", "  ")
			os.WriteFile(filepath.Join(trajDir, fmt.Sprintf("scenario_%s.json", scenarioID)), trajJSON, 0644)
		}
		fmt.Printf("  [AGENT]    got=%s correct=%v attempts=%d\n", agentCause, agentCorrect, agentAttempts)
		fmt.Println()

		records = append(records, EvalRecord{
			ScenarioID:        scenarioID,
			ActualCause:       actualCause,
			BaselineCause:     baselineCause,
			AgentCause:        agentCause,
			BaselineCorrect:   baselineCorrect,
			AgentCorrect:      agentCorrect,
			AgentAttempts:     agentAttempts,
			BaselineReasoning: baselineReasoning,
			AgentReasoning:    agentReasoning,
		})
	}

	// Summary
	fmt.Println("===== EVAL SUMMARY =====")
	fmt.Printf("Baseline score: %d/13 (%.0f%%)\n", baselineScore, float64(baselineScore)/13*100)
	fmt.Printf("Agent score:    %d/13 (%.0f%%)\n", agentScore, float64(agentScore)/13*100)
	fmt.Printf("Delta:          +%d\n", agentScore-baselineScore)

	// Write results JSON
	out, _ := json.MarshalIndent(records, "", "  ")
	os.WriteFile(filepath.Join("data", "eval_results.json"), out, 0644)
	fmt.Println("\nResults written to data/eval_results.json")
}
