package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/micro1-hackathon/rootcause/internal/models"
	"github.com/micro1-hackathon/rootcause/internal/signatures"
)

func main() {
	outDir := filepath.Join("data", "scenarios")
	
	// Check all 13 scenarios
	for i := 1; i <= 13; i++ {
		fname := filepath.Join(outDir, fmt.Sprintf("scenario_%02d.json", i))
		b, err := os.ReadFile(fname)
		if err != nil {
			fmt.Printf("Failed to read %s: %v\n", fname, err)
			continue
		}

		var s models.ScenarioData
		if err := json.Unmarshal(b, &s); err != nil {
			fmt.Printf("Failed to parse %s: %v\n", fname, err)
			continue
		}

		fmt.Printf("--- Evaluating Scenario %s: %s ---\n", s.ScenarioID, s.Description)
		
		passedCount := 0
		var passedSignatures []string

		for sigName, checkFunc := range signatures.Registry {
			res := checkFunc(s.Healthy, s.Incident)
			if res.Passed {
				passedCount++
				passedSignatures = append(passedSignatures, sigName)
			}
		}

		if s.ScenarioID == "12" {
			if passedCount != 0 {
				fmt.Printf("[FAIL] Scenario 12 (Clean) should have 0 passing signatures. Got %d: %v\n", passedCount, passedSignatures)
			} else {
				fmt.Printf("[PASS] Scenario 12 cleanly rejected all signatures.\n")
			}
		} else if s.ScenarioID == "13" {
			if passedCount != 3 {
				fmt.Printf("[FAIL] Scenario 13 (Hard) should have 3 passing signatures (lock, downstream, composite). Got %d: %v\n", passedCount, passedSignatures)
			} else {
				fmt.Printf("[PASS] Scenario 13 correctly passed 3 signatures (lock_contention + slow_downstream + lock_and_downstream): %v\n", passedSignatures)
			}
		} else {
			if passedCount != 1 {
				fmt.Printf("[FAIL] Scenario %s should have exactly 1 passing signature. Got %d: %v\n", s.ScenarioID, passedCount, passedSignatures)
			} else {
				fmt.Printf("[PASS] Scenario %s correctly passed exactly 1 signature: %v\n", s.ScenarioID, passedSignatures)
			}
		}
		fmt.Println()
	}
}
