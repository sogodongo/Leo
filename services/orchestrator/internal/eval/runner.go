package eval

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type Runner struct {
	cfg    RunnerConfig
	scorer ScorerClient
	logger *zap.Logger
}

type RunnerConfig struct {
	Concurrency int
	CaseTimeout time.Duration
	RunTimeout  time.Duration
}

type ScorerClient interface {
	Score(ctx context.Context, req ScoreRequest) (ScoreResponse, error)
}

type ScoreRequest struct {
	RunID     string
	CaseID    string
	TraceID   string
	Dimension string
	TraceData []byte
}

type ScoreResponse struct {
	Dimension string
	Score     float64
	Pass      bool
	Reason    string
}

func NewRunner(cfg RunnerConfig, scorer ScorerClient, logger *zap.Logger) *Runner {
	return &Runner{cfg: cfg, scorer: scorer, logger: logger}
}

func (r *Runner) Execute(ctx context.Context, req *RunRequest, cases []Case) (*RunResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, r.cfg.RunTimeout)
	defer cancel()

	result := &RunResult{
		RunID:      req.RunID,
		SuiteRef:   req.SuiteRef,
		CommitSHA:  req.CommitSHA,
		TotalCases: len(cases),
		StartedAt:  time.Now(),
		Scores:     make(map[string]DimensionScore),
	}

	sem := semaphore.NewWeighted(int64(r.cfg.Concurrency))
	var mu sync.Mutex
	var wg sync.WaitGroup
	caseResults := make([]CaseResult, 0, len(cases))

	for _, c := range cases {
		c := c
		if err := sem.Acquire(runCtx, 1); err != nil {
			r.logger.Warn("run context cancelled while acquiring worker slot",
				zap.String("run_id", req.RunID),
				zap.Error(err),
			)
			break
		}
		wg.Add(1)
		go func() {
			defer sem.Release(1)
			defer wg.Done()
			cr := r.runCase(runCtx, req.RunID, c)
			mu.Lock()
			caseResults = append(caseResults, cr)
			mu.Unlock()
		}()
	}

	wg.Wait()
	result.CompletedAt = time.Now()
	result.CompletedCases = len(caseResults)
	result.Scores = aggregateScores(caseResults)

	r.logger.Info("eval run complete",
		zap.String("run_id", req.RunID),
		zap.Int("total", result.TotalCases),
		zap.Int("completed", result.CompletedCases),
		zap.Duration("duration", result.CompletedAt.Sub(result.StartedAt)),
	)

	return result, nil
}

func (r *Runner) runCase(ctx context.Context, runID string, c Case) CaseResult {
	caseCtx, cancel := context.WithTimeout(ctx, r.cfg.CaseTimeout)
	defer cancel()

	start := time.Now()
	traceData, traceID, err := c.Execute(caseCtx)
	if err != nil {
		r.logger.Warn("case execution failed",
			zap.String("run_id", runID),
			zap.String("case_id", c.ID),
			zap.Error(err),
		)
		return CaseResult{CaseID: c.ID, Error: err, Latency: time.Since(start)}
	}

	scoreResp, err := r.scorer.Score(caseCtx, ScoreRequest{
		RunID:     runID,
		CaseID:    c.ID,
		TraceID:   traceID,
		Dimension: c.Dimension,
		TraceData: traceData,
	})
	if err != nil {
		return CaseResult{
			CaseID:  c.ID,
			TraceID: traceID,
			Error:   fmt.Errorf("scorer: %w", err),
			Latency: time.Since(start),
		}
	}

	return CaseResult{
		CaseID:    c.ID,
		TraceID:   traceID,
		Dimension: scoreResp.Dimension,
		Score:     scoreResp.Score,
		Pass:      scoreResp.Pass,
		Latency:   time.Since(start),
	}
}

func aggregateScores(results []CaseResult) map[string]DimensionScore {
	byDim := make(map[string][]CaseResult)
	for _, r := range results {
		byDim[r.Dimension] = append(byDim[r.Dimension], r)
	}
	scores := make(map[string]DimensionScore, len(byDim))
	for dim, cases := range byDim {
		var pass, fail, errs int
		var total float64
		for _, c := range cases {
			if c.Error != nil {
				errs++
				continue
			}
			total += c.Score
			if c.Pass {
				pass++
			} else {
				fail++
			}
		}
		scored := len(cases) - errs
		var avg float64
		if scored > 0 {
			avg = total / float64(scored)
		}
		scores[dim] = DimensionScore{
			Value:      avg,
			PassCount:  pass,
			FailCount:  fail,
			ErrorCount: errs,
		}
	}
	return scores
}

type Case struct {
	ID        string
	Dimension string
	Input     map[string]any
	Expected  map[string]any
	Execute   func(ctx context.Context) (traceData []byte, traceID string, err error)
}
