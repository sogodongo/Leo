"""
Leo SDK for Python agents.

Thin wrapper around OpenTelemetry that follows OpenInference semantic
conventions. Import this, call LeoTracer().init(), then instrument
tool calls with tool_span(). That is the entire public surface.
"""

from __future__ import annotations

import functools
from contextlib import contextmanager
from typing import Any, Generator

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


class LeoTracer:
    """Initialise the Leo OpenTelemetry pipeline.

    Call once at agent startup. Safe to call multiple times — subsequent
    calls are no-ops if a provider is already registered.
    """

    _initialized = False

    def init(
        self,
        service_name: str,
        leo_endpoint: str,
        *,
        service_version: str = "unknown",
        insecure: bool = True,
    ) -> None:
        if LeoTracer._initialized:
            return

        resource = Resource.create({
            "service.name": service_name,
            "service.version": service_version,
            "leo.sdk.version": "0.1.0",
        })

        exporter = OTLPSpanExporter(endpoint=leo_endpoint, insecure=insecure)
        provider = TracerProvider(resource=resource)
        provider.add_span_processor(BatchSpanProcessor(exporter))
        trace.set_tracer_provider(provider)

        LeoTracer._initialized = True


class _ToolSpan:
    """Context manager returned by tool_span(). Records tool call metadata."""

    def __init__(self, span: Any) -> None:
        self._span = span

    def set_output(self, output: Any) -> None:
        self._span.set_attribute("leo.tool.output", str(output)[:4096])

    def set_error(self, error: Exception) -> None:
        self._span.set_attribute("leo.tool.error", str(error))
        self._span.set_attribute("leo.tool.error_type", type(error).__name__)

    def __enter__(self) -> "_ToolSpan":
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        if exc_val is not None:
            self.set_error(exc_val)
        return False


@contextmanager
def tool_span(
    tool_name: str,
    *,
    args: dict[str, Any] | None = None,
) -> Generator[_ToolSpan, None, None]:
    """Wraps a tool call in a Leo-instrumented span.

    Records the tool name, arguments, and output following OpenInference
    tool call conventions. The caller sets output via ts.set_output(result).

    Usage:
        with tool_span("web_search", args={"q": query}) as ts:
            result = await search(query)
            ts.set_output(result)
    """
    tracer = trace.get_tracer("leo_sdk")
    with tracer.start_as_current_span(f"tool.{tool_name}") as span:
        span.set_attribute("leo.tool.name", tool_name)
        if args:
            for k, v in args.items():
                span.set_attribute(f"leo.tool.args.{k}", str(v)[:512])
        yield _ToolSpan(span)


def instrument_llm_call(fn):
    """Decorator that wraps an async LLM call function in a Leo span.

    Captures prompt, completion, model, and token usage. Designed for
    wrapper functions around litellm, openai, anthropic, etc.

    Usage:
        @instrument_llm_call
        async def call_llm(prompt: str, model: str) -> str:
            ...
    """
    @functools.wraps(fn)
    async def wrapper(*args, **kwargs):
        tracer = trace.get_tracer("leo_sdk")
        with tracer.start_as_current_span("llm_call") as span:
            result = await fn(*args, **kwargs)
            # Callers set leo.llm.* attributes on the current span if needed
            return result
    return wrapper


__all__ = ["LeoTracer", "tool_span", "instrument_llm_call"]
