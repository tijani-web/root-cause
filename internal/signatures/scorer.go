package signatures

import (
	"math"

	"github.com/micro1-hackathon/rootcause/internal/models"
)

// Score calculates a normalized 0.0–1.0 signal strength for a passing signature.
// It measures how far each key metric exceeds its threshold multiplier, then averages
// the overshoot ratios. A score near 0.5 means the metrics barely passed; near 1.0
// means the signal is overwhelmingly strong.
func Score(h, i models.Snapshot, result SignatureResult, signature string) float64 {
	if !result.Passed {
		return 0.0
	}

	switch signature {
	case "n_plus_one_query":
		return avgOvershoot(
			overshoot(float64(i.DBCallCount), float64(h.DBCallCount), 3),
			overshoot(float64(i.TotalDBTimeMs)/float64(i.TotalRequestTimeMs), 0.6, 1),
		)
	case "lock_contention":
		return avgOvershoot(
			overshoot(float64(i.LockWaitP95Ms), float64(h.LockWaitP95Ms), 5),
			overshoot(float64(i.LockHolderCount), float64(h.LockHolderCount), 2),
		)
	case "gc_pause":
		return avgOvershoot(
			overshoot(float64(i.GCEventCount), float64(h.GCEventCount), 3),
			overshoot(float64(i.MaxGCPauseMs), float64(h.MaxGCPauseMs), 4),
		)
	case "connection_pool_exhaustion":
		return avgOvershoot(
			overshoot(float64(i.CheckoutWaitP95Ms), float64(h.CheckoutWaitP95Ms), 10),
			overshoot(float64(i.PoolWaitQueue), 1, 1),
		)
	case "slow_downstream":
		return avgOvershoot(
			overshoot(float64(i.ExternalCallP95Ms), float64(h.ExternalCallP95Ms), 5),
			overshoot(float64(i.ExternalCallTimeMs)/float64(i.TotalRequestTimeMs), 0.5, 1),
		)
	case "stale_cache":
		return avgOvershoot(
			overshoot(i.DownstreamRetryRate, h.DownstreamRetryRate, 3),
			overshoot(float64(i.DataStalenessDeltaMs), float64(h.DataStalenessDeltaMs), 5),
		)
	case "thread_starvation":
		return avgOvershoot(
			overshoot(i.CPUUtilization, 0.85, 1),
			overshoot(float64(i.IOWaitTimeMs), float64(h.IOWaitTimeMs), 4),
			overshoot(float64(i.GoroutinesBlocked), float64(h.GoroutinesBlocked), 5),
		)
	case "disk_io_saturation":
		return avgOvershoot(
			overshoot(float64(i.DiskQueueDepth), float64(h.DiskQueueDepth), 4),
			overshoot(float64(i.DiskAwaitMs), float64(h.DiskAwaitMs), 3),
			overshoot(i.CPUIOWait, h.CPUIOWait, 5),
		)
	case "memory_pressure":
		return avgOvershoot(
			overshoot(i.PageFaultRate, h.PageFaultRate, 10),
			overshoot(i.SwapInRate, 0.01, 1),
		)
	case "network_retry_storm":
		return avgOvershoot(
			overshoot(i.RetryRate, h.RetryRate, 3),
			overshoot(i.DownstreamErrorRate, h.DownstreamErrorRate, 5),
			overshoot(i.ReqAmplification, 2.0, 1),
		)
	case "pagination_bug":
		return avgOvershoot(
			overshoot(float64(i.RowsFetched), float64(i.RowsDisplayed), 10),
			overshoot(float64(i.DBPayloadSizeBytes), float64(h.DBPayloadSizeBytes), 5),
			overshoot(float64(i.ExecutionTimeMs), float64(h.ExecutionTimeMs), 5),
		)
	case "lock_and_downstream":
		// Composite: average the signal strength of both sub-signatures
		lockScore := avgOvershoot(
			overshoot(float64(i.LockWaitP95Ms), float64(h.LockWaitP95Ms), 5),
			overshoot(float64(i.LockHolderCount), float64(h.LockHolderCount), 2),
		)
		downstreamScore := avgOvershoot(
			overshoot(float64(i.ExternalCallP95Ms), float64(h.ExternalCallP95Ms), 5),
			overshoot(float64(i.ExternalCallTimeMs)/float64(i.TotalRequestTimeMs), 0.5, 1),
		)
		return (lockScore + downstreamScore) / 2.0
	}

	return 1.0
}

// overshoot calculates how far a value exceeds its threshold (base * multiplier).
// Returns a normalized 0.0–1.0 score where 0.0 = at threshold, 1.0 = 2x or more above threshold.
func overshoot(actual, base, multiplier float64) float64 {
	threshold := base * multiplier
	if threshold <= 0 {
		if actual > 0 {
			return 1.0
		}
		return 0.0
	}
	ratio := actual / threshold
	if ratio <= 1.0 {
		return 0.0
	}
	// Normalize: ratio of 1.0 = 0.0, ratio of 3.0 = 1.0 (capped)
	return math.Min((ratio-1.0)/2.0, 1.0)
}

// avgOvershoot returns the average of overshoot values, shifted to the 0.5–1.0 range.
func avgOvershoot(values ...float64) float64 {
	if len(values) == 0 {
		return 0.5
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))
	return 0.5 + (avg * 0.5)
}
