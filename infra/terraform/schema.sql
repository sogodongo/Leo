-- Leo database schema. Append-only design — no updates after insert.

CREATE TABLE IF NOT EXISTS eval_runs (
    id              TEXT PRIMARY KEY,
    suite_ref       TEXT NOT NULL,
    commit_sha      TEXT NOT NULL,
    baseline_ref    TEXT NOT NULL,
    pr_number       INTEGER,
    started_at      TIMESTAMPTZ NOT NULL,
    completed_at    TIMESTAMPTZ,
    total_cases     INTEGER NOT NULL DEFAULT 0,
    completed_cases INTEGER NOT NULL DEFAULT 0,
    gate_state      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eval_runs_suite_ref    ON eval_runs (suite_ref);
CREATE INDEX IF NOT EXISTS idx_eval_runs_completed_at ON eval_runs (completed_at DESC);

CREATE TABLE IF NOT EXISTS dimension_scores (
    id             BIGSERIAL PRIMARY KEY,
    run_id         TEXT NOT NULL REFERENCES eval_runs (id),
    dimension      TEXT NOT NULL,
    score          NUMERIC(6,4) NOT NULL,
    pass_count     INTEGER NOT NULL DEFAULT 0,
    fail_count     INTEGER NOT NULL DEFAULT 0,
    error_count    INTEGER NOT NULL DEFAULT 0,
    scorer_version TEXT NOT NULL,
    baseline_score NUMERIC(6,4),
    delta          NUMERIC(6,4),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (run_id, dimension)
);

CREATE INDEX IF NOT EXISTS idx_dimension_scores_dimension ON dimension_scores (dimension, created_at DESC);

CREATE TABLE IF NOT EXISTS case_results (
    id          BIGSERIAL PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES eval_runs (id),
    case_id     TEXT NOT NULL,
    trace_id    TEXT,
    dimension   TEXT NOT NULL,
    score       NUMERIC(6,4),
    pass        BOOLEAN,
    error_msg   TEXT,
    latency_ms  INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gate_decisions (
    id          BIGSERIAL PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES eval_runs (id),
    state       TEXT NOT NULL,
    violations  JSONB NOT NULL DEFAULT '[]',
    decided_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS baselines (
    id          BIGSERIAL PRIMARY KEY,
    suite_ref   TEXT NOT NULL,
    commit_ref  TEXT NOT NULL,
    run_id      TEXT NOT NULL REFERENCES eval_runs (id),
    scores      JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (suite_ref, commit_ref)
);
