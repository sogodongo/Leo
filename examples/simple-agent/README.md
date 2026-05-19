# Simple agent example

Minimal agent instrumented with the Leo SDK.

## Run

```bash
docker compose -f ../../infra/compose/docker-compose.dev.yaml up -d
pip install -e ../../sdk/python/
python agent.py
```

Traces appear at http://localhost:3000.

## Eval

```bash
leo eval run \
  --suite ../../evals/suites/agent-v2.yaml \
  --agent agent.py
```
