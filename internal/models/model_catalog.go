package models

// ModelSpec describes a recommended model and its derived metadata.
type ModelSpec struct {
	Name                 string   `json:"name" yaml:"name"`
	Vendor               string   `json:"vendor" yaml:"vendor"`
	Family               string   `json:"family" yaml:"family"`
	TotalParamsB         float64  `json:"total_params_b" yaml:"total_params_b"`
	ActiveParamsB        float64  `json:"active_params_b" yaml:"active_params_b"`
	Precision            string   `json:"precision" yaml:"precision"`
	Instruct             bool     `json:"instruct" yaml:"instruct"`
	Interactivity        string   `json:"interactivity" yaml:"interactivity"`
	ParallelizationHints []string `json:"parallelization_hints" yaml:"parallelization_hints"`
	MinVRAMGB            int      `json:"min_vram_gb" yaml:"min_vram_gb"`
	SuggestedGPUClass    string   `json:"suggested_gpu_class" yaml:"suggested_gpu_class"`
	Rank                 int      `json:"rank" yaml:"rank"`
}
