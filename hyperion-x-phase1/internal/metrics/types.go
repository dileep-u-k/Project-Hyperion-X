package metrics

// GpuMetrics holds detailed information for a single GPU.
type GpuMetrics struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	UtilizationPct uint32 `json:"utilization_pct"`
	MemoryUsedMb   uint64 `json:"memory_used_mb"`
	MemoryTotalMb  uint64 `json:"memory_total_mb"`
	TemperatureC   uint32 `json:"temperature_c"`
}

// NodeMetrics now includes a slice of detailed GPU metrics.
type NodeMetrics struct {
	NodeName    string       `json:"node_name"`
	CPUUsagePct float64      `json:"cpu_usage_pct"`
	MemUsagePct float64      `json:"mem_usage_pct"`
	Gpus        []GpuMetrics `json:"gpus"`
}
