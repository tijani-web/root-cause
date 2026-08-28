package models

// GroundTruth represents the actual root cause of a scenario
type GroundTruth struct {
	ScenarioID     string `json:"scenario_id"`
	ActualCause    string `json:"actual_cause"`
	InjectedNoise  string `json:"injected_noise"`
}
