-- Jobs table tracks all scan jobs
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    input_type TEXT NOT NULL,
    input_path TEXT NOT NULL,
    urls TEXT,
    pod_id TEXT,
    config_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT,
    provenance_path TEXT,
    provenance_key TEXT,
    expected_scanners TEXT,
    completed_scanners TEXT,
    scanner_results TEXT,
    -- Metrics columns
    extraction_started_at TIMESTAMP,
    extraction_completed_at TIMESTAMP,
    scan_started_at TIMESTAMP,
    scan_completed_at TIMESTAMP,
    pages_scanned INTEGER DEFAULT 0,
    total_issues INTEGER DEFAULT 0,
    critical_issues INTEGER DEFAULT 0,
    serious_issues INTEGER DEFAULT 0,
    moderate_issues INTEGER DEFAULT 0,
    minor_issues INTEGER DEFAULT 0
);

-- Job events table logs all events for a job
CREATE TABLE IF NOT EXISTS job_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    event TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload_json TEXT,
    request_id TEXT,
    run_id TEXT,
    producer TEXT,
    nats_subject TEXT,
    nats_stream TEXT,
    nats_consumer TEXT,
    nats_stream_seq INTEGER,
    nats_consumer_seq INTEGER,
    nats_deliveries INTEGER,
    nats_stored_at TIMESTAMP,
    handler_status TEXT,
    handler_error TEXT,
    duration_ms INTEGER,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

-- Index for faster job lookups by state
CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);

-- Index for faster job lookups by created_at
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);

-- Index for faster event lookups by job_id
CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events(job_id);

-- Index for faster timelines per job
CREATE INDEX IF NOT EXISTS idx_job_events_job_id_timestamp ON job_events(job_id, timestamp);

-- Index for pruning old events efficiently
CREATE INDEX IF NOT EXISTS idx_job_events_timestamp ON job_events(timestamp);

-- Index for metrics queries
CREATE INDEX IF NOT EXISTS idx_jobs_completed_at ON jobs(completed_at);
