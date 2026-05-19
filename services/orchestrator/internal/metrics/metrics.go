package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	evalRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "eval_runs_total",
		Help: "Total eval runs by suite and outcome.",
	}, []string{"suite", "outcome"})

	evalRunDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "eval_run_duration_seconds",
		Help: "End-to-end eval run duration.", Buckets: []float64{30, 60, 120, 300, 600, 900, 1800},
	}, []string{"suite"})

	evalCasesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "eval_cases_total",
		Help: "Total eval cases by dimension and outcome.",
	}, []string{"dimension", "outcome"})

	workerQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "worker_queue_depth",
		Help: "Number of eval cases waiting for a worker slot.",
	})

	gateDecisionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "gate_decisions_total",
		Help: "Regression gate decisions by suite and state.",
	}, []string{"suite", "state"})

	grpcRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "leo", Subsystem: "orchestrator", Name: "grpc_requests_total",
		Help: "gRPC requests by method and status code.",
	}, []string{"method", "code"})
)

func Register() {
	prometheus.MustRegister(evalRunsTotal, evalRunDuration, evalCasesTotal,
		workerQueueDepth, gateDecisionsTotal, grpcRequestsTotal)
}

func Handler() http.HandlerFunc        { return promhttp.Handler().ServeHTTP }
func RecordEvalRun(suite, outcome string, d time.Duration) {
	evalRunsTotal.WithLabelValues(suite, outcome).Inc()
	evalRunDuration.WithLabelValues(suite).Observe(d.Seconds())
}
func RecordEvalCase(dimension, outcome string) { evalCasesTotal.WithLabelValues(dimension, outcome).Inc() }
func RecordGateDecision(suite, state string)   { gateDecisionsTotal.WithLabelValues(suite, state).Inc() }
func SetWorkerQueueDepth(d float64)            { workerQueueDepth.Set(d) }

func GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		grpcRequestsTotal.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
		return resp, err
	}
}
