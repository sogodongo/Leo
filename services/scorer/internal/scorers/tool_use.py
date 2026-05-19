from __future__ import annotations
import re
from typing import Any
import structlog
from .base import ScorerBase, ScorerInput, ScorerOutput

log = structlog.get_logger(__name__)

class ToolUseCorrectnessScorer(ScorerBase):
    dimension = "tool_use_correctness"
    version = "1.0.0"

    def score(self, inp: ScorerInput) -> ScorerOutput:
        expected = inp.dataset_case.get("expected_tool_calls", [])
        if not expected:
            return ScorerOutput(dimension=self.dimension, score=1.0, pass_=True, reason="no expectations")

        actual = inp.trace.tool_calls
        checks = []
        p = self._check_presence(expected, actual, checks)
        a = self._check_arguments(expected, actual, checks)
        o = self._check_ordering(expected, actual, checks)
        score = (p * 0.5) + (a * 0.35) + (o * 0.15)
        failed = [c for c in checks if not c["pass"]]
        return ScorerOutput(
            dimension=self.dimension, score=round(score, 4),
            pass_=not failed,
            reason="; ".join(c["reason"] for c in failed) if failed else "all checks passed",
            metadata={"checks": checks},
        )

    def _check_presence(self, expected, actual, checks):
        actual_names = {tc.name for tc in actual}
        passed = 0
        for exp in expected:
            ok = exp["name"] in actual_names
            checks.append({"type": "presence", "tool": exp["name"], "pass": ok,
                           "reason": f"tool '{exp['name']}' {'called' if ok else 'not called'}"})
            if ok: passed += 1
        return passed / len(expected) if expected else 1.0

    def _check_arguments(self, expected, actual, checks):
        by_name = {}
        for tc in actual:
            by_name.setdefault(tc.name, []).append(tc)
        passed = total = 0
        for exp in expected:
            req = exp.get("required_args", {})
            if not req: continue
            calls = by_name.get(exp["name"], [])
            ok = any(self._args_match(req, c.args) for c in calls)
            total += 1
            checks.append({"type": "arguments", "tool": exp["name"], "pass": ok,
                           "reason": f"'{exp['name']}' {'correct args' if ok else 'wrong args'}"})
            if ok: passed += 1
        return passed / total if total else 1.0

    def _check_ordering(self, expected, actual, checks):
        ordered = [e for e in expected if e.get("must_precede")]
        if not ordered: return 1.0
        names = [tc.name for tc in actual]
        passed = 0
        for exp in ordered:
            a_idx = next((i for i, n in enumerate(names) if n == exp["name"]), -1)
            b_idx = next((i for i, n in enumerate(names) if n == exp["must_precede"]), -1)
            ok = a_idx != -1 and b_idx != -1 and a_idx < b_idx
            checks.append({"type": "ordering", "tool": exp["name"], "pass": ok,
                           "reason": f"order {'ok' if ok else 'wrong'}"})
            if ok: passed += 1
        return passed / len(ordered)

    def _args_match(self, required, actual):
        for k, v in required.items():
            if k not in actual: return False
            if isinstance(v, str) and v.startswith("regex:"):
                if not re.search(v[6:], str(actual[k])): return False
            elif str(v) != str(actual[k]): return False
        return True
