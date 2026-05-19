package gate

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/your-org/leo/services/orchestrator/internal/eval"
)

// Decision is the gate outcome for a completed eval run.
type Decision struct {
	State      State
	Violations []Violation
}

type State string

const (
	StateOpen   State = "open"   // All checks passed. Merge is unblocked.
	StateClosed State = "closed" // One or more violations. Merge is blocked.
)

// Violation records a single failing gate check, with enough context to
// surface a useful PR comment and trace replay link.
type Violation struct {
	Dimension    string
	CurrentScore float64
	BaselineScore float64
	Delta        float64
	Reason       ViolationReason
	Threshold    float64
	DriftLimit   float64
}

func (v Violation) String() string {
	switch v.Reason {
	case ViolationDrift:
		return fmt.Sprintf("%s: delta %.2fpp exceeds drift limit %.2fpp (current=%.1f%%, baseline=%.1f%%)",
			v.Dimension, v.Delta*100, v.DriftLimit*100, v.CurrentScore*100, v.BaselineScore*100)
	case ViolationThreshold:
		return fmt.Sprintf("%s: score %.1f%% below threshold %.1f%%",
			v.Dimension, v.CurrentScore*100, v.Threshold*100)
	}
	return v.Dimension
}

type ViolationReason string

const (
	ViolationDrift     ViolationReason = "drift"
	ViolationThreshold ViolationReason = "threshold"
)

// Evaluator computes a gate decision from a completed eval run result
// and the suite configuration that governed it.
type Evaluator struct {
	logger *zap.Logger
}

func NewEvaluator(logger *zap.Logger) *Evaluator {
	return &Evaluator{logger: logger}
}

// Evaluate computes the gate decision. It is pure: no I/O, no side effects.
// The caller is responsible for storing the decision.
func (e *Evaluator) Evaluate(result *eval.RunResult, suite *eval.Suite) Decision {
	decision := Decision{State: StateOpen}

	for dimName, dimCfg := range suite.Dimensions {
		score, ok := result.Scores[dimName]
		if !ok {
			e.logger.Warn("missing score for configured dimension",
				zap.String("dimension", dimName),
				zap.String("run_id", result.RunID),
			)
			// A missing score for a configured dimension is treated as a block.
			// Silent omission would hide eval infrastructure failures.
			decision.Violations = append(decision.Violations, Violation{
				Dimension: dimName,
				Reason:    ViolationThreshold,
			})
			continue
		}

		baseline, hasBaseline := result.BaselineScores[dimName]

		if hasBaseline {
			delta := computeDelta(score.Value, baseline, dimCfg.Direction)
			if delta < -dimCfg.DriftLimit {
				decision.Violations = append(decision.Violations, Violation{
					Dimension:     dimName,
					CurrentScore:  score.Value,
					BaselineScore: baseline,
					Delta:         delta,
					Reason:        ViolationDrift,
					DriftLimit:    dimCfg.DriftLimit,
				})
			}
		}

		if dimCfg.BlockOnThresholdBreach && breachesThreshold(score.Value, dimCfg.Threshold, dimCfg.Direction) {
			decision.Violations = append(decision.Violations, Violation{
				Dimension:    dimName,
				CurrentScore: score.Value,
				Reason:       ViolationThreshold,
				Threshold:    dimCfg.Threshold,
			})
		}
	}

	if len(decision.Violations) > 0 {
		decision.State = StateClosed
		e.logger.Info("gate closed",
			zap.String("run_id", result.RunID),
			zap.Int("violation_count", len(decision.Violations)),
		)
	} else {
		e.logger.Info("gate open", zap.String("run_id", result.RunID))
	}

	return decision
}

// computeDelta returns the score change in the "better" direction.
// For higher_is_better, positive delta means improvement.
// For lower_is_better, positive delta means improvement (current < baseline).
func computeDelta(current, baseline float64, direction eval.Direction) float64 {
	switch direction {
	case eval.DirectionHigherIsBetter:
		return current - baseline
	case eval.DirectionLowerIsBetter:
		return baseline - current
	}
	return current - baseline
}

func breachesThreshold(score, threshold float64, direction eval.Direction) bool {
	switch direction {
	case eval.DirectionHigherIsBetter:
		return score < threshold
	case eval.DirectionLowerIsBetter:
		return score > threshold
	}
	return false
}
