from __future__ import annotations
import json, re
from typing import Any
import structlog
from .base import ScorerBase, ScorerInput, ScorerOutput

log = structlog.get_logger(__name__)

_SYS = """You are a grounding evaluator. Rate how well the agent response is grounded in the tool results.
1.0=every claim supported, 0.0=no connection to evidence.
Respond ONLY with JSON: {"grounding_score":0.0-1.0,"supported_claims":[],"unsupported_claims":[],"reasoning":"one sentence"}"""

class GroundingScorer(ScorerBase):
    dimension = "grounding_score"
    version = "1.0.0"

    def __init__(self, judge_client: Any, threshold: float = 0.80):
        self._judge = judge_client
        self._threshold = threshold

    def score(self, inp: ScorerInput) -> ScorerOutput:
        tool_results = " ".join(str(tc.result) for tc in inp.trace.tool_calls if tc.result and not tc.error)
        if not tool_results:
            return ScorerOutput(dimension=self.dimension, score=1.0, pass_=True, reason="no tool calls")

        overlap = self._overlap(inp.trace.response, tool_results)
        verdict = self._judge_call(inp.trace.response, tool_results, inp.trace.trace_id)
        judge_score = verdict.get("grounding_score", 0.0)
        blended = round((judge_score * 0.75) + (overlap * 0.25), 4)

        return ScorerOutput(
            dimension=self.dimension, score=blended, pass_=blended >= self._threshold,
            reason=verdict.get("reasoning", ""),
            metadata={"judge_score": judge_score, "overlap_score": overlap,
                      "unsupported": verdict.get("unsupported_claims", [])},
        )

    def _overlap(self, response, context):
        rt = set(re.findall(r'\b[a-z]{3,}\b', response.lower()))
        ct = set(re.findall(r'\b[a-z]{3,}\b', context.lower()))
        return len(rt & ct) / len(rt) if rt else 0.0

    def _judge_call(self, response, tool_results, trace_id):
        v = self._judge.complete(
            system=_SYS,
            user=f"[TOOL RESULTS]\n{tool_results[:8000]}\n\n[RESPONSE]\n{response}\n\n[TRACE ID: {trace_id}]",
            temperature=0, response_format={"type": "json_object"},
        )
        try:
            return json.loads(re.sub(r"^```json\s*|```$", "", v.content.strip(), flags=re.MULTILINE))
        except Exception:
            log.warning("grounding_parse_failure", trace_id=trace_id)
            return {"grounding_score": 0.0, "reasoning": "parse_error"}
