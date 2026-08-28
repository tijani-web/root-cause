package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/micro1-hackathon/rootcause/internal/scenarios"
)

func main() {
	allScenarios, truths := scenarios.GenerateAll()
	outDir := filepath.Join("data", "scenarios")

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Failed to create dir: %v\n", err)
		os.Exit(1)
	}

	for _, s := range allScenarios {
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			fmt.Printf("Marshal error %s: %v\n", s.ScenarioID, err)
			continue
		}
		fname := filepath.Join(outDir, fmt.Sprintf("scenario_%s.json", s.ScenarioID))
		if err := os.WriteFile(fname, b, 0644); err != nil {
			fmt.Printf("Write error %s: %v\n", s.ScenarioID, err)
		} else {
			fmt.Printf("Generated %s\n", fname)
		}
	}

	b, _ := json.MarshalIndent(truths, "", "  ")
	tname := filepath.Join("data", "ground_truth.json")
	if err := os.WriteFile(tname, b, 0644); err != nil {
		fmt.Printf("Write truth error: %v\n", err)
	} else {
		fmt.Printf("Generated %s\n", tname)
	}
}
