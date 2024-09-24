package types

type AutoscalerPolicy struct {
	Enabled                      bool               `mapstructure:"enabled" json:"enabled"`
	IsScopedMode                 bool               `mapstructure:"is_scoped_mode" json:"isScopedMode"`
	NodeTemplatesPartialMatching bool               `mapstructure:"node_templates_partial_matching_enabled" json:"nodeTemplatesPartialMatchingEnabled"`
	UnschedulablePods            *UnschedulablePods `mapstructure:"unschedulable_pods" json:"unschedulablePods,omitempty"`
	ClusterLimits                *ClusterLimits     `mapstructure:"cluster_limits" json:"clusterLimits,omitempty"`
	SpotInstances                *SpotInstances     `mapstructure:"spot_instances" json:"spotInstances,omitempty"`
	NodeDownscaler               *NodeDownscaler    `mapstructure:"node_downscaler" json:"nodeDownscaler,omitempty"`
}

type UnschedulablePods struct {
	Enabled         bool             `mapstructure:"enabled" json:"enabled"`
	Headroom        *Headroom        `mapstructure:"headroom" json:"headroom,omitempty"`
	HeadroomSpot    *Headroom        `mapstructure:"headroom_spot" json:"headroomSpot,omitempty"`
	NodeConstraints *NodeConstraints `mapstructure:"node_constraints" json:"nodeConstraints,omitempty"`
	CustomInstances bool             `mapstructure:"custom_instances_enabled" json:"customInstancesEnabled"`
	PodPinner       *PodPinner       `mapstructure:"pod_pinner" json:"podPinner,omitempty"`
}

type Headroom struct {
	CPUPercentage    int  `mapstructure:"cpu_percentage" json:"cpuPercentage"`
	MemoryPercentage int  `mapstructure:"memory_percentage" json:"memoryPercentage"`
	Enabled          bool `mapstructure:"enabled" json:"enabled"`
}

type NodeConstraints struct {
	MinCPUCores int  `mapstructure:"min_cpu_cores" json:"minCpuCores"`
	MaxCPUCores int  `mapstructure:"max_cpu_cores" json:"maxCpuCores"`
	MinRAMMiB   int  `mapstructure:"min_ram_mib" json:"minRamMiB"`
	MaxRAMMiB   int  `mapstructure:"max_ram_mib" json:"maxRamMiB"`
	Enabled     bool `mapstructure:"enabled" json:"enabled"`
}

type PodPinner struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
}

type ClusterLimits struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" `
	CPU     *CPU `mapstructure:"cpu" json:"cpu,omitempty"`
}

type CPU struct {
	MinCores int `mapstructure:"min_cores" json:"minCores"`
	MaxCores int `mapstructure:"max_cores" json:"maxCores"`
}

type SpotInstances struct {
	Enabled                     bool                         `mapstructure:"enabled" json:"enabled"`
	MaxReclaimRate              int                          `mapstructure:"max_reclaim_rate" json:"maxReclaimRate"`
	SpotBackups                 *SpotBackups                 `mapstructure:"spot_backups" json:"spotBackups,omitempty"`
	SpotDiversityEnabled        bool                         `mapstructure:"spot_diversity_enabled" json:"spotDiversityEnabled"`
	SpotDiversityPriceIncrease  int                          `mapstructure:"spot_diversity_price_increase_limit" json:"spotDiversityPriceIncrease"`
	SpotInterruptionPredictions *SpotInterruptionPredictions `mapstructure:"spot_interruption_predictions" json:"spotInterruptionPredictions,omitempty"`
}

type SpotBackups struct {
	Enabled                      bool `mapstructure:"enabled" json:"enabled"`
	SpotBackupRestoreRateSeconds int  `mapstructure:"spot_backup_restore_rate_seconds" json:"spotBackupRestoreRateSeconds"`
}

type SpotInterruptionPredictions struct {
	Enabled                         bool   `mapstructure:"enabled" json:"enabled"`
	SpotInterruptionPredictionsType string `mapstructure:"spot_interruption_predictions_type" json:"spotInterruptionPredictionsType"`
}

type NodeDownscaler struct {
	Enabled    bool        `mapstructure:"enabled" json:"enabled"`
	EmptyNodes *EmptyNodes `mapstructure:"empty_nodes" json:"emptyNodes,omitempty"`
	Evictor    *Evictor    `mapstructure:"evictor" json:"evictor,omitempty"`
}

type EmptyNodes struct {
	Enabled      bool `mapstructure:"enabled" json:"enabled"`
	DelaySeconds int  `mapstructure:"delay_seconds" json:"delaySeconds"`
}

type Evictor struct {
	Enabled                           bool   `mapstructure:"enabled" json:"enabled"`
	DryRun                            bool   `mapstructure:"dry_run" json:"dryRun"`
	AggressiveMode                    bool   `mapstructure:"aggressive_mode" json:"aggressiveMode"`
	ScopedMode                        bool   `mapstructure:"scoped_mode" json:"scopedMode"`
	CycleInterval                     string `mapstructure:"cycle_interval" json:"cycleInterval"`
	NodeGracePeriodMinutes            int    `mapstructure:"node_grace_period_minutes" json:"nodeGracePeriodMinutes"`
	PodEvictionFailureBackOffInterval string `mapstructure:"pod_eviction_failure_back_off_interval" json:"podEvictionFailureBackOffInterval"`
	IgnorePodDisruptionBudgets        bool   `mapstructure:"ignore_pod_disruption_budgets" json:"ignorePodDisruptionBudgets"`
}
