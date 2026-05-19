package eval_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
	"github.com/your-org/leo/services/orchestrator/internal/eval"
)

type fakeScorer struct {
	calls    atomic.Int64
	response eval.ScoreResponse
	err      error
	delay    time.Duration
}

func (f *fakeScorer) Score(ctx context.Context, req eval.ScoreRequest) (eval.ScoreResponse, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return eval.ScoreResponse{}, ctx.Err()
		}
	}
	return f.response, f.err
}

func makeCase(id, dim string) eval.Case {
	return eval.Case{ID: id, Dimension: dim, Execute: func(ctx context.Context) ([]byte, string, error) {
		return []byte(`{}`), "trc_" + id, nil
	}}
}

func makeCaseWithError(id, dim string) eval.Case {
	return eval.Case{ID: id, Dimension: dim, Execute: func(ctx context.Context) ([]byte, string, error) {
		return nil, "", errors.New("agent execution failed")
	}}
}

func makeCaseWithDelay(id, dim string, d time.Duration) eval.Case {
	return eval.Case{ID: id, Dimension: dim, Execute: func(ctx context.Context) ([]byte, string, error) {
		select {
		case <-time.After(d): return []byte(`{}`), "trc_" + id, nil
		case <-ctx.Done():    return nil, "", ctx.Err()
		}
	}}
}

func TestRunner_AllPass(t *testing.T) {
	scorer := &fakeScorer{response: eval.ScoreResponse{Dimension: "tool_use_correctness", Score: 0.95, Pass: true}}
	runner := eval.NewRunner(eval.RunnerConfig{Concurrency: 4, CaseTimeout: 5 * time.Second, RunTimeout: 30 * time.Second}, scorer, zaptest.NewLogger(t))
	req := &eval.RunRequest{RunID: "run-001", SuiteRef: "test", CommitSHA: "abc"}
	result, err := runner.Execute(context.Background(), req, []eval.Case{makeCase("c1", "tool_use_correctness"), makeCase("c2", "tool_use_correctness")})
	if err != nil { t.Fatal(err) }
	if result.CompletedCases != 2 { t.Errorf("want 2 completed, got %d", result.CompletedCases) }
	if scorer.calls.Load() != 2 { t.Errorf("want 2 scorer calls, got %d", scorer.calls.Load()) }
}

func TestRunner_CaseError(t *testing.T) {
	scorer := &fakeScorer{response: eval.ScoreResponse{Score: 1.0, Pass: true}}
	runner := eval.NewRunner(eval.RunnerConfig{Concurrency: 4, CaseTimeout: 5 * time.Second, RunTimeout: 30 * time.Second}, scorer, zaptest.NewLogger(t))
	req := &eval.RunRequest{RunID: "run-002", SuiteRef: "test", CommitSHA: "abc"}
	result, err := runner.Execute(context.Background(), req, []eval.Case{makeCase("ok", "dim"), makeCaseWithError("err", "dim")})
	if err != nil { t.Fatal(err) }
	if scorer.calls.Load() != 1 { t.Errorf("want 1 scorer call, got %d", scorer.calls.Load()) }
	if result.Scores["dim"].ErrorCount != 1 { t.Errorf("want 1 error, got %d", result.Scores["dim"].ErrorCount) }
}

func TestRunner_ConcurrencyRespected(t *testing.T) {
	var concurrent, maxConcurrent atomic.Int64
	scorer := &fakeScorer{response: eval.ScoreResponse{Score: 1.0, Pass: true}}
	runner := eval.NewRunner(eval.RunnerConfig{Concurrency: 3, CaseTimeout: 5 * time.Second, RunTimeout: 30 * time.Second}, scorer, zaptest.NewLogger(t))
	cases := make([]eval.Case, 10)
	for i := range cases {
		i := i
		cases[i] = eval.Case{ID: "c" + string(rune('0'+i)), Dimension: "d", Execute: func(ctx context.Context) ([]byte, string, error) {
			cur := concurrent.Add(1)
			defer concurrent.Add(-1)
			for { old := maxConcurrent.Load(); if cur <= old || maxConcurrent.CompareAndSwap(old, cur) { break } }
			time.Sleep(10 * time.Millisecond)
			return []byte(`{}`), "trc", nil
		}}
	}
	req := &eval.RunRequest{RunID: "run-003", SuiteRef: "test", CommitSHA: "abc"}
	if _, err := runner.Execute(context.Background(), req, cases); err != nil { t.Fatal(err) }
	if maxConcurrent.Load() > 3 { t.Errorf("concurrency exceeded limit: peak=%d limit=3", maxConcurrent.Load()) }
}
