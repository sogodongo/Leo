# ADR 002: Scorer determinism and trace replay

**Status:** Accepted
**Date:** 2024-01

## Context

For trace replay to be useful, re-running a scorer against a stored trace
must produce the same score as the original run. Two sources of non-determinism
exist in LLM-as-judge scorers: temperature > 0 and time-varying tool results.

## Decision

Two mechanisms enforce determinism:

**1. Temperature = 0 on all judge calls, with trace_id seeding**

The trace ID is embedded in the judge prompt. Same trace ID = same prompt =
same completion at temperature=0.

**2. Mocked tool results in replay**

The replay engine never re-executes tool calls. It reads stored tool results
from the original trace and injects them. The agent sees identical tool results
on every replay.

## Consequences

- LLM judge scores are deterministic per trace ID
- If the judge model is updated, scorer_version is recorded so shifts are detectable
- Tool call replay uses stored results — external API changes do not affect scores

## Alternatives considered

**Accept non-determinism, use statistical aggregation:** Rejected — multiplies
LLM cost by N with no meaningful benefit over temperature=0.
