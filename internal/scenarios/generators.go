package scenarios

import (
	"github.com/micro1-hackathon/rootcause/internal/models"
)

// GenerateAll returns all 13 scenarios and their ground truths
func GenerateAll() ([]models.ScenarioData, []models.GroundTruth) {
	var scenarios []models.ScenarioData
	var truths []models.GroundTruth

	add := func(s models.ScenarioData, actualCause, injectedNoise string) {
		scenarios = append(scenarios, s)
		truths = append(truths, models.GroundTruth{
			ScenarioID:    s.ScenarioID,
			ActualCause:   actualCause,
			InjectedNoise: injectedNoise,
		})
	}

	// 01: N+1 Query
	s1 := models.ScenarioData{
		ScenarioID:  "01",
		Description: "N+1 Query Issue",
		Healthy:     DefaultHealthySnapshot(),
	}
	s1.Incident = s1.Healthy
	s1.Incident.TotalRequestTimeMs = 400
	s1.Incident.DBCallCount = 47
	s1.Incident.AvgQueryTimeMs = 6
	s1.Incident.TotalDBTimeMs = 282
	s1.Incident.ExternalCallP95Ms = 120   // noise
	s1.Incident.ExternalCallTimeMs = 120  // noise
	add(s1, "n_plus_one_query", "slightly slow external call")

	// 02: Lock Contention
	s2 := models.ScenarioData{
		ScenarioID:  "02",
		Description: "Lock Contention",
		Healthy:     DefaultHealthySnapshot(),
	}
	s2.Incident = s2.Healthy
	s2.Incident.TotalRequestTimeMs = 500
	s2.Incident.LockWaitP95Ms = 340
	s2.Incident.LockHolderCount = 7
	s2.Incident.LockWaitSpike = true
	s2.Incident.MaxGCPauseMs = 35 // noise
	s2.Incident.GCEventCount = 2  // noise
	add(s2, "lock_contention", "mildly elevated GC pauses")

	// 03: GC Pause
	s3 := models.ScenarioData{
		ScenarioID:  "03",
		Description: "GC Pause",
		Healthy:     DefaultHealthySnapshot(),
	}
	s3.Incident = s3.Healthy
	s3.Incident.TotalRequestTimeMs = 450
	s3.Incident.GCEventCount = 8
	s3.Incident.MaxGCPauseMs = 180
	s3.Incident.GCLatencySpike = true
	s3.Incident.CheckoutWaitP95Ms = 15 // noise
	add(s3, "gc_pause", "elevated pool wait")

	// 04: Connection Pool Exhaustion
	s4 := models.ScenarioData{
		ScenarioID:  "04",
		Description: "Connection Pool Exhaustion",
		Healthy:     DefaultHealthySnapshot(),
	}
	s4.Incident = s4.Healthy
	s4.Incident.TotalRequestTimeMs = 300
	s4.Incident.PoolActive = 100
	s4.Incident.PoolWaitQueue = 12
	s4.Incident.CheckoutWaitP95Ms = 120
	s4.Incident.DiskQueueDepth = 3 // noise
	s4.Incident.DiskAwaitMs = 5    // noise
	add(s4, "connection_pool_exhaustion", "slightly slow disk")

	// 05: Slow Downstream
	s5 := models.ScenarioData{
		ScenarioID:  "05",
		Description: "Slow Downstream",
		Healthy:     DefaultHealthySnapshot(),
	}
	s5.Incident = s5.Healthy
	s5.Incident.TotalRequestTimeMs = 1000
	s5.Incident.ExternalCallP95Ms = 820
	s5.Incident.ExternalCallTimeMs = 820
	s5.Incident.LocalProcessingTimeMs = 40
	s5.Incident.DBCallCount = 10 // noise
	add(s5, "slow_downstream", "minor N+1 pattern")

	// 06: Stale Cache
	s6 := models.ScenarioData{
		ScenarioID:  "06",
		Description: "Stale Cache",
		Healthy:     DefaultHealthySnapshot(),
	}
	s6.Incident = s6.Healthy
	s6.Incident.TotalRequestTimeMs = 300
	s6.Incident.CacheHitRate = 0.91
	s6.Incident.DownstreamRetryRate = 0.05
	s6.Incident.DataStalenessDeltaMs = 3500
	s6.Incident.CPUUtilization = 0.58 // noise
	add(s6, "stale_cache", "CPU slightly elevated")

	// 07: Thread Starvation
	s7 := models.ScenarioData{
		ScenarioID:  "07",
		Description: "Thread Starvation",
		Healthy:     DefaultHealthySnapshot(),
	}
	s7.Incident = s7.Healthy
	s7.Incident.TotalRequestTimeMs = 600
	s7.Incident.CPUUtilization = 0.92
	s7.Incident.IOWaitTimeMs = 35
	s7.Incident.GoroutinesBlocked = 28
	s7.Incident.LockWaitP95Ms = 20 // noise
	add(s7, "thread_starvation", "Lock wait slightly elevated")

	// 08: Disk I/O Saturation
	s8 := models.ScenarioData{
		ScenarioID:  "08",
		Description: "Disk I/O Saturation",
		Healthy:     DefaultHealthySnapshot(),
	}
	s8.Incident = s8.Healthy
	s8.Incident.TotalRequestTimeMs = 800
	s8.Incident.DiskQueueDepth = 14
	s8.Incident.DiskAwaitMs = 31
	s8.Incident.CPUIOWait = 0.35
	s8.Incident.PageFaultRate = 2.0 // noise
	add(s8, "disk_io_saturation", "Memory slightly elevated")

	// 09: Memory Pressure
	s9 := models.ScenarioData{
		ScenarioID:  "09",
		Description: "Memory Pressure",
		Healthy:     DefaultHealthySnapshot(),
	}
	s9.Incident = s9.Healthy
	s9.Incident.TotalRequestTimeMs = 700
	s9.Incident.PageFaultRate = 12.5
	s9.Incident.SwapInRate = 0.5
	s9.Incident.PageFaultSpike = true
	s9.Incident.MaxGCPauseMs = 30 // noise
	add(s9, "memory_pressure", "GC slightly elevated")

	// 10: Network Retry Storm
	s10 := models.ScenarioData{
		ScenarioID:  "10",
		Description: "Network Retry Storm",
		Healthy:     DefaultHealthySnapshot(),
	}
	s10.Incident = s10.Healthy
	s10.Incident.TotalRequestTimeMs = 1000
	s10.Incident.RetryRate = 0.04
	s10.Incident.DownstreamErrorRate = 0.02
	s10.Incident.ReqAmplification = 3.1
	s10.Incident.ExternalCallP95Ms = 600   // noise
	s10.Incident.ExternalCallTimeMs = 600  // noise
	s10.Incident.LocalProcessingTimeMs = 180 // makes downstream signature fail
	add(s10, "network_retry_storm", "Slow downstream caused by storm")

	// 11: Off-by-One Pagination
	s11 := models.ScenarioData{
		ScenarioID:  "11",
		Description: "Off-by-One Pagination",
		Healthy:     DefaultHealthySnapshot(),
	}
	s11.Incident = s11.Healthy
	s11.Incident.TotalRequestTimeMs = 800
	s11.Incident.RowsFetched = 4800
	s11.Incident.RowsDisplayed = 20
	s11.Incident.DBPayloadSizeBytes = 50000
	s11.Incident.ExecutionTimeMs = 800
	s11.Incident.AvgQueryTimeMs = 300 // noise
	s11.Incident.TotalDBTimeMs = 600
	add(s11, "pagination_bug", "Slow overall DB time")

	// 12: Clean (No Bottleneck)
	s12 := models.ScenarioData{
		ScenarioID:  "12",
		Description: "Clean - No Bottleneck",
		Healthy:     DefaultHealthySnapshot(),
	}
	s12.Incident = s12.Healthy
	s12.Incident.TotalRequestTimeMs = 120 // very normal
	add(s12, "none", "none")

	// 13: HARD: Lock + Downstream
	s13 := models.ScenarioData{
		ScenarioID:  "13",
		Description: "HARD: Lock Contention AND Slow Downstream",
		Healthy:     DefaultHealthySnapshot(),
	}
	s13.Incident = s13.Healthy
	s13.Incident.TotalRequestTimeMs = 1000
	s13.Incident.LockWaitP95Ms = 220
	s13.Incident.LockHolderCount = 5
	s13.Incident.LockWaitSpike = true
	s13.Incident.ExternalCallP95Ms = 600
	s13.Incident.ExternalCallTimeMs = 600
	s13.Incident.LocalProcessingTimeMs = 30
	add(s13, "lock_and_downstream", "Both are genuinely failing")

	return scenarios, truths
}
