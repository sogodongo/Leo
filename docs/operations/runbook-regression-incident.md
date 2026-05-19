# Runbook: regression gate fired

**Trigger:** Leo closed the deployment gate on a PR. The GitHub Check
shows one or more eval dimensions regressed beyond their drift threshold.

## 1. Identify the regressed dimension

The PR comment lists every dimension with current score, baseline, and delta.
If missing, fetch directly:

    leo run get <run_id> --format table

## 2. Replay the regression trace

    leo trace replay <trace_id>

Replays the agent against the exact inputs and mocked tool results
from the original run. No live API calls.

## 3. Diff against baseline

    leo trace diff <regression_trace_id> <baseline_trace_id>

Look for: changed tool results, different LLM output, new spans.

## 4. Real vs environmental regression

Real: agent code or prompt changed. Fix and push — gate re-evaluates.

Environmental: dataset version changed, judge model updated, baseline stale.

    leo run get <run_id> --show-dataset-ref
    leo scorer versions --dimension hallucination_rate
    leo eval run --suite evals/suites/agent-v2.yaml --baseline-ref <sha>

## 5. Gate override (use sparingly)

    leo gate override <run_id> --reason "environmental — fix tracked in #1234"

Overrides are logged with reason and approver identity.

## 6. Escalation

If replay does not reproduce the failure, page AI reliability on-call.
Attach: run ID, trace ID, diff output, last 3 passing run IDs.
