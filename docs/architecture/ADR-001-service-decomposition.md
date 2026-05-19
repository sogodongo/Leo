# ADR 001: Service decomposition

**Status:** Accepted
**Date:** 2024-01

## Context

We needed to decide how to split Leo's capabilities across services.

## Decision

Five services: orchestrator, scorer, replay, adversarial, datasets.

The driving constraint was language fit. Scoring requires Python for
DeepEval and the ML ecosystem. Orchestration benefits from Go's goroutine
model for the worker pool. Mixing these into one process means losing
either the Python ML ecosystem or Go's concurrency model.

The second constraint was independent scalability. The scorer is the
bottleneck — LLM judge calls are slow. We need to scale scorer replicas
without scaling the orchestrator.

## Consequences

- Service-to-service calls add ~1ms latency per hop (negligible vs LLM judge latency of 2-10s)
- Each service owns its own schema subset and migrates independently
- The gRPC interface between orchestrator and scorer is a contract boundary

## Alternatives considered

**Monolith with language bindings:** Go calling Python via subprocess.
Rejected — subprocess management is fragile; gRPC is a cleaner boundary.

**Event-driven architecture:** Kafka between services. On the roadmap
for async batch eval modes. Current synchronous gRPC is simpler and
sufficient for current throughput.
