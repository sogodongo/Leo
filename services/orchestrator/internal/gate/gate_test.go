package gate_test

import (
	"testing"
	"go.uber.org/zap/zaptest"
	"github.com/your-org/leo/services/orchestrator/internal/eval"
	"github.com/your-org/leo/services/orchestrator/internal/gate"
)

func TestGate_AllPass(t *testing.T) {
	ev := gate.NewEvaluator(zaptest.NewLogger(t))
	result := &eval.RunResult{RunID: "r1",
		Scores:         map[string]eval.DimensionScore{"tool_use_correctness": {Value: 0.94}},
		BaselineScores: map[string]float64{"tool_use_correctness": 0.92},
	}
	suite := &eval.Suite{Dimensions: map[string]eval.DimensionConfig{
		"tool_use_correctness": {Threshold: 0.90, DriftLimit: 0.03, Direction: eval.DirectionHigherIsBetter},
	}}
	d := ev.Evaluate(result, suite)
	if d.State != gate.StateOpen { t.Errorf("want open, got %s", d.State) }
}

func TestGate_DriftBlocks(t *testing.T) {
	ev := gate.NewEvaluator(zaptest.NewLogger(t))
	result := &eval.RunResult{RunID: "r2",
		Scores:         map[string]eval.DimensionScore{"hallucination_rate": {Value: 0.14}},
		BaselineScores: map[string]float64{"hallucination_rate": 0.08},
	}
	suite := &eval.Suite{Dimensions: map[string]eval.DimensionConfig{
		"hallucination_rate": {Threshold: 0.10, DriftLimit: 0.03, Direction: eval.DirectionLowerIsBetter},
	}}
	d := ev.Evaluate(result, suite)
	if d.State != gate.StateClosed { t.Errorf("want closed, got %s", d.State) }
}

func TestGate_ThresholdBreachBlocks(t *testing.T) {
	ev := gate.NewEvaluator(zaptest.NewLogger(t))
	result := &eval.RunResult{RunID: "r3",
		Scores:         map[string]eval.DimensionScore{"policy_compliance": {Value: 0.95}},
		BaselineScores: map[string]float64{"policy_compliance": 0.96},
	}
	suite := &eval.Suite{Dimensions: map[string]eval.DimensionConfig{
		"policy_compliance": {Threshold: 0.98, DriftLimit: 0.03, Direction: eval.DirectionHigherIsBetter, BlockOnThresholdBreach: true},
	}}
	d := ev.Evaluate(result, suite)
	if d.State != gate.StateClosed { t.Errorf("want closed on threshold breach, got %s", d.State) }
}

func TestGate_ImprovementDoesNotBlock(t *testing.T) {
	ev := gate.NewEvaluator(zaptest.NewLogger(t))
	result := &eval.RunResult{RunID: "r4",
		Scores:         map[string]eval.DimensionScore{"tool_use_correctness": {Value: 0.97}},
		BaselineScores: map[string]float64{"tool_use_correctness": 0.92},
	}
	suite := &eval.Suite{Dimensions: map[string]eval.DimensionConfig{
		"tool_use_correctness": {Threshold: 0.90, DriftLimit: 0.03, Direction: eval.DirectionHigherIsBetter},
	}}
	d := ev.Evaluate(result, suite)
	if d.State != gate.StateOpen { t.Errorf("improvement should not block, got %s", d.State) }
}
