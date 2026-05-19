package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Version  string `envconfig:"LEO_VERSION"   default:"dev"`
	Env      string `envconfig:"LEO_ENV"       default:"development"`
	LogLevel string `envconfig:"LEO_LOG_LEVEL" default:"info"`
	HTTP     HTTPConfig
	DB       DBConfig
	S3       S3Config
	Eval     EvalConfig
	Gate     GateConfig
	ScorerAddr      string        `envconfig:"LEO_SCORER_ADDR"      required:"true"`
	ScorerTimeout   time.Duration `envconfig:"LEO_SCORER_TIMEOUT"   default:"5m"`
	DatasetAddr     string        `envconfig:"LEO_DATASET_ADDR"     required:"true"`
	AdversarialAddr string        `envconfig:"LEO_ADVERSARIAL_ADDR" required:"true"`
}

type HTTPConfig struct {
	Addr         string        `envconfig:"LEO_HTTP_ADDR"          default:":8080"`
	ReadTimeout  time.Duration `envconfig:"LEO_HTTP_READ_TIMEOUT"  default:"30s"`
	WriteTimeout time.Duration `envconfig:"LEO_HTTP_WRITE_TIMEOUT" default:"120s"`
}

type DBConfig struct {
	DSN          string        `envconfig:"LEO_DB_DSN"              required:"true"`
	MaxOpenConns int           `envconfig:"LEO_DB_MAX_OPEN_CONNS"   default:"20"`
	MaxIdleConns int           `envconfig:"LEO_DB_MAX_IDLE_CONNS"   default:"5"`
	ConnMaxLife  time.Duration `envconfig:"LEO_DB_CONN_MAX_LIFETIME" default:"1h"`
}

type S3Config struct {
	Endpoint  string `envconfig:"LEO_S3_ENDPOINT" required:"true"`
	Bucket    string `envconfig:"LEO_S3_BUCKET"   required:"true"`
	Region    string `envconfig:"LEO_S3_REGION"   default:"us-east-1"`
	AccessKey string `envconfig:"LEO_S3_ACCESS_KEY"`
	SecretKey string `envconfig:"LEO_S3_SECRET_KEY"`
}

type EvalConfig struct {
	WorkerConcurrency int           `envconfig:"LEO_EVAL_WORKER_CONCURRENCY" default:"12"`
	CaseTimeout       time.Duration `envconfig:"LEO_EVAL_CASE_TIMEOUT"       default:"3m"`
	RunTimeout        time.Duration `envconfig:"LEO_EVAL_RUN_TIMEOUT"        default:"60m"`
}

type GateConfig struct {
	FailClosedOnInfraError bool `envconfig:"LEO_GATE_FAIL_CLOSED_ON_INFRA_ERROR" default:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Eval.WorkerConcurrency < 1 || c.Eval.WorkerConcurrency > 500 {
		return fmt.Errorf("LEO_EVAL_WORKER_CONCURRENCY must be 1-500, got %d", c.Eval.WorkerConcurrency)
	}
	return nil
}
