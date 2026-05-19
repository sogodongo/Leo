# Leo

CI-native evaluation and regression-control platform for AI agents.

Leo continuously evaluates agent reliability, tool-use correctness, hallucination behavior, grounding quality, policy compliance, and adversarial robustness. It blocks deployments when agent quality regresses.

```
$ leo eval run --suite evals/suites/agent-v2.yaml --pr 847 --block-on-regression

→ Loaded 142 test cases
→ Spawning eval workers (parallelism=12)

  ┌─ Results ───────────────────────────────────────────────────────────
  │ ✓ tool_use_correctness    94.2%  +1.1pp vs baseline
  │ ✓ grounding_score         87.8%  +0.3pp vs baseline
  │ ✗ hallucination_rate      12.1%  +3.4pp vs baseline  ← REGRESSION
  │ ✓ policy_compliance       99.1%  (threshold: 98.0%)
  │ ✓ adversarial_resistance  91.0%  OWASP ASI01–ASI10
  └──────────────────────────────────────────────────────────────────────

✗ MERGE BLOCKED  1 regression detected. Deployment gate: CLOSED.
  Trace ID: trc_01HZXQP84JBV6K3NDFQ7W2MRC8
  Replay:   leo trace replay trc_01HZXQP84JBV6K3NDFQ7W2MRC8
```

---

## What Leo is

Leo is not a benchmarking app, a prompt playground, or an agent framework. It is an evaluation infrastructure layer that sits between your agent codebase and production. The core claim: **no agent reaches production without passing its eval suite**.

This is an engineering problem, not a research problem. Leo treats agent quality the same way a mature engineering organization treats software reliability — with CI gates, regression history, observability, and on-call runbooks.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  CI Layer                                                       │
│  ┌──────────────────────────────────┐  ┌──────────────────────┐│
│  │  GitHub Actions / leo CLI        │  │  Leo SDK             ││
│  │  PR hooks · merge gate           │  │  Trace instrumentation││
│  └──────────────────┬───────────────┘  └──────────┬───────────┘│
└─────────────────────┼────────────────────────────┼─────────────┘
                      │ gRPC                        │ OTLP
┌─────────────────────▼────────────────────────────▼─────────────┐
│  Orchestration Layer                                            │
│  ┌──────────────────────┐   ┌──────────────────────────────────┐│
│  │  Orchestrator (Go)   │   │  Dataset Service (Go)            ││
│  │  Run scheduling      │   │  Versioned test cases            ││
│  │  Worker pool mgmt    │   │  Sampling strategies             ││
│  │  Gate decisions      │   │  Lineage tracking                ││
│  └──────────┬───────────┘   └──────────────────────────────────┘│
└─────────────┼───────────────────────────────────────────────────┘
              │
┌─────────────▼───────────────────────────────────────────────────┐
│  Execution Layer                                                │
│  ┌──────────────────────┐   ┌──────────────────────────────────┐│
│  │  Scorer (Python)     │   │  Adversarial Engine (Python)     ││
│  │  10 pluggable dims   │   │  OWASP ASI01–ASI10               ││
│  │  LLM-as-judge        │   │  Jailbreak synthesis             ││
│  │  DeepEval + rules    │   │  Injection catalog               ││
│  └──────────────────────┘   └──────────────────────────────────┘│
│  ┌──────────────────────────────────────────────────────────────┐│
│  │  Trace Replay (Go + Python)                                  ││
│  │  Deterministic replay · Span diffing · Regression pinpointing││
│  └──────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
              │
