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

-- Scanner launch ownership is persisted separately from scanner results so a
-- process restart can distinguish work that was never claimed, work that may
-- already have created a deterministic Podman container, and work that was
-- fully launched. The row survives until the parent job is deleted.
CREATE TABLE IF NOT EXISTS scanner_launches (
    job_id TEXT NOT NULL,
    scanner_type TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    container_id TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    claimed_at TIMESTAMP,
    launched_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (job_id, scanner_type),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CHECK (state IN ('pending', 'launching', 'launched', 'failed'))
);

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

-- One-time, idempotent data migrations that require application-level JSON
-- sanitization rather than a static SQL ALTER statement.
CREATE TABLE IF NOT EXISTS maintenance_migrations (
    name TEXT PRIMARY KEY,
    completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_completed_at ON jobs(completed_at);
CREATE INDEX IF NOT EXISTS idx_terminal_events_unpublished ON terminal_events(published_at, created_at);
CREATE INDEX IF NOT EXISTS idx_scanner_launches_state ON scanner_launches(state, updated_at);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scanner_progress TEXT NOT NULL DEFAULT '{}';
