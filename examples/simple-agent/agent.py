"""
Example agent instrumented with the Leo SDK.

    docker compose -f infra/compose/docker-compose.dev.yaml up -d
    pip install -e sdk/python/
    python examples/simple-agent/agent.py
"""
import asyncio, os
from opentelemetry import trace
from leo_sdk import LeoTracer, tool_span

LeoTracer().init(
    service_name="simple-agent",
    leo_endpoint=os.getenv("LEO_ENDPOINT", "http://localhost:4317"),
)

async def web_search(query: str) -> str:
    return f"Search results for: {query}"

async def run_agent(query: str) -> str:
    tracer = trace.get_tracer(__name__)
    with tracer.start_as_current_span("agent.run") as span:
        span.set_attribute("leo.query", query)
        with tool_span("web_search", args={"q": query}) as ts:
            results = await web_search(query)
            ts.set_output(results)
        response = f"Based on search: {results[:100]}"
        span.set_attribute("leo.response", response)
        return response

async def main():
    queries = [
        "What is the capital of Kenya?",
        "Latest developments in AI safety",
        "How does retrieval-augmented generation work?",
    ]
    for q in queries:
        print(f"\nQuery: {q}")
        print(f"Response: {await run_agent(q)}")

if __name__ == "__main__":
    asyncio.run(main())
