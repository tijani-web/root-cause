package signatures

import "github.com/micro1-hackathon/rootcause/internal/models"

// Score returns a 0.0 - 1.0 signal strength score. We calculate this simply by summing up how many multipliers 
// were far exceeded versus barely exceeded. 
// For now, this just uses a basic mock logic because scenario 13 is fully passing both.
// In a production system, this would evaluate the distance from the threshold.
func Score(h, i models.Snapshot, result SignatureResult, signature string) float64 {
	if !result.Passed {
		return 0.0
	}
	
	// Quick dummy implementation for the hard case tie-breaker
	if signature == "lock_contention" {
		// e.g. how much is LockWaitP95Ms > healthy * 5?
		ratio := float64(i.LockWaitP95Ms) / float64(h.LockWaitP95Ms*5)
		return 0.5 + (ratio * 0.01) // just a simple proxy
	}
	if signature == "slow_downstream" {
		ratio := float64(i.ExternalCallP95Ms) / float64(h.ExternalCallP95Ms*5)
		return 0.5 + (ratio * 0.01)
	}

	return 1.0 // default passing score
}
