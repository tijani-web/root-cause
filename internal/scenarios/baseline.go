package scenarios

import (
	"github.com/micro1-hackathon/rootcause/internal/models"
)

// DefaultHealthySnapshot returns a standard healthy baseline snapshot.
func DefaultHealthySnapshot() models.Snapshot {
	return models.Snapshot{
		TotalRequestTimeMs:    100,
		ExecutionTimeMs:       100,
		DBCallCount:           4,
		AvgQueryTimeMs:        5,
		TotalDBTimeMs:         20,
		DBPayloadSizeBytes:    2048,
		RowsFetched:           10,
		RowsDisplayed:         10,
		LockWaitP95Ms:         5,
		LockHolderCount:       1,
		LockWaitSpike:         false,
		GCEventCount:          1,
		MaxGCPauseMs:          10,
		GCLatencySpike:        false,
		PoolActive:            20,
		PoolMax:               100,
		PoolWaitQueue:         0,
		CheckoutWaitP95Ms:     2,
		ExternalCallP95Ms:     40,
		ExternalCallTimeMs:    40,
		LocalProcessingTimeMs: 40,
		DownstreamRetryRate:   0.01,
		DownstreamErrorRate:   0.001,
		CacheHitRate:          0.95,
		DataStalenessDeltaMs:  500,
		CPUUtilization:        0.30,
		IOWaitTimeMs:          5,
		GoroutinesBlocked:     2,
		DiskQueueDepth:        1,
		DiskAwaitMs:           2,
		CPUIOWait:             0.02,
		PageFaultRate:         0.5,
		SwapInRate:            0.0,
		PageFaultSpike:        false,
		RetryRate:             0.01,
		ReqAmplification:      1.0,
	}
}