┌─────────────▼───────────────────────────────────────────────────┐
│  Observability + Storage                                        │
│  Traces → Tempo   Metrics → Prometheus → Grafana   Logs → Loki  │
│  Eval runs + scores + history → Postgres                        │
│  Raw traces + artifacts → S3-compatible object storage          │
│  Dashboard → Next.js                                            │
└─────────────────────────────────────────────────────────────────┘
```

### Service boundaries

Each service has one clear owner. No shared util packages. No cross-service imports.

| Service | Language | Responsibility |
|---|---|---|
| `orchestrator` | Go | Eval run lifecycle. Worker pool. Gate decisions. |
| `scorer` | Python | 10 eval dimensions. LLM judge. Deterministic scorers. |
| `replay` | Go + Python | Trace storage. Deterministic replay. Span diffing. |
| `adversarial` | Python | OWASP ASI attack generation. Case versioning. |
| `datasets` | Go | Versioned test cases. Sampling. Lineage. |
| `dashboard` | TypeScript | Read-only UI. Score trends. Trace viewer. |

---

## Eval dimensions

Leo scores agents across 10 independently configurable dimensions. Each dimension has its own scorer, threshold, drift limit, and block policy.

| Dimension | Scorer type | Default threshold |
|---|---|---|
| `tool_use_correctness` | Deterministic | ≥ 90% |
| `hallucination_rate` | LLM-as-judge | ≤ 10% |
| `grounding_score` | LLM-as-judge + rules | ≥ 80% |
| `policy_compliance` | Rule-based + classifier | ≥ 98% |
| `adversarial_resistance` | OWASP ASI01–ASI10 | ≥ 85% |
| `multi_agent_coordination` | Trace analysis | ≥ 85% |
| `trace_reproducibility` | Replay comparison | = 100% |
| `runtime_decision_quality` | Human-labeled paths | ≥ 75% |
| `regression_drift` | Computed | configurable |
| `reliability_score` | Composite weighted | ≥ 80% |

Scorers are pluggable. Add a scorer by implementing `ScorerBase` in `evals/scorers/` and registering it in the scorer service config.

---

## Regression gate

The gate decision is computed by the orchestrator after all scorers complete. The logic is intentionally simple:

```
for each dimension in suite config:
    delta = current_score - baseline_score
    if direction == "higher_is_better" and delta < -drift_limit:
        BLOCK
    if direction == "lower_is_better" and delta > drift_limit:
        BLOCK
    if block_on_threshold_breach and current_score < threshold:
        BLOCK
```

Baselines are stored per suite per commit ref. The `baseline_ref` field in the suite config controls which commit is used as the comparison target. Default: `main~1`.

### Suite config

```yaml
# evals/suites/agent-v2.yaml

suite: agent-v2
baseline_ref: main~1
block_on_regression: true

dimensions:
  tool_use_correctness:
    threshold: 0.90
    drift_limit: 0.03
    direction: higher_is_better

  hallucination_rate:
    threshold: 0.10
    drift_limit: 0.03
    direction: lower_is_better
    block_on_threshold_breach: true

  policy_compliance:
    threshold: 0.98
    drift_limit: 0.01
    direction: higher_is_better
    block_on_threshold_breach: true

adversarial:
  enabled: true
  injection_rate: 0.15   # 15% of cases are adversarial
  owasp_coverage: [ASI01, ASI02, ASI03, ASI04, ASI05, ASI06, ASI07, ASI08, ASI09, ASI10]

notifications:
  slack_channel: "#ai-reliability"
  post_trace_link: true
```

---

## Trace replay

Every eval run produces a full OpenTelemetry trace stored in object storage. Leo can replay any trace deterministically — replaying the exact sequence of tool calls with mocked external responses to reproduce or diff a failure.

```bash
# Replay a trace to reproduce a failure
leo trace replay trc_01HZXQP84JBV6K3NDFQ7W2MRC8

# Diff two traces — regression vs baseline
leo trace diff trc_01HZXQ... trc_01HZWP... --output json

# Re-score stored trace with updated scorers (does not re-run the agent)
leo trace rescore trc_01HZXQ... --scorers hallucination_rate,grounding_score

# Export to OTLP for external analysis
leo trace export trc_01HZXQ... --format otlp --out ./traces/
```

Replay is deterministic: a given trace ID always produces the same replay output. This matters for two reasons. First, it lets you debug regressions without re-running the agent. Second, it lets you update scorer logic and verify scores change as expected on historical traces before shipping the update.

---

## Adversarial testing

The adversarial engine generates attack cases covering the OWASP Top 10 for AI Systems (ASI01–ASI10):

| Attack class | OWASP ref | Example |
|---|---|---|
| Prompt injection | ASI01 | Injected instructions in user input |
| Insecure output | ASI02 | Agent returns sensitive data from tool results |
| Training data poisoning | ASI03 | Dataset contamination probes |
| Model denial of service | ASI04 | Pathological input sequences |
| Supply chain attacks | ASI05 | Tampered tool result injection |
| Sensitive info disclosure | ASI06 | Exfiltration via crafted queries |
| Insecure plugin design | ASI07 | Tool boundary escape attempts |
| Excessive agency | ASI08 | Scope creep via ambiguous instructions |
| Overreliance | ASI09 | Confidence manipulation attacks |
| Model theft | ASI10 | Extraction probes |

Attack cases are versioned in the dataset service. New patterns are contributed via the catalog format in `services/adversarial/internal/catalog/`.

---

## CI/CD integration

### GitHub Actions

```yaml
# .github/workflows/leo-eval.yaml
name: Leo eval
on:
  pull_request:
    branches: [main]

