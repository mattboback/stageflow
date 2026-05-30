-- Jobs table tracks all scan jobs.
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
    error_details TEXT,
    last_stage TEXT,
    total_pages INTEGER DEFAULT 0,
    current_page INTEGER DEFAULT 0,
    total_violations INTEGER DEFAULT 0,
    report_json_key TEXT,
    report_key TEXT,
    scan_stage_log_key TEXT,
    scan_recipe_key TEXT,
    extraction_stage_log_key TEXT,
    extraction_recipe_key TEXT,
    provenance_path TEXT,
    provenance_key TEXT,
    expected_scanners TEXT,
    completed_scanners TEXT,
    scanner_results TEXT,
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

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS error_details TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS last_stage TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS total_pages INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS current_page INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS total_violations INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS report_json_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS report_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scan_stage_log_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scan_recipe_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS extraction_stage_log_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS extraction_recipe_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS extraction_started_at TIMESTAMP;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS extraction_completed_at TIMESTAMP;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scan_started_at TIMESTAMP;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scan_completed_at TIMESTAMP;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS pages_scanned INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS total_issues INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS critical_issues INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS serious_issues INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS moderate_issues INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS minor_issues INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS provenance_path TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS provenance_key TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS expected_scanners TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS completed_scanners TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scanner_results TEXT;

-- Job events table logs all events for a job.
CREATE TABLE IF NOT EXISTS job_events (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT,
    payload_job_id TEXT,
    event TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload_json TEXT,
    request_id TEXT,
    run_id TEXT,
    producer TEXT,
    nats_subject TEXT,
    nats_stream TEXT,
    nats_consumer TEXT,
    nats_stream_seq BIGINT,
    nats_consumer_seq BIGINT,
    nats_deliveries BIGINT,
    nats_stored_at TIMESTAMP,
    handler_status TEXT,
    handler_error TEXT,
    duration_ms BIGINT,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

ALTER TABLE job_events ADD COLUMN IF NOT EXISTS request_id TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS run_id TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS producer TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_subject TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_stream TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_consumer TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_stream_seq BIGINT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_consumer_seq BIGINT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_deliveries BIGINT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS nats_stored_at TIMESTAMP;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS handler_status TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS handler_error TEXT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS duration_ms BIGINT;
ALTER TABLE job_events ADD COLUMN IF NOT EXISTS payload_job_id TEXT;

-- Terminal event outbox stores job.completed/job.failed until they are
-- successfully published to NATS.
CREATE TABLE IF NOT EXISTS terminal_events (
    job_id TEXT NOT NULL,
    event TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    PRIMARY KEY (job_id, event),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_completed_at ON jobs(completed_at);
CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events(job_id);
CREATE INDEX IF NOT EXISTS idx_job_events_job_id_timestamp ON job_events(job_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_job_events_timestamp ON job_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_terminal_events_unpublished ON terminal_events(published_at, created_at);
