package store

import (
	"context"
	"time"
)

type TraceStore interface {
	Load(ctx context.Context, traceID string) (*Trace, error)
	Save(ctx context.Context, trace *Trace) error
}

type Trace struct {
	TraceID     string            `json:"trace_id"`
	RunID       string            `json:"run_id"`
	CaseID      string            `json:"case_id"`
	ServiceName string            `json:"service_name"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Spans       []Span            `json:"spans"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Span struct {
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	Name         string         `json:"name"`
	Kind         string         `json:"kind"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	Attributes   map[string]any `json:"attributes,omitempty"`
	Events       []SpanEvent    `json:"events,omitempty"`
	StatusCode   string         `json:"status_code"`
	StatusMsg    string         `json:"status_message,omitempty"`
}

type SpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  time.Time      `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
