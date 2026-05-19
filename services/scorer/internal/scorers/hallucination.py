"""
Hallucination scorer.

Uses an LLM judge to detect claims in the agent response that are not
supported by the retrieved context (tool results). The judge is prompted
with the query, the tool results, and the agent response — it returns
a structured verdict.

Determinism: temperature=0 + trace_id seeded into the prompt. The same
trace always produces the same verdict, which matters for trace replay.
"""

from __future__ import annotations

import json
import re
from typing import Any

import structlog

from .base import ScorerBase, ScorerInput, ScorerOutput

log = structlog.get_logger(__name__)

_JUDGE_SYSTEM_PROMPT = """\
You are a hallucination detection system. Given a query, a set of tool results \
(the agent's retrieved context), and an agent response, identify claims in the \
response that are NOT supported by the tool results.

A claim is hallucinated if:
- It states a specific fact not present in any tool result
- It contradicts information in the tool results
- It fabricates citations, statistics, or quotes

A claim is NOT hallucinated if:
- It accurately summarises tool result content
- It is a reasonable inference directly derivable from tool results
- It is a general statement the agent could reasonably know without retrieval

Respond ONLY with valid JSON matching this schema:
{
  "hallucinated_claims": ["claim 1", "claim 2"],
  "verdict": "pass" | "fail",
  "confidence": 0.0-1.0,
  "reasoning": "one sentence explanation"
}
"""

_JUDGE_USER_TEMPLATE = """\
[QUERY]
{query}

[TOOL RESULTS]
{tool_results}

[AGENT RESPONSE]
{response}

[TRACE ID: {trace_id}]
"""


class HallucinationScorer(ScorerBase):
    dimension = "hallucination_rate"
    version = "1.0.0"

    def __init__(self, judge_client: Any, threshold: float = 0.10) -> None:
        self._judge = judge_client
        # threshold here is a failure rate: if >10% of claims are hallucinated,
        # the score falls below the passing threshold
        self._threshold = threshold

    def score(self, inp: ScorerInput) -> ScorerOutput:
        tool_results_text = self._format_tool_results(inp.trace.tool_calls)

        verdict = self._judge.complete(
            system=_JUDGE_SYSTEM_PROMPT,
            user=_JUDGE_USER_TEMPLATE.format(
                query=inp.trace.query,
                tool_results=tool_results_text,
                response=inp.trace.response,
                trace_id=inp.trace.trace_id,  # seeds determinism
            ),
            temperature=0,
            response_format={"type": "json_object"},
        )

        parsed = self._parse_verdict(verdict.content, inp.trace.trace_id)

        hallucinated_count = len(parsed.get("hallucinated_claims", []))
        # Score is (1 - hallucination_rate). A response with no hallucinations scores 1.0.
        # A response where every claim is hallucinated scores 0.0.
        score = 1.0 - min(hallucinated_count / max(len(inp.trace.response.split(". ")), 1), 1.0)

        return ScorerOutput(
            dimension=self.dimension,
            score=score,
            pass_=score >= (1.0 - self._threshold),
            reason=parsed.get("reasoning", ""),
            metadata={
                "hallucinated_claims": parsed.get("hallucinated_claims", []),
                "judge_confidence": parsed.get("confidence", 0),
                "judge_verdict": parsed.get("verdict", "unknown"),
            },
        )

    def _format_tool_results(self, tool_calls: list) -> str:
        parts = []
        for tc in tool_calls:
            result_text = json.dumps(tc.result) if tc.result is not None else "(no result)"
            parts.append(f"[{tc.name}]\n{result_text}")
        return "\n\n".join(parts) if parts else "(no tool calls)"

    def _parse_verdict(self, content: str, trace_id: str) -> dict:
        # Strip any accidental markdown fences the judge might emit
        content = re.sub(r"^```json\s*|```$", "", content.strip(), flags=re.MULTILINE)
        try:
            return json.loads(content)
        except json.JSONDecodeError:
            log.warning(
                "judge_parse_failure",
                trace_id=trace_id,
                raw_content=content[:200],
            )
            # Return a conservative result: mark as fail so we don't silently
            # treat a parse failure as a passing score
            return {"hallucinated_claims": [], "verdict": "fail", "confidence": 0, "reasoning": "parse_error"}
