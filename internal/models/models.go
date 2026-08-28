package models

// ScenarioData holds the complete scenario with healthy and incident snapshots
type ScenarioData struct {
	ScenarioID  string   `json:"scenario_id"`
	Description string   `json:"description"`
	Healthy     Snapshot `json:"healthy"`
	Incident    Snapshot `json:"incident"`
}

// Snapshot represents a moment in time (like an OTel trace/metrics snapshot)
type Snapshot struct {
	// General
	TotalRequestTimeMs int `json:"total_request_time_ms"`
	ExecutionTimeMs    int `json:"execution_time_ms"`

	// Database Metrics
	DBCallCount        int `json:"db_call_count"`
	AvgQueryTimeMs     int `json:"avg_query_time_ms"`
	TotalDBTimeMs      int `json:"total_db_time_ms"`
	DBPayloadSizeBytes int `json:"db_payload_size_bytes"`
	RowsFetched        int `json:"rows_fetched"`
	RowsDisplayed      int `json:"rows_displayed"`

	// Lock Metrics
	LockWaitP95Ms   int  `json:"lock_wait_p95_ms"`
	LockHolderCount int  `json:"lock_holder_count"`
	LockWaitSpike   bool `json:"lock_wait_spike_correlated"` // true if correlated with latency

	// GC Metrics
	GCEventCount   int  `json:"gc_event_count"`
	MaxGCPauseMs   int  `json:"max_gc_pause_ms"`
	GCLatencySpike bool `json:"gc_latency_spike_correlated"`

	// Connection Pool Metrics
	PoolActive        int `json:"pool_active"`
	PoolMax           int `json:"pool_max"`
	PoolWaitQueue     int `json:"pool_wait_queue"`
	CheckoutWaitP95Ms int `json:"checkout_wait_p95_ms"`

	// External Dependency Metrics
	ExternalCallP95Ms     int     `json:"external_call_p95_ms"`
	ExternalCallTimeMs    int     `json:"external_call_time_ms"`
	LocalProcessingTimeMs int     `json:"local_processing_time_ms"`
	DownstreamRetryRate   float64 `json:"downstream_retry_rate"`
	DownstreamErrorRate   float64 `json:"downstream_error_rate"`

	// Cache Metrics
	CacheHitRate         float64 `json:"cache_hit_rate"`
	DataStalenessDeltaMs int     `json:"data_staleness_delta_ms"`

	// CPU/Thread Metrics
	CPUUtilization    float64 `json:"cpu_utilization"`
	IOWaitTimeMs      int     `json:"io_wait_time_ms"`
	GoroutinesBlocked int     `json:"goroutines_blocked"`

	// Disk Metrics
	DiskQueueDepth int     `json:"disk_queue_depth"`
	DiskAwaitMs    int     `json:"disk_await_ms"`
	CPUIOWait      float64 `json:"cpu_iowait"`

	// Memory Metrics
	PageFaultRate  float64 `json:"page_fault_rate"`
	SwapInRate     float64 `json:"swap_in_rate"`
	PageFaultSpike bool    `json:"page_fault_spike_correlated"`

	// Network / Request Metrics
	RetryRate        float64 `json:"retry_rate"`
	ReqAmplification float64 `json:"request_amplification_factor"`
}
