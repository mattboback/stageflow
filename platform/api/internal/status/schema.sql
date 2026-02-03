CREATE TABLE IF NOT EXISTS job_status (
    job_id TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    input_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    error TEXT,
    total_pages INTEGER,
    current_page INTEGER,
    total_violations INTEGER DEFAULT 0,
    report_json_key TEXT,
    report_key TEXT,
    scan_stage_log_key TEXT,
    scan_recipe_key TEXT,
    extraction_stage_log_key TEXT,
    extraction_recipe_key TEXT,
    provenance_key TEXT,
    last_stage TEXT,
    last_error_details TEXT,
    scanner_artifacts TEXT
);

CREATE INDEX IF NOT EXISTS idx_job_status_state ON job_status(state);
CREATE INDEX IF NOT EXISTS idx_job_status_created_at ON job_status(created_at);
CREATE INDEX IF NOT EXISTS idx_job_status_updated_at ON job_status(updated_at);
