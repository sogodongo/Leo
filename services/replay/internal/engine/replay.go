package engine

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/your-org/leo/services/replay/internal/store"
)

type ReplayEngine struct {
	store  store.TraceStore
	logger *zap.Logger
}

func NewReplayEngine(s store.TraceStore, logger *zap.Logger) *ReplayEngine {
	return &ReplayEngine{store: s, logger: logger}
}

type ReplayResult struct {
	OriginalTraceID string
	ReplayTraceID   string
	Spans           []ReplayedSpan
	Diff            *TraceDiff
	Duration        time.Duration
}

type ReplayedSpan struct {
	SpanID       string
	Name         string
	Kind         string
	Attributes   map[string]any
	OriginalSpan *store.Span
	Matched      bool
}

type TraceDiff struct {
	SpanCountDelta   int
	NewSpans         []string
	MissingSpans     []string
	AttributeChanges []AttributeChange
}

type AttributeChange struct {
	SpanName  string
	Attribute string
	Original  any
	Replayed  any
}

func (e *ReplayEngine) Replay(ctx context.Context, traceID string) (*ReplayResult, error) {
	original, err := e.store.Load(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("load trace %s: %w", traceID, err)
	}

	start := time.Now()
	replayed := make([]ReplayedSpan, 0, len(original.Spans))

	for _, span := range original.Spans {
		rs := ReplayedSpan{
			SpanID:       fmt.Sprintf("rpl_%s", span.SpanID),
			Name:         span.Name,
			Kind:         span.Kind,
			Attributes:   make(map[string]any),
			OriginalSpan: &span,
		}
		for k, v := range span.Attributes {
			rs.Attributes[k] = v
		}
		if _, ok := span.Attributes["leo.tool.name"]; ok {
			rs.Attributes["leo.replay"] = true
		}
		rs.Matched = true
		replayed = append(replayed, rs)
	}

	e.logger.Info("replay complete",
		zap.String("trace_id", traceID),
		zap.Int("spans", len(replayed)),
		zap.Duration("duration", time.Since(start)),
	)

	return &ReplayResult{
		OriginalTraceID: traceID,
		ReplayTraceID:   "rpl_" + traceID,
		Spans:           replayed,
		Diff:            &TraceDiff{},
		Duration:        time.Since(start),
	}, nil
}