jobs:
  eval:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run Leo eval suite
        uses: leo-platform/leo-action@v1
        with:
          suite: evals/suites/agent-v2.yaml
          leo_url: ${{ secrets.LEO_URL }}
          leo_token: ${{ secrets.LEO_TOKEN }}

      - name: Post eval summary to PR
        if: always()
        uses: leo-platform/leo-action/comment@v1
```

The GitHub Action posts a structured eval summary to the PR as a GitHub Check, including per-dimension scores, delta from baseline, and a replay link for any regressed case.

### Gate behavior

| Condition | Gate state | PR state |
|---|---|---|
| All dimensions pass | OPEN | Mergeable |
| Any dimension regresses | CLOSED | Blocked |
| Scorer service unavailable | OPEN (configurable) | Mergeable with warning |
| Eval run times out | CLOSED | Blocked |

The fail-open/fail-closed behavior on infrastructure failures is configurable per suite. Default: fail-closed (block on uncertainty).

---

## Instrument your agent

```python
from leo_sdk import LeoTracer, tool_span
from opentelemetry import trace

# Initialise once at startup. Uses environment config if args omitted.
LeoTracer().init(
    service_name="agent-v2",
    leo_endpoint="http://leo.internal:4317",
)

async def my_agent(query: str) -> str:
    tracer = trace.get_tracer(__name__)

    with tracer.start_as_current_span("agent.run") as span:
        span.set_attribute("leo.query", query)

        # tool_span captures args, return value, latency, and errors
        with tool_span("web_search", args={"q": query}) as ts:
            result = await web_search(query)
            ts.set_output(result)

        response = await llm_call(query, context=result)
        span.set_attribute("leo.response", response)
        return response
```

The SDK follows OpenInference semantic conventions. All spans are compatible with Langfuse, Arize, and any OTLP-compatible backend.

---

## Local development

**Prerequisites:** Docker 24+, Go 1.22+, Python 3.11+, Node 20+

```bash
git clone https://github.com/your-org/leo.git
cd leo

# Copy and edit the local config
cp config/leo.example.yaml config/leo.local.yaml

# Start the full stack
docker compose -f infra/compose/docker-compose.dev.yaml up -d

# Verify all services are healthy
leo health

# Run the example eval
leo eval run --suite evals/suites/example.yaml --agent examples/agent.py
```

The dev compose stack includes Postgres, MinIO (S3-compatible), Prometheus, Grafana, Loki, and Tempo. All observability is wired by default in local dev.

### Build services individually

```bash
# Orchestrator
cd services/orchestrator && go build ./cmd/orchestrator

# Scorer  
cd services/scorer && pip install -e . && python -m scorer.cmd.serve

# Dashboard
cd services/dashboard && npm install && npm run dev
```

---

## Kubernetes deployment

```bash
# Add the Helm chart repo
helm repo add leo https://charts.leo-platform.io
helm repo update

# Install with production values
helm install leo leo/leo \
  --namespace leo-system \
  --create-namespace \
  -f infra/helm/values.prod.yaml \
  --set orchestrator.replicas=3 \
  --set scorer.replicas=5 \
  --set storage.postgres.host=your-pg-host \
  --set storage.s3.bucket=your-traces-bucket

