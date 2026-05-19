package eval

import "time"

type Direction string

const (
	DirectionHigherIsBetter Direction = "higher_is_better"
	DirectionLowerIsBetter  Direction = "lower_is_better"
)

type Suite struct {
	Name          string                     `yaml:"suite"`
	BaselineRef   string                     `yaml:"baseline_ref"`
	BlockOnReg    bool                       `yaml:"block_on_regression"`
	Dimensions    map[string]DimensionConfig `yaml:"dimensions"`
	Adversarial   AdversarialConfig          `yaml:"adversarial"`
	Notifications NotificationConfig         `yaml:"notifications"`
}

type DimensionConfig struct {
	Threshold              float64   `yaml:"threshold"`
	DriftLimit             float64   `yaml:"drift_limit"`
	Direction              Direction `yaml:"direction"`
	BlockOnThresholdBreach bool      `yaml:"block_on_threshold_breach"`
}

type AdversarialConfig struct {
	Enabled       bool     `yaml:"enabled"`
	InjectionRate float64  `yaml:"injection_rate"`
	OWASPCoverage []string `yaml:"owasp_coverage"`
}

type NotificationConfig struct {
	SlackChannel  string `yaml:"slack_channel"`
	PostTraceLink bool   `yaml:"post_trace_link"`
}

type RunRequest struct {
	RunID      string
	SuiteRef   string
	PRNumber   int
	CommitSHA  string
	AgentImage string
	Metadata   map[string]string
}

type RunResult struct {
	RunID          string
	SuiteRef       string
	CommitSHA      string
	BaselineRef    string
	StartedAt      time.Time
	CompletedAt    time.Time
	TotalCases     int
	CompletedCases int
	Scores         map[string]DimensionScore
	BaselineScores map[string]float64
	TraceID        string
}

type DimensionScore struct {
	Value         float64
	PassCount     int
	FailCount     int
	ErrorCount    int
	ScorerVersion string
}

type CaseResult struct {
	CaseID    string
	TraceID   string
	Dimension string
	Score     float64
	Pass      bool
	Error     error
	Latency   time.Duration
}
