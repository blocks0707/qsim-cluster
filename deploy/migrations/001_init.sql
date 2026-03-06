-- Initial schema for qsim-cluster
CREATE TABLE IF NOT EXISTS quantum_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    code TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'python',
    qubits INTEGER,
    depth INTEGER,
    gate_count INTEGER,
    complexity_class TEXT,
    method TEXT,
    priority TEXT NOT NULL DEFAULT 'normal',
    assigned_node TEXT,
    assigned_pool TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    execution_time_ms BIGINT,
    result_ref TEXT,
    error_message TEXT DEFAULT '',
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON quantum_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON quantum_jobs(status);
