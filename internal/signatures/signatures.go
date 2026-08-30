package signatures

import "github.com/micro1-hackathon/rootcause/internal/models"

type SignatureResult struct {
	Passed          bool
	FailedCondition string
	MatchedFields   map[string]interface{}
}

type CheckFunc func(h, i models.Snapshot) SignatureResult

var Registry = map[string]CheckFunc{
	"n_plus_one_query":           CheckNPlusOne,
	"lock_contention":            CheckLockContention,
	"gc_pause":                   CheckGCPause,
	"connection_pool_exhaustion": CheckConnectionPoolExhaustion,
	"slow_downstream":            CheckSlowDownstream,
	"stale_cache":                CheckStaleCache,
	"thread_starvation":          CheckThreadStarvation,
	"disk_io_saturation":         CheckDiskIOSaturation,
	"memory_pressure":            CheckMemoryPressure,
	"network_retry_storm":        CheckNetworkRetryStorm,
	"pagination_bug":             CheckPaginationBug,
	"lock_and_downstream":        CheckLockAndDownstream,
}

func CheckNPlusOne(h, i models.Snapshot) SignatureResult {
	if i.DBCallCount <= h.DBCallCount*3 {
		return SignatureResult{Passed: false, FailedCondition: "db_call_count not > 3x baseline"}
	}
	if float64(i.AvgQueryTimeMs) >= float64(h.AvgQueryTimeMs)*1.5 {
		return SignatureResult{Passed: false, FailedCondition: "avg_query_time is too slow (not many small calls)"}
	}
	if float64(i.TotalDBTimeMs)/float64(i.TotalRequestTimeMs) <= 0.6 {
		return SignatureResult{Passed: false, FailedCondition: "total_db_time / total_request_time <= 0.6"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"db_call_count":     i.DBCallCount,
		"avg_query_time_ms": i.AvgQueryTimeMs,
		"db_time_ratio":     float64(i.TotalDBTimeMs) / float64(i.TotalRequestTimeMs),
	}}
}

func CheckLockContention(h, i models.Snapshot) SignatureResult {
	if i.LockWaitP95Ms <= h.LockWaitP95Ms*5 {
		return SignatureResult{Passed: false, FailedCondition: "lock_wait_p95_ms not > 5x baseline"}
	}
	if i.LockHolderCount <= h.LockHolderCount*2 {
		return SignatureResult{Passed: false, FailedCondition: "lock_holder_count not > 2x baseline"}
	}
	if !i.LockWaitSpike {
		return SignatureResult{Passed: false, FailedCondition: "lock_wait_spike not temporally correlated with latency"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"lock_wait_p95_ms":  i.LockWaitP95Ms,
		"lock_holder_count": i.LockHolderCount,
		"lock_wait_spike":   i.LockWaitSpike,
	}}
}

func CheckGCPause(h, i models.Snapshot) SignatureResult {
	if i.GCEventCount <= h.GCEventCount*3 {
		return SignatureResult{Passed: false, FailedCondition: "gc_event_count not > 3x baseline"}
	}
	if i.MaxGCPauseMs <= h.MaxGCPauseMs*4 {
		return SignatureResult{Passed: false, FailedCondition: "max_gc_pause_ms not > 4x baseline"}
	}
	if !i.GCLatencySpike {
		return SignatureResult{Passed: false, FailedCondition: "gc_timestamps not within 50ms of latency spike"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"gc_event_count":   i.GCEventCount,
		"max_gc_pause_ms":  i.MaxGCPauseMs,
		"gc_latency_spike": i.GCLatencySpike,
	}}
}

func CheckConnectionPoolExhaustion(h, i models.Snapshot) SignatureResult {
	if i.PoolActive < i.PoolMax {
		return SignatureResult{Passed: false, FailedCondition: "pool_active < pool_max (not at ceiling)"}
	}
	if i.PoolWaitQueue <= 0 {
		return SignatureResult{Passed: false, FailedCondition: "pool_wait_queue <= 0 (no queuing)"}
	}
	if i.CheckoutWaitP95Ms <= h.CheckoutWaitP95Ms*10 {
		return SignatureResult{Passed: false, FailedCondition: "checkout_wait_p95_ms not > 10x baseline"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"pool_active":          i.PoolActive,
		"pool_wait_queue":      i.PoolWaitQueue,
		"checkout_wait_p95_ms": i.CheckoutWaitP95Ms,
	}}
}

func CheckSlowDownstream(h, i models.Snapshot) SignatureResult {
	if i.ExternalCallP95Ms <= h.ExternalCallP95Ms*5 {
		return SignatureResult{Passed: false, FailedCondition: "external_call_p95_ms not > 5x baseline"}
	}
	if float64(i.ExternalCallTimeMs)/float64(i.TotalRequestTimeMs) <= 0.5 {
		return SignatureResult{Passed: false, FailedCondition: "external_call_time / total_request_time <= 0.5"}
	}
	if float64(i.LocalProcessingTimeMs) >= float64(h.LocalProcessingTimeMs)*1.5 {
		return SignatureResult{Passed: false, FailedCondition: "local_processing_time >= 1.5x baseline (local code is slow)"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"external_call_p95_ms":     i.ExternalCallP95Ms,
		"external_call_time_ratio": float64(i.ExternalCallTimeMs) / float64(i.TotalRequestTimeMs),
		"local_processing_time_ms": i.LocalProcessingTimeMs,
	}}
}

