from __future__ import annotations
import abc, time
from dataclasses import dataclass, field
from typing import Any
import structlog

log = structlog.get_logger(__name__)

@dataclass
class Trace:
    trace_id: str
    run_id: str
    case_id: str
    query: str
    response: str
    tool_calls: list
    llm_calls: list
    metadata: dict[str, Any] = field(default_factory=dict)

@dataclass
class ToolCall:
    name: str
    args: dict[str, Any]
    result: Any
    error: str | None = None
    latency_ms: int = 0

@dataclass
class ScorerInput:
    trace: Trace
    dataset_case: dict[str, Any]
    dimension: str

@dataclass
class ScorerOutput:
    dimension: str
    score: float
    pass_: bool
    reason: str = ""
    scorer_version: str = "unknown"
    latency_ms: int = 0
    metadata: dict[str, Any] = field(default_factory=dict)

class ScorerBase(abc.ABC):
    @property
    @abc.abstractmethod
    def dimension(self) -> str: ...
    @property
    @abc.abstractmethod
    def version(self) -> str: ...
    @abc.abstractmethod
    def score(self, inp: ScorerInput) -> ScorerOutput: ...