# Check rollout
kubectl -n leo-system rollout status deployment/leo-orchestrator
```

See `infra/helm/` for full values documentation.

---

## Observability

Leo instruments itself with the same OpenTelemetry stack it uses to evaluate agents.

| Signal | Backend | Key metrics |
|---|---|---|
| Traces | Tempo | Eval run duration, scorer latency per dimension |
| Metrics | Prometheus + Grafana | Cases/sec, queue depth, scorer error rate, gate decisions |
| Logs | Loki | Structured JSON, always carries `run_id`, `trace_id`, `service` |

Grafana dashboards are pre-built and ship with the Helm chart. See `infra/grafana/dashboards/`.

---

## Repository layout

```
leo/
├── services/
│   ├── orchestrator/           # Go. Eval run lifecycle and gate decisions.
│   │   ├── cmd/                # Entrypoint
│   │   ├── internal/
│   │   │   ├── config/         # Runtime config, validation
│   │   │   ├── eval/           # Run scheduling, worker pool
│   │   │   ├── gate/           # Regression gate logic
│   │   │   ├── metrics/        # Prometheus instrumentation
│   │   │   └── server/         # gRPC + HTTP handlers
│   │   └── api/proto/          # Protobuf definitions
│   ├── scorer/                 # Python. 10 eval dimensions.
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── config/
│   │       ├── scorers/        # One file per dimension
│   │       ├── judge/          # LLM-as-judge client
│   │       └── metrics/
│   ├── replay/                 # Go + Python. Trace replay and diffing.
│   ├── adversarial/            # Python. OWASP ASI attack generation.
│   ├── datasets/               # Go. Versioned test case storage.
│   └── dashboard/              # TypeScript/Next.js. Read-only UI.
├── sdk/
│   ├── python/leo_sdk/         # Python instrumentation SDK
│   └── go/leo/                 # Go instrumentation SDK
├── evals/
│   ├── suites/                 # Suite configs (YAML)
│   ├── scorers/                # Custom scorer implementations
│   └── datasets/               # Test case datasets
├── infra/
│   ├── helm/leo/               # Helm chart
│   ├── compose/                # Docker Compose (dev + staging)
│   ├── terraform/              # IaC for cloud infra
│   └── grafana/dashboards/     # Pre-built Grafana dashboards
├── docs/
│   ├── architecture/           # Architecture decision records
│   ├── operations/             # Runbooks, on-call guide
│   └── runbooks/               # Incident response
├── policies/                   # Content policy configs
└── .github/workflows/          # CI definitions
```

---

## Security model

**Authentication:** All inter-service calls use mTLS. External API access requires bearer tokens issued by the orchestrator. No service accepts unauthenticated requests in production.

**Secrets:** No secrets in config files. All credentials are injected via environment variables or Kubernetes secrets. The Helm chart ships with a `secretRef` pattern — you provide the secret, Leo consumes it.

**Trace data:** Traces may contain sensitive user data. Leo stores traces with a configurable retention window (default: 90 days). Traces are encrypted at rest in S3. Per-trace access control is on the roadmap.

**Adversarial cases:** Attack prompts are stored encrypted. The adversarial engine runs in an isolated Pod with no outbound network access in production Kubernetes deployments.

---

## Failure handling

| Failure | Behavior |
|---|---|
| Scorer service down | Eval run fails, gate defaults to block (configurable) |
| LLM judge rate limit | Exponential backoff, 3 retries, then fall back to deterministic scorer |
| Agent timeout during eval | Case marked as error, counted against reliability score |
| Object storage unavailable | Traces buffered to local disk, flushed on recovery |
| Postgres unavailable | Orchestrator holds results in memory, retries for up to 5 minutes |
| Partial scorer failure | Completed dimensions reported, missing dimensions treated as block |

---

## Scaling

The orchestrator and scorer are horizontally scalable. A single orchestrator instance manages a shared job queue (Postgres-backed). Scorer workers pull from the queue; adding scorer replicas linearly increases throughput.

At 100 concurrent eval cases with 5 scorer replicas, a typical eval run completes in under 3 minutes. At 1000 cases with 20 scorer replicas, under 15 minutes. The bottleneck at scale is LLM judge throughput, not Leo infrastructure.

For very large suites, Leo supports sampling strategies that give statistically valid estimates from a subset of cases. See `datasets.sampling` in the suite config.

---

## Contributing

See `docs/architecture/` for Architecture Decision Records covering the major design choices. Pull requests should include:

- Eval coverage for new scorer dimensions
- Updated suite config documentation
- A trace replay test demonstrating the change

---

## License

Apache 2.0. See LICENSE.