func CheckStaleCache(h, i models.Snapshot) SignatureResult {
	if i.CacheHitRate <= 0.80 {
		return SignatureResult{Passed: false, FailedCondition: "cache_hit_rate <= 0.80 (cache looks unhealthy, not a stale cache trick)"}
	}
	if i.DownstreamRetryRate <= h.DownstreamRetryRate*3 {
		return SignatureResult{Passed: false, FailedCondition: "downstream_retry_rate not > 3x baseline"}
	}
	if i.DataStalenessDeltaMs <= h.DataStalenessDeltaMs*5 {
		return SignatureResult{Passed: false, FailedCondition: "data_staleness_delta_ms not > 5x baseline"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"cache_hit_rate":          i.CacheHitRate,
		"downstream_retry_rate":   i.DownstreamRetryRate,
		"data_staleness_delta_ms": i.DataStalenessDeltaMs,
	}}
}

func CheckThreadStarvation(h, i models.Snapshot) SignatureResult {
	if i.CPUUtilization <= 0.85 {
		return SignatureResult{Passed: false, FailedCondition: "cpu_utilization <= 0.85 (not hitting absolute ceiling)"}
	}
	if i.IOWaitTimeMs <= h.IOWaitTimeMs*4 {
		return SignatureResult{Passed: false, FailedCondition: "io_wait_time_ms not > 4x baseline"}
	}
	if i.GoroutinesBlocked <= h.GoroutinesBlocked*5 {
		return SignatureResult{Passed: false, FailedCondition: "goroutines_blocked not > 5x baseline"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"cpu_utilization":    i.CPUUtilization,
		"io_wait_time_ms":    i.IOWaitTimeMs,
		"goroutines_blocked": i.GoroutinesBlocked,
	}}
}

func CheckDiskIOSaturation(h, i models.Snapshot) SignatureResult {
	if i.DiskQueueDepth <= h.DiskQueueDepth*4 {
		return SignatureResult{Passed: false, FailedCondition: "disk_queue_depth not > 4x baseline"}
	}
	if i.DiskAwaitMs <= h.DiskAwaitMs*3 {
		return SignatureResult{Passed: false, FailedCondition: "disk_await_ms not > 3x baseline"}
	}
	if i.CPUIOWait <= h.CPUIOWait*5 {
		return SignatureResult{Passed: false, FailedCondition: "cpu_iowait not > 5x baseline"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"disk_queue_depth": i.DiskQueueDepth,
		"disk_await_ms":    i.DiskAwaitMs,
		"cpu_iowait":       i.CPUIOWait,
	}}
}

func CheckMemoryPressure(h, i models.Snapshot) SignatureResult {
	if i.PageFaultRate <= h.PageFaultRate*10 {
		return SignatureResult{Passed: false, FailedCondition: "page_fault_rate not > 10x baseline"}
	}
	if i.SwapInRate <= 0 {
		return SignatureResult{Passed: false, FailedCondition: "swap_in_rate <= 0 (no swap activity)"}
	}
	if !i.PageFaultSpike {
		return SignatureResult{Passed: false, FailedCondition: "page_fault_timestamps not correlated with latency spike"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"page_fault_rate":  i.PageFaultRate,
		"swap_in_rate":     i.SwapInRate,
		"page_fault_spike": i.PageFaultSpike,
	}}
}

func CheckNetworkRetryStorm(h, i models.Snapshot) SignatureResult {
	if i.RetryRate <= h.RetryRate*3 {
		return SignatureResult{Passed: false, FailedCondition: "retry_rate not > 3x baseline"}
	}
	if i.DownstreamErrorRate <= h.DownstreamErrorRate*5 {
		return SignatureResult{Passed: false, FailedCondition: "downstream_error_rate not > 5x baseline"}
	}
	if i.ReqAmplification <= 2.0 {
		return SignatureResult{Passed: false, FailedCondition: "request_amplification_factor <= 2.0"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"retry_rate":                   i.RetryRate,
		"downstream_error_rate":        i.DownstreamErrorRate,
		"request_amplification_factor": i.ReqAmplification,
	}}
}

func CheckPaginationBug(h, i models.Snapshot) SignatureResult {
	if i.RowsFetched <= i.RowsDisplayed*10 {
		return SignatureResult{Passed: false, FailedCondition: "rows_fetched <= rows_displayed * 10 (not fetching wildly more than needed)"}
	}
	if i.DBPayloadSizeBytes <= h.DBPayloadSizeBytes*5 {
		return SignatureResult{Passed: false, FailedCondition: "db_payload_size_bytes not > 5x baseline"}
	}
	if i.ExecutionTimeMs <= h.ExecutionTimeMs*5 {
		return SignatureResult{Passed: false, FailedCondition: "execution_time_ms not > 5x baseline"}
	}
	return SignatureResult{Passed: true, MatchedFields: map[string]interface{}{
		"rows_fetched":          i.RowsFetched,
		"db_payload_size_bytes": i.DBPayloadSizeBytes,
		"execution_time_ms":     i.ExecutionTimeMs,
	}}
}

// CheckLockAndDownstream is a composite signature that fires only when both
// lock_contention AND slow_downstream pass independently. This handles
// multi-signal incidents where two root causes overlap.
func CheckLockAndDownstream(h, i models.Snapshot) SignatureResult {
	lockResult := CheckLockContention(h, i)
	if !lockResult.Passed {
		return SignatureResult{Passed: false, FailedCondition: "lock_contention sub-signature failed: " + lockResult.FailedCondition}
	}
	downstreamResult := CheckSlowDownstream(h, i)
	if !downstreamResult.Passed {
		return SignatureResult{Passed: false, FailedCondition: "slow_downstream sub-signature failed: " + downstreamResult.FailedCondition}
	}
	// Merge matched fields from both sub-signatures
	merged := map[string]interface{}{}
	for k, v := range lockResult.MatchedFields {
		merged["lock_"+k] = v
	}
	for k, v := range downstreamResult.MatchedFields {
		merged["downstream_"+k] = v
	}
	return SignatureResult{Passed: true, MatchedFields: merged}
}
